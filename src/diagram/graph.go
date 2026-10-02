// graph.go — render del grafo de dependencias AST (paquete graph lo
// construye, sin LLM): clusters = archivos, nodos = símbolos, aristas =
// llamadas / referencias / importaciones. Reutiliza la paleta, el lienzo
// de texto y el rasterizador de este paquete, así que el estilo es el de
// context-diagram.png y SVG/PNG/PDF salen del mismo dibujo.
//
// Layout determinista y auto-adaptativo: los archivos se ordenan en
// columnas por dependencia (quien llama a la izquierda), cada archivo es
// un subgraph cuyos nodos pasan a 2 o 3 columnas internas si son muchos,
// el ancho entre columnas crece con la cantidad de aristas que cruzan, y
// cada arista sale/entra por un puerto distinto del nodo para no
// superponerse.
//
// El código se reparte en: graph.go (tipos + orquestación), graph_layout.go
// (métricas, columnas y posiciones), graph_draw.go (puertos, clusters,
// aristas, nodos, flechas), graph_legend.go (leyenda, título y SVG final)
// y graph_raster.go (SVG/PNG/PDF/ExportGraph).
package diagram

// GraphNode: Kind func|class|var|other; Style focus|outside|excluded.
type GraphNode struct{ ID, Cluster, Label, Kind, Style string }

// GraphCluster es un archivo (subgraph).
type GraphCluster struct{ ID, Label string }

// GraphEdge: Kind call|ref|import. En "import" From/To son ids de cluster.
type GraphEdge struct {
	From, To, Kind string
	Inferred       bool
}

// GraphLabels: todos los textos visibles, ya traducidos por quien llama.
type GraphLabels struct {
	Title                                                                          string
	Subtitle                                                                       []string
	Legend, File, Func, Class, Var, Outside, Excluded, Call, Ref, Import, Inferred string
}

// GraphData es la entrada completa del render.
type GraphData struct {
	Clusters []GraphCluster
	Nodes    []GraphNode
	Edges    []GraphEdge
	Labels   GraphLabels
}

const (
	gMargin  = 40
	gHeaderH = 34
	gPad     = 14
	gNodeH   = 28
	gNodeGap = 10
	gSubGap  = 40
	gClGap   = 36
)

type gbox struct{ x, y, w, h int }

func textW(s string, size int, bold bool) int {
	f := 0.58
	if bold {
		f = 0.64
	}
	return int(float64(len([]rune(s)))*float64(size)*f) + 1
}

func kindColor(kind string) string {
	switch kind {
	case "class":
		return colorCompiler
	case "var":
		return colorMetrics
	}
	return colorSource
}

func edgeColor(kind string) string {
	switch kind {
	case "ref":
		return colorMetrics
	case "import":
		return colorCompiler
	}
	return colorSource
}

type gport struct{ id, side string }

type garrow struct {
	x, y, dx, dy float64
	color        string
}

// gshape es un elemento SVG ya emitido junto con su caja (unidades del
// SVG): el rasterizador dibuja cada forma solo dentro de su caja, en vez
// de recorrer el lienzo entero por forma (ver rasterizeShapes).
type gshape struct {
	svg            string
	x0, y0, x1, y1 float64
}

// glayout: SVG, capa de texto, formas y tamaño del lienzo.
type glayout struct {
	svg    string
	texts  []textOp
	shapes []gshape
	w, h   int
}

// layoutGraph calcula el dibujo completo. Cada fase vive en su archivo;
// el orden es el de siempre: índices → métricas → columnas/posiciones →
// puertos → dibujo → leyenda/título/SVG.
func layoutGraph(g *GraphData) glayout {
	b := newGBuilder(g)
	b.computeMetrics()
	b.computeColumns()
	b.computePositions()
	b.computePorts()
	b.drawClusters()
	b.drawEdges()
	b.drawNodes()
	b.drawArrows()
	return b.finish()
}

// boxKnown: la arista solo se dibuja si ambos extremos existen.
func boxKnown(e GraphEdge, nbox, cbox map[string]gbox) (struct{}, bool) {
	if e.Kind == "import" {
		_, a := cbox[e.From]
		_, b := cbox[e.To]
		return struct{}{}, a && b
	}
	_, a := nbox[e.From]
	_, b := nbox[e.To]
	return struct{}{}, a && b
}
