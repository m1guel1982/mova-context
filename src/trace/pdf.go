// pdf.go — context-report.pdf, reusing the PDF generator that already
// exists in mova.local/documents (the same engine generate_pdf_document
// uses across MCP/Chat/HTTP for any other Mova Context document) - zero
// new dependencies, zero second PDF engine to maintain.
//
// All text built here is plain ASCII-leaning English (see console.go's
// header) - deliberately avoiding characters that used to render as
// literal "?" glyphs in the generated PDF (an em dash, "—", is now
// mapped correctly in documents/pdf_writer.go's WinAnsiEncoding table,
// but sticking to a plain hyphen here is a second, redundant layer of
// safety against the same class of bug reappearing for some other
// punctuation mark later).
package trace

import (
	"fmt"
	"strings"

	"mova.local/documents"
)

// WriteContextReportPDF builds a simple HTML layout (h1/h2/p/li - the
// only subset documents.GeneratePDFDocument interprets, see its own
// header) from Data and writes it to path. A discovery/audit run (no
// project.json) uses the dedicated 2-page executive layout (see
// pdf_discovery.go); a project.json run keeps the original,
// composition-focused layout below - its needs (budget, Token
// Firewall, Agents/Skills/Prompt breakdown) are genuinely different
// from an audit-mode security/governance report.
func WriteContextReportPDF(path string, d *Data) error {
	if d.StateTotals.DiscoveredFiles > 0 {
		return documents.GeneratePDFDocument(path, discoveryReportHTML(d))
	}
	return documents.GeneratePDFDocument(path, contextReportHTML(d))
}

func contextReportHTML(d *Data) string {
	var b strings.Builder
	b.WriteString("<h1>Context Report - Mova Context Trace</h1>")

	if d.IsRemote {
		fmt.Fprintf(&b, "<p><b>Repository:</b> %s</p><p><b>Branch:</b> %s</p>", d.RepoURL, orNA(d.Branch))
	} else {
		fmt.Fprintf(&b, "<p><b>Project:</b> %s</p><p><b>Task:</b> %s</p><p><b>project.json:</b> %s</p>", d.ProjectName, d.TaskName, d.ProjectJSONPath)
	}
	fmt.Fprintf(&b, "<p><b>Agent:</b> %s &middot; <b>Target model:</b> %s &middot; <b>Policy author:</b> %s</p>", orNA(d.AgentClient), orNA(d.TargetModel), orNA(d.PolicyAuthor))

	b.WriteString("<h2>Context composition</h2>")
	if len(d.Components) > 0 {
		for _, c := range d.Components {
			fmt.Fprintf(&b, "<li>%s: %s tok</li>", c.Name, formatInt(c.Tokens))
		}
	} else {
		b.WriteString("<p>No active project.json - the whole repository was scanned.</p>")
	}
	fmt.Fprintf(&b, "<p>%s</p>", TotalTokensLine(d.TotalTokens, d.Encoding))

	fmt.Fprintf(&b, "<h2>Focus</h2><li>Included: %d file(s)</li><li>Excluded: %d file(s)</li>", d.Focus.Included, d.Focus.Excluded)
	if len(d.Focus.ExcludedSuggestion) > 0 {
		fmt.Fprintf(&b, "<li>Exclude suggestion: %s</li>", strings.Join(d.Focus.ExcludedSuggestion, ", "))
	}

	if len(d.DirBreakdown) > 0 {
		b.WriteString(dirBreakdownHTML(d))
	}

	b.WriteString(governanceHTML(d))

	b.WriteString("<h2>Context Governance</h2>")
	fmt.Fprintf(&b, "<li>Sanitizer: %s</li>", onOff(d.Firewall.SanitizerOn))
	piiLine := fmt.Sprintf("<li>PII Masking: %s", onOff(d.Firewall.PIIMaskingOn))
	if d.Firewall.PIIWarning != "" {
		piiLine += " - " + d.Firewall.PIIWarning
	}
	b.WriteString(piiLine + "</li>")
	fmt.Fprintf(&b, "<li>Cache Layout Guard: %s</li>", onOff(d.Firewall.CacheGuardOn))
	fmt.Fprintf(&b, "<li>Circuit Breaker: %s</li>", onOff(d.Firewall.CircuitBreakerOn))

	b.WriteString("<h2>Budget</h2>")
	if d.MaxTokens > 0 {
		pct := float64(d.TotalTokens) / float64(d.MaxTokens) * 100
		fmt.Fprintf(&b, "<p>%s / %s tokens (%s)</p>", formatInt(d.TotalTokens), formatInt(d.MaxTokens), formatPct(pct))
	} else {
		b.WriteString("<p>N/A (no active project.json or no budget.max_tokens configured)</p>")
	}

	b.WriteString("<h2>Theoretical / Estimated Input Cost (USD)</h2>")
	fmt.Fprintf(&b, "<p>Calculated over the <b>%s final sendable tokens</b> (ALLOWED + SANITIZED in discovery mode, or the assembled project context in a project.json run) - never over the whole scanned repository. All costs are estimates, based on the prices configured in config/prices.json. They exclude output tokens, tool calls, caching, and other provider-specific charges.</p>", formatInt(costBasisTokens(d)))
	for _, c := range d.Costs {
		fmt.Fprintf(&b, "<li>%s (%s): %s</li>", c.Provider, c.Model, FormatCost(c.USD))
	}
	b.WriteString("<li>Local (Ollama / LM Studio / vLLM): $0 (local execution)</li>")

	if len(d.Findings) > MaxPDFSecurityFindings {
		fmt.Fprintf(&b, "<p><b>Full detailed audit log (%d files) exported to pii-audit-log.json</b></p>", len(d.Findings))
	}
	b.WriteString("<p>Mova Context Trace - Observable. Governed. Auditable.</p>")
	b.WriteString("<p>This report is a technical analysis. It is not a legal, privacy, or security certification.</p>")

	return b.String()
}

// dirBreakdownHTML renders the "top directories by token usage" plus
// table (see analyzer.go's buildDirBreakdown) as a simple list, since
// documents.GeneratePDFDocument does not interpret <table> markup -
// each line already carries directory, tokens, files, and percentage,
// in that fixed order, closing with the TOTAL line.
func dirBreakdownHTML(d *Data) string {
	var b strings.Builder
	b.WriteString("<h2>Top directories by token usage</h2>")
	for _, r := range d.DirBreakdown {
		fmt.Fprintf(&b, "<li>%s - %s tok, %d file(s), %s of total</li>", r.Dir, formatInt(r.Tokens), r.Files, formatPct(r.Percent))
	}
	fmt.Fprintf(&b, "<li><b>TOTAL - %s tok, %d file(s), 100.0%% of total</b></li>", formatInt(d.TotalTokens), d.Focus.Included)
	return b.String()
}
