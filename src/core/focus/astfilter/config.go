// config.go — carga dinámica del catálogo de palabras clave del AST
// (Item 1 del pedido de refactor: "Extensión del AST para Múltiples
// Palabras Clave y Lenguajes"). Reemplaza el mapa Go hardcodeado que
// este paquete tenía antes (ver languages.go, ahora reducido a un
// registro de gramáticas gotreesitter por extensión) por archivos
// JSON leídos en tiempo de ejecución desde dos lugares:
//
//  1. Los JSON EMBEBIDOS en default_config/ (vía go:embed) — el
//     catálogo de fábrica para Go, Python, JavaScript/TypeScript,
//     Java, C#, C/C++, PHP, Ruby y Rust. Siempre están disponibles,
//     incluso si la instalación no tiene una carpeta config/ast/ (por
//     ejemplo, un binario copiado suelto sin el repo al lado).
//  2. Los JSON en <root>/config/ast/keywords_<lenguaje>.json — donde
//     root es la raíz del proyecto Mova (la misma que usa
//     mova.local/i18n para config/lang/, ver Init de ese paquete).
//     Cualquier archivo encontrado ahí REEMPLAZA por completo el
//     catálogo del lenguaje con ese nombre (si ya existía entre los
//     embebidos) o AGREGA un lenguaje nuevo (si el nombre no existía) —
//     sin tocar ni recompilar el código fuente. Ese es el requisito
//     central del Item 1: soportar un lenguaje nuevo agregando un
//     archivo JSON.
//
// El formato de cada archivo (ver docs/PROJECT.md § "Filtro AST por
// palabras clave" para ejemplos completos):
//
//	{
//	  "language": "go",
//	  "extensions": [".go"],
//	  "kinds": {
//	    "func":      [{"type": "function_declaration"}, {"type": "method_declaration"}],
//	    "struct":    [{"type": "type_spec", "requires_child_type": "struct_type"}],
//	    "interface": [{"type": "type_spec", "requires_child_type": "interface_type"}]
//	  }
//	}
//
// "requires_child_type" es opcional: sin él, CUALQUIER nodo del "type"
// dado cuenta. Con él, sólo cuenta si el nodo tiene al menos un hijo
// DIRECTO cuyo propio Type() sea exactamente ese valor — así se
// distinguen formas de gramática que un mismo node-type reutiliza para
// varias palabras clave (el "type_spec" de Go cubre struct/interface/
// alias de tipo; el "lexical_declaration" de JS/TS cubre const/let;
// las asignaciones de Ruby distinguen constante de variable por el
// tipo del nodo identificador a la izquierda). Cada mapeo fue
// verificado empíricamente contra la gramática real (parseando
// fragmentos de muestra con gotreesitter), no adivinado — ver el
// campo opcional "notes" en cada JSON para las limitaciones conocidas
// de cada lenguaje (casos donde el árbol no distingue lo suficiente,
// p. ej. C/C++/C# no separan "const" de una variable normal a un solo
// nivel de profundidad).
package astfilter

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed default_config/*.json
var embeddedConfig embed.FS

// NodeMatcher is one tree-sitter node-type rule for a Kind within a
// language — see this file's header for RequiresChildType's meaning.
type NodeMatcher struct {
	Type              string `json:"type"`
	RequiresChildType string `json:"requires_child_type,omitempty"`
}

// jsonLangConfig is the on-disk shape of keywords_<language>.json.
type jsonLangConfig struct {
	Language   string                   `json:"language"`
	Extensions []string                 `json:"extensions"`
	Kinds      map[string][]NodeMatcher `json:"kinds"`
	Notes      string                   `json:"notes,omitempty"`
}

// registry holds the fully resolved, currently-active configuration:
// one langSpec per file extension, ready for astfilter.go's
// findSymbols/AllNames/ParseTree to consult via languageFor.
type registry struct {
	mu    sync.RWMutex
	byExt map[string]langSpec
}

var reg = &registry{byExt: map[string]langSpec{}}

// Init loads config/ast/ under root (see this file's header) and
// (re)builds the active registry: embedded defaults first, then
// root/config/ast/*.json layered on top, one full-language-file
// override at a time. Safe to call more than once (e.g. hot-reload,
// tests) — later calls fully replace the previous registry rather than
// merging into it, so removing a project-level override JSON and
// calling Init again correctly reverts to the embedded default.
//
// Errors in individual files are collected and returned (joined) but
// do NOT stop other files from loading — one malformed
// keywords_klingon.json must not take down AST filtering for every
// other language. Called once at startup from cli/main.go, right next
// to i18n.Init(root) (see that call site's comment for why one call
// there covers the CLI, MCP and HTTP doors alike).
func Init(root string) error {
	fresh := &registry{byExt: map[string]langSpec{}}

	var errs []string
	loadFS(fresh, embeddedConfig, "default_config", &errs)

	userDir := filepath.Join(root, "config", "ast")
	if entries, err := os.ReadDir(userDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(userDir, e.Name()))
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", e.Name(), err))
				continue
			}
			if err := loadOne(fresh, e.Name(), data); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", e.Name(), err))
			}
		}
	}

	reg.mu.Lock()
	reg.byExt = fresh.byExt
	reg.mu.Unlock()

	if len(errs) > 0 {
		return fmt.Errorf("astfilter: %d config/ast file(s) failed to load: %s", len(errs), strings.Join(errs, "; "))
	}
	return nil
}

func loadFS(into *registry, fsys embed.FS, dir string, errs *[]string) {
	entries, err := embeddedConfig.ReadDir(dir)
	if err != nil {
		*errs = append(*errs, err.Error())
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := fsys.ReadFile(dir + "/" + e.Name())
		if err != nil {
			*errs = append(*errs, fmt.Sprintf("%s: %v", e.Name(), err))
			continue
		}
		if err := loadOne(into, e.Name(), data); err != nil {
			*errs = append(*errs, fmt.Sprintf("%s: %v", e.Name(), err))
		}
	}
}

// loadOne parses one keywords_<language>.json's bytes and registers
// its extensions in into.byExt, overwriting whatever was already
// registered for each of those extensions (this is what makes a
// project-level file a full override, not a merge).
func loadOne(into *registry, filename string, data []byte) error {
	var cfg jsonLangConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if cfg.Language == "" {
		return fmt.Errorf(`missing required "language" field`)
	}
	if len(cfg.Extensions) == 0 {
		return fmt.Errorf(`missing required "extensions" field`)
	}
	if len(cfg.Kinds) == 0 {
		return fmt.Errorf(`missing required "kinds" field (must map at least one kind, e.g. "func", to a list of node types)`)
	}

	spec, ok := languageGrammar(cfg.Language)
	if !ok {
		return fmt.Errorf("unknown language %q — no gotreesitter grammar registered for it (see languages.go's grammarByName)", cfg.Language)
	}
	spec.kinds = cfg.Kinds

	for _, ext := range cfg.Extensions {
		if ext == "" {
			continue
		}
		if ext[0] != '.' {
			ext = "." + ext
		}
		into.byExt[ext] = spec
	}
	return nil
}

// languageFor looks up a grammar by extension, accepting the ext with
// or without its leading dot (filepath.Ext already includes it, but
// callers building the table by hand — e.g. tests — may not).
func languageFor(ext string) (langSpec, bool) {
	if ext == "" {
		return langSpec{}, false
	}
	if ext[0] != '.' {
		ext = "." + ext
	}
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	spec, ok := reg.byExt[ext]
	return spec, ok
}
