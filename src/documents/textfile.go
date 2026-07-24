// textfile.go implements the "Archivos de Texto y Código del Workspace"
// capability group from the original tool contract: read_file, write_file,
// and patch_file. write_file/patch_file are restricted to a known allowlist
// of extensions (see supportedExtensions below) — anything else gets a
// clear "Unsupported file type" message instead of silently writing a file
// no one asked for. read_file stays unrestricted: reading is always safe,
// regardless of extension.
package documents

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// extCategory groups a supported extension with a short human label, used
// both for the "unsupported" error message and for docs/SUPPORTED_FORMATS.md.
type extCategory struct {
	Ext   string
	Label string
}

// supportedExtensions is the single source of truth for which files
// write_file/patch_file will touch. Grouped by category purely for
// readability — all of them are checked the same way.
var supportedExtensions = []extCategory{
	// Plain / structured text
	{".txt", "Plain text"},
	{".md", "Markdown"},
	{".json", "JSON"},
	{".yml", "YAML"},
	{".yaml", "YAML"},
	{".xml", "XML"},
	{".csv", "CSV"},
	{".toml", "TOML"},
	{".ini", "INI config"},
	{".env", "Env file"},
	{".log", "Log file"},

	// Programming languages
	{".js", "JavaScript"},
	{".ts", "TypeScript"},
	{".py", "Python"},
	{".go", "Go"},
	{".cs", "C#"},
	{".java", "Java"},
	{".php", "PHP"},
	{".rb", "Ruby"},
	{".rs", "Rust"},
	{".c", "C"},
	{".cpp", "C++"},
	{".h", "C/C++ header"},
	{".kt", "Kotlin"},
	{".swift", "Swift"},
	{".sh", "Shell script"},

	// Web
	{".html", "HTML"},
	{".css", "CSS"},
	{".sql", "SQL"},
}

var supportedExtSet = buildExtSet(supportedExtensions)

func buildExtSet(list []extCategory) map[string]string {
	m := make(map[string]string, len(list))
	for _, e := range list {
		m[e.Ext] = e.Label
	}
	return m
}

// SupportedTextExt reports whether ext (as returned by filepath.Ext, with
// the leading dot) is on the write_file/patch_file allowlist.
func SupportedTextExt(ext string) bool {
	_, ok := supportedExtSet[strings.ToLower(ext)]
	return ok
}

// sortedSupportedExtensions returns every supported extension, alphabetized,
// for use in "Unsupported file type" error messages.
func sortedSupportedExtensions() []string {
	exts := make([]string, 0, len(supportedExtSet))
	for ext := range supportedExtSet {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	return exts
}

// ReadFile returns the raw text content of any file on disk — no extension
// restriction. Reading is always safe, so this covers formats outside the
// write_file/patch_file allowlist too (useful for inspecting a repo's
// existing files regardless of type).
func ReadFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read_file: %w", err)
	}
	return string(b), nil
}

// WriteFile creates a new file or fully overwrites an existing one at path.
// The extension must be on the allowlist (supportedExtensions) — anything
// else returns an English "Unsupported file type" error, per the tool
// contract's request for a clear warning in that specific case. Content is
// then validated against its extension where a cheap stdlib check exists
// (.json, .xml, .go, .csv); everything else is written as-is.
func WriteFile(path, content string) error {
	ext := strings.ToLower(filepath.Ext(path))
	if !SupportedTextExt(ext) {
		return fmt.Errorf("Unsupported file type: %s. Supported extensions: %s",
			ext, strings.Join(sortedSupportedExtensions(), ", "))
	}
	if err := ValidateTextFormat(path, content); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write_file: %w", err)
	}
	return nil
}

// PatchFile targets a specific section of an existing file and replaces it
// surgically — the rest of the document is untouched. search must appear
// in the file exactly once; zero or multiple matches are rejected so a
// patch never silently hits the wrong spot or clobbers more than intended.
// Same extension allowlist as WriteFile applies.
func PatchFile(path, search, replace string) error {
	ext := strings.ToLower(filepath.Ext(path))
	if !SupportedTextExt(ext) {
		return fmt.Errorf("Unsupported file type: %s. Supported extensions: %s",
			ext, strings.Join(sortedSupportedExtensions(), ", "))
	}
	original, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("patch_file: %w", err)
	}
	content := string(original)
	count := strings.Count(content, search)
	switch {
	case count == 0:
		return fmt.Errorf("patch_file: el texto buscado no aparece en %s", filepath.Base(path))
	case count > 1:
		return fmt.Errorf("patch_file: el texto buscado aparece %d veces en %s — no es único, sé más específico", count, filepath.Base(path))
	}
	patched := strings.Replace(content, search, replace, 1)
	if err := ValidateTextFormat(path, patched); err != nil {
		return fmt.Errorf("patch_file: el resultado quedaría inválido: %w", err)
	}
	return os.WriteFile(path, []byte(patched), 0o644)
}

// ValidateTextFormat runs the cheap stdlib check available for path's
// extension, if any: .json (encoding/json), .xml (encoding/xml tokenizing),
// .go (go/parser — real syntax validation, no third-party dependency), and
// .csv (encoding/csv — consistent field count per record). Every other
// supported extension (.md, .txt, .yml, .py, .js, .java...) has no
// dependency-free stdlib validator and is written as-is, same as any plain
// text editor would.
func ValidateTextFormat(path, content string) error {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		if !json.Valid([]byte(content)) {
			return fmt.Errorf("contenido JSON inválido para %s", filepath.Base(path))
		}
	case ".xml":
		dec := xml.NewDecoder(strings.NewReader(content))
		for {
			_, err := dec.Token()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return fmt.Errorf("XML mal formado en %s: %w", filepath.Base(path), err)
			}
		}
	case ".go":
		fset := token.NewFileSet()
		if _, err := parser.ParseFile(fset, filepath.Base(path), content, parser.AllErrors); err != nil {
			return fmt.Errorf("Go inválido en %s: %w", filepath.Base(path), err)
		}
	case ".csv":
		r := csv.NewReader(strings.NewReader(content))
		if _, err := r.ReadAll(); err != nil {
			return fmt.Errorf("CSV inválido en %s: %w", filepath.Base(path), err)
		}
	}
	return nil
}

// textWriter adapts WriteFile to IFileWriter for SaveService's
// WriterFactory — registered once per extension in supportedExtensions
// below (Markdown, plain text, JSON, YAML, every supported source-code
// extension...), so this ONE type doubles as MarkdownWriter/TextWriter/
// every other plain-text format: they all share identical semantics
// (write or append raw text, validated against the extension's own cheap
// stdlib check in ValidateTextFormat). Adding a new plain-text extension
// is therefore just one line in supportedExtensions (existing mechanism
// write_file/patch_file already used) — no new Writer type needed.
type textWriter struct{}

func (textWriter) Write(path string, opts SaveOptions) error {
	content := opts.Content
	if opts.Append {
		if existing, err := os.ReadFile(path); err == nil {
			content = string(existing) + content
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	return WriteFile(path, content) // reuses the existing allowlist + format validation, unchanged
}

func init() {
	for _, e := range supportedExtensions {
		RegisterWriter(e.Ext, textWriter{})
	}
}
