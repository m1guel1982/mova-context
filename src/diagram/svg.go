// svg.go — RenderSVG(data) turns a diagram.Data into a real, vectorial
// SVG: a vertical "visual storytelling" flow (sources → context
// compiler → context governance → agents → jobs → interfaces → metrics),
// light theme (WCAG AA contrast or better on every text/background
// pair — see the palette below), colored by category, every box drawn
// from a real field on Data — nothing here invents a stage or a
// number that Data didn't carry. png.go rasterizes this exact same SVG
// string for PNG/PDF, so all three export formats are always visually
// identical.
package diagram

import (
	"fmt"
	"strconv"
	"strings"

	"mova.local/i18n"
)

// Palette — light, high-contrast, presentation-safe. Every text/
// background pair below clears WCAG AA's 4.5:1 minimum contrast ratio
// (verified by computing relative luminance, not eyeballed) so labels
// stay legible printed, projected, or viewed on a phone.
const (
	colorBG       = "#ffffff"
	colorPanel    = "#f8fafc"
	colorText     = "#0f172a"
	colorMuted    = "#475569"
	colorSource   = "#1d4ed8" // blue
	colorCompiler = "#6d28d9" // violet
	colorFirewall = "#b45309" // amber
	colorOn       = "#15803d" // green
	colorOff      = "#64748b" // slate (disabled/off)
	colorTrigger  = "#b91c1c" // red — a limit actually tripped
	colorAgent    = "#0e7490" // cyan
	colorProvider = "#c2410c" // orange
	colorMetrics  = "#a16207" // yellow/amber
)

const canvasW = 1280

// canvas accumulates SVG body content while tracking the vertical
// cursor, so every row function only needs to know its own height.
// texts records every text draw op separately from the SVG markup:
// oksvg (png.go's rasterizer) does not implement SVG <text> at all, so
// PNG/PDF export re-draws these directly onto the raster afterwards —
// see png.go's drawTextLayer. Both paths originate from the exact same
// c.text() calls below, so SVG and PNG/PDF never drift apart.
type canvas struct {
	body  strings.Builder
	y     int
	maxY  int
	texts []textOp
}

// textOp is one label drawn at (X, Y) in the SVG's own un-scaled
// viewBox coordinates — png.go scales X/Y/Size by pngScale itself.
type textOp struct {
	X, Y  int
	Text  string
	Color string
	Size  int
	Bold  bool
}

func (c *canvas) w(format string, args ...any) {
	fmt.Fprintf(&c.body, format, args...)
}

// text is the ONLY place this package emits an SVG <text> element —
// every label in svg.go/svg_helpers.go goes through this so it's
// automatically also recorded in c.texts for PNG/PDF's raster pass.
func (c *canvas) text(x, y int, s, color string, size int, bold bool) {
	weight := "400"
	if bold {
		weight = "700"
	}
	c.w(`<text x="%d" y="%d" fill="%s" font-size="%d" font-weight="%s">%s</text>`, x, y, color, size, weight, escXML(s))
	c.texts = append(c.texts, textOp{X: x, Y: y, Text: s, Color: color, Size: size, Bold: bold})
}

func (c *canvas) advance(h int) {
	c.y += h
	if c.y > c.maxY {
		c.maxY = c.y
	}
}

// RenderSVG builds the full diagram for data. Every section below
// checks whether it has anything real to draw before drawing it.
func RenderSVG(data *Data) string {
	c := build(data)
	return c.finalize()
}

// build runs the actual layout pass, shared by RenderSVG (which only
// needs the finished SVG string) and png.go/pdf.go (which also need
// c.texts to re-draw labels on the raster afterwards — see canvas's
// own doc comment for why).
func build(data *Data) *canvas {
	c := &canvas{y: 20}
	verbose := data.DetailLevel != DetailSimple

	c.title(data)
	c.sourcesRow(data)
	if verbose {
		c.compilerRow(data)
	}
	c.firewallRow(data)
	c.agentsRow(data, verbose)
	c.interfacesRow(data)
	c.metricsRow(data)
	c.legend()
	return c
}

func (c *canvas) finalize() string {
	height := c.maxY + 40
	var out strings.Builder
	out.WriteString(fmt.Sprintf(`<svg viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg" font-family="Segoe UI, Arial, sans-serif">`, canvasW, height))
	out.WriteString(fmt.Sprintf(`<rect x="0" y="0" width="%d" height="%d" fill="%s"/>`, canvasW, height, colorBG))
	out.WriteString(c.body.String())
	out.WriteString(`</svg>`)
	return out.String()
}

func (c *canvas) title(d *Data) {
	kind := "Project"
	if d.IsGroup {
		kind = "Multi-Agent Group"
	}
	c.text(40, c.y+10, "Mova Context — "+d.ProjectName, colorText, 30, true)
	subtitle := kind
	if d.Description != "" {
		subtitle += "  ·  " + truncate(d.Description, 110)
	}
	c.text(40, c.y+36, subtitle, colorMuted, 15, false)
	c.advance(56)
	identity := fmt.Sprintf("%s: %s   ·   %s: %s   ·   %s: %s", i18n.T("reports.agent_client_label"), orNAd(d.AgentClient), i18n.T("reports.target_model_label"), orNAd(d.TargetModel), i18n.T("reports.policy_author_label"), orNAd(d.PolicyAuthor))
	c.text(40, c.y+10, identity, colorMuted, 13, false)
	c.advance(30)
}

// orNAd renders "n/a" instead of an empty string — the diagram never
// shows a blank identity field (see Data.AgentClient/TargetModel/
// PolicyAuthor's doc comments in model.go).
func orNAd(v string) string {
	if strings.TrimSpace(v) == "" {
		return "n/a"
	}
	return v
}

func (c *canvas) sectionLabel(label string) {
	c.text(40, c.y+16, strings.ToUpper(label), colorMuted, 14, true)
	c.advance(28)
}

func (c *canvas) sourcesRow(d *Data) {
	if len(d.Sources) == 0 {
		return
	}
	c.sectionLabel(i18n.T("diagrams.sources_focus"))
	labels := make([]string, len(d.Sources))
	for i, s := range d.Sources {
		labels[i] = iconFor(s.Kind) + " " + s.Path
	}
	c.boxRow(labels, colorSource, nil)
	c.arrowDown()
}

func (c *canvas) compilerRow(d *Data) {
	if len(d.Compiler) == 0 {
		return
	}
	c.sectionLabel(i18n.T("diagrams.context_compiler"))
	c.boxRow(d.Compiler, colorCompiler, nil)
	c.arrowDown()
}

func (c *canvas) firewallRow(d *Data) {
	c.sectionLabel(i18n.T("diagrams.token_firewall"))
	f := d.Firewall
	type stage struct {
		label string
		on    bool
		note  string
	}
	stages := []stage{
		{i18n.T("diagrams.sanitizer"), f.SanitizerOn, sanitizerNote(f)},
		{i18n.T("diagrams.pii_masking_optin"), f.PIIMaskingOn, i18n.T("diagrams.pii_masking_structural_note")},
		{i18n.T("diagrams.cache_layout_guard"), f.CacheGuardOn, ""},
		{i18n.T("diagrams.circuit_breaker"), f.CircuitBreakerOn, breakerNote(f)},
	}
	labels := make([]string, len(stages))
	colors := make([]string, len(stages))
	for i, s := range stages {
		state := i18n.T("diagrams.state_on")
		colors[i] = colorOn
		if !s.on {
			state = i18n.T("diagrams.state_off")
			colors[i] = colorOff
		}
		labels[i] = s.label + "\n[" + state + "]"
		if s.note != "" {
			labels[i] += "\n" + s.note
		}
	}
	c.boxRowColored(labels, colors)
	c.arrowDown()
}

func sanitizerNote(f Firewall) string {
	var parts []string
	if f.DedupeLogsOn {
		parts = append(parts, "dedupe")
	}
	if f.StripBlankOn {
		parts = append(parts, "blank-strip")
	}
	if f.StripCommentsOn {
		parts = append(parts, "comments")
	}
	return strings.Join(parts, "+")
}

func breakerNote(f Firewall) string {
	if f.MaxTokensPerRun == 0 && f.MaxMonthlyUSD == 0 {
		return ""
	}
	var parts []string
	if f.MaxTokensPerRun > 0 {
		parts = append(parts, strconv.Itoa(f.MaxTokensPerRun)+" tok/run")
	}
	if f.MaxMonthlyUSD > 0 {
		parts = append(parts, fmt.Sprintf("$%.2f/mo", f.MaxMonthlyUSD))
	}
	return strings.Join(parts, ", ")
}

func (c *canvas) agentsRow(d *Data, verbose bool) {
	label := i18n.T("diagrams.agent_label")
	if d.IsGroup {
		label = i18n.T("diagrams.multi_agent_group")
	}
	c.sectionLabel(label)
	boxW := canvasW/max(1, min(len(d.Agents), 3)) - 40
	x := 40
	rowH := 0
	for _, a := range d.Agents {
		h := c.drawAgentBox(x, c.y, boxW, a, verbose)
		if h > rowH {
			rowH = h
		}
		x += boxW + 20
		if x+boxW > canvasW {
			x = 40
			c.y += rowH + 16
			rowH = 0
		}
	}
	c.advance(rowH + 16)
	c.arrowDown()
}

func (c *canvas) drawAgentBox(x, y, w int, a AgentNode, verbose bool) int {
	lines := []string{a.Name}
	if a.Description != "" {
		lines = append(lines, truncate(a.Description, 60))
	}
	if verbose {
		if len(a.AgentRoles) > 0 {
			lines = append(lines, i18n.T("diagrams.agents_list_prefix")+strings.Join(a.AgentRoles, ", "))
		}
		if len(a.Skills) > 0 {
			lines = append(lines, i18n.T("diagrams.skills_list_prefix")+strings.Join(a.Skills, ", "))
		}
		if len(a.Tasks) > 0 {
			lines = append(lines, i18n.T("diagrams.tasks_list_prefix")+strings.Join(a.Tasks, ", "))
		}
	}
	if a.ModelName != "" {
		where := i18n.T("diagrams.cloud")
		if a.IsLocal {
			where = i18n.T("diagrams.local")
		}
		model := i18n.T("diagrams.model_prefix") + a.ModelName
		if a.Provider != "" {
			model += " (" + a.Provider + ", " + where + ")"
		} else {
			model += " (" + where + ")"
		}
		lines = append(lines, model)
	}
	piiState := i18n.T("diagrams.state_off")
	if a.PIIMasking {
		piiState = i18n.T("diagrams.state_on")
	}
	lines = append(lines, i18n.T("diagrams.pii_masking_bool", map[string]any{"value": piiState}))
	h := 40 + len(lines)*20
	c.drawBoxAt(x, y, w, h, colorAgent, lines)
	return h
}

// interfacesRow draws the four doors a diagram render (or any other
// Mova Context capability) can be triggered from, highlighting
// whichever one actually triggered THIS render (d.Origin) so the
// picture also answers "how was this run started?" — see model.go's
// Data.Origin doc comment for who sets it.
func (c *canvas) interfacesRow(d *Data) {
	c.sectionLabel(i18n.T("diagrams.available_interfaces"))
	labels := make([]string, len(d.Interfaces))
	colors := make([]string, len(d.Interfaces))
	for i, iface := range d.Interfaces {
		if iface == d.Origin {
			labels[i] = iface + "\n" + i18n.T("diagrams.this_run")
			colors[i] = colorOn
		} else {
			labels[i] = iface
			colors[i] = colorOff
		}
	}
	c.boxRowColored(labels, colors)
	c.arrowDown()
}

// metricsRow — see #12: cost figures are shown ONLY for cloud models.
// A local model (Ollama, "no local model shown = no cost", see
// AgentMetrics.IsLocal) never gets a dollar figure next to it — not
// because the number would be wrong, but because it never leaves the
// machine and showing $0.00 would misleadingly imply a real
// cloud-provider cost was computed and happened to be free. See
// svg_metrics.go for reductionPipelineRow (the detailed per-resource
// breakdown, #18) and finalSummaryRow (the headline before/after/
// savings box this function now delegates to).
func (c *canvas) metricsRow(d *Data) {
	if d.DetailLevel != DetailSimple {
		c.reductionPipelineRow(d)
	}
	c.finalSummaryRow(d)
}

func (c *canvas) legend() {
	c.text(40, c.y+10, i18n.T("diagrams.footer_disclaimer"), colorMuted, 11, false)
	c.advance(30)
}
