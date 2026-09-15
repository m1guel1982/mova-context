// console_relevance.go — the CLI's RELEVANCE and EXECUTION sections
// for a discovery run: makes --task's effect (or lack of one) and
// this run's forensic identity (execution_id/commit) visible in
// plain console output, not just in the JSON artifact. Split out of
// console.go purely to keep every file in this package under the
// 300-line limit.
package trace

import (
	"fmt"
	"strings"
)

// renderRelevanceConsole shows either the top-ranked files (task
// given) or an explicit "NOT APPLIED" notice (no task) - see
// relevance.go's RankByTask/topRelevanceRows.
func renderRelevanceConsole(d *Data) string {
	var b strings.Builder
	b.WriteString("RELEVANCE\n")
	if d.TaskName == "" || len(d.RelevanceTop) == 0 {
		b.WriteString("  Task-specific ranking: NOT APPLIED\n")
		b.WriteString("  Reason: no task or query intent was provided (see --task).\n\n")
		return b.String()
	}
	fmt.Fprintf(&b, "  Task: %s\n", d.TaskName)
	b.WriteString("  Top-ranked files (BM25 + structural + AST signal - see docs' FAQ for the exact formula):\n")
	for _, r := range d.RelevanceTop {
		role := r.Role
		if role == "" {
			role = "-"
		}
		fmt.Fprintf(&b, "  #%-3d %-50s score %-8.2f role %s\n", r.Rank, r.Path, r.Score, role)
	}
	b.WriteString("\n")
	return b.String()
}

// renderExecutionConsole prints this run's forensic identity - every
// artifact (console/report/pii-audit-log.json) carries the SAME
// execution_id, so two runs can never be confused (see
// audit_contract.go's NewExecutionID).
func renderExecutionConsole(d *Data) string {
	var b strings.Builder
	b.WriteString("EXECUTION\n")
	fmt.Fprintf(&b, "  Execution ID       %s\n", d.ExecutionID)
	fmt.Fprintf(&b, "  Repository state   %s\n", orNA(d.CommitHash))
	b.WriteString("\n  All artifacts were generated from this execution result.\n")
	return b.String()
}
