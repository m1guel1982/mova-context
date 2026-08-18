// pdf.go — RenderPDF wraps the exact same raster rasterizeDiagram
// (png.go) produces into a standalone, valid one-page PDF: an Image
// XObject holding the RGB pixels, Flate-compressed with the standard
// library's compress/zlib (no extra dependency for that part). This is
// a SEPARATE, minimal PDF writer from documents/pdf_writer.go on
// purpose — that one assembles multi-paragraph TEXT reports page by
// page; this one embeds a single large image. Reusing/extending the
// text writer for an embedded image would have meant bolting an
// Image-XObject code path onto a file that has none today, risking
// every existing docx/PDF report in the process — a new, small,
// self-contained writer is the safer change (see this feature's own
// "no romper funcionalidad existente" rule).
package diagram

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"image"
)

// RenderPDF builds data's diagram and returns a single-page PDF whose
// page is exactly that raster, scaled to fit a comfortable page size
// while preserving aspect ratio.
func RenderPDF(data *Data) ([]byte, error) {
	img, err := rasterizeDiagram(data)
	if err != nil {
		return nil, err
	}
	return imageToPDF(img)
}

// imageToPDF is deliberately format-agnostic about its caller (any
// *image.RGBA, not just a diagram) — kept general in case a future
// export (e.g. a budget report chart) wants the same wrapper, without
// this function knowing anything about diagram.Data.
func imageToPDF(img *image.RGBA) ([]byte, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	raw := make([]byte, 0, w*h*3)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			raw = append(raw, byte(r>>8), byte(g>>8), byte(bl>>8))
		}
	}
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(raw); err != nil {
		return nil, fmt.Errorf("diagram: compressing PDF image stream: %w", err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("diagram: closing PDF image stream: %w", err)
	}

	// Page geometry: fit the raster into a page at 96 DPI-equivalent
	// scale (pngScale already doubled the source resolution — the PDF
	// page itself stays a normal printable size, the image just has
	// enough pixels to look sharp).
	pageW := float64(w) / pngScale
	pageH := float64(h) / pngScale

	return buildPDF(pdfImageXObject{
		Width: w, Height: h, PageW: pageW, PageH: pageH, Compressed: compressed.Bytes(),
	}), nil
}

type pdfImageXObject struct {
	Width, Height int
	PageW, PageH  float64
	Compressed    []byte
}

// buildPDF writes the minimal set of PDF objects a single-page,
// single-image document needs: Catalog, Pages, one Page, one Image
// XObject, and the Page's content stream (just "draw the image
// full-page"). Byte offsets are tracked by hand for the xref table —
// same minimal-PDF-writer style documents/pdf_writer.go already uses,
// just with an Image XObject instead of a text content stream.
func buildPDF(x pdfImageXObject) []byte {
	var buf bytes.Buffer
	offsets := make([]int, 6) // index 1..5 used, 0 unused (object numbers start at 1)

	write := func(s string) { buf.WriteString(s) }
	writeBytes := func(b []byte) { buf.Write(b) }
	mark := func(objNum int) { offsets[objNum] = buf.Len() }

	write("%PDF-1.4\n")

	mark(1)
	write("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	mark(2)
	write("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	mark(3)
	write(fmt.Sprintf("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Resources << /XObject << /Im0 5 0 R >> >> /Contents 4 0 R >>\nendobj\n", x.PageW, x.PageH))

	content := fmt.Sprintf("q %.2f 0 0 %.2f 0 0 cm /Im0 Do Q", x.PageW, x.PageH)
	mark(4)
	write(fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(content), content))

	mark(5)
	write(fmt.Sprintf("5 0 obj\n<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /FlateDecode /Length %d >>\nstream\n", x.Width, x.Height, len(x.Compressed)))
	writeBytes(x.Compressed)
	write("\nendstream\nendobj\n")

	xrefStart := buf.Len()
	write("xref\n0 6\n0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		write(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	write("trailer\n<< /Size 6 /Root 1 0 R >>\n")
	write(fmt.Sprintf("startxref\n%d\n%%%%EOF", xrefStart))

	return buf.Bytes()
}
