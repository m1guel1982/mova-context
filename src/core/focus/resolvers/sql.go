// sql.go — SQL Resolver. Dado un nombre de tabla, devuelve solo su
// definición (CREATE TABLE ... hasta el ";" de cierre), nunca el archivo
// .sql completo. Determinista: coincidencia exacta de identificador, sin
// heurísticas de similitud (los nombres de tabla no deberían matchear por
// "LIKE" — un typo en un nombre de tabla es un error real, no algo que
// haya que tolerar).
package resolvers

import (
	"regexp"
	"sort"
	"strings"

	"mova.local/core/focus"
)

type SQLResolver struct{}

func NewSQLResolver() *SQLResolver { return &SQLResolver{} }

func createTablePattern(table string) *regexp.Regexp {
	name := regexp.QuoteMeta(table)
	// admite backticks, comillas dobles o sin comillas; "IF NOT EXISTS" opcional.
	return regexp.MustCompile(`(?i)^\s*CREATE\s+TABLE\s+(IF\s+NOT\s+EXISTS\s+)?[` + "`" + `"]?` + name + `[` + "`" + `"]?\s*\(`)
}

func (r *SQLResolver) Match(ctx focus.Context, target string) bool {
	return true // catch-all: solo aplica a archivos .sql, decidido en Resolve
}

func (r *SQLResolver) Resolve(ctx focus.Context, target string) ([]focus.ContextBlock, error) {
	symbol := stripSymbolNotation(focus.StripExact(target))
	decl := createTablePattern(symbol)
	var candidates []string
	walkFiles(ctx, ctx.RepoPath, func(p string) {
		if strings.HasSuffix(strings.ToLower(p), ".sql") {
			candidates = append(candidates, p)
		}
	})
	sort.Strings(candidates)
	for _, p := range candidates {
		content := readFile(p)
		if content == "" {
			continue
		}
		if frag, ok := extractSQLTableDef(content, decl); ok {
			return []focus.ContextBlock{{
				Source:  relOrBase(ctx.RepoPath, p),
				Kind:    "sql-def",
				Content: frag,
			}}, nil
		}
	}
	return nil, focus.ErrNotFound
}

// extractSQLTableDef returns the CREATE TABLE statement up to its closing
// ";" — nunca el archivo .sql completo, solo la definición de esa tabla.
func extractSQLTableDef(content string, decl *regexp.Regexp) (string, bool) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if !decl.MatchString(line) {
			continue
		}
		end := i
		for j := i; j < len(lines); j++ {
			end = j
			if strings.Contains(lines[j], ";") {
				break
			}
		}
		return strings.Join(lines[i:end+1], "\n"), true
	}
	return "", false
}
