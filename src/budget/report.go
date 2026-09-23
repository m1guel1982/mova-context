// report.go renders the Report (estimate.go) into mova-budget-report.md.
// Every sentence goes through i18n.T (see config/lang/{es,en}.json's
// "budget" section) so the report follows config/lang/lang_active.json
// like every other user-facing surface — see PROJECT_JSON.md/COMMANDS.md
// for the multi-language contract. Plain jargon-free sentences either
// way: whoever reads a cost report may not read the rest of Mova
// Context's other output. Every section says plainly that this is an
// estimate, never an exact bill.
package budget

import (
	"fmt"
	"path/filepath"
	"strings"

	"mova.local/core"
	"mova.local/documents"
	"mova.local/i18n"
)

// RenderMarkdown turns a Report into the final mova-budget-report.md text.
func RenderMarkdown(r *Report) string {
	var b strings.Builder

	b.WriteString(i18n.T("budget.title") + "\n\n")

	b.WriteString(i18n.T("budget.section_project") + "\n\n")
	b.WriteString(i18n.T("budget.project_name", map[string]any{"name": r.ProjectName}) + "\n\n")
	b.WriteString(i18n.T("budget.task", map[string]any{"task": r.TaskName}) + "\n\n")
	b.WriteString(i18n.T("budget.final_tokens_sent", map[string]any{"count": r.TotalTokens}) + "\n\n")

	b.WriteString(i18n.T("budget.section_tokenization") + "\n\n")
	b.WriteString(i18n.T("budget.tool_used", map[string]any{"encoding": r.Encoding}) + "\n\n")
	b.WriteString(i18n.T("budget.tokenization_note1") + "\n\n")
	b.WriteString(i18n.T("budget.tokenization_note2") + "\n\n")

	b.WriteString(deduplicationSection(r))
	b.WriteString(sanitizerSection(r))
	b.WriteString(piiMaskingSection(r))
	b.WriteString(firewallSummarySection(r))

	b.WriteString(i18n.T("budget.section_token_cost_breakdown") + "\n\n")
	b.WriteString(i18n.T("budget.token_cost_breakdown_note") + "\n\n")
	b.WriteString(componentTable(r))
	b.WriteString("\n")
	b.WriteString(fileBreakdownSection(r))

	b.WriteString(focusSection(r))
	b.WriteString(cacheLayoutSection(r))
	b.WriteString(circuitBreakerSection(r))
	b.WriteString(budgetLimitSection(r))
	b.WriteString(historicalAccuracySection(r))

	b.WriteString(i18n.T("budget.section_important") + "\n\n")
	b.WriteString(i18n.T("budget.important_note") + "\n")

	return b.String()
}

// deduplicationSection reports paragraphs automatically removed across
// the whole context (see core.BuildContextSections) — a real, automatic
// optimization, not a suggestion the developer has to act on.
func deduplicationSection(r *Report) string {
	if r.DuplicatesRemoved == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(i18n.T("budget.section_deduplication") + "\n\n")
	b.WriteString(i18n.T("budget.deduplication_note", map[string]any{"count": r.DuplicatesRemoved}) + "\n\n")
	return b.String()
}

// focusSection reports the whole-repo-vs-focus comparison, only present
// when --focus was requested.
func focusSection(r *Report) string {
	if r.Focus == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(i18n.T("budget.section_context_optimization") + "\n\n")
	b.WriteString(i18n.T("budget.without_focus", map[string]any{"count": r.Focus.TokensWithoutFocus}) + "\n\n")
	b.WriteString(i18n.T("budget.with_focus", map[string]any{"count": r.Focus.TokensWithFocus}) + "\n\n")
	if r.Focus.SavingsPercent >= 0 {
		b.WriteString(i18n.T("budget.estimated_savings", map[string]any{"pct": fmt.Sprintf("%.1f", r.Focus.SavingsPercent)}) + "\n\n")
	} else {
		b.WriteString(i18n.T("budget.no_savings", map[string]any{"pct": fmt.Sprintf("%.1f", -r.Focus.SavingsPercent)}) + "\n\n")
	}
	b.WriteString(i18n.T("budget.focus_explainer") + "\n\n")
	return b.String()
}

// budgetLimitSection shows the configured "budget": {"max_tokens": N}
// ceiling (if any) and whether this run is within it — informational
// here; the actual enforcement that stops execution lives in limit.go's
// EnforceLimit, called before sending anything to a model.
func budgetLimitSection(r *Report) string {
	if r.MaxTokens <= 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(i18n.T("budget.section_budget_limit") + "\n\n")
	b.WriteString(i18n.T("budget.configured_limit", map[string]any{"count": r.MaxTokens}) + "\n\n")
	b.WriteString(i18n.T("budget.current_context", map[string]any{"count": r.TotalTokens}) + "\n\n")

	percent := (float64(r.TotalTokens) / float64(r.MaxTokens)) * 100
	diff := r.TotalTokens - r.MaxTokens
	b.WriteString(i18n.T("budget.usage_pct", map[string]any{"pct": fmt.Sprintf("%.1f", percent)}) + "\n\n")
	if diff > 0 {
		b.WriteString(i18n.T("budget.over_limit_by", map[string]any{"count": diff}) + "\n\n")
	} else {
		b.WriteString(i18n.T("budget.headroom_left", map[string]any{"count": -diff}) + "\n\n")
	}

	if r.OverBudget {
		b.WriteString(i18n.T("budget.status_over_budget") + "\n\n")
		b.WriteString(budgetRecommendations(r))
	} else {
		b.WriteString(i18n.T("budget.status_within_budget") + "\n\n")
	}

	b.WriteString(providerComparisonNote(r))
	return b.String()
}

// budgetRecommendations turns the per-component breakdown already
// computed above into concrete, ordered suggestions — pointing first at
// whichever component actually uses the most tokens, instead of a
// generic "reduce context" hint.
func budgetRecommendations(r *Report) string {
	var b strings.Builder
	b.WriteString(i18n.T("budget.recommendations_title") + "\n\n")

	biggest := biggestComponent(r)
	if biggest != nil && biggest.Tokens > 0 {
		b.WriteString(i18n.T("budget.recommendation_biggest", map[string]any{"name": biggest.Name, "count": biggest.Tokens}) + "\n")
	}
	b.WriteString(i18n.T("budget.recommendation_use_focus") + "\n")
	b.WriteString(i18n.T("budget.recommendation_remove_unused") + "\n")
	if r.DuplicatesRemoved == 0 {
		b.WriteString(i18n.T("budget.recommendation_check_duplicates") + "\n")
	}
	b.WriteString(i18n.T("budget.recommendation_raise_limit", map[string]any{"count": r.TotalTokens}) + "\n\n")
	return b.String()
}

func biggestComponent(r *Report) *ComponentBreakdown {
	var max *ComponentBreakdown
	for i := range r.Components {
		if max == nil || r.Components[i].Tokens > max.Tokens {
			max = &r.Components[i]
		}
	}
	return max
}

// providerComparisonNote explains, in plain language, why the same
// context can report a different token count per provider — asked for
// explicitly so a non-technical reader understands this isn't an error.
func providerComparisonNote(r *Report) string {
	var b strings.Builder
	b.WriteString(i18n.T("budget.provider_diff_title") + "\n\n")
	b.WriteString(i18n.T("budget.provider_diff_intro", map[string]any{"encoding": r.Encoding}) + "\n\n")
	b.WriteString(i18n.T("budget.provider_diff_openai") + "\n")
	b.WriteString(i18n.T("budget.provider_diff_gemini") + "\n")
	b.WriteString(i18n.T("budget.provider_diff_claude") + "\n\n")
	b.WriteString(i18n.T("budget.provider_diff_see_historical") + "\n\n")
	return b.String()
}

// historicalAccuracySection is the Feedback Loop: compares the local
// tiktoken-go estimate against real Cloud API usage recorded for this
// project (see history.go) — always present so the report explains, the
// first time, why it says "No historical data".
func historicalAccuracySection(r *Report) string {
	var b strings.Builder
	b.WriteString(i18n.T("budget.section_historical_accuracy") + "\n\n")
	b.WriteString(i18n.T("budget.historical_accuracy_intro") + "\n\n")
	b.WriteString(i18n.T("budget.historical_table_header") + "\n")
	for _, acc := range r.HistoricalAccuracy {
		if acc.HasData {
			b.WriteString(fmt.Sprintf("| %s | %+.1f%% |\n", acc.Provider, acc.DeviationPercent))
		} else {
			b.WriteString(fmt.Sprintf("| %s | %s |\n", acc.Provider, i18n.T("budget.historical_no_data")))
		}
	}
	b.WriteString("\n" + i18n.T("budget.historical_footer") + "\n\n")
	return b.String()
}

// componentTable builds the per-component breakdown table plus a TOTAL
// row — columns are the provider/model list from prices.json, taken from
// TotalCosts so the table always matches whatever config/prices.json
// currently declares, with zero hardcoded provider names. The "TOTAL"
// row label is kept as a plain identifier rather than routed through
// i18n — same rule markdown.go documents for context-report.md: a
// table's structural labels stay stable so a mixed-language repo of
// generated reports is still diffable/greppable, while the column
// header (budget.component_col_header) and every sentence of PROSE
// around this table is fully translated.
func componentTable(r *Report) string {
	var b strings.Builder
	b.WriteString(i18n.T("budget.component_col_header"))
	for _, c := range r.TotalCosts {
		fmt.Fprintf(&b, " %s %s (USD) |", c.Provider, c.Model)
	}
	b.WriteString("\n|---|---|")
	for range r.TotalCosts {
		b.WriteString("---|")
	}
	b.WriteString("\n")

	for _, comp := range r.Components {
		fmt.Fprintf(&b, "| %s | %d |", comp.Name, comp.Tokens)
		for _, c := range comp.Costs {
			fmt.Fprintf(&b, " $%.4f |", c.USD)
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "| **TOTAL** | **%d** |", r.TotalTokens)
	for _, c := range r.TotalCosts {
		fmt.Fprintf(&b, " **$%.4f** |", c.USD)
	}
	b.WriteString("\n")

	if r.CLPRate > 0 && len(r.TotalCosts) > 0 {
		b.WriteString("\n" + i18n.T("budget.approx_total_clp", map[string]any{
			"rate": fmt.Sprintf("%.0f", r.CLPRate), "clp": fmt.Sprintf("%.0f", r.TotalCosts[0].CLP),
			"provider": r.TotalCosts[0].Provider, "model": r.TotalCosts[0].Model,
		}) + "\n")
	}
	return b.String()
}

// BudgetReportPath resolves where mova-budget-report.md is written for
// this project — project.json's "budget_path" (see "11. budget_path" in
// the spec this implements), or projects/<project>/mova-budget-report.md
// by default — the same location config/prices.json's old "report_path"
// pointed existing example projects at, so removing "report_path" from
// config/prices.json (see prices.go) changes no project that didn't
// already rely on a custom path.
func BudgetReportPath(root, project string, proj *core.Project) string {
	if proj != nil && proj.BudgetPath != "" {
		// Cross-platform absolute check (documents.IsAbsCrossPlatform)
		// instead of plain filepath.IsAbs — same fix as core.MemoryPath,
		// so a Windows/UNC/Unix absolute path in budget_path is honored
		// regardless of which OS built the running binary.
		if documents.IsAbsCrossPlatform(proj.BudgetPath) {
			if normalized, err := documents.NormalizeAbsPath(proj.BudgetPath); err == nil {
				return normalized
			}
			return proj.BudgetPath
		}
		return filepath.Join(root, proj.BudgetPath)
	}
	return filepath.Join(root, "projects", project, "mova-budget-report.md")
}

// WriteReport renders the report and writes it to BudgetReportPath,
// reusing documents.WriteFile — the same writer every other create/edit
// tool in Mova Context uses, so the report gets the same directory-
// creation behavior as any other generated file. Returns the resolved
// path it wrote to.
func WriteReport(root, project string, proj *core.Project, report *Report) (string, error) {
	content := RenderMarkdown(report)
	path := BudgetReportPath(root, project, proj)
	if err := documents.WriteFile(path, content); err != nil {
		return "", err
	}
	return path, nil
}
