// actions_budget.go — the job engine's "budget" action: when a job
// declares {"budget": {"focus": true}}, it produces mova-budget-report.md
// exactly like `mova budget --focus` does (see budget.BuildReport/
// budget.WriteReport) — the same component `mova budget` and the
// "estimate_budget" MCP tool already use. Runs against the job's first
// declared task (or the project's default_task if the job declares
// none), since a budget report needs some task context to size.
package jobs

import "mova.local/budget"

func runBudget(jc *jobContext, res *Result) {
	if jc.Spec.Budget == nil {
		return
	}
	task := ""
	if len(jc.Spec.Tasks) > 0 && jc.Spec.Tasks[0] != "*" {
		task = jc.Spec.Tasks[0]
	}

	report, err := budget.BuildReport(jc.Adapter, jc.Root, jc.Project, task, jc.Spec.Budget.Focus)
	if err != nil {
		res.fail("budget: %v", err)
		return
	}
	path, err := budget.WriteReport(jc.Root, jc.Project, jc.Proj, report)
	if err != nil {
		res.fail("budget: could not write report: %v", err)
		return
	}
	res.log("✓ budget report: %s (%d tokens)", path, report.TotalTokens)
}
