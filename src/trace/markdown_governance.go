// markdown_governance.go — the Context Governance & Traceability
// Engine sections of context-report.md: STATUS, CONTEXT DECISION,
// SECURITY FINDINGS & ACTIONS, WHAT ENTERED / WHAT STAYED OUT, POLICY,
// and MODEL COMPATIBILITY (see reporte.md for the shape this mirrors).
// Split out of markdown.go purely to respect the 300-line-per-file
// limit — RenderContextReportMarkdown (markdown.go) calls
// RenderGovernanceMarkdown once, right after the composition section.
package trace

import (
	"fmt"
	"strings"

	"mova.local/i18n"
)

// RenderGovernanceMarkdown returns "" when there is nothing to show
// (a project.json run that has not been wired into the per-file state
// machine yet — see AnalyzeLocal's doc comment) so callers can just
// concatenate its result unconditionally.
func RenderGovernanceMarkdown(d *Data) string {
	if d.StateTotals.DiscoveredFiles == 0 {
		return ""
	}
	var b strings.Builder
	c := d.StateTotals
	imp := d.SecurityImpact

	fmt.Fprintf(&b, "\n## %s\n\n**%s**\n\n%s: `%s` · %s: `%s` · %s: `%s`\n\n", i18n.T("reports.status"), d.GovernanceStatus, i18n.T("reports.execution_id"), d.ExecutionID, i18n.T("reports.repository_state"), orNA(d.CommitHash), i18n.T("reports.f_task"), taskOrNone(d.TaskName))
	fmt.Fprintf(&b, "%s: `%s` · %s: `%s` · %s: `%s`\n\n", i18n.T("reports.agent_client_label"), orNA(d.AgentClient), i18n.T("reports.target_model_label"), orNA(d.TargetModel), i18n.T("reports.policy_author_label"), orNA(d.PolicyAuthor))

	b.WriteString("## " + i18n.T("reports.context_decision") + "\n\n")
	fmt.Fprintf(&b, "| %s | %s | %s |\n|---|---:|---:|\n", i18n.T("reports.c_stage"), i18n.T("reports.c_files"), i18n.T("reports.c_tokens"))
	fmt.Fprintf(&b, "| "+i18n.T("reports.s_discovered_scanned")+" | %s | %s |\n", formatInt(c.DiscoveredFiles), formatInt(c.DiscoveredTokens))
	fmt.Fprintf(&b, "| "+i18n.T("reports.state_candidate")+" | %s | %s |\n", formatInt(c.CandidateFiles), formatInt(c.CandidateTokens))
	fmt.Fprintf(&b, "| "+i18n.T("reports.state_allowed")+" | %s | %s |\n", formatInt(c.AllowedFiles), formatInt(c.AllowedTokens))
	fmt.Fprintf(&b, "| "+i18n.T("reports.s_would_sanitize_audit")+" | %s | %s |\n", formatInt(c.SanitizedFiles), formatInt(c.SanitizedTokens))
	fmt.Fprintf(&b, "| "+i18n.T("reports.state_blocked")+" | %s | %s |\n", formatInt(c.BlockedFiles), formatInt(c.BlockedTokens))
	fmt.Fprintf(&b, "| "+i18n.T("reports.state_excluded")+" | %s | %s |\n\n", formatInt(c.ExcludedFiles), formatInt(c.ExcludedTokens))

	if d.Focus.Included != c.DiscoveredFiles {
		gap := d.Focus.Included - c.DiscoveredFiles
		fmt.Fprintf(&b, "_Focus (%s files) is the repository SCOPE considered before content-level discovery; Discovery (%s files) is what was actually analyzed as text/image. The %s-file gap could not be read (binary/unreadable/empty) and is counted under EXCLUDED above._\n\n", formatInt(d.Focus.Included), formatInt(c.DiscoveredFiles), formatInt(gap))
	}

	b.WriteString("### " + i18n.T("reports.h_sendable") + "\n\n")
	fmt.Fprintf(&b, "| | %s |\n|---|---:|\n", i18n.T("reports.c_tokens"))
	fmt.Fprintf(&b, "| "+i18n.T("reports.r_scanned")+" | %s |\n", formatInt(c.DiscoveredTokens))
	fmt.Fprintf(&b, "| "+i18n.T("reports.r_allowed")+" | %s |\n", formatInt(c.AllowedTokens))
	fmt.Fprintf(&b, "| "+i18n.T("reports.r_would_sanitize")+" | %s |\n", formatInt(c.SanitizedTokens))
	fmt.Fprintf(&b, "| "+i18n.T("reports.r_transformed")+" | %s |\n\n", formatInt(imp.TokensActuallyRedacted))
	fmt.Fprintf(&b, "**Safe to send RIGHT NOW: %s tok (%s reduction)** - only ALLOWED content.\n\n", formatInt(c.SafeToSendNowTokens()), formatPct(c.SafeReductionPercent()))
	fmt.Fprintf(&b, "Sendable AFTER sanitization is actually applied: %s tok (%s reduction) - WOULD_SANITIZE content is **not** safe to send as-is until a real sanitization pass runs (e.g. via a `project.json` Context Governance).\n\n", formatInt(c.FinalSendableTokens()), formatPct(c.ReductionPercent()))

	totalIndicatorFiles := imp.FilesWithAnyIndicator
	fmt.Fprintf(&b, "_Files with a possible PII/secret indicator: **%s** = would_sanitize (%s) + blocked (%s) + detected-but-allowed (%s, no active rule modifies that pattern - see `config/policy/security.json`)._\n\n",
		formatInt(totalIndicatorFiles), formatInt(c.SanitizedFiles), formatInt(c.BlockedFiles), formatInt(imp.DetectedNotActioned))

	if d.TaskName == "" {
		b.WriteString("_No task was provided (see `--task`): CANDIDATE equals every DISCOVERED file that passed the existing focus/ignore rules, so this reduction reflects security/policy exclusions only, not task relevance._\n\n")
	} else {
		fmt.Fprintf(&b, "_Task: **%s** — CANDIDATE was narrowed by BM25 relevance ranking (see the Ranking & Selection section below)._\n\n", d.TaskName)
	}

	b.WriteString(renderExclusionMarkdown(d))
	b.WriteString(renderPolicyMarkdown(d))
	b.WriteString(renderRelevanceMarkdown(d))
	b.WriteString(renderModelCompatMarkdown(d))
	b.WriteString(renderNextStepsMarkdown(d))
	return b.String()
}

// renderRelevanceMarkdown is the "Ranking & Selection" block —
// transparent, auditable exposure of --task's BM25 ranking (or an
// explicit "not applied" note with no task) so the effect of --task
// is never a black box.
func renderRelevanceMarkdown(d *Data) string {
	var b strings.Builder
	b.WriteString("## " + i18n.T("reports.ranking_selection") + "\n\n")
	if d.TaskName == "" || len(d.RelevanceTop) == 0 {
		b.WriteString(i18n.T("reports.task_not_applied") + "\n\n")
		return b.String()
	}
	fmt.Fprintf(&b, "%s", i18n.T("reports.task_label", map[string]any{"task": d.TaskName}))
	b.WriteString("| " + i18n.T("reports.c_rank") + " | " + i18n.T("reports.c_file") + " | " + i18n.T("reports.c_score") + " |\n|---:|---|---:|\n")
	for _, r := range d.RelevanceTop {
		fmt.Fprintf(&b, "| %d | %s | %.2f |\n", r.Rank, r.Path, r.Score)
	}
	b.WriteString("\n_Score = BM25 lexical relevance + a small structural/filename boost - see `docs/i18n/en/context-trace.md`'s FAQ for the exact formula. This is deterministic and explainable, not a semantic/embedding search._\n\n")
	return b.String()
}

// renderNextStepsMarkdown is context-report's "Actionable Next Steps"
// section: dynamic, data-driven suggestions (never invented) computed
// straight from d.DirBreakdown - the same table the console's "TOP
// DIRECTORIES BY TOKEN USAGE" and the suggested project.json's
// `exclude` list already use (see project_gen.go), so all three can
// never disagree about which directory is the biggest opportunity.
func renderNextStepsMarkdown(d *Data) string {
	if len(d.DirBreakdown) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## " + i18n.T("reports.h_next_steps") + "\n\n")
	shown := 0
	for _, r := range d.DirBreakdown {
		if r.Percent < 1.0 || shown >= 5 {
			continue
		}
		fmt.Fprintf(&b, "- %s\n", i18n.T("reports.n_exclude_saves", map[string]any{"dir": r.Dir, "pct": formatPct(r.Percent), "tok": formatInt(r.Tokens), "files": r.Files}))
		shown++
	}
	if shown == 0 {
		b.WriteString("- " + i18n.T("reports.n_none") + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

func renderExclusionMarkdown(d *Data) string {
	if len(d.ExclusionReasons) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## " + i18n.T("reports.h_stayed_out") + "\n\n")
	b.WriteString("| " + i18n.T("reports.c_reason") + " | " + i18n.T("reports.c_files") + " | " + i18n.T("reports.c_tokens") + " |\n|---|---:|---:|\n")
	for _, r := range d.ExclusionReasons {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", r.Reason, formatInt(r.Files), formatInt(r.Tokens))
	}
	b.WriteString("\n")
	return b.String()
}

func renderPolicyMarkdown(d *Data) string {
	var b strings.Builder
	b.WriteString("## " + i18n.T("reports.h_policy") + "\n\n")
	fmt.Fprintf(&b, "- %s: `%s`\n", i18n.T("reports.policy_source_label"), orNA(d.PolicySource))
	fmt.Fprintf(&b, "- %s: %s\n", i18n.T("reports.policy_version_label"), orNA(d.PolicyVersion))
	b.WriteString("- " + i18n.T("reports.f_decision_authority") + ": Mova Context Policy Engine\n\n")
	return b.String()
}

func renderModelCompatMarkdown(d *Data) string {
	if len(d.ModelCompat) == 0 {
		return ""
	}
	var b strings.Builder
	final := d.StateTotals.FinalSendableTokens()
	fmt.Fprintf(&b, "## %s\n\n%s: %s tokens\n\n", i18n.T("reports.h_model_compat"), i18n.T("reports.f_final_context"), formatInt(final))
	b.WriteString("| " + i18n.T("reports.c_provider") + " | " + i18n.T("reports.c_model") + " | " + i18n.T("reports.c_window") + " | " + i18n.T("reports.c_fits") + " |\n|---|---|---:|---|\n")
	for _, r := range d.ModelCompat {
		window := "-"
		fits := i18n.T("reports.v_fits_unknown")
		if r.ContextWindow > 0 {
			window = formatInt(r.ContextWindow)
			if r.Fits {
				fits = i18n.T("reports.v_yes")
			} else {
				fits = i18n.T("reports.v_no")
			}
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", r.Provider, r.Model, window, fits)
	}
	b.WriteString("\n_Provider/model configuration may impose additional limits._\n\n")
	return b.String()
}
