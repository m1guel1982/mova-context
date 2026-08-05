// actions_tasks.go — the job engine's "tasks" action: builds the exact
// same context `mova run`/`mova chat`/MCP get_full_context build via
// mova.local/budget.BuildGatedContext — one assembly, one Token
// Firewall pipeline (Sanitizer → Circuit Breaker → the Budget gate),
// every transport, every caller, jobs included. "*" runs every task
// declared in the project's own "tasks" map, sorted, so job output is
// deterministic across runs.
package jobs

import (
	"sort"
	"strings"

	"mova.local/budget"
	"mova.local/core"
)

// runTasks executes spec.Tasks against jc.Proj/jc.Adapter, appending each
// task's assembled context to jc.TaskOutput. A task any Token Firewall
// stage rejects is skipped (its error recorded in res) — same "print
// nothing" rule `mova run` follows, applied per-task so one over-budget
// task doesn't block the rest of the job.
func runTasks(jc *jobContext, res *Result) {
	if len(jc.Spec.Tasks) == 0 {
		return
	}
	names := jc.Spec.Tasks
	if len(names) == 1 && names[0] == "*" {
		names = allTaskNames(jc.Proj)
	}

	var out []string
	for _, name := range names {
		gated := budget.BuildGatedContext(jc.Adapter, jc.Root, jc.Project, name)
		if gated.Sanitize.LinesRemoved > 0 || gated.Sanitize.BlankRemoved > 0 {
			res.log("· sanitizer: task %q cleaned (%d repeated line(s), %d blank-line run(s))", name, gated.Sanitize.LinesRemoved, gated.Sanitize.BlankRemoved)
		}
		if gated.Err != nil {
			res.fail("task %q: %v", name, gated.Err)
			continue
		}
		out = append(out, "## Task: "+name+"\n\n"+gated.Text)
		res.log("✓ task %q executed (%d tokens)", name, gated.Tokens)
	}
	jc.TaskOutput = strings.Join(out, "\n\n---\n\n")
}

// allTaskNames lists proj.Tasks' keys, sorted — used for the "*" wildcard.
func allTaskNames(proj *core.Project) []string {
	names := make([]string, 0, len(proj.Tasks))
	for k := range proj.Tasks {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
