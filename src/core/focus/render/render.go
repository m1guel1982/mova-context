// render.go — capa de presentación del Focus Resolution Engine (edición
// Community, gratis). Antes vivía exclusivamente en el módulo comercial
// (mova.local/compiler/focus/render); ahora vive acá para que `focus`
// funcione sin -tags premium. La edición Premium (mova.local/compiler/focus)
// reutiliza DefaultResolvers()/NewEngineWithResolvers de este mismo
// paquete y antepone su propio SemanticResolver — nunca duplica esta
// lógica.
package render

import (
	"fmt"
	"path/filepath"
	"regexp" 
	"strings"

	"mova.local/core/focus"
	"mova.local/core/focus/resolvers"
)

// resolveRepoPath applies the workflow.md "repo" resolution rules.
func resolveRepoPath(root, repo string) string {
	if repo == "" || repo == "." {
		return root
	}
	if filepath.IsAbs(repo) {
		return repo
	}
	return filepath.Join(root, repo)
}

// DefaultResolvers construye la lista de resolvers Community en el orden
// de prioridad por defecto: File → Directory → JSON → SQL → CodeSymbol →
// Markdown → Legal → Memory → Fallback. Expuesta (mayúscula) para que la
// edición Premium pueda anteponer su SemanticResolver sin reimplementar
// ni reordenar esta lista.
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
		resolvers.NewFallbackResolver(),
	}
}

// NewEngineWithResolvers arma un *focus.Engine registrando primero extra
// (por ejemplo, el SemanticResolver de la edición Premium) y luego los
// resolvers Community de DefaultResolvers() — la cascada siempre termina
// en la misma red de seguridad "LIKE simple", en cualquier edición.
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
// texto "FOCUS:item\n<contenido>\n" concatenado, junto con ScanStats:
// evidencia real de cuántos archivos tocó el escaneo del repo, por qué el
// resto no entró, y cuántos párrafos duplicados se quitaron. extraExclude
// son carpetas adicionales a ignorar, si el llamador quiere pasar alguna
// — se SUMAN a las que ya se ignoran siempre, nunca las reemplazan.
func RenderFocusContext(root, repo string, items []string, extraExclude []string) (string, focus.ScanStats) {
	return renderFocusContext(root, repo, items, extraExclude, defaultEngine())
}

// RenderFocusContextWithEngine es igual que RenderFocusContext pero recibe
// un *focus.Engine ya armado — usado por la edición Premium para pasar un
// engine con el SemanticResolver antepuesto (ver NewEngineWithResolvers).
func RenderFocusContextWithEngine(root, repo string, items []string, extraExclude []string, engine *focus.Engine) (string, focus.ScanStats) {
	return renderFocusContext(root, repo, items, extraExclude, engine)
}

// proseKinds son los tipos de ContextBlock que contienen prosa/documentación
// — únicos candidatos a la deduplicación de párrafos exactos (ver
// dedupParagraphs). Código, SQL y nodos JSON NUNCA se tocan: un bloque de
// código idéntico repetido a propósito (ej. dos funciones que comparten
// una misma línea de import) no es un "duplicado de contenido", es
// estructura del lenguaje.
var proseKinds = map[string]bool{
	"doc-section": true, "legal-article": true, "chronological": true,
	"bounded-excerpt": true,
}

// proseFileExt — extensiones de archivo que, cuando el Kind es "file"
// (entrega de archivo completo), se consideran prosa para efectos de
// deduplicación. Código fuente (.go, .py, .js, ...) nunca entra acá.
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

var multiBlank = regexp.MustCompile(`\n{2,}`)

// normalizeParagraph colapsa espacios/saltos de línea para comparar
// "¿es el mismo texto?" sin que un espacio extra al final de línea cuente
// como una diferencia real — pero SIN cambiar ni una palabra del
// contenido que efectivamente se emite (la normalización es solo para la
// comparación, ver dedupParagraphs).
func normalizeParagraph(p string) string {
	return strings.Join(strings.Fields(p), " ")
}

// dedupParagraphs quita, de un bloque de prosa, cualquier párrafo
// (separado por línea en blanco) cuyo texto normalizado sea IDÉNTICO a
// uno que YA se emitió antes en este mismo render — nunca una
// reformulación, nunca un "parecido": exactamente lo que
// compiler/dedup hace para AGENT/SKILL/PROMPT, aplicado ahora también a
// FOCUS (que antes quedaba completamente afuera de cualquier
// deduplicación — ver docs/i18n/{es,en}/focus-engine.md, sección
// "Duplicados"). seen se comparte entre TODOS los items de `focus` de una
// misma compilación: si "manual.md" completo y luego "Artículo 3" por
// separado repiten el mismo párrafo, la segunda aparición se quita y se
// cuenta en removed.
func dedupParagraphs(content string, seen map[string]bool) (string, int) {
	paragraphs := multiBlank.Split(content, -1)
	var kept []string
	removed := 0
	for _, p := range paragraphs {
		if strings.TrimSpace(p) == "" {
			kept = append(kept, p)
			continue
		}
		key := normalizeParagraph(p)
		if seen[key] {
			removed++
			continue
		}
		seen[key] = true
		kept = append(kept, p)
	}
	return strings.Join(kept, "\n\n"), removed
}

func renderFocusContext(root, repo string, items []string, extraExclude []string, engine *focus.Engine) (string, focus.ScanStats) {
	if len(items) == 0 {
		return "", focus.ScanStats{}
	}
	repoPath := resolveRepoPath(root, repo)
	stats := &focus.ScanStats{}
	ctx := focus.Context{RepoPath: repoPath, ExcludeDirs: extraExclude, Stats: stats}

	included := map[string]bool{}       // dedup: un mismo archivo referenciado por 2 focus items cuenta una vez
	seenParagraphs := map[string]bool{} // dedup: mismo párrafo de prosa ya emitido antes en este render
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
		}
	}
	stats.FilesIncluded = len(included)
	return sb.String(), *stats
}

// renderResult traduce []ContextBlock al formato de texto que arma
// contexto.txt/BuildContext. Aplica dedupParagraphs SOLO a bloques de
// prosa (ver isProse) — código, SQL, JSON e índices de directorio se
// devuelven intactos, byte a byte, siempre.
// renderResult traduce []ContextBlock al formato de texto que arma
// contexto.txt/BuildContext. Un mismo item de `focus` puede devolver MÁS
// DE UN bloque (ver MarkdownResolver/LegalResolver: "Órdenes" puede
// matchear tres headings distintos) — todos se renderizan, en el mismo
// orden en que el resolver los devolvió, separados por una línea en
// blanco. Aplica dedupParagraphs SOLO a bloques de prosa (ver isProse) —
// código, SQL, JSON e índices de directorio se devuelven intactos, byte a
// byte, siempre.
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

// renderBlock renderiza UN ContextBlock — la lógica que antes vivía
// directamente en renderResult, ahora factorizada para poder aplicarse a
// cada bloque de una lista con más de un resultado.
func renderBlock(b focus.ContextBlock, seenParagraphs map[string]bool, stats *focus.ScanStats) string {
	switch b.Kind {
	case "file":
		if isProse(b.Source, b.Kind) {
			deduped, removed := dedupParagraphs(b.Content, seenParagraphs)
			stats.DuplicatesRemoved += removed
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
			deduped, removed := dedupParagraphs(content, seenParagraphs)
			stats.DuplicatesRemoved += removed
			content = deduped
			if strings.TrimSpace(content) == "" && removed > 0 {
				// Todo el contenido de este target ya se había incluido
				// antes (por otro item de `focus`, o repetido de forma
				// literal dentro del propio archivo fuente) — se dice
				// explícitamente en vez de dejar un bloque vacío sin
				// explicación.
				return fmt.Sprintf("  (%s)\n  [duplicado — ya incluido antes en este contexto, ver contexto.report]", b.Source)
			}
		}
		return fmt.Sprintf("  (%s)\n%s", b.Source, content)
	}
}
