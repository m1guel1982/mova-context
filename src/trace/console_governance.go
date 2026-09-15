// console_governance.go — the CLI's ASCII governance summary table:
// DISCOVERED/CANDIDATE/ALLOWED/WOULD_SANITIZE/BLOCKED/EXCLUDED
// file+token counts, the audit-mode-honest sendable-context breakdown,
// the PII-indicator reconciliation, and estimated cost for both the
// "safe right now" and "after sanitization" scenarios. Split out of
// console.go purely to keep every file in this package under the
// 300-line limit.
package trace

import (
	"fmt"
	"strings"

	"mova.local/i18n"
)

const boxWidth = 66

func boxLine(s string) string {
	if len(s) > boxWidth {
		s = s[:boxWidth]
	}
	return "| " + s + strings.Repeat(" ", boxWidth-len(s)) + " |\n"
}

// cheapestCost returns the lowest non-zero-provider cost in costs, or
// a zero CostRow if costs is empty.
func cheapestCost(costs []CostRow) (CostRow, bool) {
	if len(costs) == 0 {
		return CostRow{}, false
	}
	cheapest := costs[0]
	for _, c := range costs {
		if c.USD < cheapest.USD {
			cheapest = c
		}
	}
	return cheapest, true
}

// renderGovernanceConsole draws the boxed GOVERNANCE summary shown
// right after CONTEXT GOVERNANCE in a discovery run (see console.go's
// renderConsoleRemote) — the CLI counterpart of context-report's own
// "Context decision" section (markdown_governance.go).
func renderGovernanceConsole(d *Data) string {
	c := d.StateTotals
	imp := d.SecurityImpact
	var b strings.Builder

	border := "+" + strings.Repeat("-", boxWidth+2) + "+\n"
	b.WriteString(i18n.T("reports.governance_title") + "\n")
	b.WriteString(border)
	b.WriteString(boxLine(i18n.T("reports.status_label", map[string]any{"status": d.GovernanceStatus})))
	b.WriteString(boxLine(""))
	row := i18n.T("reports.files_tok_row")
	b.WriteString(boxLine(fmt.Sprintf(row, i18n.T("reports.state_discovered"), formatInt(c.DiscoveredFiles), formatInt(c.DiscoveredTokens))))
	b.WriteString(boxLine(fmt.Sprintf(row, i18n.T("reports.state_candidate"), formatInt(c.CandidateFiles), formatInt(c.CandidateTokens))))
	b.WriteString(boxLine(fmt.Sprintf(row, i18n.T("reports.state_allowed"), formatInt(c.AllowedFiles), formatInt(c.AllowedTokens))))
	b.WriteString(boxLine(fmt.Sprintf(row, i18n.T("reports.state_would_sanitize"), formatInt(c.SanitizedFiles), formatInt(c.SanitizedTokens))))
	b.WriteString(boxLine(fmt.Sprintf(row, i18n.T("reports.state_blocked"), formatInt(c.BlockedFiles), formatInt(c.BlockedTokens))))
	b.WriteString(boxLine(fmt.Sprintf(row, i18n.T("reports.state_excluded"), formatInt(c.ExcludedFiles), formatInt(c.ExcludedTokens))))
	b.WriteString(boxLine(""))

	// Audit-mode-honest sendable breakdown (see sanitize.StateCounts'
	// SafeToSendNowTokens/FinalSendableTokens doc comments): the exact
	// 4-part breakdown requested to remove the "is 9M tokens safe to
	// send?" ambiguity.
	b.WriteString(boxLine(i18n.T("reports.sendable_breakdown_title")))
	b.WriteString(boxLine(fmt.Sprintf(i18n.T("reports.repo_context_scanned"), formatInt(c.DiscoveredTokens))))
	b.WriteString(boxLine(fmt.Sprintf(i18n.T("reports.policy_allowed_safe_now"), formatInt(c.AllowedTokens))))
	b.WriteString(boxLine(fmt.Sprintf(i18n.T("reports.would_require_sanitization"), formatInt(c.SanitizedTokens))))
	b.WriteString(boxLine(fmt.Sprintf(i18n.T("reports.actually_transformed"), formatInt(imp.TokensActuallyRedacted))))
	b.WriteString(boxLine(""))
	b.WriteString(boxLine(i18n.T("reports.safe_to_send_now", map[string]any{"tokens": formatInt(c.SafeToSendNowTokens()), "pct": formatPct(c.SafeReductionPercent())})))
	b.WriteString(boxLine(i18n.T("reports.sendable_after_sanitization", map[string]any{"tokens": formatInt(c.FinalSendableTokens()), "pct": formatPct(c.ReductionPercent())})))
	b.WriteString(boxLine(""))

	// PII/secret indicator reconciliation (see SecurityImpact.DetectedNotActioned's
	// doc comment) - closes the "1832 indicators vs 1470 WOULD_SANITIZE" gap.
	totalIndicatorFiles := imp.FilesWithAnyIndicator
	b.WriteString(boxLine(i18n.T("reports.files_with_indicator", map[string]any{
		"total": formatInt(totalIndicatorFiles), "would_sanitize": formatInt(c.SanitizedFiles),
		"blocked": formatInt(c.BlockedFiles), "detected": formatInt(imp.DetectedNotActioned),
	})))
	b.WriteString(boxLine(""))

	if cheapest, ok := cheapestCost(d.CostsSafeNow); ok {
		b.WriteString(boxLine(i18n.T("reports.cost_safe_now_lowest", map[string]any{"cost": FormatCost(cheapest.USD), "provider": cheapest.Provider})))
	}
	if cheapest, ok := cheapestCost(d.Costs); ok {
		b.WriteString(boxLine(i18n.T("reports.cost_after_sanitization_lowest", map[string]any{"cost": FormatCost(cheapest.USD), "provider": cheapest.Provider})))
	}
	b.WriteString(boxLine(i18n.T("reports.local_zero_cost")))
	b.WriteString(border)
	b.WriteString("\n")
	return b.String()
}
