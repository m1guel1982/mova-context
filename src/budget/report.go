// report.go renders the Report (estimate.go) into mova-budget-report.md.
// The report content is deliberately English-only, written in plain,
// jargon-free sentences — whoever reads a cost report may not read the
// rest of Mova Context's other output. Every section says plainly that
// this is an estimate, never an exact bill.
package budget

import (
	"fmt"
	"path/filepath"
	"strings"

	"mova.local/core"
	"mova.local/documents"
)

// RenderMarkdown turns a Report into the final mova-budget-report.md text.
func RenderMarkdown(r *Report) string {
	var b strings.Builder

	b.WriteString("# Mova Budget Report\n\n")

	b.WriteString("## Project\n\n")
	b.WriteString(fmt.Sprintf("Project name: %s\n\n", r.ProjectName))
	b.WriteString(fmt.Sprintf("Task: %s\n\n", r.TaskName))

	b.WriteString("## Tokenization\n\n")
	b.WriteString(fmt.Sprintf("Tool used: tiktoken-go (encoding: %s)\n\n", r.Encoding))
	b.WriteString("Token counts are a local estimate computed with tiktoken-go — the same open-source library OpenAI itself publishes. Every calculation in this report runs on this machine: nothing in your project is sent anywhere to produce it.\n\n")
	b.WriteString("For OpenAI models, this count is typically an exact match to what the OpenAI API bills you for. For Claude and Gemini, no official local tokenizer is publicly available, so this report reuses the same encoding as a close approximation — real counts from those providers are usually very close, but can differ, especially for non-English text or dense code. See \"Historical Token Accuracy\" below for this project's own measured difference.\n\n")

	b.WriteString(deduplicationSection(r))

	b.WriteString("## Token & Cost Breakdown\n\n")
	b.WriteString("This is where your token budget is actually going — one row per piece of the context declared in project.json (agents, skills, prompt, focus, memory), plus fixed engine overhead. Use it to see, in both tokens and dollars, exactly what is worth trimming first.\n\n")
	b.WriteString(componentTable(r))
	b.WriteString("\n")

	b.WriteString(focusSection(r))
	b.WriteString(budgetLimitSection(r))
	b.WriteString(historicalAccuracySection(r))

	b.WriteString("## Important\n\n")
	b.WriteString("These are estimates based on token counts computed locally and on the prices configured in config/prices.json. Real costs can vary depending on provider, model, caching, discounts, and commercial policies. This report is a tool to help you see where your token budget goes and what to optimize — it does not replace checking your actual invoice with each provider, and it is not a guarantee of exact pricing.\n")

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
	b.WriteString("## Deduplication\n\n")
	b.WriteString(fmt.Sprintf("Mova Context automatically removed %d duplicated paragraph(s) that appeared more than once across agents, skills, prompt, focus, and memory. This happens automatically on every run — no configuration needed, and no content is ever summarized or reworded, only exact repeats are removed.\n\n", r.DuplicatesRemoved))
	return b.String()
}

// focusSection reports the whole-repo-vs-focus comparison, only present
// when --focus was requested.
func focusSection(r *Report) string {
	if r.Focus == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Context Optimization\n\n")
	b.WriteString(fmt.Sprintf("Without focus (entire repository):\nTokens: %d\n\n", r.Focus.TokensWithoutFocus))
	b.WriteString(fmt.Sprintf("With focus (only the files selected in project.json):\nTokens: %d\n\n", r.Focus.TokensWithFocus))
	if r.Focus.SavingsPercent >= 0 {
		b.WriteString(fmt.Sprintf("Estimated savings: %.1f%% fewer tokens\n\n", r.Focus.SavingsPercent))
	} else {
		b.WriteString(fmt.Sprintf("No savings here: focus adds about %.1f%% formatting overhead instead. This happens when focus already covers the entire repository — there is nothing left to exclude. Savings appear once focus is narrowed to only the files that are actually relevant to the task, on a repository with more files than focus selects.\n\n", -r.Focus.SavingsPercent))
	}
	b.WriteString("Focus is not automatic compression — nothing is summarized or rewritten. Focus simply gives the developer full control over exactly which files are sent to the model, instead of dumping the entire repository. Benefits: lower cost, less noise, better precision, more privacy, and more control over the project.\n\n")
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
	b.WriteString("## Budget Limit\n\n")
	b.WriteString(fmt.Sprintf("Configured limit: %d tokens\n\n", r.MaxTokens))
	b.WriteString(fmt.Sprintf("Current context: %d tokens\n\n", r.TotalTokens))

	percent := (float64(r.TotalTokens) / float64(r.MaxTokens)) * 100
	diff := r.TotalTokens - r.MaxTokens
	b.WriteString(fmt.Sprintf("Usage: %.1f%% of the configured limit\n\n", percent))
	if diff > 0 {
		b.WriteString(fmt.Sprintf("Over the limit by: %d tokens\n\n", diff))
	} else {
		b.WriteString(fmt.Sprintf("Headroom left: %d tokens\n\n", -diff))
	}

	if r.OverBudget {
		b.WriteString("Status: OVER BUDGET. Sending this context to a model is blocked until it fits — this is enforced automatically on every `mova run`, `mova chat`, and MCP/HTTP call, never just a warning (see mova.local/budget.EnforceLimit).\n\n")
		b.WriteString(budgetRecommendations(r))
	} else {
		b.WriteString("Status: within budget.\n\n")
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
	b.WriteString("Recommendations:\n\n")

	biggest := biggestComponent(r)
	if biggest != nil && biggest.Tokens > 0 {
		b.WriteString(fmt.Sprintf("- %q is the largest piece of this context (%d tokens) — trim it first in project.json.\n", biggest.Name, biggest.Tokens))
	}
	b.WriteString("- Use `--focus` (or narrow an existing \"focus\" list) so only the files actually needed for this task are included.\n")
	b.WriteString("- Remove agents/skills from project.json's \"use\" lists that this task does not need.\n")
	if r.DuplicatesRemoved == 0 {
		b.WriteString("- Check agents/skills/prompt for repeated paragraphs — Mova Context removes exact duplicates automatically, but only what it can find.\n")
	}
	b.WriteString(fmt.Sprintf("- Raise \"budget\": {\"max_tokens\": %d} in project.json (or the task) only if this context size is genuinely intended.\n\n", r.TotalTokens))
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
	b.WriteString("### Why token counts differ between providers\n\n")
	b.WriteString(fmt.Sprintf("This report uses one encoding (%s, via tiktoken-go) for every provider, because Claude and Gemini do not publish a local tokenizer. In practice:\n\n", r.Encoding))
	b.WriteString("- OpenAI/GPT: this estimate is normally an exact or near-exact match, since tiktoken-go is OpenAI's own tokenizer.\n")
	b.WriteString("- Google Gemini: typically close, small differences come from how Gemini splits certain punctuation, code, and non-English text.\n")
	b.WriteString("- Anthropic Claude: usually the largest gap, since Claude uses its own (unpublished) tokenizer, which tends to segment text somewhat differently.\n\n")
	b.WriteString("See \"Historical Token Accuracy\" below — once real API calls have been recorded for this project, the deviation for each provider is measured directly instead of assumed.\n\n")
	return b.String()
}

// historicalAccuracySection is the Feedback Loop: compares the local
// tiktoken-go estimate against real Cloud API usage recorded for this
// project (see history.go) — always present so the report explains, the
// first time, why it says "No historical data".
func historicalAccuracySection(r *Report) string {
	var b strings.Builder
	b.WriteString("## Historical Token Accuracy\n\n")
	b.WriteString("Mova Budget compares local token estimation (tiktoken-go) with real cloud API usage collected from this project.\n\n")
	b.WriteString("| Provider | Average deviation |\n|---|---|\n")
	for _, acc := range r.HistoricalAccuracy {
		if acc.HasData {
			b.WriteString(fmt.Sprintf("| %s | %+.1f%% |\n", acc.Provider, acc.DeviationPercent))
		} else {
			b.WriteString(fmt.Sprintf("| %s | No historical data |\n", acc.Provider))
		}
	}
	b.WriteString("\nHistorical accuracy is automatically calibrated using previous cloud requests from this project. Actual costs may vary depending on provider billing policies and tokenizer updates.\n\n")
	return b.String()
}

// componentTable builds the per-component breakdown table plus a TOTAL
// row — columns are the provider/model list from prices.json, taken from
// TotalCosts so the table always matches whatever config/prices.json
// currently declares, with zero hardcoded provider names.
func componentTable(r *Report) string {
	var b strings.Builder
	b.WriteString("| Component | Tokens |")
	for _, c := range r.TotalCosts {
		b.WriteString(fmt.Sprintf(" %s %s (USD) |", c.Provider, c.Model))
	}
	b.WriteString("\n|---|---|")
	for range r.TotalCosts {
		b.WriteString("---|")
	}
	b.WriteString("\n")

	for _, comp := range r.Components {
		b.WriteString(fmt.Sprintf("| %s | %d |", comp.Name, comp.Tokens))
		for _, c := range comp.Costs {
			b.WriteString(fmt.Sprintf(" $%.4f |", c.USD))
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("| **TOTAL** | **%d** |", r.TotalTokens))
	for _, c := range r.TotalCosts {
		b.WriteString(fmt.Sprintf(" **$%.4f** |", c.USD))
	}
	b.WriteString("\n")

	if r.CLPRate > 0 && len(r.TotalCosts) > 0 {
		b.WriteString(fmt.Sprintf("\nApproximate total in CLP (exchange rate %.0f): $%.0f CLP (using %s %s)\n",
			r.CLPRate, r.TotalCosts[0].CLP, r.TotalCosts[0].Provider, r.TotalCosts[0].Model))
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
		if filepath.IsAbs(proj.BudgetPath) {
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
