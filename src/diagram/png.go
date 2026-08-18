// png.go — RenderPNG rasterizes the diagram into real PNG bytes.
// oksvg/rasterx (github.com/srwiley/oksvg, github.com/srwiley/rasterx —
// pure Go, no cgo, the only new third-party dependency this feature
// introduces, see go.mod) rasterize every SHAPE (boxes, accent bars,
// arrows) straight from the SVG this package already generates. oksvg
// does not implement SVG <text> at all, though — verified by actually
// rendering a first version of this file's output and finding every
// label silently missing, not assumed from documentation — so text is
// drawn in a SECOND pass, directly onto the same raster.
//
// That second pass uses a REAL vector font (golang.org/x/image/font/
// opentype + the "Go" font family already embedded in the same x/image
// module oksvg pulls in — golang.org/x/image/font/gofont/{goregular,
// gobold} — no separate font file to ship). An earlier version of this
// file used golang.org/x/image/font/basicfont's fixed 7x13-pixel bitmap
// face instead: it only scaled glyph SPACING, not the glyphs themselves
// (font.Drawer has no built-in scaling), so anything above ~13px came
// out as tiny letters with huge gaps — a real, found-by-rendering-and-
// looking bug, not a style preference. A proper outline font fixes
// that (true per-size rendering, anti-aliased) and, as a bonus, removes
// the old bitmap face's ASCII-only limitation — Spanish accents render
// correctly now, no more transliteration needed for the raster path.
package diagram

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

// pngScale multiplies the SVG's own pixel dimensions before
// rasterizing — SVG text at 1:1 CSS pixels looks soft once rasterized,
// so PNG/PDF export renders at 2x and lets the viewer downscale, the
// same trick browsers use for "retina" screenshots.
const pngScale = 2.0

// RenderPNG builds data's diagram straight to PNG bytes — shapes via
// oksvg/rasterx, labels via the real-font overlay described above.
// Kept as one call (taking Data, not an already-built SVG string) so
// callers never need to know about the two-pass shapes+text split.
func RenderPNG(data *Data) ([]byte, error) {
	img, err := rasterizeDiagram(data)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("diagram: encoding PNG: %w", err)
	}
	return buf.Bytes(), nil
}

// rasterizeDiagram is shared by RenderPNG and pdf.go (which embeds the
// exact same raster into a one-page PDF, so PNG and PDF are always
// pixel-identical).
func rasterizeDiagram(data *Data) (*image.RGBA, error) {
	c := build(data)
	svgText := c.finalize()

	// IgnoreErrorMode, not WarnErrorMode: <text> elements are handled
	// entirely by drawTextLayer below, on purpose (see this file's
	// header) — every "cannot process text element" warning oksvg would
	// otherwise print here is expected noise, not a real problem.
	icon, err := oksvg.ReadIconStream(strings.NewReader(svgText), oksvg.IgnoreErrorMode)
	if err != nil {
		return nil, fmt.Errorf("diagram: parsing generated SVG for raster export: %w", err)
	}
	w := int(icon.ViewBox.W * pngScale)
	h := int(icon.ViewBox.H * pngScale)
	icon.SetTarget(0, 0, float64(w), float64(h))

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
	raster := rasterx.NewDasher(w, h, scanner)
	icon.Draw(raster, 1.0)

	drawTextLayer(img, c.texts)
	return img, nil
}

// ── Real vector fonts (parsed once, process-wide) ──────────────────

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
			panic("diagram: embedded Go Regular font failed to parse: " + err.Error()) // can only happen if the vendored font bytes are corrupted — a build-time problem, not a runtime one
		}
		regularFont = f
	})
	return regularFont
}

func loadBoldFont() *opentype.Font {
	boldFontOnce.Do(func() {
		f, err := opentype.Parse(gobold.TTF)
		if err != nil {
			panic("diagram: embedded Go Bold font failed to parse: " + err.Error())
		}
		boldFont = f
	})
	return boldFont
}

// faceCache avoids re-hinting/re-scaling a font.Face for every single
// label at the same size — one diagram typically repeats only a
// handful of distinct (size, bold) pairs (title, section labels, box
// title line, box body lines...) across dozens of text ops.
type faceCache struct {
	faces map[string]font.Face
}

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
	// DPI*pngScale makes NewFace's internal pixel scale come out to
	// exactly (size * pngScale) px — see this function's own doc
	// comment in the file header: point-size math done once here
	// instead of hand-rolled advance-scaling like the old bitmap path.
	face, err := opentype.NewFace(src, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     72 * pngScale,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil // extremely unlikely (NewFace only errors on a nil font) — drawTextLayer skips a label rather than panic mid-render
	}
	fc.faces[key] = face
	return face
}

// drawTextLayer draws every recorded label onto img with a real,
// anti-aliased outline font (see this file's header) — bold lines use
// the actual Go Bold face, not a fake double-draw offset.
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

// parseHexColor parses this package's own "#rrggbb" palette constants
// (see svg.go) — always well-formed since they're Go string literals,
// so a parse failure only means "fall back to opaque black" rather
// than ever needing to surface an error to the caller.
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
