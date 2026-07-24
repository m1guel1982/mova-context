// edit_apply.go — the format-agnostic pieces of "modify an existing
// file" shared by cli/nl_edit.go (interactive `mova chat`) and
// mcp/nl_edit.go (chat_completion over MCP/HTTP): reading a file's current
// content in whichever way makes sense for its format, building the
// prompt asking a model for the updated version, and cleaning up its
// reply. Both callers still do their own confirmation UX (interactive
// y/n vs. an explicit "apply_edits" argument) — see those two files.
//
// Design choice, stated plainly: this regenerates a file's FULL content
// rather than computing a binary patch. For source code and plain text
// that means the model rewrites the whole file (reviewed via DiffLines
// before anything is written); for .docx/.pdf it means the CONTENT is
// preserved and re-laid-out through the same generator save_service.go
// already uses (docx.go/pdf.go), not a byte-for-byte patch of the
// original file — formatting/layout gets regenerated, not preserved
// verbatim. .xlsx works the same way but is the least reliable of the
// three: ReadDocumentLayer's extraction is a best-effort text view of the
// sheet, so round-tripping it through an edit can lose structure that a
// precise cell-level tool wouldn't. For anything that must be preserved
// byte-for-byte outside the text content itself, prefer a direct tool
// (patch_file for source code, or editing the source markdown/HTML and
// regenerating) over a natural-language request.
package documents

import (
	"os"
	"path/filepath"
	"strings"
)

// regeneratedExts are formats ReadEditableContent extracts as plain text
// and BuildEditPrompt/callers then regenerate from scratch via Save/
// WriterFor — as opposed to plain text/code, which is edited and written
// back as itself. Kept as its own small set (rather than "everything not
// on the text allowlist") so the caveat above is explicit about exactly
// which formats it applies to.
var regeneratedExts = map[string]bool{
	".docx": true,
	".pdf":  true,
	".xlsx": true,
}

// IsRegeneratedFormat reports whether path's extension is edited by
// full-content regeneration (.docx/.pdf/.xlsx) rather than as plain text.
func IsRegeneratedFormat(path string) bool {
	return regeneratedExts[strings.ToLower(filepath.Ext(path))]
}

// ReadEditableContent returns path's current content as editable text:
// the raw file for plain text/code, or ReadDocumentLayer's extracted text
// layer for .docx/.pdf/.xlsx (see the package doc comment above for the
// tradeoff that implies for those three formats).
func ReadEditableContent(path string) (string, error) {
	if IsRegeneratedFormat(path) {
		return ReadDocumentLayer(path)
	}
	return ReadFile(path)
}

// BuildEditPrompt constructs the message sent to the model: the file's
// current content, the user's request verbatim, and an explicit
// instruction to reply with ONLY the complete new content — no
// explanation, no markdown fences (ExtractEditedContent strips them
// defensively anyway, since models don't always follow this perfectly).
func BuildEditPrompt(path, currentContent, userRequest string) string {
	var b strings.Builder
	b.WriteString("You are editing an existing file at \"" + path + "\".\n\n")
	b.WriteString("CURRENT CONTENT:\n---\n")
	b.WriteString(currentContent)
	b.WriteString("\n---\n\n")
	b.WriteString("Requested change: " + userRequest + "\n\n")
	b.WriteString("Reply with ONLY the complete new content for this file, reflecting that change. " +
		"Do not include any explanation, preamble, or markdown code fences — just the file's new content, in full, exactly as it should be written to disk.")
	return b.String()
}

// ExtractEditedContent cleans up a model's reply for BuildEditPrompt:
// strips a single leading/trailing ``` fence if the model added one
// despite being asked not to (common enough behavior to guard against
// defensively rather than reject the reply outright).
func ExtractEditedContent(reply string) string {
	text := strings.TrimSpace(reply)
	if !strings.HasPrefix(text, "```") {
		return text
	}
	lines := strings.Split(text, "\n")
	if len(lines) < 2 {
		return text
	}
	// Drop the opening fence (optionally tagged, e.g. ```go).
	lines = lines[1:]
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// ResolveExistingFile resolves ref (a path or bare filename, same rules
// as ResolveFilePath) and reports whether it already exists on disk —
// this existence check is deliberately what tells "edit" intent apart
// from "create" intent in cli/nl_edit.go and mcp/nl_edit.go: a request to
// modify something that doesn't exist yet isn't an edit.
func ResolveExistingFile(root, repo, ref string) (full string, ambiguous []string, exists bool, err error) {
	full, ambiguous, err = ResolveFilePath(root, repo, ref)
	if err != nil || len(ambiguous) > 0 {
		return full, ambiguous, false, err
	}
	_, statErr := os.Stat(full)
	return full, nil, statErr == nil, nil
}
