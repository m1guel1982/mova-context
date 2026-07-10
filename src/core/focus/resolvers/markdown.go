// markdown.go — Markdown Resolver.
//
// Antes usaba strings.Contains(strings.ToLower(line), strings.ToLower(q))
// para decidir si un heading calzaba con el target pedido: coincidencia
// exacta de substring, sensible a acentos. Ahora usa focus.LikeContains
// (ver match.go) — insensible a mayúsculas Y acentos, el "LIKE simple" que
// pide la edición Community. Ejemplo real (projects/i18n-demo): buscar
// "articulo 3" (sin tilde) ahora SÍ encuentra "## Artículo 3 —
// Pluralización"; antes daba "not found".
//
// Dos capacidades más, agregadas para que `focus` se comporte como un
// verdadero LIKE de SQL:
//
//  1. Multi-match: un mismo item de `focus` (ej. "Órdenes") devuelve TODAS
//     las secciones cuyo heading matchea, no solo la primera — antes
//     "Capítulo 1 — Órdenes" tapaba a "Artículo 5 — Creación de órdenes" y
//     "Artículo 6 — Cancelación de órdenes"; ahora las tres se incluyen.
//  2. Modo exacto: un item que empieza con "=" (ver focus.IsExact) usa
//     focus.LikeExact — sensible a mayúsculas Y acentos — en vez de
//     focus.LikeContains. "=Órdenes" SOLO matchea el heading que contiene
//     "Órdenes" tal cual (mismo caso, mismo acento); "ordenes" o "ÓRDENES"
//     no cuentan como el mismo texto en este modo.
package resolvers

import (
	"regexp"
	"sort"
	"strings"

	"mova.local/core/focus"
)

// markdownHeadingPattern matches Markdown headings ("#" a "######").
var markdownHeadingPattern = regexp.MustCompile(`(?i)^\s*#{1,6}\s+.*$`)

// headingMatches decide si una línea de heading matchea query, en modo
// exacto (case/acento sensible) o LIKE (insensible) según exact.
func headingMatches(line, query string, exact bool) bool {
	if exact {
		return focus.LikeExact(line, query)
	}
	return focus.LikeContains(line, query)
}

// extractAllDocSections devuelve CADA sección (desde un heading que
// matchea hasta el siguiente heading del mismo patrón, sin incluirlo) —
// no solo la primera. En modo exacto (case/acento sensible) usa
// focus.LikeExact; si no, focus.LikeContains (LIKE simple, insensible).
func extractAllDocSections(headingRE *regexp.Regexp, content, query string, exact bool) []string {
	lines := strings.Split(content, "\n")

	var allHeadings []int
	var matchedHeadings []int
	for i, line := range lines {
		if !headingRE.MatchString(line) {
			continue
		}
		allHeadings = append(allHeadings, i)
		if headingMatches(line, query, exact) {
			matchedHeadings = append(matchedHeadings, i)
		}
	}
	if len(matchedHeadings) == 0 {
		return nil
	}

	var sections []string
	for _, start := range matchedHeadings {
		end := len(lines)
		for _, h := range allHeadings {
			if h > start {
				end = h
				break
			}
		}
		sections = append(sections, strings.Join(lines[start:end], "\n"))
	}
	return sections
}

// MarkdownResolver busca, en todos los archivos del repo, TODAS las
// secciones delimitadas por encabezados Markdown ("#", "##", ...) cuyo
// título matchea el símbolo pedido — LIKE simple por defecto (sin
// distinguir mayúsculas ni acentos), o exacto si el item viene marcado
// con "=" (ver focus.IsExact). Catch-all: la certeza la da Resolve.
type MarkdownResolver struct{}

func NewMarkdownResolver() *MarkdownResolver { return &MarkdownResolver{} }

func (r *MarkdownResolver) Match(ctx focus.Context, target string) bool { return true }

func (r *MarkdownResolver) Resolve(ctx focus.Context, target string) ([]focus.ContextBlock, error) {
	exact := focus.IsExact(target)
	symbol := stripSymbolNotation(focus.StripExact(target))
	var candidates []string
	walkFiles(ctx, ctx.RepoPath, func(p string) { candidates = append(candidates, p) })
	sort.Strings(candidates)

	var blocks []focus.ContextBlock
	for _, p := range candidates {
		content := readFile(p)
		if content == "" {
			continue
		}
		for _, frag := range extractAllDocSections(markdownHeadingPattern, content, symbol, exact) {
			blocks = append(blocks, focus.ContextBlock{
				Source:  relOrBase(ctx.RepoPath, p),
				Kind:    "doc-section",
				Content: frag,
			})
		}
	}
	if len(blocks) == 0 {
		return nil, focus.ErrNotFound
	}
	return blocks, nil
}
