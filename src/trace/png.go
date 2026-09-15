// png.go — RenderDiagramPNG rasterizes buildChart's output (see svg.go)
// into real PNG bytes: shapes via oksvg/rasterx (pure Go, no cgo - the
// exact same pair mova.local/diagram/png.go already depends on, see
// go.mod's comment there; nothing new added to the module for this),
// text via a second pass with a real vector font (oksvg does not
// implement <text> at all - see svg.go's header for how that was
// found). This is a small, local duplicate of diagram/png.go's
// shapes+text technique rather than a shared helper, because
// diagram/png.go's font-loading/drawing functions are unexported and
// tied to that package's own Data model - see SOURCE.md § 21 for why
// that's an intentional, not accidental, separation.
package trace

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strconv"
	"strings"
	"sync"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// pngScale renders at 2x and lets the viewer downscale - text drawn at
// 1:1 raster pixels looks soft otherwise, the same reasoning
// diagram/png.go's own pngScale constant documents.
const pngScale = 2.0

// RenderDiagramPNG is context-diagram.png (see write_outputs.go):
// the governance pipeline flowchart for a discovery run (see
// pipeline_diagram.go), or the composition chart for a project.json
// run (see svg.go) — buildPipelineChart picks automatically.
func RenderDiagramPNG(d *Data) ([]byte, error) {
	layout := buildPipelineChart(d)

	svgDoc := fmt.Sprintf(`<svg width="%d" height="%d" viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg">%s</svg>`,
		layout.Width, layout.Height, layout.Width, layout.Height, layout.ShapesSVG)

	icon, err := oksvg.ReadIconStream(strings.NewReader(svgDoc), oksvg.IgnoreErrorMode)
	if err != nil {
		return nil, fmt.Errorf("trace: parsing generated diagram SVG for raster export: %w", err)
	}
	w := int(icon.ViewBox.W * pngScale)
	h := int(icon.ViewBox.H * pngScale)
	icon.SetTarget(0, 0, float64(w), float64(h))

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
	raster := rasterx.NewDasher(w, h, scanner)
	icon.Draw(raster, 1.0)

	drawTextLayer(img, layout.Texts)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("trace: encoding diagram PNG: %w", err)
	}
	return buf.Bytes(), nil
}

var (
	regularFontOnce sync.Once
	regularFont     *opentype.Font
	boldFontOnce    sync.Once
	boldFont        *opentype.Font
)

func loadRegularFont() *opentype.Font {
	regularFontOnce.Do(func() {
		f, err := opentype.Parse(goregular.TTF)
		if err != nil {
			panic("trace: embedded Go Regular font failed to parse: " + err.Error())
		}
		regularFont = f
	})
	return regularFont
}

func loadBoldFont() *opentype.Font {
	boldFontOnce.Do(func() {
		f, err := opentype.Parse(gobold.TTF)
		if err != nil {
			panic("trace: embedded Go Bold font failed to parse: " + err.Error())
		}
		boldFont = f
	})
	return boldFont
}

type faceCache struct{ faces map[string]font.Face }

func newFaceCache() *faceCache { return &faceCache{faces: map[string]font.Face{}} }

func (fc *faceCache) get(size int, bold bool) font.Face {
	key := fmt.Sprintf("%d-%v", size, bold)
	if f, ok := fc.faces[key]; ok {
		return f
	}
	src := loadRegularFont()
	if bold {
		src = loadBoldFont()
	}
	face, err := opentype.NewFace(src, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     72 * pngScale,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil
	}
	fc.faces[key] = face
	return face
}

// drawTextLayer draws every recorded label with a real, anti-aliased
// outline font - see this file's header.
func drawTextLayer(img *image.RGBA, texts []textOp) {
	fc := newFaceCache()
	for _, t := range texts {
		face := fc.get(t.Size, t.Bold)
		if face == nil {
			continue
		}
		col := parseHexColor(t.Color)
		drawer := &font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(col),
			Face: face,
			Dot:  fixed.P(int(float64(t.X)*pngScale), int(float64(t.Y)*pngScale)),
		}
		drawer.DrawString(t.Text)
	}
}

func parseHexColor(hex string) color.Color {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return color.Black
	}
	r, err1 := strconv.ParseUint(hex[0:2], 16, 8)
	g, err2 := strconv.ParseUint(hex[2:4], 16, 8)
	b, err3 := strconv.ParseUint(hex[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return color.Black
	}
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}
