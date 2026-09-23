// pipeline_diagram.go — context-diagram.png's PRIMARY content for a
// "discovery" (no active project.json) run: the governance pipeline
// flowchart requested for `mova context-trace` — REPOSITORY ->
// CONTEXT SELECTION -> SECURITY SCAN -> POLICY ENGINE -> FINAL CONTEXT
// -> TARGET LLM (Local/Cloud) — with the real file/token counts from
// sanitize.StateCounts at every stage (see governance.go). Reuses the
// exact same chartLayout/textOp shapes png.go already knows how to
// rasterize (see svg.go) — this is a second LAYOUT builder, not a
// second rendering engine.
package trace

import (
	"fmt"

	"mova.local/i18n"
)

// buildPipelineChart lays out the governance flowchart. Falls back to
// the plain composition chart (buildChart) whenever there is no
// meaningful StateTotals to show (e.g. a local project.json run that
// hasn't been wired into the per-file state machine yet — see
// AnalyzeLocal's doc comment).
func buildPipelineChart(d *Data) chartLayout {
	if d.StateTotals.DiscoveredFiles == 0 {
		return buildChart(d)
	}

	const w = 760
	var shapes []string
	var texts []textOp
	y := 30

	rect := func(x, ry, bw, bh int, fill string, radius int) {
		shapes = append(shapes, fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" rx="%d" fill="%s"/>`, x, ry, bw, bh, radius, fill))
	}
	stroke := func(x, ry, bw, bh int, col string, radius int) {
		shapes = append(shapes, fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" rx="%d" fill="none" stroke="%s" stroke-width="2"/>`, x, ry, bw, bh, radius, col))
	}
	line := func(x, ry, length int, vertical bool) {
		if vertical {
			shapes = append(shapes, fmt.Sprintf(`<rect x="%d" y="%d" width="2" height="%d" fill="%s"/>`, x, ry, length, chartMuted))
		} else {
			shapes = append(shapes, fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="2" fill="%s"/>`, x, ry, length, chartMuted))
		}
	}
	text := func(x, ty int, s string, size int, bold bool, color string) {
		texts = append(texts, textOp{X: x, Y: ty, Text: s, Size: size, Bold: bold, Color: color})
	}
	centerText := func(cx, ty int, s string, size int, bold bool, color string) {
		text(cx-len(s)*size/4, ty, s, size, bold, color)
	}

	rect(0, 0, w, 10000, chartBG, 0)

	text(30, y, "Mova Context Trace", 20, true, chartText)
	y += 20
	text(30, y, headerSubtitle(d), 12, false, chartMuted)
	y += 16
	text(30, y, i18n.T("reports.execution_commit_task", map[string]any{
		"execution": d.ExecutionID, "commit": orNA(d.CommitHash), "task": orTaskNone(d.TaskName),
	}), 10, false, chartMuted)
	y += 16
	text(30, y, fmt.Sprintf("%s: %s   ·   %s: %s   ·   %s: %s", i18n.T("reports.agent_client_label"), orNA(d.AgentClient), i18n.T("reports.target_model_label"), orNA(d.TargetModel), i18n.T("reports.policy_author_label"), orNA(d.PolicyAuthor)), 10, false, chartMuted)
	y += 26

	box := func(label, sub, color string) {
		bx, rightMargin, bh := 190, 40, 56
		bw := w - bx - rightMargin
		rect(bx, y, bw, bh, chartPanel, 10)
		stroke(bx, y, bw, bh, color, 10)
		centerText(bx+bw/2, y+22, label, 14, true, chartText)
		centerText(bx+bw/2, y+40, sub, 12, false, chartMuted)
		y += bh
		line(w/2, y, 18, true)
		y += 18
	}

	c := d.StateTotals
	box(i18n.T("diagrams.repository"), i18n.T("diagrams.repository_files_tok", map[string]any{"files": formatInt(c.DiscoveredFiles), "tokens": formatInt(c.DiscoveredTokens)}), chartBorder)
	box(i18n.T("diagrams.context_selection"), i18n.T("diagrams.candidate_tok", map[string]any{"tokens": formatInt(c.CandidateTokens)}), "#1d4ed8")

	// Branch labels (EXCLUDED / SANITIZED / BLOCKED) drawn as small
	// side notes to the LEFT of the SECURITY SCAN box rather than as
	// separate connected boxes, keeping the flowchart readable at this
	// size. x=15 (not x=30) and the box's left edge moved to x=190
	// (see box() above) together guarantee these labels never collide
	// with the box border, even for large token counts.
	branchY := y
	text(15, branchY, i18n.T("diagrams.excluded_tok", map[string]any{"tokens": formatInt(c.ExcludedTokens)}), 11, false, chartMuted)
	text(15, branchY+16, i18n.T("diagrams.would_sanitize_tok", map[string]any{"tokens": formatInt(c.SanitizedTokens)}), 11, false, chartWarn)
	text(15, branchY+32, i18n.T("diagrams.blocked_tok", map[string]any{"tokens": formatInt(c.BlockedTokens)}), 11, false, chartOver)

	box(i18n.T("diagrams.security_scan"), d.GovernanceStatus, "#6d28d9")
	box(i18n.T("diagrams.final_context"), i18n.T("diagrams.final_context_box", map[string]any{
		"safe": formatInt(c.SafeToSendNowTokens()), "would_sanitize": formatInt(c.SanitizedTokens),
	}), chartOK)

	bx, bw, bh := 60, w-120, 70
	rect(bx, y, bw/2-10, bh, chartPanel, 10)
	stroke(bx, y, bw/2-10, bh, chartOffColor, 10)
	centerText(bx+(bw/2-10)/2, y+18, i18n.T("diagrams.local_inference"), 13, true, chartText)
	centerText(bx+(bw/2-10)/2, y+34, i18n.T("diagrams.api_cost_zero"), 11, false, chartMuted)
	centerText(bx+(bw/2-10)/2, y+48, i18n.T("diagrams.infra_cost_not_included"), 9, false, chartMuted)

	rx := bx + bw/2 + 10
	rect(rx, y, bw/2-10, bh, chartPanel, 10)
	stroke(rx, y, bw/2-10, bh, chartOffColor, 10)
	centerText(rx+(bw/2-10)/2, y+16, i18n.T("diagrams.external_model"), 13, true, chartText)
	centerText(rx+(bw/2-10)/2, y+30, i18n.T("diagrams.estimated_input_cost_label"), 9, false, chartMuted)
	for i, cl := range externalCostLines(d, 2) {
		centerText(rx+(bw/2-10)/2, y+30+12*(i+1), cl, 10, false, chartMuted)
	}
	y += bh + 18
	line(w/2, y-18, 18, true)

	centerText(w/2, y+16, i18n.T("diagrams.target_llm"), 14, true, chartText)
	y += 40

	var sb string
	for _, s := range shapes {
		sb += s
	}
	return chartLayout{Width: w, Height: y + 20, ShapesSVG: sb, Texts: texts}
}

// externalCostLines returns up to max short "Model: $X.XX" lines from
// d.Costs (already computed over the final sendable context only —
// see costBasisTokens) - the SAME numbers the report/console cost
// tables show, never a value recomputed here (see the traceability
// requirement: diagram/report/console must never disagree).
func externalCostLines(d *Data, max int) []string {
	var lines []string
	for i, c := range d.Costs {
		if i >= max {
			break
		}
		lines = append(lines, fmt.Sprintf("%s: %s", c.Model, FormatCost(c.USD)))
	}
	if len(lines) == 0 {
		lines = []string{i18n.T("diagrams.see_context_report_pricing")}
	}
	return lines
}
