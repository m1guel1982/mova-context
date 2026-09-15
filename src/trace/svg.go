// svg.go — builds the visual composition chart shown in diagram.png
// (see png.go for the actual rasterizer): a light-themed card matching
// mova.local/diagram's own palette (white background, the same
// blue/violet/amber/cyan accents — see diagram/svg.go's colorSource,
// colorCompiler, colorFirewall, colorAgent constants), so
// context-trace's diagram looks like it belongs to the same product
// instead of a one-off dark chart.
//
// oksvg (the pure-Go SVG rasterizer png.go uses) does not implement
// <text> at all — verified by rendering an early version of this file
// and finding every label missing, the same discovery diagram/png.go's
// header documents. So this file builds two separate things: a
// SHAPES-ONLY svg string (rects, safe for oksvg) and a parallel list
// of textOp values that png.go draws in a second pass with a real
// vector font.
package trace

import (
	"fmt"
	"strings"

	"mova.local/i18n"
)

// textOp is one label to draw in the font-rendering pass — same shape
// as diagram/png.go's own (unexported, so not reusable across
// packages) textOp, duplicated here on purpose rather than exporting
// diagram's internals just for this.
type textOp struct {
	X, Y  int
	Text  string
	Size  int
	Bold  bool
	Color string
}

const (
	chartW        = 900
	chartBG       = "#ffffff"
	chartPanel    = "#f8fafc"
	chartText     = "#0f172a"
	chartMuted    = "#475569"
	chartBorder   = "#e2e8f0"
	chartOK       = "#15803d"
	chartWarn     = "#b45309"
	chartOver     = "#b91c1c"
	chartOffColor = "#64748b"
)

var compositionColors = map[string]string{
	"Agents":               "#1d4ed8", // blue - matches diagram's colorSource
	"Skills":               "#6d28d9", // violet - matches diagram's colorCompiler
	"Prompt":               "#b45309", // amber - matches diagram's colorFirewall
	"Focus":                "#0e7490", // cyan - matches diagram's colorAgent
	"Memory":               "#7e22ce", // purple - matches diagram's colorJob
	"Repository (scanned)": "#1d4ed8",
}

// chartLayout is what png.go needs to rasterize: shapes it can hand to
// oksvg as-is, plus the labels to draw on top, plus the final canvas
// size.
type chartLayout struct {
	Width, Height int
	ShapesSVG     string
	Texts         []textOp
}

// buildChart lays out the whole diagram: header, composition bar +
// legend, an optional "top directories" mini bar chart (remote mode
// only, see analyzer.go's DirBreakdown), and a budget gauge at the
// bottom.
func buildChart(d *Data) chartLayout {
	var shapes strings.Builder
	var texts []textOp
	y := 0

	rect := func(x, ry, w, h int, fill string, radius int) {
		fmt.Fprintf(&shapes, `<rect x="%d" y="%d" width="%d" height="%d" rx="%d" fill="%s"/>`, x, ry, w, h, radius, fill)
	}
	rectStroke := func(x, ry, w, h int, stroke string, radius int) {
		fmt.Fprintf(&shapes, `<rect x="%d" y="%d" width="%d" height="%d" rx="%d" fill="none" stroke="%s" stroke-width="1"/>`, x, ry, w, h, radius, stroke)
	}
	text := func(x, ty int, s string, size int, bold bool, color string) {
		texts = append(texts, textOp{X: x, Y: ty, Text: s, Size: size, Bold: bold, Color: color})
	}

	rect(0, 0, chartW, 10000, chartBG, 0) // background; height trimmed to the real total at the end

	// ── Header ──────────────────────────────────────────────────────
	y = 34
	text(30, y, "Mova Context Trace", 22, true, chartText)
	y += 22
	subtitle := headerSubtitle(d)
	text(30, y, subtitle, 13, false, chartMuted)
	y += 20
	identity := fmt.Sprintf("Agent: %s   ·   Target model: %s   ·   Policy author: %s", orNA(d.AgentClient), orNA(d.TargetModel), orNA(d.PolicyAuthor))
	text(30, y, identity, 11, false, chartMuted)
	y += 26

	// ── Composition bar ─────────────────────────────────────────────
	rows := d.Components
	if len(rows) == 0 {
		rows = []ComponentRow{{Name: "Repository (scanned)", Tokens: d.TotalTokens}}
	}
	total := d.TotalTokens
	if total == 0 {
		total = 1
	}
	barX, barW, barH := 30, chartW-60, 34
	rect(barX, y, barW, barH, chartPanel, 8)
	x := barX
	for _, r := range rows {
		w := int(float64(barW) * float64(r.Tokens) / float64(total))
		rect(x, y, w, barH, colorFor(r.Name), 0)
		x += w
	}
	rectStroke(barX, y, barW, barH, chartBorder, 8)
	y += barH + 22

	text(30, y, "CONTEXT COMPOSITION", 12, true, chartMuted)
	y += 20
	for _, r := range rows {
		pct := float64(r.Tokens) / float64(total) * 100
		fmt.Fprintf(&shapes, `<rect x="30" y="%d" width="14" height="14" rx="3" fill="%s"/>`, y-12, colorFor(r.Name))
		text(52, y, fmt.Sprintf("%s - %s tok (%s)", r.Name, formatInt(r.Tokens), formatPct(pct)), 13, false, chartText)
		y += 24
	}
	y += 10

	// ── Top directories (remote mode only) ──────────────────────────
	if len(d.DirBreakdown) > 0 {
		y = appendDirChart(&shapes, &texts, d, y)
	}

	// ── Budget gauge ─────────────────────────────────────────────────
	y = appendBudgetGauge(&shapes, &texts, d, y)

	// ── Footer ───────────────────────────────────────────────────────
	y += 10
	rectStroke(0, 0, chartW, y, chartBorder, 0)
	text(30, y-6, i18n.T("reports.generated_by_footer"), 10, false, chartMuted)
	y += 16

	return chartLayout{Width: chartW, Height: y, ShapesSVG: shapes.String(), Texts: texts}
}

func headerSubtitle(d *Data) string {
	if d.IsRemote {
		return i18n.T("reports.repository_branch", map[string]any{"repo": d.RepoURL, "branch": orNA(d.Branch)})
	}
	return i18n.T("reports.project_task", map[string]any{"project": d.ProjectName, "task": d.TaskName})
}

func colorFor(name string) string {
	if c, ok := compositionColors[name]; ok {
		return c
	}
	return chartOffColor
}

// maxDirChartRows caps the visual mini bar chart tighter than the
// report table (maxDirRows) - a diagram is meant to be read in a few
// seconds, not scrolled.
const maxDirChartRows = 8

func appendDirChart(shapes *strings.Builder, texts *[]textOp, d *Data, y int) int {
	text := func(x, ty int, s string, size int, bold bool, color string) {
		*texts = append(*texts, textOp{X: x, Y: ty, Text: s, Size: size, Bold: bold, Color: color})
	}

	// 1. Título de la sección
	text(30, y+10, "TOP DIRECTORIES BY TOKEN USAGE", 12, true, chartMuted)

	// Aumentamos el margen vertical de 20 a 28px para dar suficiente aire
	// entre el título y la primera barra azul, eliminando el traslape visual.
	y += 28

	rows := d.DirBreakdown
	if len(rows) > maxDirChartRows {
		rows = rows[:maxDirChartRows]
	}
	maxTokens := 1
	for _, r := range rows {
		if r.Tokens > maxTokens {
			maxTokens = r.Tokens
		}
	}
	barMaxW := 260
	for _, r := range rows {
		w := int(float64(barMaxW) * float64(r.Tokens) / float64(maxTokens))

		// 2. Ancho mínimo aumentado de 2px a 6px para que subdirectorios
		// pequeños pero importantes sigan siendo claramente visibles en el diagrama.
		if w < 6 {
			w = 6
		}

		fmt.Fprintf(shapes, `<rect x="180" y="%d" width="%d" height="14" rx="3" fill="#1d4ed8"/>`, y-11, w)
		text(30, y, truncateDir(r.Dir, 20), 12, false, chartText)
		text(180+barMaxW+10, y, fmt.Sprintf("%s tok (%s)", formatInt(r.Tokens), formatPct(r.Percent)), 11, false, chartMuted)
		y += 22
	}
	return y + 8
}

func appendBudgetGauge(shapes *strings.Builder, texts *[]textOp, d *Data, y int) int {
	text := func(x, ty int, s string, size int, bold bool, color string) {
		*texts = append(*texts, textOp{X: x, Y: ty, Text: s, Size: size, Bold: bold, Color: color})
	}

	text(30, y+10, "BUDGET", 12, true, chartMuted)
	y += 20

	barX, barW, barH := 30, chartW-60, 22
	if d.MaxTokens <= 0 {
		fmt.Fprintf(shapes, `<rect x="%d" y="%d" width="%d" height="%d" rx="6" fill="none" stroke="%s" stroke-width="1"/>`, barX, y, barW, barH, chartBorder)
		text(38, y+15, "N/A - no token limit configured", 12, false, chartMuted)
		return y + barH + 10
	}

	pct := float64(d.TotalTokens) / float64(d.MaxTokens)
	fill := chartOK
	if pct >= 1.0 {
		fill = chartOver
	} else if pct >= 0.8 {
		fill = chartWarn
	}
	fmt.Fprintf(shapes, `<rect x="%d" y="%d" width="%d" height="%d" rx="6" fill="%s"/>`, barX, y, barW, barH, chartPanel)
	filledW := int(float64(barW) * minFloat(pct, 1.0))
	fmt.Fprintf(shapes, `<rect x="%d" y="%d" width="%d" height="%d" rx="6" fill="%s"/>`, barX, y, filledW, barH, fill)
	fmt.Fprintf(shapes, `<rect x="%d" y="%d" width="%d" height="%d" rx="6" fill="none" stroke="%s" stroke-width="1"/>`, barX, y, barW, barH, chartBorder)
	text(38, y+16, fmt.Sprintf("%s / %s tokens (%s)", formatInt(d.TotalTokens), formatInt(d.MaxTokens), formatPct(pct*100)), 12, true, "#ffffff")
	return y + barH + 10
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
