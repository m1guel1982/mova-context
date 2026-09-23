// report_pipeline.go — the three new mova-budget-report.md sections
// for Context Governance (Sanitizer/Cache Layout/Circuit Breaker), kept
// in their own file so report.go (already close to the 300-line limit)
// doesn't have to grow for this. Called from RenderMarkdown (report.go)
// — same report, same file, same generation call, just three more
// sections appended when there's something to say. Every sentence goes
// through i18n.T (see config/lang/{es,en}.json's "budget" section) —
// same rule as report.go.
package budget

import (
	"fmt"
	"strings"

	"mova.local/i18n"
)

// sanitizerSection reports what the Sanitizer stage actually removed —
// present only when it removed something, same "don't pad the report
// with an empty section" rule deduplicationSection already follows.
func sanitizerSection(r *Report) string {
	s := r.SanitizeStats
	if s.LinesRemoved == 0 && s.BlankRemoved == 0 && s.CommentsRemoved == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(i18n.T("budget.section_sanitizer") + "\n\n")
	b.WriteString(i18n.T("budget.sanitizer_intro") + "\n\n")
	if s.LinesRemoved > 0 {
		b.WriteString(i18n.T("budget.sanitizer_lines_removed", map[string]any{"count": s.LinesRemoved}) + "\n")
	}
	if s.BlankRemoved > 0 {
		b.WriteString(i18n.T("budget.sanitizer_blank_removed", map[string]any{"count": s.BlankRemoved}) + "\n")
	}
	if s.CommentsRemoved > 0 {
		b.WriteString(i18n.T("budget.sanitizer_comments_removed", map[string]any{"count": s.CommentsRemoved}) + "\n")
	}
	if s.CharsRemoved > 0 {
		approxTokens := s.CharsRemoved / 4
		b.WriteString("\n" + i18n.T("budget.sanitizer_approx_savings", map[string]any{"tokens": approxTokens, "chars": s.CharsRemoved}) + "\n\n")
	}
	return b.String()
}

// piiMaskingSection reports the PII Masking stage's result — present
// only when it actually masked something (i.e. the project explicitly
// enabled it AND at least one token cleared the threshold). Always
// carries the legal/technical disclaimer so the report itself never
// implies a compliance guarantee this heuristic stage doesn't make.
func piiMaskingSection(r *Report) string {
	p := r.PIIStats
	if p.TokensMasked == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(i18n.T("budget.section_pii_masking") + "\n\n")
	b.WriteString(i18n.T("budget.pii_masking_result", map[string]any{"masked": p.TokensMasked, "scanned": p.TokensScanned}) + "\n\n")
	b.WriteString(i18n.T("budget.pii_masking_disclaimer") + "\n\n")
	return b.String()
}

// cacheLayoutSection reports the Cache Layout Guard's result — present
// only when "cache_hint" is enabled (r.CacheLayout != nil).
func cacheLayoutSection(r *Report) string {
	if r.CacheLayout == nil {
		return ""
	}
	l := r.CacheLayout
	var b strings.Builder
	b.WriteString(i18n.T("budget.section_cache_layout") + "\n\n")
	b.WriteString(i18n.T("budget.cache_layout_intro") + "\n\n")
	b.WriteString(i18n.T("budget.cache_static_prefix", map[string]any{"count": l.StaticTokens}) + "\n\n")
	b.WriteString(i18n.T("budget.cache_prefix_fingerprint", map[string]any{"hash": l.Hash}) + "\n\n")
	if l.StaticTokens > 0 {
		b.WriteString(i18n.T("budget.cache_estimated_reuse", map[string]any{"count": int(float64(l.StaticTokens) * 0.9)}) + "\n\n")
	}
	if l.StaticTokens < 1024 {
		b.WriteString(i18n.T("budget.cache_below_minimum") + "\n\n")
	}
	b.WriteString(i18n.T("budget.cache_probability_note") + "\n\n")
	return b.String()
}

// circuitBreakerSection reports the spend-governance gate's status —
// present only when at least one ceiling ("max_tokens_per_run" /
// "max_monthly_usd") is actually configured.
func circuitBreakerSection(r *Report) string {
	cb := r.CircuitBreaker
	if !cb.Checked {
		return ""
	}
	var b strings.Builder
	b.WriteString(i18n.T("budget.section_circuit_breaker") + "\n\n")
	if cb.RunLimit > 0 {
		status := i18n.T("budget.circuit_status_ok")
		if cb.RunExceeded {
			status = i18n.T("budget.circuit_status_over")
		}
		b.WriteString(i18n.T("budget.circuit_per_run_limit", map[string]any{"used": cb.RunTokens, "limit": cb.RunLimit, "status": status}) + "\n\n")
	}
	if cb.MonthUSDLimit > 0 {
		status := i18n.T("budget.circuit_status_ok")
		if cb.MonthExceeded {
			status = i18n.T("budget.circuit_status_over")
		}
		b.WriteString(i18n.T("budget.circuit_monthly_spend", map[string]any{
			"spent": fmt.Sprintf("%.2f", cb.MonthUSDSpent), "limit": fmt.Sprintf("%.2f", cb.MonthUSDLimit), "status": status,
		}) + "\n\n")
	}
	if cb.Message != "" {
		if cb.Aborted {
			b.WriteString(i18n.T("budget.circuit_status_aborted", map[string]any{"message": cb.Message}) + "\n\n")
		} else {
			b.WriteString(i18n.T("budget.circuit_status_warn", map[string]any{"message": cb.Message}) + "\n\n")
		}
	} else {
		b.WriteString(i18n.T("budget.circuit_status_within") + "\n\n")
	}
	return b.String()
}

// firewallSummarySection is the "big picture" comparison: total tokens
// and estimated cost before vs. after the whole Context Governance pipeline
// ran, plus the two percentages the person actually cares about. Only
// rendered when core.DetailedReportsEnabled populated RawTokens.
func firewallSummarySection(r *Report) string {
	if r.RawTokens == 0 || r.RawTokens <= r.TotalTokens {
		return ""
	}
	var b strings.Builder
	b.WriteString(i18n.T("budget.section_governance_summary") + "\n\n")
	b.WriteString(i18n.T("budget.governance_summary_intro") + "\n\n")
	b.WriteString(i18n.T("budget.governance_summary_header") + "\n")
	b.WriteString(i18n.T("budget.governance_summary_tokens_row", map[string]any{
		"before": r.RawTokens, "after": r.TotalTokens, "pct": fmt.Sprintf("%.1f", r.MemorySavingsPercent),
	}) + "\n")
	for i, before := range r.RawCosts {
		if i >= len(r.TotalCosts) {
			break
		}
		after := r.TotalCosts[i]
		b.WriteString(i18n.T("budget.governance_summary_cost_row", map[string]any{
			"model": before.Model, "before": fmt.Sprintf("%.4f", before.USD), "after": fmt.Sprintf("%.4f", after.USD),
			"pct": fmt.Sprintf("%.1f", r.CostSavingsPercent),
		}) + "\n")
	}
	b.WriteString("\n" + i18n.T("budget.governance_summary_footer") + "\n\n")
	return b.String()
}

// fileBreakdownSection lists tokens per individual Focus file — useful
// for spotting which single file is driving the cost up. Only rendered
// when core.DetailedReportsEnabled populated FileBreakdown.
func fileBreakdownSection(r *Report) string {
	if len(r.FileBreakdown) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(i18n.T("budget.section_file_breakdown") + "\n\n")
	b.WriteString(i18n.T("budget.file_breakdown_header") + "\n")
	for _, f := range r.FileBreakdown {
		fmt.Fprintf(&b, "| %s | %d |\n", f.Name, f.Tokens)
	}
	b.WriteString("\n")
	return b.String()
}
