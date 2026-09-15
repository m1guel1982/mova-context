// nl_read.go — natural-language "show me this file's content" for the
// MCP/HTTP "chat_completion" tool, mirroring cli/nl_read.go. No model
// call needed — resolves the path the same way nl_edit.go does
// (documents.ResolveExistingFile) and returns the content directly.
package mcp

import (
	"fmt"
	"strings"

	"mova.local/core"
	"mova.local/documents"
)

func applyNaturalLanguageRead(root string, proj *core.Project, message string) (reply string, handled bool) {
	target, ok := documents.DetectReadIntent(message)
	if !ok {
		return "", false
	}
	repo := ""
	if proj != nil {
		repo = proj.Repo
	}
	full, ambiguous, exists, err := documents.ResolveExistingFile(root, repo, target)
	if err != nil {
		return fmt.Sprintf("[Read] Error resolving %q: %s\n", target, err.Error()), true
	}
	if len(ambiguous) > 0 {
		return fmt.Sprintf("[Read] %q matches more than one file — be more specific (%s).\n", target, strings.Join(ambiguous, ", ")), true
	}
	if !exists {
		return fmt.Sprintf("[Read] %q does not exist.\n", target), true
	}
	content, err := documents.ReadEditableContent(full)
	if err != nil {
		return fmt.Sprintf("[Read] Could not read %q: %s\n", full, err.Error()), true
	}
	return fmt.Sprintf("[Read] %s:\n%s\n", full, content), true
}
