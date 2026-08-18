// svg_metrics.go — the token-reduction pipeline visualization (#18):
// per-resource token breakdown (Agents/Skills/Prompt/Focus/Memory) and
// the progressive effect of Deduplication -> Sanitizer -> PII Masking
// -> Budget & Focus, ending in one unmissable before/after/savings/
// cost-saved summary. Every number comes straight from budget.Report
// (via AgentMetrics — see model.go/build.go's fillAgentMetricsFromReport)
// — nothing here recomputes a token count or a percentage a second time.
package diagram

import (
	"fmt"
	"strings"

	"mova.local/budget"
)

// reductionPipelineRow draws one box per agent with real Report data,
// showing where its tokens actually came from and what each reduction
// stage did to them. Skipped entirely in "simple" detail mode (see
// build()) — this is the expanded, "verbose" view; simple mode still
// gets the compact finalSummaryRow.
func (c *canvas) reductionPipelineRow(d *Data) {
	if d.Metrics == nil {
		return
	}
	agentsWithData := 0
	for _, am := range d.Metrics.PerAgent {
		if am.RawTokens > 0 || len(am.Components) > 0 {
			agentsWithData++
		}
	}
	if agentsWithData == 0 {
		return
	}
	c.sectionLabel("Token Reduction Pipeline (per resource)")
	labels := make([]string, 0, len(d.Metrics.PerAgent))
	for _, am := range d.Metrics.PerAgent {
		if am.RawTokens == 0 && len(am.Components) == 0 {
			continue
		}
		labels = append(labels, pipelineLines(am))
	}
	c.boxRow(labels, colorMetrics, nil)
	c.arrowDown()
}

// pipelineLines builds one box's full text for one agent's reduction
// pipeline — see this file's header for the honest "what's measured
// together vs. separately" note this respects: Deduplication happens
// INSIDE context assembly, before RawTokens is even measured, so its
// paragraph count is shown as its own fact, not folded into the
// Sanitizer/PII delta below it.
func pipelineLines(am AgentMetrics) string {
	var lines []string
	if am.Name != "" {
		lines = append(lines, am.Name)
	}
	for _, comp := range am.Components {
		if comp.Tokens > 0 {
			lines = append(lines, fmt.Sprintf("  %s: %d tok", comp.Name, comp.Tokens))
		}
	}
	if am.DuplicatesRemoved > 0 {
		lines = append(lines, fmt.Sprintf("Deduplication: %d duplicate paragraph(s) removed", am.DuplicatesRemoved))
	}
	if am.SanitizedLines > 0 {
		lines = append(lines, fmt.Sprintf("Sanitizer: %d repeated line(s) collapsed", am.SanitizedLines))
	}
	if am.PIIScanned > 0 {
		lines = append(lines, fmt.Sprintf("PII Masking: %d/%d token(s) pseudonymized (privacy, not a token reducer)", am.PIIMasked, am.PIIScanned))
	}
	if fc := am.FocusComparison; fc != nil && fc.TokensWithoutFocus > 0 {
		lines = append(lines, fmt.Sprintf("Budget & Focus: %d tok (whole repo) -> %d tok (focused), %.0f%% narrower", fc.TokensWithoutFocus, fc.TokensWithFocus, fc.SavingsPercent))
	}
	if am.RawTokens > 0 {
		lines = append(lines, fmt.Sprintf("Before Firewall: %d tok -> After: %d tok (%.0f%% reduction)", am.RawTokens, am.Tokens, am.SavingsPercent))
	}
	return strings.Join(lines, "\n")
}

// finalSummaryRow is the one box nobody can miss: raw input tokens,
// optimized tokens actually sent, overall savings, and — cloud models
// only, see #12 — the money that reduction saved. Always drawn (even
// in "simple" detail mode), since this is the headline result the
// whole feature exists to show (see this feature's own value-
// proposition note in README.md).
func (c *canvas) finalSummaryRow(d *Data) {
	c.sectionLabel("Final Summary")
	if d.Metrics == nil {
		c.text(40, c.y+10, fmt.Sprintf("No report data available yet — run \"mova budget %s\" first.", d.ProjectName), colorMuted, 13, false)
		c.advance(30)
		return
	}
	lines := []string{fmt.Sprintf("Tokens sent to the LLM (optimized): %d", d.Metrics.TotalTokens)}

	rawTotal, savingsPct := aggregateRaw(d.Metrics.PerAgent)
	if rawTotal > d.Metrics.TotalTokens {
		lines = append(lines, fmt.Sprintf("Tokens before Mova Context's reduction: %d", rawTotal))
		lines = append(lines, fmt.Sprintf("Overall savings: %.0f%% fewer tokens sent", savingsPct))
	}

	if anyCloudAgent(d.Metrics.PerAgent) {
		for _, cost := range d.Metrics.Costs {
			if cost.USD > 0 {
				lines = append(lines, fmt.Sprintf("%s/%s: $%.4f USD (optimized)", cost.Provider, cost.Model, cost.USD))
			}
		}
		if rawCost := aggregateRawCost(d.Metrics.PerAgent); rawCost != nil && rawCost.USD > 0 {
			for _, cost := range d.Metrics.Costs {
				if cost.Provider == rawCost.Provider && cost.Model == rawCost.Model && rawCost.USD > cost.USD {
					lines = append(lines, fmt.Sprintf("Estimated money saved (%s/%s): $%.4f USD", rawCost.Provider, rawCost.Model, rawCost.USD-cost.USD))
					break
				}
			}
		}
	}

	for _, am := range d.Metrics.PerAgent {
		seg := fmt.Sprintf("%s: %d tok", am.Name, am.Tokens)
		if am.IsLocal {
			seg += " (local — no cost)"
		} else if best := cheapestCost(am.Costs); best != nil {
			seg += fmt.Sprintf(", cheapest: %s/%s $%.4f", best.Provider, best.Model, best.USD)
		}
		if am.CircuitTriggered {
			seg += " [!] circuit breaker tripped"
		}
		lines = append(lines, seg)
	}

	h := 30 + len(lines)*20
	c.drawBoxAt(40, c.y, canvasW-80, h, colorMetrics, lines)
	c.advance(h + 10)
}

func anyCloudAgent(agents []AgentMetrics) bool {
	if len(agents) == 0 {
		return true // no per-agent breakdown available — default to showing the aggregate rather than silently hiding it
	}
	for _, a := range agents {
		if !a.IsLocal {
			return true
		}
	}
	return false
}

func cheapestCost(costs []budget.ModelCost) *budget.ModelCost {
	var best *budget.ModelCost
	for i := range costs {
		if costs[i].USD <= 0 {
			continue
		}
		if best == nil || costs[i].USD < best.USD {
			best = &costs[i]
		}
	}
	return best
}

// aggregateRaw sums every agent's RawTokens/current Tokens to produce
// one overall "before" figure and savings percentage for a whole
// group — a plain sum, not a re-derivation (each addend is already a
// real budget.Report.RawTokens/TotalTokens).
func aggregateRaw(agents []AgentMetrics) (rawTotal int, savingsPercent float64) {
	var finalTotal int
	for _, a := range agents {
		if a.RawTokens > 0 {
			rawTotal += a.RawTokens
			finalTotal += a.Tokens
		}
	}
	if rawTotal > 0 && rawTotal > finalTotal {
		savingsPercent = float64(rawTotal-finalTotal) / float64(rawTotal) * 100
	}
	return rawTotal, savingsPercent
}

// aggregateRawCost picks the first agent with a non-empty RawCosts —
// used only for the single-project case (a group's per-provider raw
// cost isn't summed, since mixing providers across agents into one
// dollar figure would misrepresent which provider actually costs
// what); good enough for the common case this line serves.
func aggregateRawCost(agents []AgentMetrics) *budget.ModelCost {
	for _, a := range agents {
		if a.IsLocal {
			continue
		}
		for i := range a.RawCosts {
			if a.RawCosts[i].USD > 0 {
				return &a.RawCosts[i]
			}
		}
	}
	return nil
}
