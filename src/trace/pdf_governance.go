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
	b.WriteString("<h2>1. Executive summary</h2>")
	fmt.Fprintf(&b, "<p><b>%s</b></p>", d.GovernanceStatus)
	fmt.Fprintf(&b, "<p>Repository: %s (branch: %s) - Task: %s</p>", orNA(d.RepoURL), orNA(d.Branch), orTaskNone(d.TaskName))
	fmt.Fprintf(&b, "<p>Execution ID: %s - Repository state: %s</p>", d.ExecutionID, orNA(d.CommitHash))
	fmt.Fprintf(&b, "<p>%s: %s - %s: %s - %s: %s</p>", i18n.T("reports.agent_client_label"), orNA(d.AgentClient), i18n.T("reports.target_model_label"), orNA(d.TargetModel), i18n.T("reports.policy_author_label"), orNA(d.PolicyAuthor))

	// 2. Reduction Metrics
	b.WriteString("<h2>2. Reduction metrics</h2>")
	fmt.Fprintf(&b, "<li>Discovered (scanned): %s file(s), %s tok</li>", formatInt(c.DiscoveredFiles), formatInt(c.DiscoveredTokens))
	fmt.Fprintf(&b, "<li>Candidate: %s file(s), %s tok</li>", formatInt(c.CandidateFiles), formatInt(c.CandidateTokens))
	if d.Focus.Included != c.DiscoveredFiles {
		fmt.Fprintf(&b, "<p><i>Focus (%s files) is the scope considered before content-level discovery; the %s-file gap could not be read (binary/unreadable/empty) and is counted under Excluded.</i></p>", formatInt(d.Focus.Included), formatInt(d.Focus.Included-c.DiscoveredFiles))
	}

	// 3. Decisions Made
	b.WriteString("<h2>3. Decisions made</h2>")
	fmt.Fprintf(&b, "<li>Allowed: %s file(s), %s tok</li>", formatInt(c.AllowedFiles), formatInt(c.AllowedTokens))
	fmt.Fprintf(&b, "<li>Would sanitize (audit mode - no file rewritten): %s file(s), %s tok</li>", formatInt(c.SanitizedFiles), formatInt(c.SanitizedTokens))
	fmt.Fprintf(&b, "<li>Blocked: %s file(s), %s tok</li>", formatInt(c.BlockedFiles), formatInt(c.BlockedTokens))
	fmt.Fprintf(&b, "<li>Excluded: %s file(s), %s tok</li>", formatInt(c.ExcludedFiles), formatInt(c.ExcludedTokens))
	b.WriteString("<h3>Sendable context breakdown (audit mode - nothing written to disk)</h3>")
	fmt.Fprintf(&b, "<li>Repository context (scanned): %s tok</li>", formatInt(c.DiscoveredTokens))
	fmt.Fprintf(&b, "<li>Policy allowed (safe now): %s tok</li>", formatInt(c.AllowedTokens))
	fmt.Fprintf(&b, "<li>Would require sanitization: %s tok</li>", formatInt(c.SanitizedTokens))
	fmt.Fprintf(&b, "<li>Actually transformed/redacted: %s tok</li>", formatInt(imp.TokensActuallyRedacted))
	fmt.Fprintf(&b, "<p><b>Safe to send RIGHT NOW: %s tok (%s reduction)</b></p>", formatInt(c.SafeToSendNowTokens()), formatPct(c.SafeReductionPercent()))
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
	b.WriteString("<h2>4. Ranking &amp; selection</h2>")
	if d.TaskName == "" || len(d.RelevanceTop) == 0 {
		b.WriteString("<p>Task-specific ranking: <b>NOT APPLIED</b> - no task or query intent was provided.</p>")
		return b.String()
	}
	fmt.Fprintf(&b, "<p>Task: <b>%s</b></p>", d.TaskName)
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
	b.WriteString("<h2>5. Next actions</h2>")
	shown := 0
	for _, r := range d.DirBreakdown {
		if r.Percent < 1.0 || shown >= 5 {
			continue
		}
		fmt.Fprintf(&b, "<li>Excluding %s/ would save an estimated %s of the current context (%s tokens, %d file(s)).</li>", r.Dir, formatPct(r.Percent), formatInt(r.Tokens), r.Files)
		shown++
	}
	if shown == 0 {
		b.WriteString("<li>No single directory concentrates enough tokens to suggest an exclusion.</li>")
	}
	return b.String()
}

func exclusionHTML(d *Data) string {
	if len(d.ExclusionReasons) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<h3>What stayed out?</h3>")
	for _, r := range d.ExclusionReasons {
		fmt.Fprintf(&b, "<li>%s - %s file(s), %s tok</li>", r.Reason, formatInt(r.Files), formatInt(r.Tokens))
	}
	return b.String()
}

func policyHTML(d *Data) string {
	var b strings.Builder
	b.WriteString("<h3>Policy (supplementary)</h3>")
	fmt.Fprintf(&b, "<li>Policy source: %s</li>", orNA(d.PolicySource))
	fmt.Fprintf(&b, "<li>Policy version: %s</li>", orNA(d.PolicyVersion))
	b.WriteString("<li>Decision authority: Mova Context Policy Engine</li>")
	return b.String()
}

func modelCompatHTML(d *Data) string {
	if len(d.ModelCompat) == 0 {
		return ""
	}
	var b strings.Builder
	final := d.StateTotals.FinalSendableTokens()
	fmt.Fprintf(&b, "<h3>Model compatibility (supplementary)</h3><p>Final context: %s tokens</p>", formatInt(final))
	for _, r := range d.ModelCompat {
		window, fits := "-", "unknown (context_window not configured)"
		if r.ContextWindow > 0 {
			window = formatInt(r.ContextWindow)
			if r.Fits {
				fits = "yes"
			} else {
				fits = "no"
			}
		}
		fmt.Fprintf(&b, "<li>%s / %s - window: %s - fits: %s</li>", r.Provider, r.Model, window, fits)
	}
	b.WriteString("<p>Provider/model configuration may impose additional limits.</p>")
	return b.String()
}
