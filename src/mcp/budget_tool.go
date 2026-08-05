// budget_tool.go — expone `mova budget`/`mova run --count` como la tool
// MCP "estimate_budget", reachable idénticamente desde stdio y HTTP
// (mismo Process(), ver server.go) — el mismo mova.local/orchestrator.Count
// (group-aware generalization of mova.local/budget.BuildReport) que usa
// el comando CLI `mova run --count` y el slash command /budget del chat
// REPL. Una sola implementación, todas las puertas de entrada. `project`
// puede ser un proyecto normal o un grupo multiagente (tiene su propio
// config.json) — orchestrator.Count decide cuál es y, si es un grupo,
// suma un estimado por cada agente en vez de fallar.
package mcp

import (
	"fmt"
	"strconv"

	"mova.local/budget"
	"mova.local/core"
	"mova.local/orchestrator"
)

func budgetTool(adapter core.Adapter, root string, args map[string]any) (string, error) {
	project := str(args, "project")
	task := str(args, "task")
	withFocus, _ := strconv.ParseBool(str(args, "focus"))

	result, err := orchestrator.Count(adapter, root, project, task, withFocus)
	if err != nil {
		return "", err
	}

	if !result.IsGroup {
		proj, err := adapter.GetProject(project)
		if err != nil {
			return "", err
		}
		return formatBudgetReport(root, project, proj, result.Report)
	}
	return formatGroupBudget(project, result), nil
}

func formatBudgetReport(root, project string, proj *core.Project, report *budget.Report) (string, error) {
	path, err := budget.WriteReport(root, project, proj, report)
	if err != nil {
		return "", err
	}
	summary := fmt.Sprintf("✓ mova-budget-report.md generated: %s\n\nTotal tokens: %d (%s)\n", path, report.TotalTokens, report.Encoding)
	for _, c := range report.TotalCosts {
		summary += fmt.Sprintf("  %s/%s: $%.4f %s\n", c.Provider, c.Model, c.USD, report.Currency)
	}
	if report.Focus != nil {
		summary += fmt.Sprintf("\nFocus savings: %.1f%% fewer tokens (%d → %d)\n",
			report.Focus.SavingsPercent, report.Focus.TokensWithoutFocus, report.Focus.TokensWithFocus)
	}
	return summary, nil
}

func formatGroupBudget(group string, result orchestrator.CountResult) string {
	summary := fmt.Sprintf("Group: %s (%d agents)\n\n", group, len(result.Agents))
	for _, ac := range result.Agents {
		if ac.Err != nil {
			summary += fmt.Sprintf("  %-40s error: %s\n", ac.Project, ac.Err.Error())
			continue
		}
		summary += fmt.Sprintf("  %-40s %8d tokens  (task: %s)\n", ac.Project, ac.Report.TotalTokens, ac.Report.TaskName)
	}
	summary += fmt.Sprintf("\nTOTAL — one run of every agent: %d tokens (approx.)\n", result.TotalTokens)
	for _, c := range result.TotalCosts {
		summary += fmt.Sprintf("  %s/%s: $%.4f\n", c.Provider, c.Model, c.USD)
	}
	return summary
}
