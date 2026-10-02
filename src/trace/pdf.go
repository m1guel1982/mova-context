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
	"regexp"
	"strings"

	"mova.local/documents"
	"mova.local/i18n"
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

// mdInlineToHTML converts the small subset of inline Markdown used by
// shared i18n strings (like reports.cost_basis_note or
// reports.full_audit_log_exported, written once and reused by BOTH
// markdown.go's Markdown output and this file's HTML/PDF output) into
// the plain <b> tags documents.GeneratePDFDocument actually
// understands (see this file's header comment — it only interprets
// h1/h2/p/li). Without this conversion, a translated string containing
// "**text**" or "`code`" would render as literal asterisks/backticks
// in the generated PDF instead of being interpreted.
func mdInlineToHTML(s string) string {
	s = boldRe.ReplaceAllString(s, "<b>$1</b>")
	s = codeRe.ReplaceAllString(s, "$1")
	return s
}

var (
	boldRe = regexp.MustCompile(`\*\*(.+?)\*\*`)
	codeRe = regexp.MustCompile("`([^`]+)`")
)

func contextReportHTML(d *Data) string {
	var b strings.Builder
	b.WriteString("<h1>" + i18n.T("reports.h_report_title") + "</h1>")

	if d.IsRemote {
		fmt.Fprintf(&b, "<p><b>%s:</b> %s</p><p><b>%s:</b> %s</p>", i18n.T("reports.f_repository"), d.RepoURL, i18n.T("reports.f_branch"), orNA(d.Branch))
	} else {
		fmt.Fprintf(&b, "<p><b>%s:</b> %s</p><p><b>%s:</b> %s</p><p><b>project.json:</b> %s</p>", i18n.T("reports.f_project"), d.ProjectName, i18n.T("reports.f_task"), d.TaskName, d.ProjectJSONPath)
	}
	fmt.Fprintf(&b, "<p><b>%s:</b> %s &middot; <b>%s:</b> %s &middot; <b>%s:</b> %s</p>", i18n.T("reports.agent_client_label"), orNA(d.AgentClient), i18n.T("reports.target_model_label"), orNA(d.TargetModel), i18n.T("reports.policy_author_label"), orNA(d.PolicyAuthor))

	b.WriteString("<h2>" + i18n.T("reports.h_composition") + "</h2>")
	if len(d.Components) > 0 {
		for _, c := range d.Components {
			fmt.Fprintf(&b, "<li>%s: %s tok</li>", c.Name, formatInt(c.Tokens))
		}
	} else {
		b.WriteString("<p>" + mdInlineToHTML(i18n.T("reports.no_active_project_scanned")) + "</p>")
	}
	fmt.Fprintf(&b, "<p>%s</p>", TotalTokensLine(d.TotalTokens, d.Encoding))

	fmt.Fprintf(&b, "<h2>%s</h2><li>%s: %d</li><li>%s: %d</li>", i18n.T("reports.h_focus"), i18n.T("reports.f_files_included"), d.Focus.Included, i18n.T("reports.f_files_excluded"), d.Focus.Excluded)
	if len(d.Focus.ExcludedSuggestion) > 0 {
		fmt.Fprintf(&b, "<li>%s: %s</li>", i18n.T("reports.f_exclude_suggestion"), strings.Join(d.Focus.ExcludedSuggestion, ", "))
	}

	if len(d.DirBreakdown) > 0 {
		b.WriteString(dirBreakdownHTML(d))
	}

	b.WriteString(governanceHTML(d))

	b.WriteString("<h2>" + i18n.T("reports.h_governance") + "</h2>")
	fmt.Fprintf(&b, "<li>"+i18n.T("reports.f_sanitizer")+": %s</li>", onOff(d.Firewall.SanitizerOn))
	piiLine := fmt.Sprintf("<li>"+i18n.T("reports.f_pii_masking")+": %s", onOff(d.Firewall.PIIMaskingOn))
	if d.Firewall.PIIWarning != "" {
		piiLine += " - " + d.Firewall.PIIWarning
	}
	b.WriteString(piiLine + "</li>")
	fmt.Fprintf(&b, "<li>"+i18n.T("reports.f_cache_guard")+": %s</li>", onOff(d.Firewall.CacheGuardOn))
	fmt.Fprintf(&b, "<li>"+i18n.T("reports.f_circuit_breaker")+": %s</li>", onOff(d.Firewall.CircuitBreakerOn))

	b.WriteString("<h2>" + i18n.T("reports.h_budget") + "</h2>")
	if d.MaxTokens > 0 {
		pct := float64(d.TotalTokens) / float64(d.MaxTokens) * 100
		fmt.Fprintf(&b, "<p>%s / %s tokens (%s)</p>", formatInt(d.TotalTokens), formatInt(d.MaxTokens), formatPct(pct))
	} else {
		b.WriteString("<p>" + mdInlineToHTML(i18n.T("reports.budget_not_configured_full")) + "</p>")
	}

	b.WriteString("<h2>" + i18n.T("reports.c_cost") + "</h2>")
	fmt.Fprintf(&b, "<p>%s</p>", mdInlineToHTML(strings.TrimSpace(i18n.T("reports.cost_basis_note", map[string]any{"tokens": formatInt(costBasisTokens(d))}))))
	for _, c := range d.Costs {
		fmt.Fprintf(&b, "<li>%s (%s): %s</li>", c.Provider, c.Model, FormatCost(c.USD))
	}
	b.WriteString("<li>Local (Ollama / LM Studio / vLLM): " + i18n.T("reports.v_local_exec") + "</li>")

	if len(d.Findings) > MaxPDFSecurityFindings {
		fmt.Fprintf(&b, "<p>%s</p>", mdInlineToHTML(i18n.T("reports.full_audit_log_exported", map[string]any{"count": len(d.Findings)})))
	}
	b.WriteString("<p>" + i18n.T("reports.trace_tagline") + "</p>")
	b.WriteString("<p>" + i18n.T("reports.report_disclaimer") + "</p>")

	return b.String()
}

// dirBreakdownHTML renders the "top directories by token usage" plus
// table (see analyzer.go's buildDirBreakdown) as a simple list, since
// documents.GeneratePDFDocument does not interpret <table> markup -
// each line already carries directory, tokens, files, and percentage,
// in that fixed order, closing with the TOTAL line.
func dirBreakdownHTML(d *Data) string {
	var b strings.Builder
	b.WriteString("<h2>" + i18n.T("reports.h_top_dirs") + "</h2>")
	for _, r := range d.DirBreakdown {
		fmt.Fprintf(&b, "<li>%s</li>", i18n.T("reports.f_dir_line", map[string]any{"dir": r.Dir, "tok": formatInt(r.Tokens), "files": r.Files, "pct": formatPct(r.Percent)}))
	}
	fmt.Fprintf(&b, "<li><b>%s</b></li>", i18n.T("reports.f_dir_total", map[string]any{"tok": formatInt(d.TotalTokens), "files": d.Focus.Included}))
	return b.String()
}
