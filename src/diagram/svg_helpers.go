// svg_helpers.go — low-level drawing primitives for svg.go: rounded
// boxes with multi-line text, auto-centered rows of boxes, arrows,
// and the small string utilities (XML escaping, truncation, icon
// lookup) every row function in svg.go calls. Kept in its own file so
// svg.go stays about the diagram's STRUCTURE, this one about pixels.
package diagram

import (
	"strings"
)

const (
	boxPadding  = 12
	boxLineH    = 20
	boxMinW     = 160
	rowGapY     = 16
	arrowGapY   = 26
	charWidthPx = 7 // rough monospace-ish estimate, good enough for box sizing
)

// boxRow draws labels as same-colored boxes, evenly spaced, wrapping
// to a new line of boxes if they would overflow canvasW. Each label
// may contain "\n" for multi-line box text.
func (c *canvas) boxRow(labels []string, color string, notes []string) {
	colors := make([]string, len(labels))
	for i := range colors {
		colors[i] = color
	}
	c.boxRowColored(labels, colors)
}

// boxRowColored is boxRow with a per-box color (used by the Token
// Firewall row, where ON/OFF/triggered states need different colors).
func (c *canvas) boxRowColored(labels []string, colors []string) {
	x := 40
	rowH := 0
	for i, label := range labels {
		lines := strings.Split(label, "\n")
		w := boxWidthFor(lines)
		h := boxPadding*2 + len(lines)*boxLineH
		if x+w > canvasW-40 && x > 40 {
			x = 40
			c.y += rowH + rowGapY
			rowH = 0
		}
		c.drawBoxAt(x, c.y, w, h, colors[i], lines)
		if h > rowH {
			rowH = h
		}
		x += w + 20
	}
	c.advance(rowH)
}

// drawBoxAt draws one rounded rectangle at (x, y) sized (w, h), a
// colored left accent bar, and each of lines top-to-bottom inside it.
// The first line is drawn bold (the box's title); the rest muted.
func (c *canvas) drawBoxAt(x, y, w, h int, color string, lines []string) {
	c.w(`<rect x="%d" y="%d" width="%d" height="%d" rx="10" fill="%s" stroke="%s" stroke-width="2.5"/>`,
		x, y, w, h, colorPanel, color)
	c.w(`<rect x="%d" y="%d" width="8" height="%d" rx="2" fill="%s"/>`, x, y, h, color)
	for i, line := range lines {
		ty := y + boxPadding + (i+1)*boxLineH - 6
		size, bold, fill := 13, false, colorText
		if i == 0 {
			size, bold = 14, true
		} else {
			fill = colorMuted
		}
		c.text(x+16, ty, truncate(line, w/charWidthPx), fill, size, bold)
	}
}

// arrowDown draws one downward connector at canvas center, advancing
// the cursor past it — the visual "then" between two rows/sections.
func (c *canvas) arrowDown() {
	x := canvasW / 2
	y1, y2 := c.y+4, c.y+arrowGapY-6
	c.w(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="3"/>`, x, y1, x, y2, colorText)
	c.w(`<polygon points="%d,%d %d,%d %d,%d" fill="%s"/>`, x-8, y2-10, x+8, y2-10, x, y2, colorText)
	c.advance(arrowGapY)
}

// boxWidthFor sizes a box to its widest line, clamped to [boxMinW, canvasW-80].
func boxWidthFor(lines []string) int {
	widest := 0
	for _, l := range lines {
		if len(l) > widest {
			widest = len(l)
		}
	}
	w := widest*charWidthPx + boxPadding*2 + 8
	if w < boxMinW {
		w = boxMinW
	}
	if maxW := canvasW - 80; w > maxW {
		w = maxW
	}
	return w
}

func iconFor(kind string) string {
	switch kind {
	case "dir":
		return "[DIR]"
	case "glob":
		return "[GLOB]"
	case "symbol":
		return "[SYM]"
	default:
		return "[FILE]"
	}
}

func truncate(s string, max int) string {
	if max < 4 || len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// escXML escapes the handful of characters that would otherwise break
// SVG's XML syntax — deliberately not a full HTML escaper since diagram
// text is plain labels, never markup.
func escXML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
