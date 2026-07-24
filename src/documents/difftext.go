// difftext.go — a small, dependency-free line diff, used to show exactly
// what a proposed edit would change BEFORE it's written to disk (see
// cli/nl_edit.go and mcp/nl_edit.go). This is what makes natural-language
// edits precise and reviewable instead of a blind overwrite: the person
// (or the MCP/HTTP caller) sees the real +/- lines, not just "done".
package documents

import (
	"fmt"
	"strings"
)

// DiffResult is a rendered preview of the difference between two versions
// of a file's content.
type DiffResult struct {
	Text         string // unified-diff-like preview: "- old line" / "+ new line", unchanged lines omitted
	Changed      bool
	LinesAdded   int
	LinesRemoved int
}

// diffOp is one line-level operation in the edit script between two texts.
type diffOp struct {
	kind byte // '-' removed (only in old), '+' added (only in new)
	line string
}

// maxDiffCells bounds the O(len(old)*len(new)) LCS table below so a
// pathological pair of huge files can't exhaust memory — past this size,
// DiffLines falls back to a coarse summary instead of a full line diff.
const maxDiffCells = 2_000_000

// maxDiffLinesShown caps how many changed lines are rendered, so a
// sweeping rewrite doesn't flood the chat with thousands of +/- lines.
const maxDiffLinesShown = 200

// DiffLines compares oldText and newText line by line and renders a
// compact preview: only changed lines, prefixed "-"/"+", unchanged lines
// omitted to keep it readable. Identical texts return Changed: false.
func DiffLines(oldText, newText string) DiffResult {
	if oldText == newText {
		return DiffResult{Text: "(no changes)", Changed: false}
	}

	oldLines := strings.Split(oldText, "\n")
	newLines := strings.Split(newText, "\n")

	if len(oldLines)*len(newLines) > maxDiffCells {
		return DiffResult{
			Changed: true,
			Text: fmt.Sprintf(
				"(file too large to line-diff in full: %d -> %d lines; review the full new content before applying)",
				len(oldLines), len(newLines)),
		}
	}

	ops := lcsLineDiff(oldLines, newLines)

	var b strings.Builder
	added, removed, shown := 0, 0, 0
	for _, op := range ops {
		if shown >= maxDiffLinesShown {
			b.WriteString("... (diff truncated — apply to see the full result)\n")
			break
		}
		switch op.kind {
		case '-':
			b.WriteString("- " + op.line + "\n")
			removed++
		case '+':
			b.WriteString("+ " + op.line + "\n")
			added++
		}
		shown++
	}
	return DiffResult{Text: b.String(), Changed: true, LinesAdded: added, LinesRemoved: removed}
}

// lcsLineDiff computes a classic Longest-Common-Subsequence line diff via
// dynamic programming: dp[i][j] is the LCS length of a[i:] and b[j:].
// Walking the table from (0,0) then reconstructs the minimal edit script.
// O(n*m) time and space — fine for source files and documents; see
// maxDiffCells above for the size guard on pathological inputs.
func lcsLineDiff(a, b []string) []diffOp {
	n, m := len(a), len(b)
	dp := make([][]int32, n+1)
	for i := range dp {
		dp[i] = make([]int32, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	var ops []diffOp
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			ops = append(ops, diffOp{'-', a[i]})
			i++
		default:
			ops = append(ops, diffOp{'+', b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, diffOp{'-', a[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, diffOp{'+', b[j]})
	}
	return ops
}
