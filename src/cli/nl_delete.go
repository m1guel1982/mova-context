// nl_delete.go — natural-language file/directory DELETION, shared by
// `mova chat` and `mova ui chat` (see chat_cmd.go / tui_chat.go's
// nlDeleteCmd). Fixes a real bug found in QA: "elimina el archivo
// test.cpp" used to match editVerbRe (see documents/edit_intent.go,
// before this file existed) and go through the EDIT flow — the model
// was asked to produce a diff, which just emptied the file's content
// instead of actually removing it. Delete intent (documents.
// DetectDeleteIntent) is now checked BEFORE edit intent (see
// chat_cmd.go's dispatch order) and reuses runChatDelete — the exact
// same confirmation flow `/delete` already uses, so this never deletes
// silently.
package main

import (
	"fmt"
	"strings"

	"mova.local/core"
	"mova.local/documents"
)

// handleNaturalLanguageDelete returns true when it handled the message
// itself. No model call involved — deletion never needs generated
// content, only a path and a yes/no (via readLine — a *bufio.Scanner
// wrapper on `mova chat`, a menuChan-backed read on `mova ui chat`).
func handleNaturalLanguageDelete(root string, proj *core.Project, line string, state *chatFileState, readLine readLineFunc, emit func(string)) bool {
	if emit == nil {
		emit = consolePrint
	}
	intent := documents.DetectDeleteIntent(line)
	if !intent.VerbDetected {
		return false
	}
	repo := ""
	if proj != nil {
		repo = proj.Repo
	}
	// "elimina el CONTENIDO de X" / "elimina el TEXTO de X" empties the
	// file instead of removing it — no confirmation needed, this never
	// destroys the file itself, only its content (see
	// documents.ClearContent's doc comment).
	if intent.ClearContentOnly {
		target := intent.Targets
		if len(target) == 0 && state.lastFile != "" {
			target = []string{state.lastFile}
		}
		for _, t := range target {
			msg, err := documents.ClearContent(root, repo, t)
			if err != nil {
				emit("[Clear] Error: " + err.Error() + "\n")
				continue
			}
			emit("[Clear] " + msg + "\n")
		}
		return true
	}
	targets := intent.Targets
	if len(targets) == 0 {
		if state.lastFile == "" {
			return false // nothing to fall back to — might not even be about a file
		}
		targets = []string{state.lastFile}
	}
	var quoted []string
	for _, t := range targets {
		quoted = append(quoted, fmt.Sprintf("%q", strings.Trim(t, `"'`)))
	}
	runChatDelete(root, proj, strings.Join(quoted, " "), readLine, emit)
	return true
}
