// rename_service.go — RenameService: the single entry point behind
// chat's natural-language rename ("renombra X a Y"), the "rename_path"
// MCP/HTTP tool, and (eventually) any future "/rename" command — same
// SINGLE-entry-point convention as save_service.go/delete_service.go.
// Added because neither existed before: a rename request used to just
// get a plain-text refusal from the model, since there was no
// underlying capability at all (see docs/PROJECT.md's changelog note).
package documents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// hasPathSeparator reports whether s contains a directory separator of
// EITHER style ("/" or "\") — checked explicitly rather than relying on
// the host OS's own filepath.Separator, since a rename target typed on
// Linux/macOS might still be a Windows-style path (and vice versa) when
// project.json or the person's own message uses one — same
// cross-platform posture as IsAbsCrossPlatform (pathresolve.go).
func hasPathSeparator(s string) bool {
	return strings.ContainsAny(s, `/\`)
}

// siblingPath builds "same directory as fromFile, but named newName" —
// used for a bare rename target ("renombra X a Y" with no path in Y).
// Preserves fromFile's OWN separator style (\ for a Windows-style path,
// / otherwise) so a Windows path renamed in place stays a Windows path.
func siblingPath(fromFile, newName string) string {
	sep := "/"
	if strings.Contains(fromFile, `\`) && !strings.Contains(fromFile, "/") {
		sep = `\`
	}
	dir := filepath.ToSlash(filepath.Dir(fromFile))
	if sep == `\` {
		dir = strings.ReplaceAll(dir, "/", `\`)
	}
	return strings.TrimRight(dir, `/\`) + sep + newName
}

// RenameRequest is what every door builds before calling Rename.
type RenameRequest struct {
	From    string // existing file or directory, resolved the same way delete_path/ResolveExistingFile already do
	To      string // new name or path — a bare name ("adios.txt") is resolved next to From, same directory; a path moves it
	Repo    string
	Confirm bool // must be true for Rename to actually touch the filesystem — same two-phase convention as Delete
}

// RenameResult mirrors DeleteResult's shape (Pending/Prompt/Message) so
// every door already familiar with Delete's contract needs no new
// mental model for Rename.
type RenameResult struct {
	Pending bool
	Prompt  string
	From    string
	To      string
	Message string
}

// Rename resolves req.From (must already exist — file or directory) and
// req.To (a bare name resolves next to From; anything path-shaped
// resolves the same way Save's destination does), then either returns a
// confirmation prompt (Confirm: false) or performs the rename
// (Confirm: true).
func Rename(root string, req RenameRequest) (RenameResult, error) {
	fromFile, ambiguous, exists, err := ResolveExistingFile(root, req.Repo, req.From)
	isDir := false
	if err != nil || !exists {
		// Not a file — try as a directory before giving up, since
		// "renombra el directorio X" is just as valid as a file rename.
		if dirFull, dirAmbiguous, dirErr := ResolveDirectoryPath(root, req.Repo, req.From); dirErr == nil {
			if info, statErr := os.Stat(dirFull); statErr == nil && info.IsDir() {
				fromFile, ambiguous, exists, err, isDir = dirFull, dirAmbiguous, true, nil, true
			}
		}
	}
	if err != nil {
		return RenameResult{}, err
	}
	if len(ambiguous) > 0 {
		return RenameResult{Message: FormatAmbiguousMessage(req.From, ambiguous)}, nil
	}
	if !exists {
		return RenameResult{Message: fmt.Sprintf("%q does not exist.", req.From)}, nil
	}

	toPath := req.To
	// A bare name (no path separator) renames IN PLACE — same
	// directory as From — rather than being (mis)interpreted as
	// relative to the repo root, which would silently MOVE the file.
	if !hasPathSeparator(toPath) {
		toPath = siblingPath(fromFile, toPath)
	} else if isDir {
		toPath, ambiguous, err = ResolveDirectoryPath(root, req.Repo, toPath)
	} else {
		toPath, ambiguous, err = ResolveFilePath(root, req.Repo, toPath)
	}
	if err != nil {
		return RenameResult{}, err
	}
	if len(ambiguous) > 0 {
		return RenameResult{Message: FormatAmbiguousMessage(req.To, ambiguous)}, nil
	}

	if !req.Confirm {
		return RenameResult{
			Pending: true, From: fromFile, To: toPath,
			Prompt: fmt.Sprintf("Rename %q to %q? (Y/N)", fromFile, toPath),
		}, nil
	}
	if err := os.Rename(fromFile, toPath); err != nil {
		return RenameResult{}, err
	}
	return RenameResult{From: fromFile, To: toPath, Message: fmt.Sprintf("✓ renamed: %s -> %s", fromFile, toPath)}, nil
}
