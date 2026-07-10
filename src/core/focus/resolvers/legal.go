// legal.go — Legal Document Resolver.
//
// Reutiliza extractAllDocSections (markdown.go, ya con LIKE simple — ver
// match.go) con el patrón jerárquico legal: Título, Capítulo, Sección,
// Artículo, Inciso. Registrado después de MarkdownResolver en el orden de
// prioridad por defecto. Igual que MarkdownResolver: devuelve TODAS las
// secciones que matchean (no solo la primera) y soporta el marcador "="
// de coincidencia exacta (ver focus.IsExact / focus.LikeExact).
package resolvers

import (
	"regexp"
	"sort"

	"mova.local/core/focus"
)

// legalHeadingPattern matches jerarquía de documentos legales/manuales en
// español: Título, Capítulo, Sección, Artículo, Inciso.
var legalHeadingPattern = regexp.MustCompile(`(?i)^\s*(t[íi]tulo|cap[íi]tulo|secci[óo]n|art[íi]culo|inciso)\s+\S.*$`)

// LegalResolver busca, en todos los archivos del repo, TODOS los
// artículos/secciones de un documento legal cuyo encabezado matchea el
// símbolo pedido — ejemplo: focus: ["articulo 5"] (sin tilde) en un
// proyecto sobre Ley 21.719 sigue encontrando "Artículo 5". Con "=" al
// inicio del item, solo cuenta una coincidencia literal (mismo caso,
// mismo acento) — ver focus.IsExact.
type LegalResolver struct{}

func NewLegalResolver() *LegalResolver { return &LegalResolver{} }

func (r *LegalResolver) Match(ctx focus.Context, target string) bool { return true }

func (r *LegalResolver) Resolve(ctx focus.Context, target string) ([]focus.ContextBlock, error) {
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
		for _, frag := range extractAllDocSections(legalHeadingPattern, content, symbol, exact) {
			blocks = append(blocks, focus.ContextBlock{
				Source:  relOrBase(ctx.RepoPath, p),
				Kind:    "legal-article",
				Content: frag,
			})
		}
	}
	if len(blocks) == 0 {
		return nil, focus.ErrNotFound
	}
	return blocks, nil
}
