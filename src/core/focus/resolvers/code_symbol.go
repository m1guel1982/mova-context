// code_symbol.go — Code Symbol Resolver.
//
// Heurística estructural (brace/indent matching) — no es un parser real,
// pero es suficiente y determinista para escopar una única declaración en
// los estilos de código habituales.
//
// Community (LIKE simple): además del match exacto de identificador
// (`func LoadTranslations(...)`), si nada calza exactamente se intenta una
// segunda pasada más tolerante — LikeContains sobre la línea completa de
// declaración — para no fallar por un typo menor o una mayúscula/acento
// distinta (ej.: "loadtranslations" sin camelCase). Sigue siendo búsqueda
// TEXTUAL: nunca infiere que un nombre distinto está relacionado si no
// aparece escrito — eso es "Like Semántico" (Premium, embeddings).
package resolvers

import (
	"regexp"
	"sort"
	"strings"

	"mova.local/core/focus"
)

// CodeSymbolResolver busca una función/clase/método/interfaz por nombre en
// todos los archivos del repo. Es un resolver "catch-all": no puede saber de
// antemano si el símbolo existe sin recorrer el repo, así que Match siempre
// es true para targets que no son ruta de archivo/directorio ya resuelta —
// la decisión real ocurre en Resolve, y si no encuentra nada devuelve
// focus.ErrNotFound para que el motor continúe con el siguiente resolver.
type CodeSymbolResolver struct{}

func NewCodeSymbolResolver() *CodeSymbolResolver { return &CodeSymbolResolver{} }

func (r *CodeSymbolResolver) Match(ctx focus.Context, target string) bool {
	return true // catch-all: la certeza real la da Resolve
}

func (r *CodeSymbolResolver) Resolve(ctx focus.Context, target string) ([]focus.ContextBlock, error) {
	exact := focus.IsExact(target)
	symbol := stripSymbolNotation(focus.StripExact(target))
	var candidates []string
	walkFiles(ctx, ctx.RepoPath, func(p string) { candidates = append(candidates, p) })
	sort.Strings(candidates)

	// Pasada 1: coincidencia exacta de identificador (regex de límite de
	// palabra) — mismo comportamiento de siempre, prioridad más alta.
	for _, p := range candidates {
		content := readFile(p)
		if content == "" {
			continue
		}
		if frag, ok := extractCodeSymbol(content, symbol); ok {
			return []focus.ContextBlock{{
				Source:  relOrBase(ctx.RepoPath, p),
				Kind:    "code-symbol",
				Content: frag,
			}}, nil
		}
	}

	// "=" (focus.IsExact): solo la pasada 1 cuenta. Si no calzó
	// exactamente, se reporta not found — nunca cae a la pasada 2 LIKE.
	if exact {
		return nil, focus.ErrNotFound
	}

	// Pasada 2 (LIKE simple, Community): si nada calzó exactamente,
	// se intenta encontrar una línea de declaración cuyo texto CONTENGA
	// el símbolo pedido de forma tolerante a mayúsculas/acentos — sin
	// inventar relaciones semánticas, solo texto más permisivo.
	for _, p := range candidates {
		content := readFile(p)
		if content == "" {
			continue
		}
		if frag, ok := extractCodeSymbolLike(content, symbol); ok {
			return []focus.ContextBlock{{
				Source:  relOrBase(ctx.RepoPath, p),
				Kind:    "code-symbol",
				Content: frag,
			}}, nil
		}
	}
	return nil, focus.ErrNotFound
}

// codeDeclPattern matches a function/class/method/interface declaration line
// for the requested symbol across common brace languages, plus Python.
func codeDeclPattern(symbol string) *regexp.Regexp {
	name := regexp.QuoteMeta(symbol)
	return regexp.MustCompile(
		`(?i)^\s*(func|def|class|interface|type|public|private|protected|static|export|async)?[\s\w<>\[\],.*]*\b` + name + `\b\s*[(<:{]`)
}

// declLinePattern reconoce líneas que TIENEN forma de declaración (para la
// pasada 2, LIKE), sin exigir que el nombre calce exactamente por límite
// de palabra — solo que la línea empiece con una palabra clave de
// declaración habitual.
var declLinePattern = regexp.MustCompile(`(?i)^\s*(func|def|class|interface|type|public|private|protected|static|export|async)\b`)

// extractCodeSymbol extracts a function/class/method body via brace or
// indentation matching.
func extractCodeSymbol(content, symbol string) (string, bool) {
	lines := strings.Split(content, "\n")
	decl := codeDeclPattern(symbol)
	for i, line := range lines {
		if !decl.MatchString(line) {
			continue
		}
		if strings.Contains(line, "{") {
			return extractBraceBlock(lines, i), true
		}
		if strings.TrimSpace(line) != "" {
			if frag := extractIndentBlock(lines, i); frag != "" {
				return frag, true // Python-style / brace-less header
			}
		}
	}
	return "", false
}

// extractCodeSymbolLike is the Community "LIKE simple" fallback pass: any
// declaration-shaped line whose text loosely contains symbol (accent- and
// case-insensitive) counts as a match.
func extractCodeSymbolLike(content, symbol string) (string, bool) {
	if strings.TrimSpace(symbol) == "" {
		return "", false
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if !declLinePattern.MatchString(line) {
			continue
		}
		if !focus.LikeContains(line, symbol) {
			continue
		}
		if strings.Contains(line, "{") {
			return extractBraceBlock(lines, i), true
		}
		if frag := extractIndentBlock(lines, i); frag != "" {
			return frag, true
		}
	}
	return "", false
}

// extractBraceBlock returns lines from start to the matching closing brace.
func extractBraceBlock(lines []string, start int) string {
	depth := 0
	end := start
	started := false
	for i := start; i < len(lines); i++ {
		depth += strings.Count(lines[i], "{") - strings.Count(lines[i], "}")
		if strings.Contains(lines[i], "{") {
			started = true
		}
		end = i
		if started && depth <= 0 {
			break
		}
	}
	return strings.Join(lines[start:end+1], "\n")
}

// extractIndentBlock returns the header line plus every deeper-indented line
// that follows (Python def/class style).
func extractIndentBlock(lines []string, start int) string {
	base := indentOf(lines[start])
	end := start
	for i := start + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			end = i
			continue
		}
		if indentOf(lines[i]) <= base {
			break
		}
		end = i
	}
	if end == start {
		return ""
	}
	return strings.Join(lines[start:end+1], "\n")
}

func indentOf(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}
