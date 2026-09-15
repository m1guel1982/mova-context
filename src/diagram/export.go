// export.go — Export writes the requested formats for data to disk,
// creating outDir first if it doesn't exist yet. Cross-platform by
// construction: it only ever goes through path/filepath + os.MkdirAll,
// exactly like every other file-writing path in this codebase (see
// e.g. documents/utils.go's ensureDir) — no OS-specific branching, so
// the same code path already works on Windows/Linux/macOS the way the
// rest of Mova Context does.
package diagram

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidFormats are the export formats CLI/HTTP/MCP callers may pass to
// Export — kept as an explicit whitelist so an unrecognized format
// name fails clearly instead of silently writing nothing.
var ValidFormats = []string{"svg", "png", "pdf"}

// splitFilePathTarget recognizes "path" as a full file destination
// (not a directory) when it ends in ".<format>" — e.g.
// "./diagramas/evidencia.png" with format "png" splits into
// ("./diagramas", "evidencia"). ok is false for anything else
// (including a bare directory, or an extension that doesn't match
// format), so Export's normal directory behavior is unaffected.
func splitFilePathTarget(path, format string) (dir, name string, ok bool) {
	if path == "" || !strings.EqualFold(filepath.Ext(path), "."+format) {
		return "", "", false
	}
	base := filepath.Base(path)
	name = strings.TrimSuffix(base, filepath.Ext(base))
	if name == "" {
		return "", "", false
	}
	dir = filepath.Dir(path)
	return dir, name, true
}

// Export renders data once and writes each requested format to
// outDir/baseName.<format>, returning the absolute paths written (in
// the same order as formats) or the first error encountered. An empty
// outDir defaults to the current working directory — same "empty
// means here" convention `mova run`'s other output flags already use.
//
// Convenience: when exactly one format was requested and outDir itself
// ends in ".<format>" (e.g. "--path ./diagramas/evidencia.png" with
// "--export png"), outDir is treated as the FULL destination file
// path — its directory becomes the real outDir and its file name
// (without extension) becomes baseName — instead of being (mis)read
// as a directory literally named "evidencia.png". This is what makes
// `mova run <project> --diagram --export png --path <file>.png` write
// exactly that file, one command, no surprises.
func Export(data *Data, formats []string, outDir, baseName string) ([]string, error) {
	if len(formats) == 1 {
		if dir, name, ok := splitFilePathTarget(outDir, formats[0]); ok {
			outDir, baseName = dir, name
		}
	}
	if len(formats) == 0 {
		formats = []string{"svg"}
	}
	if outDir == "" {
		outDir = "."
	}
	if baseName == "" {
		baseName = sanitizeFileName(data.ProjectName)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("diagram: creating output directory %q: %w", outDir, err)
	}

	var written []string
	for _, format := range formats {
		path := filepath.Join(outDir, baseName+"."+format)
		var content []byte
		var err error
		switch format {
		case "svg":
			content = []byte(RenderSVG(data))
		case "png":
			content, err = RenderPNG(data)
		case "pdf":
			content, err = RenderPDF(data)
		default:
			return written, fmt.Errorf("diagram: unknown export format %q — valid formats: svg, png, pdf", format)
		}
		if err != nil {
			return written, fmt.Errorf("diagram: rendering %s: %w", format, err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return written, fmt.Errorf("diagram: writing %s: %w", path, err)
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			abs = path
		}
		written = append(written, abs)
	}
	return written, nil
}

// sanitizeFileName turns a project/group name (which may contain "/"
// for a group agent, e.g. "02-pii-compliance-governance/ai-privacy-
// reviewer") into a single safe file name component — "/" and other
// path separators become "-" so Export never accidentally writes
// outside outDir or fails on Windows' stricter path rules.
func sanitizeFileName(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			out = append(out, '-')
		default:
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return "diagram"
	}
	return string(out)
}
