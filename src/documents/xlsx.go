package documents

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

 

// Cell declares its underlying data type explicitly (string, number, or
// boolean) to avoid JSON-unmarshalling ambiguity in the Go compiler — per
// the "Estructura Estricta de Documentos" rule.
type Cell struct {
	Type  string `json:"type"` // "string" | "number" | "boolean"
	Value any    `json:"value"`
}

// SheetsData maps a sheet name to its rows of typed cells.
type SheetsData map[string][][]Cell

// GenerateExcelReport compiles sheetsData into a real .xlsx file at path.
// Uses only the standard library (archive/zip + encoding/xml) — an .xlsx is
// a zip of OOXML spreadsheet parts, same idea as GenerateWordContract.
func GenerateExcelReport(path string, sheetsData SheetsData) error {
	if len(sheetsData) == 0 {
		return fmt.Errorf("generate_excel_report: sheets_data está vacío")
	}

	if err := ensureDir(path); err != nil {
		return fmt.Errorf("generate_excel_report: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("generate_excel_report: %w", err)
	}
	defer f.Close()

	names := sortedSheetNames(sheetsData)
	zw := zip.NewWriter(f)

	parts := map[string]string{
		"[Content_Types].xml":        xlsxContentTypes(names),
		"_rels/.rels":                xlsxRootRels,
		"xl/workbook.xml":            xlsxWorkbookXML(names),
		"xl/_rels/workbook.xml.rels": xlsxWorkbookRels(names),
	}
	for i, name := range names {
		parts[fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1)] = xlsxSheetXML(sheetsData[name])
	}
	for name, content := range parts {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(content)); err != nil {
			return err
		}
	}
	return zw.Close()
}

func sortedSheetNames(data SheetsData) []string {
	names := make([]string, 0, len(data))
	for name := range data {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func xlsxContentTypes(sheets []string) string {
	overrides := ""
	for i := range sheets {
		overrides += fmt.Sprintf(`<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, i+1)
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
		overrides + `
</Types>`
}

const xlsxRootRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`

func xlsxWorkbookXML(sheets []string) string {
	entries := ""
	for i, name := range sheets {
		entries += fmt.Sprintf(`<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, escapeXMLAttr(name), i+1, i+1)
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets>` + entries + `</sheets>
</workbook>`
}

func xlsxWorkbookRels(sheets []string) string {
	entries := ""
	for i := range sheets {
		entries += fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, i+1, i+1)
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + entries + `</Relationships>`
}

func xlsxSheetXML(rows [][]Cell) string {
	var body string
	for r, row := range rows {
		body += fmt.Sprintf(`<row r="%d">`, r+1)
		for c, cell := range row {
			ref := colLetter(c) + strconv.Itoa(r+1)
			body += cellXML(ref, cell)
		}
		body += `</row>`
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>` + body + `</sheetData>
</worksheet>`
}

func cellXML(ref string, cell Cell) string {
	switch cell.Type {
	case "number":
		return fmt.Sprintf(`<c r="%s"><v>%v</v></c>`, ref, cell.Value)
	case "boolean":
		v := "0"
		if b, ok := cell.Value.(bool); ok && b {
			v = "1"
		}
		return fmt.Sprintf(`<c r="%s" t="b"><v>%s</v></c>`, ref, v)
	default: // "string" and anything else
		return fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`,
			ref, escapeXMLAttr(fmt.Sprintf("%v", cell.Value)))
	}
}

// colLetter converts a 0-based column index to spreadsheet letters (0→A,
// 25→Z, 26→AA...).
func colLetter(i int) string {
	s := ""
	i++
	for i > 0 {
		i--
		s = string(rune('A'+i%26)) + s
		i /= 26
	}
	return s
}

var xmlEscaper = strings.NewReplacer(
	`&`, "&amp;",
	`<`, "&lt;",
	`>`, "&gt;",
	`"`, "&quot;",
	`'`, "&apos;",
)

func escapeXMLAttr(s string) string {
	return xmlEscaper.Replace(s)
}

// xlsxWriter adapts GenerateExcelReport to IFileWriter for SaveService's
// WriterFactory (".xlsx" — see save_service.go). GenerateExcelReport
// itself is untouched and still reachable directly by the legacy
// generate_excel_report MCP tool, which keeps requiring the strict
// SheetsData JSON shape.
type xlsxWriter struct{}

func (xlsxWriter) Write(path string, opts SaveOptions) error {
	sheets, err := sheetsFromContent(opts.Content)
	if err != nil {
		return err
	}
	return GenerateExcelReport(path, sheets)
}

func init() { RegisterWriter(".xlsx", xlsxWriter{}) }

// sheetsFromContent lets /save accept whichever shape is easiest for
// whoever produced the content: SheetsData's own typed-cell JSON (for
// power users/MCP callers who already know it, tried first), falling
// back to plain CSV/TSV text — one sheet named "Sheet1", each cell
// auto-typed as a number when it parses as one and a string otherwise.
// This means a model that just answers a report with a CSV/TSV table
// never has to learn SheetsData's JSON shape to get a working .xlsx out
// of `/save`.
func sheetsFromContent(content string) (SheetsData, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil, fmt.Errorf("generate_excel_report: el contenido está vacío")
	}
	if strings.HasPrefix(trimmed, "{") {
		var sheets SheetsData
		if err := json.Unmarshal([]byte(trimmed), &sheets); err == nil && len(sheets) > 0 {
			return sheets, nil
		}
		// no era SheetsData válido — sigue el intento de CSV/TSV de abajo
	}

	delim := rune(',')
	if strings.Count(trimmed, "\t") > strings.Count(trimmed, ",") {
		delim = '\t'
	}
	r := csv.NewReader(strings.NewReader(trimmed))
	r.Comma = delim
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("generate_excel_report: el contenido no es JSON de sheets_data válido ni CSV/TSV interpretable: %w", err)
	}

	rows := make([][]Cell, len(records))
	for i, rec := range records {
		row := make([]Cell, len(rec))
		for j, v := range rec {
			if n, err := strconv.ParseFloat(v, 64); err == nil {
				row[j] = Cell{Type: "number", Value: n}
			} else {
				row[j] = Cell{Type: "string", Value: v}
			}
		}
		rows[i] = row
	}
	return SheetsData{"Sheet1": rows}, nil
}