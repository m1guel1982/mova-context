// graph_draw.go — puertos de las aristas y dibujo de clusters, aristas,
// nodos y flechas. Extraído de layoutGraph sin cambiar la lógica.
package diagram

import "math"

func (b *gbuilder) boxOf(id string, isCluster bool) gbox {
	if isCluster {
		c := b.cbox[id]
		return gbox{c.x, c.y, c.w, gHeaderH}
	}
	return b.nbox[id]
}

// sideOf: por qué lado sale y entra una arista.
func (b *gbuilder) sideOf(e GraphEdge) (string, string) {
	if e.Kind != "import" && b.clusterOf(e.From, false) == b.clusterOf(e.To, false) { // mismo archivo: por el pasillo entre sub-columnas
		a, c := b.nbox[e.From], b.nbox[e.To]
		switch {
		case c.x > a.x:
			return "R", "L"
		case c.x < a.x:
			return "L", "R"
		}
		return "R", "R"
	}
	da := b.depth[b.clusterOf(e.From, e.Kind == "import")]
	db := b.depth[b.clusterOf(e.To, e.Kind == "import")]
	switch {
	case db > da:
		return "R", "L"
	case db < da:
		return "L", "R"
	}
	return "R", "R"
}

// computePorts cuenta cuántas aristas usa cada puerto (cada una tomará
// una ordenada distinta).
func (b *gbuilder) computePorts() {
	b.total, b.used = map[gport]int{}, map[gport]int{}
	for _, e := range b.g.Edges {
		if _, ok := boxKnown(e, b.nbox, b.cbox); !ok || e.Kind == "import" {
			continue
		}
		s1, s2 := b.sideOf(e)
		b.total[gport{e.From, s1}]++
		b.total[gport{e.To, s2}]++
	}
}

func (b *gbuilder) port(id, side string, isCluster bool) (float64, float64) {
	bx := b.boxOf(id, isCluster)
	p := gport{id, side}
	n, j := b.total[p], b.used[p]
	b.used[p]++
	off := (float64(j) - float64(n-1)/2) * 5
	lim := float64(bx.h)/2 - 4
	off = math.Max(-lim, math.Min(lim, off))
	x := float64(bx.x)
	if side == "R" {
		x = float64(bx.x + bx.w)
	}
	return x, float64(bx.y) + float64(bx.h)/2 + off
}

func (b *gbuilder) drawClusters() {
	for _, cl := range b.g.Clusters {
		bx := b.cbox[cl.ID]
		b.add(float64(bx.x), float64(bx.y), float64(bx.x+bx.w), float64(bx.y+bx.h), `<rect x="%d" y="%d" width="%d" height="%d" rx="10" fill="%s" stroke="#cbd5e1" stroke-width="2"/>`, bx.x, bx.y, bx.w, bx.h, colorPanel)
		b.add(float64(bx.x), float64(bx.y), float64(bx.x+bx.w), float64(bx.y+gHeaderH), `<rect x="%d" y="%d" width="%d" height="%d" rx="10" fill="#e8eef5"/>`, bx.x, bx.y, bx.w, gHeaderH)
		b.add(float64(bx.x), float64(bx.y), float64(bx.x+8), float64(bx.y+gHeaderH), `<rect x="%d" y="%d" width="8" height="%d" rx="2" fill="%s"/>`, bx.x, bx.y, gHeaderH, colorAgent)
		b.c.text(bx.x+18, bx.y+22, cl.Label, colorText, 14, true)
	}
}

func (b *gbuilder) drawEdges() {
	bulge, lane := 0, 0
	topUse := map[string]int{}
	topPort := func(id string) float64 {
		bx := b.cbox[id]
		x := bx.x + 26 + 16*topUse[id]
		topUse[id]++
		return float64(min(x, bx.x+bx.w-16))
	}
	for _, e := range b.g.Edges {
		if _, ok := boxKnown(e, b.nbox, b.cbox); !ok {
			continue
		}
		if e.Kind == "import" { // carril horizontal sobre todos los archivos, baja al destino
			x1, x2 := topPort(e.From), topPort(e.To)
			yl := float64(b.clTop - 12 - 9*lane)
			lane++
			b.add(fmin(x1, x2), yl, fmax(x1, x2), float64(max(b.cbox[e.From].y, b.cbox[e.To].y)), `<path d="M %.1f %d L %.1f %.1f L %.1f %.1f L %.1f %d" fill="none" stroke="%s" stroke-width="2.4" stroke-dasharray="9,5"/>`,
				x1, b.cbox[e.From].y, x1, yl, x2, yl, x2, b.cbox[e.To].y, edgeColor("import"))
			b.arrows = append(b.arrows, garrow{x2, float64(b.cbox[e.To].y), 0, 1, edgeColor("import")})
			continue
		}
		s1, s2 := b.sideOf(e)
		x1, y1 := b.port(e.From, s1, false)
		x2, y2 := b.port(e.To, s2, false)
		var c1x, c2x float64
		switch {
		case s1 == "R" && s2 == "L":
			dx := math.Max(40, (x2-x1)*0.45)
			c1x, c2x = x1+dx, x2-dx
		case s1 == "L" && s2 == "R":
			dx := math.Max(40, (x1-x2)*0.45)
			c1x, c2x = x1-dx, x2+dx
		default:
			bx := math.Max(x1, x2) + 16 + float64(bulge%4)*7
			bulge++
			c1x, c2x = bx, bx
		}
		col := edgeColor(e.Kind)
		width, dash := "1.6", ""
		if e.Inferred {
			dash = ` stroke-dasharray="6,4"`
		}
		b.add(fmin(x1, c1x, c2x, x2), fmin(y1, y2), fmax(x1, c1x, c2x, x2), fmax(y1, y2), `<path d="M %.1f %.1f C %.1f %.1f %.1f %.1f %.1f %.1f" fill="none" stroke="%s" stroke-width="%s"%s/>`, x1, y1, c1x, y1, c2x, y2, x2, y2, col, width, dash)
		b.arrows = append(b.arrows, garrow{x2, y2, x2 - c2x, 0, col})
	}
}

func (b *gbuilder) drawNodes() {
	for _, n := range b.g.Nodes {
		bx := b.nbox[n.ID]
		stroke, fill, tcol, dash := kindColor(n.Kind), colorPanel, colorText, ""
		switch n.Style {
		case "outside":
			stroke, fill, tcol, dash = "#94a3b8", colorBG, colorMuted, ` stroke-dasharray="6,4"`
		case "excluded":
			stroke, fill, tcol, dash = colorTrigger, "#fef2f2", colorTrigger, ` stroke-dasharray="3,3"`
		}
		b.add(float64(bx.x), float64(bx.y), float64(bx.x+bx.w), float64(bx.y+bx.h), `<rect x="%d" y="%d" width="%d" height="%d" rx="8" fill="%s" stroke="%s" stroke-width="2"%s/>`, bx.x, bx.y, bx.w, bx.h, fill, stroke, dash)
		b.add(float64(bx.x), float64(bx.y), float64(bx.x+7), float64(bx.y+bx.h), `<rect x="%d" y="%d" width="7" height="%d" rx="2" fill="%s"/>`, bx.x, bx.y, bx.h, stroke)
		b.c.text(bx.x+16, bx.y+19, truncate(n.Label, bx.w/7), tcol, 13, n.Style != "outside")
	}
}

func (b *gbuilder) drawArrows() {
	for _, a := range b.arrows {
		norm := math.Hypot(a.dx, a.dy)
		if norm == 0 {
			continue
		}
		ux, uy := a.dx/norm, a.dy/norm
		bx, by := a.x-ux*10, a.y-uy*10
		b.add(fmin(a.x, bx-uy*5, bx+uy*5), fmin(a.y, by+ux*5, by-ux*5), fmax(a.x, bx-uy*5, bx+uy*5), fmax(a.y, by+ux*5, by-ux*5), `<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s"/>`, a.x, a.y, bx-uy*5, by+ux*5, bx+uy*5, by-ux*5, a.color)
	}
}
