// nl_rename.go — natural-language "rename X to Y", shared by `mova
// chat` and `mova ui chat`. Works for files AND directories, on any
// absolute path style (C:\..., \\server\share\..., /mnt/...) — see
// documents.Rename.
package main

import (
	"fmt"
	"strings"

	"mova.local/core"
	"mova.local/documents"
)

func handleNaturalLanguageRename(root string, proj *core.Project, line string, readLine readLineFunc, emit func(string)) bool {
	if emit == nil {
		emit = consolePrint
	}
	from, to, ok := documents.DetectRenameIntent(line)
	if !ok {
		return false
	}
	repo := ""
	if proj != nil {
		repo = proj.Repo
	}
	result, err := documents.Rename(root, documents.RenameRequest{From: from, To: to, Repo: repo, Confirm: false})
	if err != nil {
		emit("[Rename] Error: " + err.Error() + "\n")
		return true
	}
	if !result.Pending {
		emit("[Rename] " + result.Message + "\n")
		return true
	}
	emit(result.Prompt + " ")
	text, ok := readLine()
	if !ok {
		emit("\n[Rename] Cancelled.\n")
		return true
	}
	answer := strings.ToLower(strings.TrimSpace(text))
	if answer != "y" && answer != "yes" && answer != "s" && answer != "si" && answer != "sí" {
		emit("[Rename] Cancelled.\n")
		return true
	}
	result, err = documents.Rename(root, documents.RenameRequest{From: from, To: to, Repo: repo, Confirm: true})
	if err != nil {
		emit("[Rename] Error: " + err.Error() + "\n")
		return true
	}
	emit(fmt.Sprintf("[Rename] %s\n", result.Message))
	return true
}
