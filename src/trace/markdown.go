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
	b.WriteString("# " + i18n.T("reports.h_report_title") + "\n\n")

	if d.IsRemote {
		fmt.Fprintf(&b, "**%s:** %s\n\n**%s:** %s\n\n", i18n.T("reports.f_repository"), d.RepoURL, i18n.T("reports.f_branch"), orNA(d.Branch))
	} else {
		fmt.Fprintf(&b, "**%s:** %s\n\n**%s:** %s\n\n**project.json:** `%s`\n\n", i18n.T("reports.f_project"), d.ProjectName, i18n.T("reports.f_task"), d.TaskName, d.ProjectJSONPath)
	}
	fmt.Fprintf(&b, "**%s:** `%s`  ·  **%s:** `%s`  ·  **%s:** `%s`\n\n", i18n.T("reports.agent_client_label"), orNA(d.AgentClient), i18n.T("reports.target_model_label"), orNA(d.TargetModel), i18n.T("reports.policy_author_label"), orNA(d.PolicyAuthor))
	fmt.Fprintf(&b, "**%s:** `%s`  ·  **%s:** `%s`\n\n", i18n.T("reports.policy_source_label"), orNA(d.PolicySource), i18n.T("reports.policy_version_label"), orNA(d.PolicyVersion))

	if len(d.Components) > 0 {
		b.WriteString("## " + i18n.T("reports.h_composition") + "\n\n| " + i18n.T("reports.c_layer") + " | " + i18n.T("reports.c_tokens") + " |\n|---|---:|\n")
		for _, c := range d.Components {
			fmt.Fprintf(&b, "| %s | %s tok |\n", c.Name, formatInt(c.Tokens))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("## " + i18n.T("reports.h_composition") + "\n\n" + i18n.T("reports.no_active_project_scanned") + "\n\n")
	}
	fmt.Fprintf(&b, "%s\n\n", TotalTokensLine(d.TotalTokens, d.Encoding))

	fmt.Fprintf(&b, "## %s\n\n- %s: **%d**\n- %s: **%d**\n", i18n.T("reports.h_focus"), i18n.T("reports.f_files_included"), d.Focus.Included, i18n.T("reports.f_files_excluded"), d.Focus.Excluded)
	if len(d.Focus.ExcludedSuggestion) > 0 {
		fmt.Fprintf(&b, "- %s: %s\n", i18n.T("reports.f_exclude_suggestion"), strings.Join(d.Focus.ExcludedSuggestion, ", "))
	}

	if len(d.DirBreakdown) > 0 {
		b.WriteString("\n")
		b.WriteString(renderDirBreakdownMarkdown(d))
	}

	b.WriteString(RenderGovernanceMarkdown(d))

	b.WriteString("\n## " + i18n.T("reports.h_governance") + "\n\n")
	b.WriteString(renderFirewallTable(d.Firewall))

	if d.MaxTokens > 0 {
		pct := float64(d.TotalTokens) / float64(d.MaxTokens) * 100
		fmt.Fprintf(&b, "\n## %s\n\n%s / %s tokens (%s)\n", i18n.T("reports.h_budget"), formatInt(d.TotalTokens), formatInt(d.MaxTokens), formatPct(pct))
	} else {
		b.WriteString("\n## " + i18n.T("reports.h_budget") + "\n\n" + i18n.T("reports.budget_not_configured_full") + "\n")
	}

	b.WriteString("\n## " + i18n.T("reports.h_cost") + "\n\n")
	fmt.Fprintf(&b, "%s", i18n.T("reports.cost_basis_note", map[string]any{"tokens": formatInt(costBasisTokens(d))}))
	b.WriteString("| " + i18n.T("reports.c_provider") + " | " + i18n.T("reports.c_model") + " | " + i18n.T("reports.c_cost") + " |\n|---|---|---:|\n")
	for _, c := range d.Costs {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", c.Provider, c.Model, FormatCost(c.USD))
	}
	b.WriteString("| Local | Ollama / LM Studio / vLLM | " + i18n.T("reports.v_local_exec") + " |\n")

	if len(d.Findings) > MaxPDFSecurityFindings {
		fmt.Fprintf(&b, "\n%s\n", i18n.T("reports.full_audit_log_exported", map[string]any{"count": len(d.Findings)}))
	}
	b.WriteString("\n---\n\n" + i18n.T("reports.trace_tagline") + "\n\n" + i18n.T("reports.report_disclaimer") + "\n")

	return b.String()
}

// renderDirBreakdownMarkdown is the "plus" table the person asked
// for: which top-level directories are consuming the most tokens, how
// many files each one has, and what share of the total that is - so
// it's obvious at a glance where excluding a folder would help the
// most, with a TOTAL row closing the table.
func renderDirBreakdownMarkdown(d *Data) string {
	var b strings.Builder
	b.WriteString("## " + i18n.T("reports.h_top_dirs") + "\n\n")
	b.WriteString("| " + i18n.T("reports.c_directory") + " | " + i18n.T("reports.c_tokens") + " | " + i18n.T("reports.c_files") + " | " + i18n.T("reports.c_pct_total") + " |\n|---|---:|---:|---:|\n")
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
	b.WriteString("| " + i18n.T("reports.c_stage") + " | " + i18n.T("reports.c_status") + " |\n|---|---|\n")
	fmt.Fprintf(&b, "| "+i18n.T("reports.f_sanitizer")+" | %s |\n", onOff(f.SanitizerOn))
	piiCell := onOff(f.PIIMaskingOn)
	if f.PIIWarning != "" {
		piiCell += "  \n" + f.PIIWarning
	}
	fmt.Fprintf(&b, "| "+i18n.T("reports.f_pii_masking")+" | %s |\n", piiCell)
	fmt.Fprintf(&b, "| "+i18n.T("reports.f_cache_guard")+" | %s |\n", onOff(f.CacheGuardOn))
	fmt.Fprintf(&b, "| "+i18n.T("reports.f_circuit_breaker")+" | %s |\n", onOff(f.CircuitBreakerOn))
	return b.String()
}
