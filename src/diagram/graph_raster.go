// graph_raster.go — salidas del grafo: SVG, PNG, PDF y ExportGraph, más
// el rasterizador por formas. Movido tal cual desde graph.go (que superaba
// las 300 líneas); sin cambios de comportamiento.
package diagram

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// RenderGraphSVG devuelve el SVG del grafo.
func RenderGraphSVG(g *GraphData) string {
	return layoutGraph(g).svg
}

// graphRaster rasteriza el grafo; reduce la escala solo si el lienzo
// superaría ~36 M de píxeles (memoria acotada aunque haya cientos de nodos).
type rasterResult struct {
	img   *image.RGBA
	scale float64
}

func graphRaster(g *GraphData) (*rasterResult, error) {
	l := layoutGraph(g)
	scale := pngScale
	switch n := len(l.shapes); { // grafos densos: lienzo grande, no hace falta tanta resolución
	case n > 900:
		scale = 1
	case n > 400:
		scale = 1.5
	}
	if px := float64(l.w*l.h) * scale * scale; px > 36e6 {
		scale = math.Max(0.5, math.Sqrt(36e6/float64(l.w*l.h)))
	}
	img, err := rasterizeShapes(l, scale)
	if err != nil {
		return nil, err
	}
	return &rasterResult{img, scale}, nil
}

// rasterizeShapes pinta cada forma en su propia caja (oksvg/rasterx
// recorren el lienzo completo por cada forma: con cientos de formas y
// millones de píxeles eso tardaba minutos) y la compone en orden sobre el
// fondo; el texto va encima, igual que en rasterizeSVG. El resultado es el
// mismo dibujo, en una fracción del tiempo.
func rasterizeShapes(l glayout, scale float64) (*image.RGBA, error) {
	W, H := int(float64(l.w)*scale), int(float64(l.h)*scale)
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: hexRGBA(colorBG)}, image.Point{}, draw.Src)

	type job struct {
		svg  string
		r    image.Rectangle
		tile *image.RGBA
		err  error
	}
	jobs := make([]job, 0, len(l.shapes))
	for _, s := range l.shapes {
		r := image.Rect(
			max(0, int(math.Floor((s.x0-4)*scale))), max(0, int(math.Floor((s.y0-4)*scale))),
			min(W, int(math.Ceil((s.x1+4)*scale))), min(H, int(math.Ceil((s.y1+4)*scale))))
		if r.Dx() > 0 && r.Dy() > 0 {
			jobs = append(jobs, job{svg: s.svg, r: r})
		}
	}
	render := func(j *job) {
		pw, ph := j.r.Dx(), j.r.Dy()
		// viewBox siempre en 0,0 (oksvg no traslada bien un origen distinto) y
		// la forma se desplaza con un translate hasta su posición en la caja.
		doc := fmt.Sprintf(`<svg viewBox="0 0 %g %g" xmlns="http://www.w3.org/2000/svg"><g transform="translate(%g %g)">%s</g></svg>`,
			float64(pw)/scale, float64(ph)/scale, -float64(j.r.Min.X)/scale, -float64(j.r.Min.Y)/scale, j.svg)
		icon, err := oksvg.ReadIconStream(strings.NewReader(doc), oksvg.IgnoreErrorMode)
		if err != nil {
			j.err = fmt.Errorf("diagram: parsing graph shape: %w", err)
			return
		}
		icon.SetTarget(0, 0, float64(pw), float64(ph))
		j.tile = image.NewRGBA(image.Rect(0, 0, pw, ph))
		icon.Draw(rasterx.NewDasher(pw, ph, rasterx.NewScannerGV(pw, ph, j.tile, j.tile.Bounds())), 1.0)
	}

	// Lotes: las formas de un lote se pintan en paralelo (son independientes)
	// y se componen en orden original (el orden es el z-order); el lote se
	// corta a ~64 MB de teselas para acotar la memoria en grafos enormes.
	const batchBytes = 64 << 20
	for start := 0; start < len(jobs); {
		end, bytes := start, 0
		for end < len(jobs) && (end == start || bytes+jobs[end].r.Dx()*jobs[end].r.Dy()*4 <= batchBytes) {
			bytes += jobs[end].r.Dx() * jobs[end].r.Dy() * 4
			end++
		}
		var wg sync.WaitGroup
		next := make(chan int, end-start)
		for i := start; i < end; i++ {
			next <- i
		}
		close(next)
		for w := 0; w < min(runtime.NumCPU(), end-start); w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range next {
					render(&jobs[i])
				}
			}()
		}
		wg.Wait()
		for i := start; i < end; i++ {
			if jobs[i].err != nil {
				return nil, jobs[i].err
			}
			draw.Draw(img, jobs[i].r, jobs[i].tile, image.Point{}, draw.Over)
			jobs[i].tile = nil
		}
		start = end
	}
	drawTextLayer(img, l.texts, scale)
	return img, nil
}

func hexRGBA(h string) color.RGBA {
	var r, g, b uint8
	fmt.Sscanf(strings.TrimPrefix(h, "#"), "%02x%02x%02x", &r, &g, &b)
	return color.RGBA{r, g, b, 255}
}

// RenderGraphPNG devuelve el grafo como PNG.
func RenderGraphPNG(g *GraphData) ([]byte, error) {
	r, err := graphRaster(g)
	if err != nil {
		return nil, err
	}
	return encodePNG(r.img)
}

// RenderGraphPDF devuelve el grafo como PDF de una página.
func RenderGraphPDF(g *GraphData) ([]byte, error) {
	r, err := graphRaster(g)
	if err != nil {
		return nil, err
	}
	return imageToPDFScaled(r.img, r.scale)
}

// ExportGraph escribe el grafo en path; el formato sale de la extensión
// (.png/.svg/.pdf) y las carpetas intermedias se crean.
func ExportGraph(g *GraphData, path string) error {
	var data []byte
	var err error
	switch strings.ToLower(filepath.Ext(path)) {
	case ".svg":
		data = []byte(RenderGraphSVG(g))
	case ".png":
		data, err = RenderGraphPNG(g)
	case ".pdf":
		data, err = RenderGraphPDF(g)
	default:
		return fmt.Errorf("diagram: unsupported graph format %q", filepath.Ext(path))
	}
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// Atómico: se escribe a un temporal y se renombra, así un cierre del
	// proceso (Ctrl+C, exit) o un visor abierto nunca dejan un grafo a medias.
	tmp := fmt.Sprintf("%s.tmp-%d", path, os.Getpid())
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil { // Windows no reemplaza siempre en un rename
		_ = os.Remove(path)
		if err = os.Rename(tmp, path); err != nil {
			_ = os.Remove(tmp)
			return err
		}
	}
	return nil
}

func fmin(first float64, rest ...float64) float64 {
	for _, v := range rest {
		first = math.Min(first, v)
	}
	return first
}

func fmax(first float64, rest ...float64) float64 {
	for _, v := range rest {
		first = math.Max(first, v)
	}
	return first
}
