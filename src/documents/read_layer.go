package documents

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ReadDocumentLayer extracts the plain-text layer from a .docx, .xlsx, or
// .pdf file and returns it to the caller's context buffer, per the original
// "extracción nativa" tool contract. Uses only the standard library:
// archive/zip + regexp for OOXML parts, compress/zlib for PDF FlateDecode
// streams.
func ReadDocumentLayer(path string) (string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".docx":
		return readDocxLayer(path)
	case ".xlsx":
		return readXlsxLayer(path)
	case ".pdf":
		return readPDFLayer(path)
	default:
		return "", fmt.Errorf("read_document_layer: extensión no soportada: %s", filepath.Ext(path))
	}
}

// --- DOCX ------------------------------------------------------------------

var wTextRe = regexp.MustCompile(`(?s)<w:t[^>]*>(.*?)</w:t>`)

func readDocxLayer(path string) (string, error) {
	raw, err := readZipPart(path, "word/document.xml")
	if err != nil {
		return "", fmt.Errorf("read_document_layer (docx): %w", err)
	}
	var sb strings.Builder
	for _, para := range strings.Split(raw, "</w:p>") {
		matches := wTextRe.FindAllStringSubmatch(para, -1)
		if len(matches) == 0 {
			continue
		}
		for _, m := range matches {
			sb.WriteString(unescapeXMLText(m[1]))
		}
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

// --- XLSX --------------------------------------------------------------------

var sharedStringRe = regexp.MustCompile(`(?s)<si>(.*?)</si>`)
var cellRe = regexp.MustCompile(`(?s)<c[^>]*r="([A-Z]+\d+)"[^>]*?( t="([a-zA-Z]+)")?[^>]*>(.*?)</c>`)
var cellValueRe = regexp.MustCompile(`(?s)<v>(.*?)</v>`)
var cellInlineRe = regexp.MustCompile(`(?s)<is>.*?<t[^>]*>(.*?)</t>.*?</is>`)

func readXlsxLayer(path string) (string, error) {
	shared, _ := readZipPart(path, "xl/sharedStrings.xml")
	sharedStrings := parseSharedStrings(shared)

	var sb strings.Builder
	for i := 1; ; i++ {
		sheetPath := fmt.Sprintf("xl/worksheets/sheet%d.xml", i)
		content, err := readZipPart(path, sheetPath)
		if err != nil {
			break // no more sheets
		}
		sb.WriteString(fmt.Sprintf("## Sheet%d\n", i))
		sb.WriteString(renderSheetText(content, sharedStrings))
		sb.WriteString("\n")
	}
	if sb.Len() == 0 {
		return "", fmt.Errorf("read_document_layer (xlsx): no se encontraron hojas")
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

func parseSharedStrings(xmlContent string) []string {
	if xmlContent == "" {
		return nil
	}
	matches := sharedStringRe.FindAllStringSubmatch(xmlContent, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		text := wTextRe.FindAllStringSubmatch(m[1], -1)
		var sb strings.Builder
		for _, t := range text {
			sb.WriteString(t[1])
		}
		out = append(out, unescapeXMLText(sb.String()))
	}
	return out
}

func renderSheetText(sheetXML string, sharedStrings []string) string {
	rows := regexp.MustCompile(`(?s)<row[^>]*>(.*?)</row>`).FindAllStringSubmatch(sheetXML, -1)
	var sb strings.Builder
	for _, row := range rows {
		cells := cellRe.FindAllStringSubmatch(row[1], -1)
		var values []string
		for _, c := range cells {
			cellType := c[3]
			values = append(values, resolveCellText(cellType, c[4], sharedStrings))
		}
		sb.WriteString(strings.Join(values, "\t"))
		sb.WriteString("\n")
	}
	return sb.String()
}

func resolveCellText(cellType, inner string, sharedStrings []string) string {
	if cellType == "inlineStr" {
		if m := cellInlineRe.FindStringSubmatch(inner); m != nil {
			return unescapeXMLText(m[1])
		}
		return ""
	}
	m := cellValueRe.FindStringSubmatch(inner)
	if m == nil {
		return ""
	}
	if cellType == "s" {
		var idx int
		fmt.Sscanf(m[1], "%d", &idx)
		if idx >= 0 && idx < len(sharedStrings) {
			return sharedStrings[idx]
		}
		return ""
	}
	return unescapeXMLText(m[1])
}

// --- shared zip helper -------------------------------------------------------

func readZipPart(path, partName string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name == partName {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()
			b, err := io.ReadAll(rc)
			if err != nil {
				return "", err
			}
			return string(b), nil
		}
	}
	return "", fmt.Errorf("part not found: %s", partName)
}

func unescapeXMLText(s string) string {
	r := strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&apos;", "'")
	return r.Replace(s)
}

// --- PDF (best effort) --------------------------------------------------------

var streamRe = regexp.MustCompile(`(?s)<<(.*?)>>\s*stream\r?\n(.*?)\r?\nendstream`)
var tjRe = regexp.MustCompile(`\(((?:\\.|[^()\\])*)\)\s*Tj`)
var arrayTjRe = regexp.MustCompile(`(?s)\[(.*?)\]\s*TJ`)
var arrayStringRe = regexp.MustCompile(`\(((?:\\.|[^()\\])*)\)`)

// readPDFLayer does a best-effort extraction: it decompresses FlateDecode
// content streams and pulls text shown via the Tj/TJ operators. It does not
// implement a full PDF parser (font encoding tables, CMaps, etc.) — that is
// out of scope for a native text-layer read, but it reliably recovers text
// from PDFs generated by GeneratePDFDocument and most simple text PDFs.
func readPDFLayer(path string) (string, error) {
	rawBytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read_document_layer (pdf): %w", err)
	}
	raw := string(rawBytes)
	var sb strings.Builder
	for _, m := range streamRe.FindAllStringSubmatch(raw, -1) {
		dict, body := m[1], m[2]
		content := body
		if strings.Contains(dict, "FlateDecode") {
			if decompressed, err := zlibDecompress([]byte(body)); err == nil {
				content = string(decompressed)
			}
		}
		sb.WriteString(extractPDFText(content))
	}
	text := strings.TrimSpace(sb.String())
	if text == "" {
		return "", fmt.Errorf("read_document_layer (pdf): no se pudo extraer texto (PDF escaneado o con codificación no soportada)")
	}
	return text, nil
}

func extractPDFText(content string) string {
	var sb strings.Builder
	for _, m := range tjRe.FindAllStringSubmatch(content, -1) {
		sb.WriteString(unescapePDFString(m[1]))
		sb.WriteString(" ")
	}
	for _, arr := range arrayTjRe.FindAllStringSubmatch(content, -1) {
		for _, m := range arrayStringRe.FindAllStringSubmatch(arr[1], -1) {
			sb.WriteString(unescapePDFString(m[1]))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func unescapePDFString(s string) string {
	r := strings.NewReplacer(`\(`, "(", `\)`, ")", `\\`, `\`)
	return r.Replace(s)
}

func zlibDecompress(data []byte) ([]byte, error) {
	zr, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}
