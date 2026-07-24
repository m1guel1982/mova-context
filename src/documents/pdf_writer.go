// pdf_writer.go — the low-level PDF 1.4 byte writer: hand-written objects,
// xref table, and trailer, plus WinAnsiEncoding (CP1252) text encoding for
// accented Spanish/Latin characters. See pdf.go for the public API and
// HTML text extraction that feeds into this.
package documents

import (
	"bytes"
	"fmt"
	"strings"
)

// encodePDFWinAnsi converts a UTF-8 string into a PDF string literal
// compatible with WinAnsiEncoding (CP1252).
func encodePDFWinAnsi(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			sb.WriteString(`\\`)
		case '(':
			sb.WriteString(`\(`)
		case ')':
			sb.WriteString(`\)`)
		default:
			if r < 128 {
				sb.WriteRune(r)
				continue
			}
			sb.WriteString(fmt.Sprintf("\\%03o", winAnsiCode(r)))
		}
	}
	return sb.String()
}

// winAnsiCode maps the accented Spanish/Latin characters this generator
// actually needs to their WinAnsiEncoding (CP1252) byte value; anything
// else falls back to '?' (unsupported in WinAnsiEncoding).
func winAnsiCode(r rune) byte {
	switch r {
	case 'Á':
		return 0xC1
	case 'É':
		return 0xC9
	case 'Í':
		return 0xCD
	case 'Ó':
		return 0xD3
	case 'Ú':
		return 0xDA
	case 'á':
		return 0xE1
	case 'é':
		return 0xE9
	case 'í':
		return 0xED
	case 'ó':
		return 0xF3
	case 'ú':
		return 0xFA
	case 'Ñ':
		return 0xD1
	case 'ñ':
		return 0xF1
	case 'Ü':
		return 0xDC
	case 'ü':
		return 0xFC
	case '¿':
		return 0xBF
	case '¡':
		return 0xA1
	case 'º':
		return 0xBA
	case 'ª':
		return 0xAA
	default:
		return '?'
	}
}

// pdfWriter accumulates page content streams and produces the final PDF
// 1.4 byte stream (objects + xref table + trailer) by hand.
type pdfWriter struct {
	pageStreams []string
}

func newPDFWriter() *pdfWriter { return &pdfWriter{} }

func (w *pdfWriter) addPage(blocks []textBlock) {
	var sb strings.Builder
	y := 740
	for _, b := range blocks {
		font := "F1"
		if b.bold {
			font = "F2"
		}
		for _, line := range strings.Split(b.text, "\n") {
			encodedLine := encodePDFWinAnsi(line)
			sb.WriteString(fmt.Sprintf("BT /%s %d Tf 56 %d Td (%s) Tj ET\n", font, b.size, y, encodedLine))
			y -= b.size + 4
		}
		y -= 4
	}
	w.pageStreams = append(w.pageStreams, sb.String())
}

func (w *pdfWriter) build() []byte {
	var buf bytes.Buffer
	var offsets []int
	buf.WriteString("%PDF-1.4\n")

	nPages := len(w.pageStreams)
	pageObjStart := 3
	contentObjStart := pageObjStart + nPages
	fontRegular := contentObjStart + nPages
	fontBold := fontRegular + 1
	totalObjs := fontBold

	writeObj := func(n int, body string) {
		offsets = append(offsets, buf.Len())
		buf.WriteString(fmt.Sprintf("%d 0 obj\n%s\nendobj\n", n, body))
	}

	kids := make([]string, nPages)
	for i := 0; i < nPages; i++ {
		kids[i] = fmt.Sprintf("%d 0 R", pageObjStart+i)
	}
	writeObj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	writeObj(2, fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), nPages))

	for i := 0; i < nPages; i++ {
		writeObj(pageObjStart+i, fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 %d 0 R /F2 %d 0 R >> >> /Contents %d 0 R >>",
			fontRegular, fontBold, contentObjStart+i))
	}
	for i, stream := range w.pageStreams {
		writeObj(contentObjStart+i, fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(stream), stream))
	}

	// Font objects declare /Encoding /WinAnsiEncoding so encodePDFWinAnsi's
	// octal escapes above are interpreted correctly by PDF readers.
	writeObj(fontRegular, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	writeObj(fontBold, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")

	xrefStart := buf.Len()
	buf.WriteString(fmt.Sprintf("xref\n0 %d\n", totalObjs+1))
	buf.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", off))
	}
	buf.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", totalObjs+1, xrefStart))

	return buf.Bytes()
}
