// memory.go — Memory Resolver.
//
// Busca bloques cronológicos (bitácoras, call center, finanzas, salud —
// cualquier bloque que empiece con una fecha) que mencionen el símbolo
// pedido. La comparación de contenido usa LIKE simple (ver match.go):
// insensible a mayúsculas y acentos.
package resolvers

import (
	"regexp"
	"sort"
	"strings"

	"mova.local/core/focus"
)

// datedLinePattern matches lines starting a chronological entry.
var datedLinePattern = regexp.MustCompile(`^\s*\**\[?(\d{4}-\d{2}-\d{2}|\d{2}/\d{2}/\d{4})\]?\**`)

// extractChronological returns dated blocks whose text matches the query
// — LIKE simple (case/acento insensible) por defecto, o exacto si exact.
func extractChronological(content, query string, exact bool) (string, bool) {
	lines := strings.Split(content, "\n")
	var blocks []string
	var cur []string
	flush := func() {
		if len(cur) == 0 {
			return
		}
		text := strings.Join(cur, " ")
		matched := focus.LikeContains(text, query)
		if exact {
			matched = focus.LikeExact(text, query)
		}
		if matched {
			blocks = append(blocks, strings.Join(cur, "\n"))
		}
		cur = nil
	}
	for _, line := range lines {
		if datedLinePattern.MatchString(line) {
			flush()
		}
		cur = append(cur, line)
	}
	flush()
	if len(blocks) == 0 {
		return "", false
	}
	return strings.Join(blocks, "\n---\n"), true
}

// MemoryResolver busca bloques cronológicos (entradas fechadas) que
// mencionen el símbolo pedido — bitácoras, call center, finanzas, salud.
type MemoryResolver struct{}

func NewMemoryResolver() *MemoryResolver { return &MemoryResolver{} }

func (r *MemoryResolver) Match(ctx focus.Context, target string) bool { return true }

func (r *MemoryResolver) Resolve(ctx focus.Context, target string) ([]focus.ContextBlock, error) {
	exact := focus.IsExact(target)
	symbol := stripSymbolNotation(focus.StripExact(target))
	var candidates []string
	walkFiles(ctx, ctx.RepoPath, func(p string) { candidates = append(candidates, p) })
	sort.Strings(candidates)
	for _, p := range candidates {
		if frag, ok := extractChronological(readFile(p), symbol, exact); ok {
			return []focus.ContextBlock{{
				Source:  relOrBase(ctx.RepoPath, p),
				Kind:    "chronological",
				Content: frag,
			}}, nil
		}
	}
	return nil, focus.ErrNotFound
}
