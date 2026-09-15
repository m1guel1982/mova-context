// nl_rename.go — natural-language "rename X to Y" for MCP/HTTP's
// chat_completion, same two-phase (apply_rename) convention as
// nl_delete.go/nl_edit.go. Works for files AND directories, on any
// absolute path style (C:\..., \\server\share\..., /mnt/...) — see
// documents.Rename / documents/pathresolve.go's IsAbsCrossPlatform.
package mcp

import (
	"fmt"
	"strings"

	"mova.local/core"
	"mova.local/documents"
)

func applyNaturalLanguageRename(statusLog *strings.Builder, root string, proj *core.Project, message string, applyRename bool) (reply string, handled bool) {
	from, to, ok := documents.DetectRenameIntent(message)
	if !ok {
		return "", false
	}
	repo := ""
	if proj != nil {
		repo = proj.Repo
	}
	result, err := documents.Rename(root, documents.RenameRequest{From: from, To: to, Repo: repo, Confirm: applyRename})
	if err != nil {
		return fmt.Sprintf("[Rename] Error: %s\n", err.Error()), true
	}
	if result.Message != "" && !result.Pending {
		statusLog.WriteString("[Rename] " + result.Message + "\n")
		return "[Rename] " + result.Message + "\n", true
	}
	if result.Pending {
		return fmt.Sprintf("[Rename] %s\n[Rename] Not applied — call chat_completion again with \"apply_rename\": true to confirm (no interactive prompt on this door).\n", result.Prompt), true
	}
	return "[Rename] " + result.Message + "\n", true
}
