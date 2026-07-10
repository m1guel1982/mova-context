// fallback.go — Fallback Resolver (bounded excerpt).
//
// Último recurso antes de reportar "not found". Ventana deliberadamente
// pequeña — el propósito de "focus" es evitar enviar contenido no
// relacionado, así que el fallback nunca vuelca un archivo completo, solo
// un contexto acotado alrededor de la primera ocurrencia del símbolo.
// Debe registrarse ÚLTIMO en el Engine: si cualquier otro resolver
// estructural (código, doc, cronológico) encuentra algo, este nunca se
// ejecuta.
//
// Community (LIKE simple), dos pasadas, de más a menos estricta:
//  1. LikeContains: el target completo aparece como substring (sin
//     distinguir mayúsculas/acentos) — igual que antes, solo tolerante a
//     acentos.
//  2. LikeContainsAllWords: si (1) no encontró nada y el target tiene más
//     de una palabra, se acepta un archivo donde TODAS las palabras del
//     target aparecen en algún lugar (no necesariamente juntas ni en
//     orden). Sigue siendo búsqueda textual — nunca infiere una palabra
//     que no está escrita — pero es más amigable con targets como
//     "carga traducciones" cuando el texto real dice "traducciones se
//     cargan desde locales/".
package resolvers

import (
	"sort"
	"strings"

	"mova.local/core/focus"
)

// FallbackResolver is the last-resort resolver: nearby textual excerpt.
type FallbackResolver struct{}

func NewFallbackResolver() *FallbackResolver { return &FallbackResolver{} }

func (r *FallbackResolver) Match(ctx focus.Context, target string) bool { return true }

func (r *FallbackResolver) Resolve(ctx focus.Context, target string) ([]focus.ContextBlock, error) {
	symbol := stripSymbolNotation(focus.StripExact(target))
	var candidates []string
	walkFiles(ctx, ctx.RepoPath, func(p string) { candidates = append(candidates, p) })
	sort.Strings(candidates)

	// Pasada 1: substring LIKE del target completo.
	for _, p := range candidates {
		content := readFile(p)
		if content == "" {
			continue
		}
		if frag, ok := boundedExcerpt(content, symbol); ok {
			return []focus.ContextBlock{{
				Source:  relOrBase(ctx.RepoPath, p),
				Kind:    "bounded-excerpt",
				Content: frag,
			}}, nil
		}
	}

	// Pasada 2: todas las palabras del target, en cualquier orden.
	if len(strings.Fields(symbol)) > 1 {
		for _, p := range candidates {
			content := readFile(p)
			if content == "" {
				continue
			}
			if frag, ok := boundedExcerptAllWords(content, symbol); ok {
				return []focus.ContextBlock{{
					Source:  relOrBase(ctx.RepoPath, p),
					Kind:    "bounded-excerpt",
					Content: frag,
				}}, nil
			}
		}
	}
	return nil, focus.ErrNotFound
}

// boundedExcerpt is a small window of context around the first textual
// occurrence of query (LIKE simple: sin distinguir mayúsculas/acentos).
func boundedExcerpt(content, query string) (string, bool) {
	if strings.TrimSpace(query) == "" {
		return "", false
	}
	idx := indexLike(content, query)
	if idx == -1 {
		return "", false
	}
	return windowAround(content, idx), true
}

// boundedExcerptAllWords finds the first line containing every word of
// query (in any order) and returns a window around it.
func boundedExcerptAllWords(content, query string) (string, bool) {
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		if focus.LikeContainsAllWords(l, query) {
			s, e := i-10, i+10
			if s < 0 {
				s = 0
			}
			if e > len(lines) {
				e = len(lines)
			}
			return strings.Join(lines[s:e], "\n"), true
		}
	}
	return "", false
}

// indexLike finds the byte index of the first LIKE (accent/case
// insensitive) occurrence of needle in haystack, or -1.
func indexLike(haystack, needle string) int {
	// La normalización (acentos/mayúsculas) puede cambiar la longitud en
	// bytes de un carácter (p. ej. "Á" UTF-8 son 2 bytes, "a" es 1), así
	// que no podemos usar directamente el índice devuelto por
	// strings.Index sobre el texto normalizado como índice válido sobre
	// el texto original. Para esta ventana de contexto (que se calcula
	// en líneas, no en bytes) alcanza con ubicar la LÍNEA que matchea.
	lines := strings.Split(haystack, "\n")
	pos := 0
	for _, l := range lines {
		if focus.LikeContains(l, needle) {
			return pos
		}
		pos += len(l) + 1
	}
	return -1
}

// windowAround returns the ~20-line window of content surrounding the
// line containing byte offset idx.
func windowAround(content string, idx int) string {
	lines := strings.Split(content, "\n")
	pos, count := 0, 0
	for i, l := range lines {
		count += len(l) + 1
		if count >= idx {
			pos = i
			break
		}
	}
	s, e := pos-10, pos+10
	if s < 0 {
		s = 0
	}
	if e > len(lines) {
		e = len(lines)
	}
	return strings.Join(lines[s:e], "\n")
}
