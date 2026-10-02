// graph_legend.go — leyenda, título y ensamblado final del SVG.
// Extraído de layoutGraph sin cambiar la lógica.
package diagram

import (
	"fmt"
	"strings"
)

func (b *gbuilder) finish() glayout {
	g, c := b.g, b.c
	w := max(b.maxRight+gMargin, 980)
	ly := b.maxBottom + 40
	c.text(gMargin, ly, g.Labels.Legend, colorText, 14, true)
	ly += 16
	type sw struct{ label, stroke, fill, dash string }
	lx, rowH := gMargin, 26
	for _, s := range []sw{
		{g.Labels.File, colorAgent, "#e8eef5", ""}, {g.Labels.Func, colorSource, colorPanel, ""},
		{g.Labels.Class, colorCompiler, colorPanel, ""}, {g.Labels.Var, colorMetrics, colorPanel, ""},
		{g.Labels.Outside, "#94a3b8", colorBG, "6,4"}, {g.Labels.Excluded, colorTrigger, "#fef2f2", "3,3"},
	} {
		d := ""
		if s.dash != "" {
			d = ` stroke-dasharray="` + s.dash + `"`
		}
		b.add(float64(lx), float64(ly), float64(lx+22), float64(ly+14), `<rect x="%d" y="%d" width="22" height="14" rx="4" fill="%s" stroke="%s" stroke-width="2"%s/>`, lx, ly, s.fill, s.stroke, d)
		c.text(lx+30, ly+12, s.label, colorMuted, 13, false)
		lx += 30 + textW(s.label, 13, false) + 28
	}
	ly += rowH
	lx = gMargin
	type ln struct{ label, color, dash string }
	for _, l := range []ln{
		{g.Labels.Call, colorSource, ""}, {g.Labels.Ref, colorMetrics, ""},
		{g.Labels.Import, colorCompiler, "9,5"}, {g.Labels.Inferred, colorSource, "6,4"},
	} {
		d := ""
		if l.dash != "" {
			d = ` stroke-dasharray="` + l.dash + `"`
		}
		b.add(float64(lx), float64(ly+7), float64(lx+30), float64(ly+7), `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="2"%s/>`, lx, ly+7, lx+30, ly+7, l.color, d)
		b.add(float64(lx+27), float64(ly+3), float64(lx+36), float64(ly+11), `<polygon points="%d,%d %d,%d %d,%d" fill="%s"/>`, lx+36, ly+7, lx+27, ly+3, lx+27, ly+11, l.color)
		c.text(lx+44, ly+12, l.label, colorMuted, 13, false)
		lx += 44 + textW(l.label, 13, false) + 28
	}
	h := ly + rowH + gMargin
	if lx+gMargin > w {
		w = lx + gMargin
	}

	// ---- título (al final para conocer w) ---------------------------------
	c.text(gMargin, gMargin+22, g.Labels.Title, colorText, 28, true)
	for i, s := range g.Labels.Subtitle {
		c.text(gMargin, gMargin+52+22*i, s, colorMuted, 14, false)
	}

	var out strings.Builder
	fmt.Fprintf(&out, `<svg viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg" font-family="Segoe UI, Arial, sans-serif">`, w, h)
	fmt.Fprintf(&out, `<rect x="0" y="0" width="%d" height="%d" fill="%s"/>`, w, h, colorBG)
	out.WriteString(c.body.String())
	out.WriteString(`</svg>`)
	return glayout{out.String(), c.texts, b.shapes, w, h}
}
