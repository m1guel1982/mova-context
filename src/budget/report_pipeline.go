// report_pipeline.go — the three new mova-budget-report.md sections
// for the Token Firewall (Sanitizer/Cache Layout/Circuit Breaker),
// kept in their own file so report.go (already close to the 300-line
// limit) doesn't have to grow for this. Called from RenderMarkdown
// (report.go) — same report, same file, same generation call, just
// three more sections appended when there's something to say.
package budget

import (
	"fmt"
	"strings"
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
	b.WriteString("## Sanitizer\n\n")
	b.WriteString("Mova Context removed repetitive noise from Focus/Memory before counting or sending anything — 100% deterministic, no AI involved, microseconds of work:\n\n")
	if s.LinesRemoved > 0 {
		b.WriteString(fmt.Sprintf("- %d repeated line(s)/header(s) collapsed (e.g. repeated log lines, duplicated file headers).\n", s.LinesRemoved))
	}
	if s.BlankRemoved > 0 {
		b.WriteString(fmt.Sprintf("- %d run(s) of excess blank lines collapsed.\n", s.BlankRemoved))
	}
	if s.CommentsRemoved > 0 {
		b.WriteString(fmt.Sprintf("- %d line(s) of large comment blocks omitted (only when \"strip_comments\" is enabled).\n", s.CommentsRemoved))
	}
	if s.CharsRemoved > 0 {
		approxTokens := s.CharsRemoved / 4
		b.WriteString(fmt.Sprintf("\nApproximate savings: ~%d tokens (~%d characters) — already reflected in every count above.\n\n", approxTokens, s.CharsRemoved))
	}
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
	b.WriteString("## Cache Layout Guard\n\n")
	b.WriteString("\"cache_hint\" is enabled: the system prompt sent to the model is laid out as a stable prefix (agents + skills + prompt) followed by everything that changes every run (timestamp, focus, memory) — this is what lets a Cloud provider's own prompt caching actually trigger.\n\n")
	b.WriteString(fmt.Sprintf("Static prefix: %d tokens\n\n", l.StaticTokens))
	b.WriteString(fmt.Sprintf("Prefix fingerprint: `%s` — compare this value run to run; an unchanged fingerprint means the prefix is byte-identical and a real provider cache is likely to hit.\n\n", l.Hash))
	if l.StaticTokens > 0 {
		b.WriteString(fmt.Sprintf("Estimated tokens reused on a cache hit: ~%d (~90%% of the static prefix — Anthropic's own published discount for cached input; other providers vary, see COMMANDS.md § Token Firewall for the per-provider table).\n\n", int(float64(l.StaticTokens)*0.9)))
	}
	if l.StaticTokens < 1024 {
		b.WriteString("Note: this prefix is under ~1,024 tokens, the approximate minimum several providers require before caching kicks in (the exact minimum varies by provider/model and changes over time) — caching may not activate yet for this project, but there is no downside to leaving \"cache_hint\" on.\n\n")
	}
	b.WriteString("This increases the PROBABILITY of a cache hit — it does not guarantee one. Actual caching depends on the provider, the model, and whether you make another call using the same prefix again while the provider's cache window is still open.\n\n")
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
	b.WriteString("## Circuit Breaker\n\n")
	if cb.RunLimit > 0 {
		status := "OK"
		if cb.RunExceeded {
			status = "OVER LIMIT"
		}
		b.WriteString(fmt.Sprintf("Per-run limit: %d / %d tokens (%s)\n\n", cb.RunTokens, cb.RunLimit, status))
	}
	if cb.MonthUSDLimit > 0 {
		status := "OK"
		if cb.MonthExceeded {
			status = "OVER LIMIT"
		}
		b.WriteString(fmt.Sprintf("Monthly spend: $%.2f / $%.2f (%s) — tracked in mova-spend.json, resets automatically at the start of each calendar month.\n\n", cb.MonthUSDSpent, cb.MonthUSDLimit, status))
	}
	if cb.Message != "" {
		if cb.Aborted {
			b.WriteString(fmt.Sprintf("Status: %s **This stops execution before anything is sent to a model** (\"on_exceed\": \"abort\").\n\n", cb.Message))
		} else {
			b.WriteString(fmt.Sprintf("Status: %s This is a warning only (\"on_exceed\": \"warn\", the default) — execution continues.\n\n", cb.Message))
		}
	} else {
		b.WriteString("Status: within budget.\n\n")
	}
	return b.String()
}

// firewallSummarySection is the "big picture" comparison: total tokens
// and estimated cost before vs. after the whole Token Firewall pipeline
// ran, plus the two percentages the person actually cares about. Only
// rendered when core.DetailedReportsEnabled populated RawTokens.
func firewallSummarySection(r *Report) string {
	if r.RawTokens == 0 || r.RawTokens <= r.TotalTokens {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Token Firewall — Summary\n\n")
	b.WriteString("Before vs. after the full pipeline (Sanitizer + Cache Layout Guard awareness + Circuit Breaker check):\n\n")
	b.WriteString("| | Before | After | Savings |\n|---|---|---|---|\n")
	b.WriteString(fmt.Sprintf("| Tokens | %d | %d | %.1f%% |\n", r.RawTokens, r.TotalTokens, r.MemorySavingsPercent))
	for i, before := range r.RawCosts {
		if i >= len(r.TotalCosts) {
			break
		}
		after := r.TotalCosts[i]
		b.WriteString(fmt.Sprintf("| Cost (%s) | $%.4f | $%.4f | %.1f%% |\n", before.Model, before.USD, after.USD, r.CostSavingsPercent))
	}
	b.WriteString("\nThese are estimates from the local tokenizer and config/prices.json — consistent and useful for comparing before/after, not a guarantee of what a provider will actually bill (see the Historical Accuracy section below for how close local estimates have run to real usage on this project).\n\n")
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
	b.WriteString("## Tokens per File (Focus)\n\n")
	b.WriteString("| File | Tokens |\n|---|---|\n")
	for _, f := range r.FileBreakdown {
		b.WriteString(fmt.Sprintf("| %s | %d |\n", f.Name, f.Tokens))
	}
	b.WriteString("\n")
	return b.String()
}
