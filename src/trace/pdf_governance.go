// pdf_governance.go — the HTML fragment for context-report.pdf's
// Context Governance & Traceability Engine sections; the PDF-flavored
// twin of markdown_governance.go, built from the exact same Data
// fields so the .md and .pdf exports never disagree on a number.
package trace

import (
	"fmt"
	"strings"

	"mova.local/i18n"
)

// governanceHTML returns "" for a run with no governance state to
// show (see RenderGovernanceMarkdown's doc comment - same condition).
// Organized into 5 blocks: Executive Summary, Reduction Metrics,
// Decisions Made (incl. the audit-mode sendable breakdown and PII
// indicator reconciliation), Ranking & Selection, Next Actions - plus
// Policy/Model-Compatibility as supplementary info after them. The
// "Security & PII" block was REMOVED from this report by request -
// its data still lives in full in pii-audit-log.json (always written,
// see write_outputs.go), just not duplicated here.
func governanceHTML(d *Data) string {
	if d.StateTotals.DiscoveredFiles == 0 {
		return ""
	}
	c := d.StateTotals
	imp := d.SecurityImpact
	var b strings.Builder

	// 1. Executive Summary
	b.WriteString("<h2>1. " + i18n.T("reports.h_exec_summary") + "</h2>")
	fmt.Fprintf(&b, "<p><b>%s</b></p>", d.GovernanceStatus)
	fmt.Fprintf(&b, "<p>%s: %s (%s: %s) - %s: %s</p>", i18n.T("reports.f_repository"), orNA(d.RepoURL), i18n.T("reports.f_branch"), orNA(d.Branch), i18n.T("reports.f_task"), orTaskNone(d.TaskName))
	fmt.Fprintf(&b, "<p>%s: %s - %s: %s</p>", i18n.T("reports.execution_id"), d.ExecutionID, i18n.T("reports.repository_state"), orNA(d.CommitHash))
	fmt.Fprintf(&b, "<p>%s: %s - %s: %s - %s: %s</p>", i18n.T("reports.agent_client_label"), orNA(d.AgentClient), i18n.T("reports.target_model_label"), orNA(d.TargetModel), i18n.T("reports.policy_author_label"), orNA(d.PolicyAuthor))

	// 2. Reduction Metrics
	b.WriteString("<h2>2. " + i18n.T("reports.h_reduction_metrics") + "</h2>")
	fmt.Fprintf(&b, "<li>%s</li>", filesTokLine(i18n.T("reports.s_discovered_scanned"), c.DiscoveredFiles, c.DiscoveredTokens))
	fmt.Fprintf(&b, "<li>%s</li>", filesTokLine(i18n.T("reports.state_candidate"), c.CandidateFiles, c.CandidateTokens))
	if d.Focus.Included != c.DiscoveredFiles {
		fmt.Fprintf(&b, "<p><i>Focus (%s files) is the scope considered before content-level discovery; the %s-file gap could not be read (binary/unreadable/empty) and is counted under Excluded.</i></p>", formatInt(d.Focus.Included), formatInt(d.Focus.Included-c.DiscoveredFiles))
	}

	// 3. Decisions Made
	b.WriteString("<h2>3. " + i18n.T("reports.h_decisions") + "</h2>")
	fmt.Fprintf(&b, "<li>%s</li>", filesTokLine(i18n.T("reports.state_allowed"), c.AllowedFiles, c.AllowedTokens))
	fmt.Fprintf(&b, "<li>%s</li>", filesTokLine(i18n.T("reports.s_would_sanitize_audit"), c.SanitizedFiles, c.SanitizedTokens))
	fmt.Fprintf(&b, "<li>%s</li>", filesTokLine(i18n.T("reports.state_blocked"), c.BlockedFiles, c.BlockedTokens))
	fmt.Fprintf(&b, "<li>%s</li>", filesTokLine(i18n.T("reports.state_excluded"), c.ExcludedFiles, c.ExcludedTokens))
	b.WriteString("<h3>" + i18n.T("reports.h_sendable") + "</h3>")
	fmt.Fprintf(&b, "<li>"+i18n.T("reports.r_scanned")+": %s tok</li>", formatInt(c.DiscoveredTokens))
	fmt.Fprintf(&b, "<li>"+i18n.T("reports.r_allowed")+": %s tok</li>", formatInt(c.AllowedTokens))
	fmt.Fprintf(&b, "<li>"+i18n.T("reports.r_would_sanitize")+": %s tok</li>", formatInt(c.SanitizedTokens))
	fmt.Fprintf(&b, "<li>"+i18n.T("reports.r_transformed")+": %s tok</li>", formatInt(imp.TokensActuallyRedacted))
	fmt.Fprintf(&b, "<p><b>%s</b></p>", i18n.T("reports.safe_to_send_now", map[string]any{"tokens": formatInt(c.SafeToSendNowTokens()), "pct": formatPct(c.SafeReductionPercent())}))
	fmt.Fprintf(&b, "<p>Sendable AFTER sanitization is actually applied: %s tok (%s reduction) - not safe as-is until a real sanitization pass runs.</p>", formatInt(c.FinalSendableTokens()), formatPct(c.ReductionPercent()))
	totalIndicatorFiles := imp.FilesWithAnyIndicator
	fmt.Fprintf(&b, "<p><i>Files with a possible PII/secret indicator: %s = would_sanitize (%s) + blocked (%s) + detected-but-allowed (%s).</i></p>",
		formatInt(totalIndicatorFiles), formatInt(c.SanitizedFiles), formatInt(c.BlockedFiles), formatInt(imp.DetectedNotActioned))

	// 4. Ranking & Selection
	b.WriteString(relevanceHTML(d))

	// 6. Next Actions
	b.WriteString(exclusionHTML(d))
	b.WriteString(nextStepsHTML(d))

	b.WriteString(policyHTML(d))
	b.WriteString(modelCompatHTML(d))
	return b.String()
}

func orTaskNone(task string) string {
	if task == "" {
		return "none"
	}
	return task
}

// relevanceHTML is the PDF-flavored twin of renderRelevanceMarkdown.
func relevanceHTML(d *Data) string {
	var b strings.Builder
	b.WriteString("<h2>4. " + strings.ReplaceAll(i18n.T("reports.ranking_selection"), "&", "&amp;") + "</h2>")
	if d.TaskName == "" || len(d.RelevanceTop) == 0 {
		b.WriteString("<p>" + i18n.T("reports.task_not_applied") + "</p>")
		return b.String()
	}
	fmt.Fprintf(&b, "<p>%s: <b>%s</b></p>", i18n.T("reports.f_task"), d.TaskName)
	for _, r := range d.RelevanceTop {
		fmt.Fprintf(&b, "<li>#%d %s - score %.2f</li>", r.Rank, r.Path, r.Score)
	}
	return b.String()
}

// nextStepsHTML is the PDF-flavored twin of renderNextStepsMarkdown -
// same data, same 5-item/1%-threshold rule, so the .md and .pdf
// exports never disagree.
func nextStepsHTML(d *Data) string {
	if len(d.DirBreakdown) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<h2>5. " + i18n.T("reports.next_actions") + "</h2>")
	shown := 0
	for _, r := range d.DirBreakdown {
		if r.Percent < 1.0 || shown >= 5 {
			continue
		}
		fmt.Fprintf(&b, "<li>%s</li>", mdInlineToHTML(i18n.T("reports.n_exclude_saves", map[string]any{"dir": r.Dir, "pct": formatPct(r.Percent), "tok": formatInt(r.Tokens), "files": r.Files})))
		shown++
	}
	if shown == 0 {
		b.WriteString("<li>" + i18n.T("reports.n_none") + "</li>")
	}
	return b.String()
}

func exclusionHTML(d *Data) string {
	if len(d.ExclusionReasons) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<h3>" + i18n.T("reports.h_stayed_out") + "</h3>")
	for _, r := range d.ExclusionReasons {
		fmt.Fprintf(&b, "<li>%s - %s file(s), %s tok</li>", r.Reason, formatInt(r.Files), formatInt(r.Tokens))
	}
	return b.String()
}

func policyHTML(d *Data) string {
	var b strings.Builder
	b.WriteString("<h3>" + i18n.T("reports.h_policy") + " (" + i18n.T("reports.f_supplementary") + ")</h3>")
	fmt.Fprintf(&b, "<li>%s: %s</li>", i18n.T("reports.policy_source_label"), orNA(d.PolicySource))
	fmt.Fprintf(&b, "<li>%s: %s</li>", i18n.T("reports.policy_version_label"), orNA(d.PolicyVersion))
	b.WriteString("<li>" + i18n.T("reports.f_decision_authority") + ": Mova Context Policy Engine</li>")
	return b.String()
}

func modelCompatHTML(d *Data) string {
	if len(d.ModelCompat) == 0 {
		return ""
	}
	var b strings.Builder
	final := d.StateTotals.FinalSendableTokens()
	fmt.Fprintf(&b, "<h3>%s (%s)</h3><p>%s: %s tokens</p>", i18n.T("reports.h_model_compat"), i18n.T("reports.f_supplementary"), i18n.T("reports.f_final_context"), formatInt(final))
	for _, r := range d.ModelCompat {
		window, fits := "-", i18n.T("reports.v_fits_unknown")
		if r.ContextWindow > 0 {
			window = formatInt(r.ContextWindow)
			if r.Fits {
				fits = i18n.T("reports.v_yes")
			} else {
				fits = i18n.T("reports.v_no")
			}
		}
		fmt.Fprintf(&b, "<li>%s / %s - %s: %s - %s: %s</li>", r.Provider, r.Model, i18n.T("reports.c_window"), window, i18n.T("reports.c_fits"), fits)
	}
	b.WriteString("<p>Provider/model configuration may impose additional limits.</p>")
	return b.String()
}
