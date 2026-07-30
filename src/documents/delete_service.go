// delete_service.go — DeleteService: the SINGLE entry point behind the
// unified "delete" capability, exactly the same role SaveService (see
// save_service.go) already plays for `/save`. None of the three doors
// (chat's `/delete`, the MCP "delete_path" tool, `POST /delete`) ever
// removes a file or directory itself — they build a DeleteRequest and
// call documents.Delete(), which resolves every requested path through
// the SAME path-resolution rules every other document tool already uses
// (pathresolve.go), decides file vs. directory by asking the filesystem
// (never by guessing from the string), and never touches disk without an
// explicit confirmation.
//
// Confirmation works the same way apply_edits already does for
// natural-language editing (see edit_apply.go): there is no interactive
// y/n available on MCP/HTTP, so Delete() itself never blocks waiting for
// input. Called with Confirm=false it only returns the exact prompt text
// to show ("Delete \"x\"? (Y/N)" — one line per path, in request order)
// and deletes nothing; called again with Confirm=true it performs the
// deletion. The interactive CLI door is the only one that loops asking
// y/n itself (see cli/delete_cmd.go) — MCP/HTTP require the caller to
// pass confirm:true once the person has agreed, same convention as
// apply_edits. The PROMPT TEXT and the deletion logic are identical
// everywhere; only how each door collects the "yes" differs, exactly
// like the rest of Mova Context's confirmation story.
package documents

import (
	"fmt"
	"os"
	"strings"
)

// DeleteRequest is the one format-agnostic input every door normalizes
// its own arguments into before calling Delete(). See cli/delete_cmd.go
// (parses "/delete ..."), mcp/documents_tool.go ("delete_path" tool
// args), and http/server.go (POST /delete JSON body).
type DeleteRequest struct {
	// Paths are the files/directories to delete, in the order given.
	// Each one is resolved the same way write_file/create_directory
	// already resolve paths (pathresolve.go): absolute paths honored
	// as-is, everything else resolved under Repo.
	Paths []string
	// Repo scopes relative paths — same convention every other
	// document tool already uses (project.repo, or "." for the Mova
	// root).
	Repo string
	// Confirm must be true for Delete to actually remove anything.
	// false (the default) only returns the confirmation prompt(s).
	Confirm bool
}

// DeleteItem describes one resolved path awaiting or having received a
// delete decision.
type DeleteItem struct {
	Requested string // exactly what the caller asked for
	Resolved  string // absolute path after pathresolve.go's rules
	IsDir     bool
	Existed   bool
}

// DeleteResult is what all three doors report back to whoever asked.
type DeleteResult struct {
	// Pending is true when Delete only returned confirmation prompts —
	// nothing was removed yet.
	Pending bool
	Items   []DeleteItem
	// Prompt is the exact "Delete \"x\"? (Y/N)" text (one or several
	// lines) to show when Pending is true — identical wording on
	// every door, see FormatDeletePrompt.
	Prompt string
	// Deleted lists the resolved paths actually removed (only set when
	// Pending is false and Confirm was true).
	Deleted []string
	Message string
}

// Delete resolves every path in req.Paths and either returns the
// confirmation prompt (req.Confirm == false) or performs the deletion
// (req.Confirm == true). This is the ONLY function chat/MCP/HTTP call
// for "remove a file or directory" — see the package doc comment above.
func Delete(root string, req DeleteRequest) (DeleteResult, error) {
	paths := dedupeNonEmpty(req.Paths)
	if len(paths) == 0 {
		return DeleteResult{}, fmt.Errorf(`"delete" needs at least one path (file or directory)`)
	}

	items := make([]DeleteItem, 0, len(paths))
	for _, p := range paths {
		item, err := resolveDeleteItem(root, req.Repo, p)
		if err != nil {
			return DeleteResult{}, err
		}
		items = append(items, item)
	}

	if !req.Confirm {
		return DeleteResult{
			Pending: true,
			Items:   items,
			Prompt:  FormatDeletePrompt(items),
			Message: FormatDeletePrompt(items),
		}, nil
	}

	var deleted []string
	var missing []string
	for _, item := range items {
		if !item.Existed {
			missing = append(missing, item.Requested)
			continue
		}
		var err error
		if item.IsDir {
			err = os.RemoveAll(item.Resolved)
		} else {
			err = os.Remove(item.Resolved)
		}
		if err != nil {
			return DeleteResult{}, fmt.Errorf("could not delete %q: %w", item.Requested, err)
		}
		deleted = append(deleted, item.Resolved)
	}

	msg := formatDeleteSummary(deleted, missing)
	return DeleteResult{Items: items, Deleted: deleted, Message: msg}, nil
}

// resolveDeleteItem resolves one requested path to an absolute location
// and finds out whether it's a file or a directory — without assuming
// either: a trailing "/" ("logs/") is an explicit directory hint (same
// convention `/save -d` already uses), otherwise the path is resolved as
// a file location and then checked against the filesystem, so an
// existing directory referenced without a trailing slash ("logs") still
// resolves correctly.
func resolveDeleteItem(root, repo, requested string) (DeleteItem, error) {
	trimmed := strings.TrimSpace(requested)
	asDir := strings.HasSuffix(trimmed, "/") || strings.HasSuffix(trimmed, "\\")

	var full string
	var err error
	if asDir {
		full, ambiguous, rerr := ResolveDirectoryPath(root, repo, strings.TrimRight(trimmed, "/\\"))
		if rerr != nil {
			return DeleteItem{}, rerr
		}
		if len(ambiguous) > 0 {
			return DeleteItem{}, fmt.Errorf("%s", FormatAmbiguousMessage(trimmed, ambiguous))
		}
		return statDeleteItem(requested, full, true)
	}

	full, ambiguous, err := ResolveFilePath(root, repo, trimmed)
	if err != nil {
		return DeleteItem{}, err
	}
	if len(ambiguous) > 0 {
		return DeleteItem{}, fmt.Errorf("%s", FormatAmbiguousMessage(ambiguousDirLabel(trimmed), ambiguous))
	}
	return statDeleteItem(requested, full, false)
}

func statDeleteItem(requested, full string, hintDir bool) (DeleteItem, error) {
	info, statErr := os.Stat(full)
	if statErr != nil {
		return DeleteItem{Requested: requested, Resolved: full, IsDir: hintDir, Existed: false}, nil
	}
	return DeleteItem{Requested: requested, Resolved: full, IsDir: info.IsDir(), Existed: true}, nil
}

// FormatDeletePrompt renders the exact confirmation text for one or many
// items — identical wording whether it is shown by the interactive CLI,
// returned as an MCP tool result, or sent back in an HTTP JSON body:
//
//	Delete "archivo.txt"?
//	(Y/N)
//
// and, for several items, one such pair of lines per item, in order.
// Directories are shown with a trailing "/" regardless of how the caller
// originally typed them, so the person always sees what kind of thing
// is about to be removed.
func FormatDeletePrompt(items []DeleteItem) string {
	var b strings.Builder
	for _, item := range items {
		label := deleteLabel(item)
		b.WriteString(fmt.Sprintf("Delete %q?\n(Y/N)\n", label))
	}
	return strings.TrimRight(b.String(), "\n")
}

func deleteLabel(item DeleteItem) string {
	name := strings.TrimRight(strings.ReplaceAll(item.Requested, "\\", "/"), "/")
	if idx := strings.LastIndex(name, "/"); idx != -1 {
		name = name[idx+1:]
	}
	if item.IsDir {
		return name + "/"
	}
	return name
}

func formatDeleteSummary(deleted, missing []string) string {
	var b strings.Builder
	for _, d := range deleted {
		b.WriteString("✓ deleted: " + d + "\n")
	}
	for _, m := range missing {
		b.WriteString("⚠ not found, skipped: " + m + "\n")
	}
	if b.Len() == 0 {
		return "Nothing to delete."
	}
	return strings.TrimRight(b.String(), "\n")
}

func dedupeNonEmpty(paths []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}
