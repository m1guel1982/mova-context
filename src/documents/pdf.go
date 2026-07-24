// pdf.go — GeneratePDFDocument compiles a simple HTML/CSS layout into a
// real .pdf file, standard library only: no third-party rendering engine.
// It renders the HTML's text content as paginated body copy — CSS is used
// only to detect heading tags for bold sizing, not for full visual
// layout. See pdf_writer.go for the low-level PDF 1.4 byte writer this
// calls into.
package documents

import (
	"fmt"
	"html"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

// GeneratePDFDocument compiles layoutHTMLCSS into a real .pdf file at path.
func GeneratePDFDocument(path, layoutHTMLCSS string) error {
	if err := ensureDir(path); err != nil {
		return fmt.Errorf("generate_pdf_document: could not create directory: %w", err)
	}

	blocks := htmlToBlocks(layoutHTMLCSS)
	if len(blocks) == 0 {
		return fmt.Errorf("generate_pdf_document: no text found in layout_html_css")
	}
	pages := paginate(blocks, linesPerPage)

	pdf := newPDFWriter()
	for _, page := range pages {
		pdf.addPage(page)
	}
	data := pdf.build()

	return os.WriteFile(path, data, 0o644)
}

type textBlock struct {
	text string
	bold bool
	size int
}

const linesPerPage = 50

var tagRe = regexp.MustCompile(`(?is)<(/?)(h1|h2|h3|p|li|br|div)[^>]*>`)
var anyTagRe = regexp.MustCompile(`(?s)<[^>]*>`)

// htmlToBlocks does a best-effort text extraction: block-level tags become
// line breaks, h1/h2/h3 become bold larger text, everything else becomes
// normal paragraphs.
func htmlToBlocks(htmlCSS string) []textBlock {
	// Strip <style>/<script> content entirely — CSS rules aren't rendered.
	htmlCSS = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`).ReplaceAllString(htmlCSS, "")
	htmlCSS = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`).ReplaceAllString(htmlCSS, "")

	var blocks []textBlock
	segments := splitOnTags(htmlCSS)
	for _, seg := range segments {
		text := strings.TrimSpace(anyTagRe.ReplaceAllString(seg.content, ""))
		text = html.UnescapeString(text)
		if text == "" {
			continue
		}
		size := 11
		bold := false
		switch seg.tag {
		case "h1":
			size, bold = 22, true
		case "h2":
			size, bold = 16, true
		case "h3":
			size, bold = 13, true
		}
		blocks = append(blocks, textBlock{text: text, bold: bold, size: size})
	}
	return blocks
}

type segment struct {
	tag     string
	content string
}

// splitOnTags breaks the document into per-block segments using tagRe as
// boundaries, tagging each with the most recent opening block tag seen.
func splitOnTags(htmlCSS string) []segment {
	matches := tagRe.FindAllStringSubmatchIndex(htmlCSS, -1)
	var out []segment
	lastEnd := 0
	currentTag := "p"
	for _, m := range matches {
		if m[0] > lastEnd {
			out = append(out, segment{tag: currentTag, content: htmlCSS[lastEnd:m[0]]})
		}
		closing := htmlCSS[m[2]:m[3]] == "/"
		tagName := strings.ToLower(htmlCSS[m[4]:m[5]])
		if !closing {
			currentTag = tagName
		}
		lastEnd = m[1]
	}
	if lastEnd < len(htmlCSS) {
		out = append(out, segment{tag: currentTag, content: htmlCSS[lastEnd:]})
	}
	return out
}

func paginate(blocks []textBlock, perPage int) [][]textBlock {
	var pages [][]textBlock
	var current []textBlock
	lineCount := 0
	for _, b := range blocks {
		wrapped := wrapText(b.text, 80)
		if lineCount+len(wrapped) > perPage && len(current) > 0 {
			pages = append(pages, current)
			current = nil
			lineCount = 0
		}
		current = append(current, textBlock{text: strings.Join(wrapped, "\n"), bold: b.bold, size: b.size})
		lineCount += len(wrapped) + 1
	}
	if len(current) > 0 {
		pages = append(pages, current)
	}
	return pages
}

func wrapText(text string, width int) []string {
	words := strings.Fields(text)
	var lines []string
	var line string
	for _, w := range words {
		if line == "" {
			line = w
			continue
		}
		// utf8.RuneCountInString gives the exact visual width, accounting
		// for accented/UTF-8 characters correctly.
		if utf8.RuneCountInString(line)+1+utf8.RuneCountInString(w) > width {
			lines = append(lines, line)
			line = w
		} else {
			line += " " + w
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
}

// pdfWriterAdapter adapts GeneratePDFDocument to IFileWriter for
// SaveService's WriterFactory (".pdf" — see save_service.go).
type pdfWriterAdapter struct{}

func (pdfWriterAdapter) Write(path string, opts SaveOptions) error {
	return GeneratePDFDocument(path, contentAsHTML(opts.Content))
}

func init() { RegisterWriter(".pdf", pdfWriterAdapter{}) }

func contentAsHTML(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return trimmed
	}
	if strings.Contains(trimmed, "<") && strings.Contains(trimmed, ">") {
		return trimmed // already looks like HTML/XML — send as-is
	}
	var b strings.Builder
	for _, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		b.WriteString("<p>" + html.EscapeString(line) + "</p>\n")
	}
	return b.String()
}
