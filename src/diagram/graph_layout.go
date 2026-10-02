// graph_layout.go — fases de cálculo del layout del grafo: índices,
// métricas por cluster, columnas por dependencia y posiciones.
// Extraído de layoutGraph sin cambiar la lógica.
package diagram

import (
	"fmt"
	"math"
	"sort"
)

type gcmetric struct{ cols, rows, nodeW, w, h int }

// gbuilder acumula el estado que antes eran variables locales de
// layoutGraph.
type gbuilder struct {
	g          *GraphData
	c          *canvas
	shapes     []gshape
	nodeIdx    map[string]int
	byCluster  map[string][]int
	clusterIdx map[string]int
	cm         []gcmetric

	adj      map[string]map[string]bool
	depth    map[string]int
	cols     map[int][]string
	maxDepth int
	incoming map[int]int

	clTop, maxBottom, maxRight int
	cbox, nbox                 map[string]gbox

	total, used map[gport]int
	arrows      []garrow
}

func newGBuilder(g *GraphData) *gbuilder {
	b := &gbuilder{g: g, c: &canvas{}, nodeIdx: map[string]int{}, byCluster: map[string][]int{}, clusterIdx: map[string]int{}}
	for i, n := range g.Nodes {
		b.nodeIdx[n.ID] = i
		b.byCluster[n.Cluster] = append(b.byCluster[n.Cluster], i)
	}
	for i, cl := range g.Clusters {
		b.clusterIdx[cl.ID] = i
	}
	return b
}

// add emite un elemento SVG y registra su caja para el rasterizador.
func (b *gbuilder) add(x0, y0, x1, y1 float64, format string, args ...any) {
	el := fmt.Sprintf(format, args...)
	b.c.body.WriteString(el)
	b.shapes = append(b.shapes, gshape{el, x0, y0, x1, y1})
}

func (b *gbuilder) clusterOf(id string, isCluster bool) string {
	if isCluster {
		return id
	}
	if i, ok := b.nodeIdx[id]; ok {
		return b.g.Nodes[i].Cluster
	}
	return ""
}

// computeMetrics: columnas internas, ancho de nodo y tamaño de cada cluster.
func (b *gbuilder) computeMetrics() {
	g := b.g
	b.cm = make([]gcmetric, len(g.Clusters))
	for i, cl := range g.Clusters {
		idxs := b.byCluster[cl.ID]
		cols := 1
		if len(idxs) > 24 {
			cols = 3
		} else if len(idxs) > 10 {
			cols = 2
		}
		nw := 150
		for _, ni := range idxs {
			if w := textW(g.Nodes[ni].Label, 13, true) + 44; w > nw {
				nw = w
			}
		}
		rows := (len(idxs) + cols - 1) / cols
		if rows == 0 {
			rows = 1
		}
		w := 2*gPad + cols*nw + (cols-1)*gSubGap
		if tw := textW(cl.Label, 14, true) + 40; tw > w {
			w = tw
		}
		b.cm[i] = gcmetric{cols, rows, nw, w, gHeaderH + gPad + rows*(gNodeH+gNodeGap) - gNodeGap + gPad}
	}
}

// computeColumns: dependencias entre clusters → profundidad (columna).
func (b *gbuilder) computeColumns() {
	g := b.g
	b.adj = map[string]map[string]bool{}
	for _, e := range g.Edges {
		a, c := b.clusterOf(e.From, e.Kind == "import"), b.clusterOf(e.To, e.Kind == "import")
		if a == "" || c == "" || a == c {
			continue
		}
		if b.adj[a] == nil {
			b.adj[a] = map[string]bool{}
		}
		b.adj[a][c] = true
	}
	ids := make([]string, 0, len(g.Clusters))
	for _, cl := range g.Clusters {
		ids = append(ids, cl.ID)
	}
	sort.Strings(ids)
	sortedAdj := func(u string) []string {
		out := make([]string, 0, len(b.adj[u]))
		for v := range b.adj[u] {
			out = append(out, v)
		}
		sort.Strings(out)
		return out
	}
	state := map[string]int{}
	back := map[[2]string]bool{}
	var post []string
	var dfs func(u string)
	dfs = func(u string) {
		state[u] = 1
		for _, v := range sortedAdj(u) {
			switch state[v] {
			case 0:
				dfs(v)
			case 1:
				back[[2]string{u, v}] = true // ciclo: esta arista no cuenta para las columnas
			}
		}
		state[u] = 2
		post = append(post, u)
	}
	for _, u := range ids {
		if state[u] == 0 {
			dfs(u)
		}
	}
	b.depth = map[string]int{}
	for i := len(post) - 1; i >= 0; i-- {
		u := post[i]
		for _, v := range sortedAdj(u) {
			if !back[[2]string{u, v}] && b.depth[u]+1 > b.depth[v] {
				b.depth[v] = b.depth[u] + 1
			}
		}
	}
	b.cols = map[int][]string{}
	for _, id := range ids {
		b.cols[b.depth[id]] = append(b.cols[b.depth[id]], id)
		if b.depth[id] > b.maxDepth {
			b.maxDepth = b.depth[id]
		}
	}
	b.incoming = map[int]int{} // aristas que llegan a cada columna (ancho del canal)
	for _, e := range g.Edges {
		a, c := b.clusterOf(e.From, e.Kind == "import"), b.clusterOf(e.To, e.Kind == "import")
		if a != "" && c != "" && a != c {
			b.incoming[b.depth[c]]++
		}
	}
}

// computePositions: caja de cada cluster (por columna, ordenado por
// baricentro de sus predecesores) y de cada nodo dentro de su cluster.
func (b *gbuilder) computePositions() {
	g := b.g
	top := gMargin + 40 + 22*len(g.Labels.Subtitle) + 26
	importCount := 0 // las importaciones viajan por carriles sobre los archivos: no cruzan títulos ajenos
	for _, e := range g.Edges {
		_, a := b.clusterIdx[e.From]
		_, c := b.clusterIdx[e.To]
		if e.Kind == "import" && a && c {
			importCount++
		}
	}
	if importCount > 0 {
		top += min(180, 10+9*importCount)
	}
	b.clTop = top
	b.cbox = map[string]gbox{}
	preds := map[string][]string{}
	for u := range b.adj {
		for v := range b.adj[u] {
			preds[v] = append(preds[v], u)
		}
	}
	colX := gMargin
	b.maxBottom, b.maxRight = top, 0
	for d := 0; d <= b.maxDepth; d++ {
		col := b.cols[d]
		bary := map[string]float64{}
		for _, id := range col {
			sum, n := 0.0, 0
			for _, p := range preds[id] {
				if bx, ok := b.cbox[p]; ok {
					sum += float64(bx.y + bx.h/2)
					n++
				}
			}
			bary[id] = math.MaxFloat64
			if n > 0 {
				bary[id] = sum / float64(n)
			}
		}
		sort.SliceStable(col, func(i, j int) bool {
			if bary[col[i]] != bary[col[j]] {
				return bary[col[i]] < bary[col[j]]
			}
			return col[i] < col[j]
		})
		y, colW := top, 0
		for _, id := range col {
			m := b.cm[b.clusterIdx[id]]
			b.cbox[id] = gbox{colX, y, m.w, m.h}
			y += m.h + gClGap
			if m.w > colW {
				colW = m.w
			}
		}
		if y-gClGap > b.maxBottom {
			b.maxBottom = y - gClGap
		}
		if colX+colW > b.maxRight {
			b.maxRight = colX + colW
		}
		colX += colW + 110 + min(180, 5*b.incoming[d+1])
	}
	b.nbox = map[string]gbox{}
	for _, cl := range g.Clusters {
		bx, m := b.cbox[cl.ID], b.cm[b.clusterIdx[cl.ID]]
		for k, ni := range b.byCluster[cl.ID] {
			col, row := k/m.rows, k%m.rows
			b.nbox[g.Nodes[ni].ID] = gbox{bx.x + gPad + col*(m.nodeW+gSubGap), bx.y + gHeaderH + gPad + row*(gNodeH+gNodeGap), m.nodeW, gNodeH}
		}
	}
}
