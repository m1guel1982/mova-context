// render.go — capa de presentación del Focus Resolution Engine.
package resolvers

import (
	"fmt"
	"path/filepath"
	"strings"

	"mova.local/core/focus"
	"mova.local/core/focus/resolvers"
	"mova.local/dedup"
	"mova.local/documents"
)

// ResolveRepoPath aplica las reglas de resolución de ruta del repositorio —
// las MISMAS que documents.IsAbsCrossPlatform/NormalizeAbsPath ya usan
// para "repo" en write_file/create_directory/generate_* (ver
// documents/pathresolve.go), en vez del filepath.IsAbs nativo que este
// archivo usaba antes: filepath.IsAbs es específico del SO en el que
// corre Mova ahora mismo, así que un "repo": "C:\\proyecto" en
// project.json se ignoraba silenciosamente (se unía mal con root) si
// Mova corría en Linux/macOS, y viceversa con una ruta UNC. Con
// IsAbsCrossPlatform, un repo absoluto Windows/UNC/Unix se reconoce
// igual sin importar el SO anfitrión — mismo comportamiento que ya
// tenían los demás caminos que resuelven "repo".
func ResolveRepoPath(root, repo string) string {
	if repo == "" || repo == "." {
		return root
	}
	if documents.IsAbsCrossPlatform(repo) {
		if normalized, err := documents.NormalizeAbsPath(repo); err == nil {
			return normalized
		}
		// Ruta absoluta reconocida pero de un SO distinto al actual
		// (p.ej. letra de unidad Windows corriendo en Linux) — cae al
		// join relativo a root en vez de devolver una ruta inválida,
		// mismo criterio de "nunca romper el proceso" que el resto de
		// esta base de código.
	}
	return filepath.Join(root, repo)
}

// AbsUnder devuelve la ruta absoluta de source: si ya es absoluta (target
// fuera del repo) se respeta tal cual, si no se une a repoPath — nunca
// se antepone el repo a una ruta que ya lo trae.
func AbsUnder(repoPath, source string) string {
	if documents.IsAbsCrossPlatform(source) {
		return filepath.Clean(source)
	}
	return filepath.Join(repoPath, filepath.FromSlash(strings.ReplaceAll(source, `\`, "/")))
}

// ResolveExcludeTargets resuelve cada entrada de `exclude` a los archivos
// reales a los que apunta (solo para el log de debug). Usa su propio
// índice, sin aplicar `exclude`, para poder mostrar lo que se excluye.
func ResolveExcludeTargets(root, repo string, exclude []string) []focus.ResolvedTarget {
	ctx := focus.Context{RepoPath: ResolveRepoPath(root, repo), Index: &focus.FileIndex{}}
	out := make([]focus.ResolvedTarget, 0, len(exclude))
	for _, ex := range exclude {
		out = append(out, focus.ResolvedTarget{Name: ex, Paths: resolvers.LocateTarget(ctx, ex)})
	}
	return out
}

// DefaultResolvers construye la lista de resolvers Community en orden de prioridad.
func DefaultResolvers() []focus.Resolver {
	return []focus.Resolver{
		resolvers.NewAstSymbolResolver(),
		resolvers.NewFileResolver(),
		resolvers.NewDirectoryResolver(),
		resolvers.NewJSONResolver(),
		resolvers.NewSQLResolver(),
		resolvers.NewCodeSymbolResolver(),
		resolvers.NewMarkdownResolver(),
		resolvers.NewLegalResolver(),
		resolvers.NewMemoryResolver(),
		resolvers.NewGlobResolver(),
		resolvers.NewFallbackResolver(),
	}
}

// NewEngineWithResolvers arma un *focus.Engine registrando primero extra resolvers y luego los Community.
func NewEngineWithResolvers(extra ...focus.Resolver) *focus.Engine {
	e := focus.New()
	for _, r := range extra {
		e.RegisterResolver(r)
	}
	for _, r := range DefaultResolvers() {
		e.RegisterResolver(r)
	}
	return e
}

func defaultEngine() *focus.Engine {
	return NewEngineWithResolvers()
}

// RenderFocusContext resuelve cada item de focus y devuelve el bloque de
// texto compilado. exclude implementa la clave "exclude" de
// project.json (misma sintaxis multiplataforma que focus/items, pero
// para EXCLUSIÓN — ver core/focus/resolvers/exclude.go); nil/vacío
// desactiva la exclusión, comportamiento idéntico al de antes de que
// "exclude" existiera.
func RenderFocusContext(root, repo string, items []string, extraExclude []string, exclude []string) (string, focus.ScanStats) {
	return renderFocusContext(root, repo, items, extraExclude, exclude, defaultEngine(), nil)
}

// RenderFocusContextWithEngine permite pasar un Engine personalizado.
func RenderFocusContextWithEngine(root, repo string, items []string, extraExclude []string, exclude []string, engine *focus.Engine) (string, focus.ScanStats) {
	return renderFocusContext(root, repo, items, extraExclude, exclude, engine, nil)
}

// RenderFocusContextWithSeen (REQUERIDO POR engine.go)
// Recibe un mapa externo de párrafos ya procesados para deduplicar contra AGENTS, PROMPT, etc.
func RenderFocusContextWithSeen(root, repo string, items []string, extraExclude []string, exclude []string, seen map[string]bool) (string, focus.ScanStats) {
	return renderFocusContext(root, repo, items, extraExclude, exclude, defaultEngine(), seen)
}

var proseKinds = map[string]bool{
	"doc-section": true, "legal-article": true, "chronological": true,
	"bounded-excerpt": true,
}

var proseFileExt = map[string]bool{
	".md": true, ".markdown": true, ".txt": true, ".rst": true,
}

func isProse(source, kind string) bool {
	if proseKinds[kind] {
		return true
	}
	if kind == "file" {
		return proseFileExt[strings.ToLower(filepath.Ext(source))]
	}
	return false
}

func renderFocusContext(root, repo string, items []string, extraExclude []string, exclude []string, engine *focus.Engine, seenParagraphs map[string]bool) (string, focus.ScanStats) {
	if len(items) == 0 {
		return "", focus.ScanStats{}
	}
	repoPath := ResolveRepoPath(root, repo)
	stats := &focus.ScanStats{}
	ctx := focus.Context{RepoPath: repoPath, ExcludeDirs: extraExclude, Exclude: exclude, Stats: stats, Index: &focus.FileIndex{}}

	// dirLike reusa el Match() real de Glob/DirectoryResolver — nunca
	// re-implementa "¿esto es un directorio?" con una heurística propia
	// (p. ej. contar blocks, que se equivoca apenas un directorio
	// resuelve a exactamente 1 archivo) — un solo lugar decide qué es
	// "directorio" (fsutil.go/file.go) y aquí solo se consulta.
	globR := resolvers.NewGlobResolver()
	dirR := resolvers.NewDirectoryResolver()
	dirLike := func(item string) bool {
		return globR.Match(ctx, item) || dirR.Match(ctx, item)
	}

	included := map[string]bool{}
	if seenParagraphs == nil {
		seenParagraphs = map[string]bool{}
	}
	var sb strings.Builder
	for _, item := range items {
		sb.WriteString("FOCUS:" + item + "\n")
		blocks, err := engine.Resolve(ctx, item)
		if err == nil {
			for i := range blocks {
				if blocks[i].Kind == "file" && blocks[i].Source != "" {
					blocks[i].Content = resolvers.ApplyAstSymbolExcludes(exclude, blocks[i].Source, blocks[i].Content)
				}
			}
		}
		sb.WriteString(renderResult(item, blocks, err, seenParagraphs, stats))
		sb.WriteString("\n")
		if err == nil {
			for _, b := range blocks {
				if b.Source != "" {
					included[b.Source] = true
				}
			}
			// Un ítem por TARGET (nunca por archivo individual): "dir"
			// cuando el target ES un directorio o un patrón glob/"."
			// (sin importar a cuántos archivos haya expandido — incluso
			// 1), "file" para cualquier otra resolución puntual (un
			// archivo, un símbolo, una sección de documento, ...).
			kind := "file"
			if dirLike(item) {
				kind = "dir"
			}
			var paths []string
			if kind == "file" {
				for _, b := range blocks {
					if b.Source != "" {
						paths = append(paths, AbsUnder(repoPath, b.Source))
					}
				}
			}
			stats.RecordFocusItem(item, kind, len(blocks), paths)
		}
	}
	stats.FilesIncluded = len(included)
	return sb.String(), *stats
}

func renderResult(item string, blocks []focus.ContextBlock, err error, seenParagraphs map[string]bool, stats *focus.ScanStats) string {
	if err != nil || len(blocks) == 0 {
		return "  not found: " + item
	}
	rendered := make([]string, 0, len(blocks))
	for _, b := range blocks {
		rendered = append(rendered, renderBlock(b, seenParagraphs, stats))
	}
	return strings.Join(rendered, "\n\n")
}

func renderBlock(b focus.ContextBlock, seenParagraphs map[string]bool, stats *focus.ScanStats) string {
	switch b.Kind {
	case "file":
		if isProse(b.Source, b.Kind) {
			deduped, removed, chars := dedup.Paragraphs(b.Content, seenParagraphs)
			stats.DuplicatesRemoved += removed
			stats.DuplicatesRemovedChars += chars

			if strings.TrimSpace(deduped) == "" && removed > 0 {
				return fmt.Sprintf("  [duplicado — %s ya se incluyó antes en este contexto, ver contexto.report]", b.Source)
			}
			return deduped
		}
		return b.Content
	case "dir-index":
		return "  " + b.Content
	default:
		content := b.Content
		if isProse(b.Source, b.Kind) {
			deduped, removed, chars := dedup.Paragraphs(content, seenParagraphs)
			stats.DuplicatesRemoved += removed
			stats.DuplicatesRemovedChars += chars

			content = deduped
			if strings.TrimSpace(content) == "" && removed > 0 {
				return fmt.Sprintf("  (%s)\n  [duplicado — ya incluido antes en este contexto, ver contexto.report]", b.Source)
			}
		}
		return fmt.Sprintf("  (%s)\n%s", b.Source, content)
	}
}
