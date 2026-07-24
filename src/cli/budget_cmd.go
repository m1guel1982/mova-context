// budget_cmd.go — `mova budget [project] [task] [--focus]`.
//
// Estima el costo en tokens/USD del contexto real de un proyecto (el
// mismo que arma `mova run`: agents+skills+prompt+focus+memory — ver
// mova.local/budget y core.BuildContextSections), lo desglosa por
// componente, y escribe mova-budget-report.md. --focus además compara el
// repo completo sin filtrar contra solo lo que `focus` selecciona.
//
// Todo el cálculo es local (tiktoken-go embebido): no llama ningún LLM ni
// API externa, no envía una sola línea del repo fuera de esta máquina.
package main

import (
	"fmt"

	"mova.local/budget"
	"mova.local/core"
)

func runBudget(root, project, task string, withFocus bool) {
	fa := core.NewFileAdapter(root)
	proj, err := fa.GetProject(project)
	must(err)
	adapter := newAdapter(root, proj)

	report, err := budget.BuildReport(adapter, root, project, task, withFocus)
	must(err)

	prices, err := budget.LoadPrices(root)
	must(err)
	path, err := budget.WriteReport(root, prices, report)
	must(err)

	consolePrint(fmt.Sprintf("✓ mova-budget-report.md generated: %s\n\n", path))
	consolePrint(fmt.Sprintf("Total tokens: %d (%s)\n", report.TotalTokens, report.Encoding))
	for _, c := range report.TotalCosts {
		consolePrint(fmt.Sprintf("  %s/%s: $%.4f %s\n", c.Provider, c.Model, c.USD, report.Currency))
	}
	if report.Focus != nil {
		consolePrint(fmt.Sprintf("\nFocus savings: %.1f%% fewer tokens (%d → %d)\n",
			report.Focus.SavingsPercent, report.Focus.TokensWithoutFocus, report.Focus.TokensWithFocus))
	}
}
