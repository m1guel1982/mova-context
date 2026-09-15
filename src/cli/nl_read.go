// nl_read.go — natural-language "show me this file's content", shared
// by `mova chat` and `mova ui chat`: resolves the path the same way
// nl_edit.go does (documents.ResolveExistingFile) and prints the
// content directly, no model call needed. Another real gap found in
// QA (the model would otherwise just answer "I can't access files").
package main

import (
	"fmt"

	"mova.local/core"
	"mova.local/documents"
)

func handleNaturalLanguageRead(root string, proj *core.Project, line string, state *chatFileState, emit func(string)) bool {
	if emit == nil {
		emit = consolePrint
	}
	target, ok := documents.DetectReadIntent(line)
	if !ok {
		return false
	}
	repo := ""
	if proj != nil {
		repo = proj.Repo
	}
	full, ambiguous, exists, err := documents.ResolveExistingFile(root, repo, target)
	if err != nil {
		emit(fmt.Sprintf("[Read] Error resolving %q: %s\n", target, err.Error()))
		return true
	}
	if len(ambiguous) > 0 {
		emit(fmt.Sprintf("[Read] %q matches more than one file — be more specific (%s).\n", target, ambiguous))
		return true
	}
	if !exists {
		emit(fmt.Sprintf("[Read] %q does not exist.\n", target))
		return true
	}
	content, err := documents.ReadEditableContent(full)
	if err != nil {
		emit(fmt.Sprintf("[Read] Could not read %q: %s\n", full, err.Error()))
		return true
	}
	emit(fmt.Sprintf("[Read] %s:\n%s\n", full, content))
	state.lastFile = full
	return true
}
