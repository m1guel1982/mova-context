// pdf_discovery.go — the 2-page executive PDF for a discovery/audit
// run (see pdf.go's WriteContextReportPDF dispatch). The underlying
// PDF writer (mova.local/documents) has no page-break control - page
// count is a direct function of how much text is generated (see
// documents/pdf.go's fixed linesPerPage=50) - so hitting the 2-page
// target means being economical with content here, not fighting the
// renderer. Exhaustive detail (every finding, every excluded file)
// lives in pii-audit-log.json; this PDF is deliberately an executive
// summary, never a JSON dump.
package trace

import (
	"fmt"
	"strings"

	"mova.local/i18n"
)

func discoveryReportHTML(d *Data) string {
	var b strings.Builder
	c := d.StateTotals

	// ── PAGE 1: EXECUTIVE / CONTEXT RESULT ──────────────────────────
	b.WriteString("<h1>MOVA CONTEXT TRACE</h1>")
	fmt.Fprintf(&b, "<p>%s (branch: %s) - Task: %s</p>", orNA(d.RepoURL), orNA(d.Branch), orTaskNone(d.TaskName))
	fmt.Fprintf(&b, "<p>Execution ID: %s - Commit: %s</p>", d.ExecutionID, orNA(d.CommitHash))

	b.WriteString("<h2>Context reduction</h2>")
	fmt.Fprintf(&b, "<li>Scanned: %s files, %s tok</li>", formatInt(c.DiscoveredFiles), formatInt(c.DiscoveredTokens))
	fmt.Fprintf(&b, "<li>Candidate: %s files, %s tok (%s reduction)</li>", formatInt(c.CandidateFiles), formatInt(c.CandidateTokens), formatPct(c.ReductionPercent()))

	b.WriteString("<h2>Selection summary</h2>")
	fmt.Fprintf(&b, "<li>Candidate: %s files / %s tok</li>", formatInt(c.CandidateFiles), formatInt(c.CandidateTokens))
	if excl := excludedByTask(d); excl != nil {
		fmt.Fprintf(&b, "<li>Excluded by task: %s files / %s tok</li>", formatInt(excl.Files), formatInt(excl.Tokens))
	}

	b.WriteString("<h2>Top relevant files</h2>")
	if len(d.RelevanceTop) == 0 {
		b.WriteString("<p>No task given - ranking not applied.</p>")
	} else {
		max := 10
		if len(d.RelevanceTop) < max {
			max = len(d.RelevanceTop)
		}
		for _, r := range d.RelevanceTop[:max] {
			fmt.Fprintf(&b, "<li>#%d %s - %.2f</li>", r.Rank, truncatePath(r.Path, 58), r.Score)
		}
	}

	b.WriteString("<h2>Cost estimate</h2>")
	b.WriteString(costEstimateHTML(d))

	// ── PAGE 2: GOVERNANCE / SECURITY / TRACEABILITY ────────────────
	b.WriteString("<h2>Policy decisions</h2>")
	fmt.Fprintf(&b, "<li>Allowed: %s files / %s tok</li>", formatInt(c.AllowedFiles), formatInt(c.AllowedTokens))
	fmt.Fprintf(&b, "<li>Would sanitize: %s files / %s tok</li>", formatInt(c.SanitizedFiles), formatInt(c.SanitizedTokens))
	fmt.Fprintf(&b, "<li>Blocked: %s files / %s tok</li>", formatInt(c.BlockedFiles), formatInt(c.BlockedTokens))

	b.WriteString("<h2>Audit mode breakdown</h2>")
	fmt.Fprintf(&b, "<li>Actually transformed: 0 tok</li>")
	fmt.Fprintf(&b, "<li>Safe now: %s tok</li>", formatInt(c.SafeToSendNowTokens()))
	fmt.Fprintf(&b, "<li>After sanitization: %s tok</li>", formatInt(c.FinalSendableTokens()))

	b.WriteString("<h2>Security metrics</h2>")
	imp := d.SecurityImpact
	fmt.Fprintf(&b, "<li>PII indicators: %d - Secret indicators: %d</li>", imp.FilesWithPotentialPII, imp.FilesWithPotentialSecrets)
	fmt.Fprintf(&b, "<li>Detected not actioned: %d - Tokens redacted: 0</li>", imp.DetectedNotActioned)

	b.WriteString("<h2>Repository composition</h2>")
	for _, r := range topDirsForPDF(d, 5) {
		fmt.Fprintf(&b, "<li>%s - %s</li>", r.Dir, formatPct(r.Percent))
	}

	b.WriteString("<h2>Traceability &amp; interpretation</h2>")
	fmt.Fprintf(&b, "<li>execution_id: %s</li>", d.ExecutionID)
	fmt.Fprintf(&b, "<li>commit: %s - task: %s</li>", orNA(d.CommitHash), orTaskNone(d.TaskName))
	if len(d.IgnorePatterns) > 0 {
		fmt.Fprintf(&b, "<li>ignore: %s</li>", strings.Join(d.IgnorePatterns, ", "))
	}
	fmt.Fprintf(&b, "<p><b>%s</b></p>", i18n.T("reports.audit_mode_note"))
	if len(d.Findings) > 0 {
		fmt.Fprintf(&b, "<p>Full detailed findings (%d) in pii-audit-log.json.</p>", len(d.Findings))
	}

	return b.String()
}

// excludedByTask returns the "Not relevant to task" row from
// ExclusionReasons, or nil when absent (no --task given).
func excludedByTask(d *Data) *ExclusionReasonRow {
	for _, r := range d.ExclusionReasons {
		if strings.Contains(strings.ToLower(r.Reason), "task") {
			row := r
			return &row
		}
	}
	return nil
}

// costEstimateHTML lists the 4 headline models the spec names
// explicitly (Claude Haiku, Claude Sonnet, GPT-4.1, Gemini Flash) when
// present in d.Costs, falling back to whatever IS configured so this
// never renders an empty section just because prices.json uses
// different model keys.
func costEstimateHTML(d *Data) string {
	var b strings.Builder
	shown := 0
	wanted := []string{"claude-haiku", "claude-sonnet", "gpt-4.1", "gemini-flash", "gemini-pro", "gpt-4o"}
	for _, w := range wanted {
		for _, c := range d.Costs {
			if strings.Contains(strings.ToLower(c.Model), w) {
				fmt.Fprintf(&b, "<li>%s: %s</li>", c.Model, FormatCost(c.USD))
				shown++
				break
			}
		}
		if shown >= 4 {
			break
		}
	}
	if shown == 0 {
		for i, c := range d.Costs {
			if i >= 4 {
				break
			}
			fmt.Fprintf(&b, "<li>%s: %s</li>", c.Model, FormatCost(c.USD))
		}
	}
	b.WriteString("<li>Local (Ollama/LM Studio/vLLM): $0</li>")
	return b.String()
}

func truncatePath(p string, max int) string {
	if len(p) <= max {
		return p
	}
	return "..." + p[len(p)-max+3:]
}

// topDirsForPDF returns the top-N DirBreakdown rows by token share.
func topDirsForPDF(d *Data, n int) []DirRow {
	if len(d.DirBreakdown) < n {
		n = len(d.DirBreakdown)
	}
	return d.DirBreakdown[:n]
}
