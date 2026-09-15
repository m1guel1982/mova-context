// markdown.go — Markdown content for context-report.md and
// budget-report.md (the two text OUTPUT artifacts — see COMMANDS.md §
// context-trace). Never rewrites mova-budget-report.md (the
// "canonical" report from `mova budget`, see budget/report.go) — this
// is its own artifact, meant to sit alongside the PDF, not replace it.
// Table/section headers here are still plain English by design (a
// generated Markdown file is often committed to a repo or shared
// outside the chat UI, where a mixed-language document reads worse
// than a consistently-English one) — but any PROSE sentence long
// enough to actually explain something to a person (see
// reports.cost_basis_note) goes through i18n.T like every other
// user-facing surface, since that's exactly the kind of text a
// Spanish-speaking user benefits from reading in their own language.
package trace

import (
	"fmt"
	"strings"

	"mova.local/i18n"
)

// RenderContextReportMarkdown builds the executive context report:
// token breakdown per layer and Context Governance status.
func RenderContextReportMarkdown(d *Data) string {
	var b strings.Builder
	b.WriteString("# Context Report - Mova Context Trace\n\n")

	if d.IsRemote {
		fmt.Fprintf(&b, "**Repository:** %s\n\n**Branch:** %s\n\n", d.RepoURL, orNA(d.Branch))
	} else {
		fmt.Fprintf(&b, "**Project:** %s\n\n**Task:** %s\n\n**project.json:** `%s`\n\n", d.ProjectName, d.TaskName, d.ProjectJSONPath)
	}
	fmt.Fprintf(&b, "**Agent:** `%s`  ·  **Target model:** `%s`  ·  **Policy author:** `%s`\n\n", orNA(d.AgentClient), orNA(d.TargetModel), orNA(d.PolicyAuthor))

	if len(d.Components) > 0 {
		b.WriteString("## Context composition\n\n| Layer | Tokens |\n|---|---:|\n")
		for _, c := range d.Components {
			fmt.Fprintf(&b, "| %s | %s tok |\n", c.Name, formatInt(c.Tokens))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("## Context composition\n\nNo active `project.json` - the whole repository was scanned.\n\n")
	}
	fmt.Fprintf(&b, "%s\n\n", TotalTokensLine(d.TotalTokens, d.Encoding))

	fmt.Fprintf(&b, "## Focus\n\n- Files included: **%d**\n- Files excluded: **%d**\n", d.Focus.Included, d.Focus.Excluded)
	if len(d.Focus.ExcludedSuggestion) > 0 {
		fmt.Fprintf(&b, "- Exclude suggestion: %s\n", strings.Join(d.Focus.ExcludedSuggestion, ", "))
	}

	if len(d.DirBreakdown) > 0 {
		b.WriteString("\n")
		b.WriteString(renderDirBreakdownMarkdown(d))
	}

	b.WriteString(RenderGovernanceMarkdown(d))

	b.WriteString("\n## Context Governance\n\n")
	b.WriteString(renderFirewallTable(d.Firewall))

	if d.MaxTokens > 0 {
		pct := float64(d.TotalTokens) / float64(d.MaxTokens) * 100
		fmt.Fprintf(&b, "\n## Budget\n\n%s / %s tokens (%s)\n", formatInt(d.TotalTokens), formatInt(d.MaxTokens), formatPct(pct))
	} else {
		b.WriteString("\n## Budget\n\nN/A (no active `project.json` or no `budget.max_tokens` configured)\n")
	}

	b.WriteString("\n## Estimated input cost\n\n")
	fmt.Fprintf(&b, "%s", i18n.T("reports.cost_basis_note", map[string]any{"tokens": formatInt(costBasisTokens(d))}))
	b.WriteString("| Provider | Model | Theoretical / Estimated Input Cost (USD) |\n|---|---|---:|\n")
	for _, c := range d.Costs {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", c.Provider, c.Model, FormatCost(c.USD))
	}
	b.WriteString("| Local | Ollama / LM Studio / vLLM | $0 (local execution) |\n")

	if len(d.Findings) > MaxPDFSecurityFindings {
		fmt.Fprintf(&b, "\n**Full detailed audit log (%d files) exported to `pii-audit-log.json`**\n", len(d.Findings))
	}
	b.WriteString("\n---\n\nMova Context Trace - Observable. Governed. Auditable.\n\nThis report is a technical analysis. It is not a legal, privacy, or security certification.\n")

	return b.String()
}

// renderDirBreakdownMarkdown is the "plus" table the person asked
// for: which top-level directories are consuming the most tokens, how
// many files each one has, and what share of the total that is - so
// it's obvious at a glance where excluding a folder would help the
// most, with a TOTAL row closing the table.
func renderDirBreakdownMarkdown(d *Data) string {
	var b strings.Builder
	b.WriteString("## Top directories by token usage\n\n")
	b.WriteString("| Directory | Tokens | Files | % of total |\n|---|---:|---:|---:|\n")
	for _, r := range d.DirBreakdown {
		fmt.Fprintf(&b, "| %s | %s | %d | %s |\n", r.Dir, formatInt(r.Tokens), r.Files, formatPct(r.Percent))
	}
	fmt.Fprintf(&b, "| **TOTAL** | **%s** | **%d** | **100.0%%** |\n", formatInt(d.TotalTokens), d.Focus.Included)
	return b.String()
}

// costRowsBudgetSection used to live here as RenderBudgetReportMarkdown,
// a second, separate OUTPUT artifact (budget-report.md). Rule 4 of the
// context-trace governance spec requires EXACTLY two physical output
// artifacts (context-report.{md,pdf} and context-diagram.png) - so its
// content (the estimated-cost table) was folded directly into
// RenderContextReportMarkdown's "Estimated input cost" section above,
// and this second file is no longer written (see write_outputs.go).

func renderFirewallTable(f FirewallStatus) string {
	var b strings.Builder
	b.WriteString("| Stage | Status |\n|---|---|\n")
	fmt.Fprintf(&b, "| Sanitizer | %s |\n", onOff(f.SanitizerOn))
	piiCell := onOff(f.PIIMaskingOn)
	if f.PIIWarning != "" {
		piiCell += "  \n" + f.PIIWarning
	}
	fmt.Fprintf(&b, "| PII Masking | %s |\n", piiCell)
	fmt.Fprintf(&b, "| Cache Layout Guard | %s |\n", onOff(f.CacheGuardOn))
	fmt.Fprintf(&b, "| Circuit Breaker | %s |\n", onOff(f.CircuitBreakerOn))
	return b.String()
}
