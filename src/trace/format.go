// format.go — shared text formatting for every report/console surface
// in this package: cost amounts (always explicit "USD", 2 decimals for
// anything a person would actually pay, 4 decimals only for the
// fractions-of-a-cent case where 2 decimals would round to $0.00 and
// hide a real cost), and thousands-separated token counts. One
// function per concern, called from console.go/markdown.go/pdf.go/
// png.go so the three surfaces can never drift into showing a
// different number of decimals for the same value.
package trace

import (
	"fmt"
	"strings"

	"mova.local/i18n"
)

// FormatCost renders amount as a currency string, always in USD and
// always unambiguous about it:
//   - exactly 0        -> "No cost (local execution)"
//   - >= $0.01          -> "$X.XX USD"      (what a person actually pays)
//   - > 0 but < $0.01   -> "$X.XXXX USD"    (keeps sub-cent estimates
//     visible instead of rounding a real cost down to "$0.00 USD")
func FormatCost(amount float64) string {
	if amount == 0 {
		return i18n.T("reports.no_cost_local")
	}
	if amount >= 0.01 {
		return fmt.Sprintf("$%.2f USD", amount)
	}
	return fmt.Sprintf("$%.4f USD", amount)
}

// formatInt adds a thousands separator (","), e.g. 16969539 -> "16,969,539".
func formatInt(n int) string {
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

// formatPct renders a percentage with one decimal, e.g. 41.532 -> "41.5%".
func formatPct(p float64) string {
	return fmt.Sprintf("%.1f%%", p)
}

// TotalTokensLine is the single sentence every surface (console,
// context-report, budget-report) shows for "how many tokens did this
// analysis count, and with which tokenizer" — one shared wording so it
// never reads slightly differently between the three.
func TotalTokensLine(totalTokens int, encoding string) string {
	return i18n.T("reports.total_tokens_encoding", map[string]any{"tokens": formatInt(totalTokens), "encoding": orNA(encoding)})
}

// costBasisTokens is the single definition of "how many tokens is the
// cost table actually computed over" - used identically by
// console.go, markdown.go, and pdf.go so the three outputs can never
// disagree (see prompt requirement: same content, only the
// presentation format changes). Discovery mode (StateTotals
// populated) uses the final sendable context (ALLOWED + SANITIZED); a
// project.json run uses the assembled context total (d.TotalTokens) -
// the same number budget.EstimateCost was actually called with in
// analyzer.go, never recomputed here.
func costBasisTokens(d *Data) int {
	if d.StateTotals.DiscoveredFiles > 0 {
		return d.StateTotals.FinalSendableTokens()
	}
	return d.TotalTokens
}

// taskOrNone renders d.TaskName for the INPUT section - "None" (not
// an empty string) when no --task was given, matching the exact
// baseline contract ("Task: None").
func taskOrNone(task string) string {
	if task == "" {
		return "None"
	}
	return task
}

func onOff(v bool) string {
	if v {
		return "ON"
	}
	return "OFF"
}

func orNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}
