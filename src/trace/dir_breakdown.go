// dir_breakdown.go — top-level-directory token accounting helpers
// used by AnalyzeRemote (see analyzer.go): which directory a path
// belongs to, and the resulting DirRow table with exclude
// suggestions. Split out of analyzer.go purely to keep every file in
// this package under the 300-line limit.
package trace

import (
	"path/filepath"
	"sort"

	"mova.local/budget"
)

// componentsFromReport builds the Agents/Skills/Prompt/Focus/Memory
// row breakdown for a project.json run (see AnalyzeLocal).
func componentsFromReport(report *budget.Report) []ComponentRow {
	byName := map[string]int{}
	for _, c := range report.Components {
		byName[c.Name] = c.Tokens
	}
	rows := make([]ComponentRow, 0, len(componentOrder))
	for _, name := range componentOrder {
		rows = append(rows, ComponentRow{Name: name, Tokens: byName[name]})
	}
	return rows
}

// costRowsFromModelCosts adapts budget.ModelCost into this package's
// own CostRow shape.
func costRowsFromModelCosts(costs []budget.ModelCost) []CostRow {
	rows := make([]CostRow, 0, len(costs))
	for _, c := range costs {
		rows = append(rows, CostRow{Provider: c.Provider, Model: c.Model, USD: c.USD})
	}
	return rows
}

func topLevelDir(repoDir, path string) string {
	rel, err := filepath.Rel(repoDir, path)
	if err != nil {
		return "(repo root)"
	}
	rel = filepath.ToSlash(rel)
	if i := indexByte(rel, '/'); i >= 0 {
		return rel[:i]
	}
	return "(repo root)"
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

const maxDirRows = 20

func buildDirBreakdown(dirTokens, dirFiles map[string]int, totalTokens int) []DirRow {
	rows := make([]DirRow, 0, len(dirTokens))
	for dir, tokens := range dirTokens {
		pct := 0.0
		if totalTokens > 0 {
			pct = float64(tokens) / float64(totalTokens) * 100
		}
		rows = append(rows, DirRow{Dir: dir, Tokens: tokens, Files: dirFiles[dir], Percent: pct})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Tokens > rows[j].Tokens })

	if len(rows) <= maxDirRows {
		return rows
	}
	kept := rows[:maxDirRows-1]
	other := DirRow{Dir: "(other directories)"}
	for _, r := range rows[maxDirRows-1:] {
		other.Tokens += r.Tokens
		other.Files += r.Files
		other.Percent += r.Percent
	}
	return append(kept, other)
}

func suggestedExcludeDirs(byDir map[string]int) []string {
	type pair struct {
		name  string
		count int
	}
	pairs := make([]pair, 0, len(byDir))
	for name, count := range byDir {
		pairs = append(pairs, pair{name, count})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].count > pairs[j].count })
	out := make([]string, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, p.name)
	}
	return out
}
