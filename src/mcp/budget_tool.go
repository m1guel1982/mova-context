// budget_tool.go — expone `mova budget` como la tool MCP "estimate_budget",
// reachable idénticamente desde stdio y HTTP (mismo Process(), ver
// server.go) — la misma mova.local/budget.BuildReport que usa el comando
// CLI y el slash command /budget del chat REPL. Una sola implementación,
// tres puertas de entrada.
package mcp

import (
	"fmt"
	"strconv"

	"mova.local/budget"
	"mova.local/core"
)

func budgetTool(adapter core.Adapter, root string, args map[string]any) (string, error) {
	project := str(args, "project")
	task := str(args, "task")
	withFocus, _ := strconv.ParseBool(str(args, "focus"))

	report, err := budget.BuildReport(adapter, root, project, task, withFocus)
	if err != nil {
		return "", err
	}
	proj, err := adapter.GetProject(project)
	if err != nil {
		return "", err
	}
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
