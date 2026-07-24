// save_service.go — SaveService: the SINGLE entry point behind `/save`
// (chat), the "save" MCP tool, and the `POST /save` HTTP endpoint (see
// mcp/save_tool.go and http/server.go). None of those three doors ever
// picks a generator function by name (GenerateWordContract vs
// GeneratePDFDocument vs GenerateExcelReport vs WriteFile...) — they
// build a SaveRequest and call documents.Save(), which resolves the
// right IFileWriter from the file's own extension via WriterFor
// (WriterFactory) and hands it the content. That's the whole contract.
//
// Adding a NEW output format later (say .pptx) never touches this file,
// SaveRequest/SaveResult, the chat command, the MCP tool, or the HTTP
// handler — it only means writing one small type that satisfies
// IFileWriter and calling RegisterWriter(".pptx", ...) from that new
// file's own init(), the exact same pattern docx.go/pdf.go/xlsx.go/
// svg.go/textfile.go already use below. Open/Closed by construction.
//
// The individual generators this delegates to (GenerateWordContract,
// GeneratePDFDocument, GenerateExcelReport, GenerateVectorGraphic,
// WriteFile) are UNCHANGED and still reachable directly — the legacy MCP
// tools (generate_word_contract, generate_pdf_document, ...) keep calling
// them exactly as before, for backward compatibility (see
// mcp/documents_tool.go). SaveService is a thin routing layer on top, not
// a replacement.
package documents

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SaveOptions is what an IFileWriter actually receives — deliberately
// smaller than SaveRequest (no Path/Directory/Repo: by the time a Writer
// runs, the path is already resolved and its parent directories already
// exist).
type SaveOptions struct {
	Content   string
	Append    bool
	Overwrite bool
	Encoding  string // reserved: every built-in writer assumes UTF-8 today
}

// IFileWriter is the WriterFactory's extension point — see the package
// doc comment above for how to add a new one.
type IFileWriter interface {
	Write(path string, opts SaveOptions) error
}

// writerRegistry maps a lowercase extension (with its leading dot) to the
// IFileWriter that handles it. Never populated directly from outside —
// always through RegisterWriter, generally from an init() next to the
// writer's own type (see the bottom of docx.go/pdf.go/xlsx.go/svg.go/
// textfile.go).
var writerRegistry = map[string]IFileWriter{}

// RegisterWriter wires ext to w. Last registration for a given extension
// wins, so a project-specific writer could deliberately override a
// built-in one if that's ever needed.
func RegisterWriter(ext string, w IFileWriter) {
	writerRegistry[strings.ToLower(ext)] = w
}

// WriterFor is the WriterFactory itself: resolves path's extension to its
// IFileWriter. No switch statement, no per-format branching — just a map
// lookup, so registering a new extension is the entire integration.
func WriterFor(path string) (IFileWriter, bool) {
	w, ok := writerRegistry[strings.ToLower(filepath.Ext(path))]
	return w, ok
}

// RegisteredExtensions lists every extension with a Writer registered,
// alphabetized — used for the "Unsupported file type" message and for
// `/save` 's own help text, so that list is never hand-maintained twice.
func RegisteredExtensions() []string {
	exts := make([]string, 0, len(writerRegistry))
	for e := range writerRegistry {
		exts = append(exts, e)
	}
	sort.Strings(exts)
	return exts
}

// SaveRequest is the one format-agnostic input every door (chat's
// /save, MCP's "save" tool, POST /save) normalizes its own arguments
// into before calling Save(). See cli/chat_cmd.go (parses "/save ..."),
// mcp/save_tool.go ("save" tool args), and http/server.go (POST /save
// JSON body) for how each door builds one of these.
type SaveRequest struct {
	// Path is required unless Directory is set — a repo-relative or
	// absolute path to the file to create/edit. Its extension picks the
	// Writer via WriterFor.
	Path string
	// Directory, when set (Path left empty), means "only create this
	// directory" — e.g. `/save -d "docs/backend"`. No Writer involved.
	Directory string
	Content   string
	Append    bool
	Overwrite bool
	// OverwriteExplicit distinguishes "the caller explicitly passed
	// overwrite:false" from "the caller didn't mention overwrite at
	// all" — only the former triggers the exists-already guard below,
	// so a plain `/save "docs/readme.md"` keeps behaving the way
	// write_file always has (silently overwrites) — backward compatible
	// by default.
	OverwriteExplicit bool
	// Repo scopes relative paths — same convention every other document
	// tool already uses (project.repo, or "." for the Mova root).
	Repo string
}

// SaveResult is what all three doors report back to whoever asked.
type SaveResult struct {
	Path    string
	Message string
}

// Save resolves req's path (smart directory-search rules from
// pathresolve.go — identical to every existing document tool), picks the
// right Writer by extension, guarantees parent directories exist, and
// writes. This is the ONLY function chat/MCP/HTTP call for "create or
// edit a file or directory" — see the package doc comment for why.
func Save(root string, req SaveRequest) (SaveResult, error) {
	if req.Directory != "" {
		full, ambiguous, err := ResolveDirectoryPath(root, req.Repo, req.Directory)
		if err != nil {
			return SaveResult{}, err
		}
		if len(ambiguous) > 0 {
			return SaveResult{Message: FormatAmbiguousMessage(req.Directory, ambiguous)}, nil
		}
		if err := CreateDirectory(full); err != nil {
			return SaveResult{}, err
		}
		return SaveResult{Path: full, Message: "✓ directory created: " + full}, nil
	}

	if strings.TrimSpace(req.Path) == "" {
		return SaveResult{}, fmt.Errorf(`/save necesita un "path" (archivo a crear/editar) o un "directory" (solo crear la carpeta)`)
	}

	full, ambiguous, err := ResolveFilePath(root, req.Repo, req.Path)
	if err != nil {
		return SaveResult{}, err
	}
	if len(ambiguous) > 0 {
		return SaveResult{Message: FormatAmbiguousMessage(ambiguousDirLabel(req.Path), ambiguous)}, nil
	}

	writer, ok := WriterFor(full)
	if !ok {
		return SaveResult{}, fmt.Errorf("Unsupported file type: %s. Supported extensions: %s",
			strings.ToLower(filepath.Ext(full)), strings.Join(RegisteredExtensions(), ", "))
	}

	if !req.Append && req.OverwriteExplicit && !req.Overwrite {
		if _, statErr := os.Stat(full); statErr == nil {
			return SaveResult{}, fmt.Errorf(`%s already exists — pass "overwrite": true or "append": true to modify it`, filepath.Base(full))
		}
	}

	opts := SaveOptions{Content: req.Content, Append: req.Append, Overwrite: req.Overwrite}
	if err := writer.Write(full, opts); err != nil {
		return SaveResult{}, err
	}
	return SaveResult{Path: full, Message: saveMessageFor(full)}, nil
}

// saveMessageFor keeps the exact same success wording every existing
// generate_*/write_file tool already returned, so scripts or people
// parsing that text don't see anything change under /save.
func saveMessageFor(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".docx":
		return "Word document generated: " + path
	case ".pdf":
		return "PDF generated: " + path
	case ".xlsx":
		return "Excel generated: " + path
	case ".svg":
		return "SVG generated: " + path
	default:
		return "file saved: " + path
	}
}

// ambiguousDirLabel extracts just the directory portion of a "dir/file.ext"
// request, so the disambiguation question names the ambiguous folder
// ("config") instead of the whole requested path ("config/ajustes.json").
// Same rule mcp/documents_tool.go's resolveSmartFile already applied —
// factored here so Save() doesn't need the mcp package (that would be an
// import cycle: mcp already imports documents).
func ambiguousDirLabel(requested string) string {
	cleaned := strings.ReplaceAll(requested, "\\", "/")
	if idx := strings.LastIndex(cleaned, "/"); idx != -1 {
		return cleaned[:idx]
	}
	return cleaned
}

// FormatAmbiguousMessage renders the "which folder did you mean?" prompt
// shared by every path-resolving tool. Exported so mcp/documents_tool.go's
// existing resolveSmartDir/resolveSmartFile can delegate to this instead
// of keeping their own duplicate copy (see documents_tool.go).
func FormatAmbiguousMessage(name string, candidates []string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d folders named %q. Which one should I use?\n", len(candidates), name))
	for i, c := range candidates {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, c))
	}
	sb.WriteString("Ask again with the full path (one of the above, or a new one).")
	return sb.String()
}
