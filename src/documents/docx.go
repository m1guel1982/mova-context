package documents

import (
	"archive/zip"
	"fmt"
	"html"
	"os"
	"strconv"
	"strings"
)

 
// GenerateWordContract compiles markdownContent into a real .docx file at
// path. Supports headings (#, ##, ###), **bold** runs, and plain paragraphs
// — the strongly-structured subset the original tool contract asks for.
// Uses only the standard library: a .docx is just a zip of OOXML parts, so
// no third-party dependency is needed to write one.
func GenerateWordContract(path, markdownContent string) error {
	if err := ensureDir(path); err != nil {
		return fmt.Errorf("generate_word_contract: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("generate_word_contract: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	parts := map[string]string{
		"[Content_Types].xml": contentTypesXML,
		"_rels/.rels":         relsXML,
		"word/document.xml":   documentXML(markdownContent),
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

// docxWriter adapts GenerateWordContract to IFileWriter so SaveService's
// WriterFactory can dispatch to it purely by ".docx" extension — see
// save_service.go. GenerateWordContract itself is untouched and still
// used directly by the legacy generate_word_contract MCP tool.
type docxWriter struct{}

func (docxWriter) Write(path string, opts SaveOptions) error {
	return GenerateWordContract(path, opts.Content)
}

func init() { RegisterWriter(".docx", docxWriter{}) }

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`

const relsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

func documentXML(markdown string) string {
	var body strings.Builder
	for _, line := range strings.Split(markdown, "\n") {
		body.WriteString(paragraphXML(line))
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>` + body.String() + `
    <w:sectPr/>
  </w:body>
</w:document>`
}

func paragraphXML(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return `<w:p/>`
	}
	if level, text := headingLevel(trimmed); level > 0 {
		size := strconv.Itoa(48 - (level-1)*8) // half-points: H1=48(24pt), H2=40, H3=32
		return fmt.Sprintf(`<w:p><w:pPr><w:rPr><w:b/><w:sz w:val="%s"/></w:rPr></w:pPr>%s</w:p>`,
			size, runsXML(text, true))
	}
	return `<w:p>` + runsXML(trimmed, false) + `</w:p>`
}

func headingLevel(line string) (int, string) {
	for level := 3; level >= 1; level-- {
		prefix := strings.Repeat("#", level) + " "
		if strings.HasPrefix(line, prefix) {
			return level, strings.TrimPrefix(line, prefix)
		}
	}
	return 0, line
}

// runsXML splits **bold** spans into separate <w:r> runs; forceBold makes
// every run bold (used for headings) in addition to any ** markers.
func runsXML(text string, forceBold bool) string {
	var sb strings.Builder
	parts := strings.Split(text, "**")
	for i, part := range parts {
		if part == "" {
			continue
		}
		bold := forceBold || i%2 == 1
		sb.WriteString(runXML(part, bold))
	}
	return sb.String()
}

func runXML(text string, bold bool) string {
	rpr := ""
	if bold {
		rpr = "<w:rPr><w:b/></w:rPr>"
	}
	return fmt.Sprintf(`<w:r>%s<w:t xml:space="preserve">%s</w:t></w:r>`, rpr, html.EscapeString(text))
}