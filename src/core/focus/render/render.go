// render.go — capa de presentación del Focus Resolution Engine.
package resolvers

import (
	"fmt"
	"path/filepath"
	"strings"

	"mova.local/core/focus"
	"mova.local/core/focus/resolvers"
	"mova.local/dedup"
)

// resolveRepoPath aplica las reglas de resolución de ruta del repositorio.
func resolveRepoPath(root, repo string) string {
	if repo == "" || repo == "." {
		return root
	}
	if filepath.IsAbs(repo) {
		return repo
	}
	return filepath.Join(root, repo)
}

// DefaultResolvers construye la lista de resolvers Community en orden de prioridad.
func DefaultResolvers() []focus.Resolver {
	return []focus.Resolver{
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
	repoPath := resolveRepoPath(root, repo)
	stats := &focus.ScanStats{}
	ctx := focus.Context{RepoPath: repoPath, ExcludeDirs: extraExclude, Exclude: exclude, Stats: stats}

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
			stats.RecordFocusItem(item, kind, len(blocks))
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
