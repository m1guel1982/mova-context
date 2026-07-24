// nl_edit.go — natural-language file EDITING for `mova chat`, the same
// experience Claude Console gives: "modify the login function", "change
// the intro paragraph of report.docx" propose a real, reviewable change
// and ask before writing anything — nothing is applied silently. Detection
// lives in mova.local/documents (DetectEditIntent, shared with the MCP
// door — see mcp/nl_edit.go); reading/prompting/cleanup also shared
// (ReadEditableContent/BuildEditPrompt/ExtractEditedContent).
//
// Flow per detected message:
//  1. Resolve every mentioned path to an EXISTING file (this, not the verb
//     alone, is what tells "edit" apart from nl_save.go's "create" — a
//     request naming something that doesn't exist yet isn't an edit).
//     No path mentioned at all falls back to chatFileState.lastFile — the
//     last file this chat created or edited — so "now fix the typo in the
//     third paragraph" works without repeating the filename.
//  2. Ask the model for the complete new content (BuildEditPrompt), given
//     the file's current content and the request.
//  3. Show the exact diff (documents.DiffLines) — never apply anything
//     the person hasn't seen.
//  4. Ask "Apply this change? (y/n)"; with more than one file, ask once
//     "Apply to ALL N files? (y/n)" first as a shortcut, falling back to
//     per-file confirmation on "n".
package main

import (
	"bufio"
	"fmt"
	"strings"

	"mova.local/core"
	"mova.local/documents"
	"mova.local/models"
)

// chatFileState tracks the last file this chat session created or edited,
// so a follow-up like "now fix the typo" doesn't need to repeat a path.
// Deliberately just one field — a full undo/redo history is out of scope.
type chatFileState struct {
	lastFile string
}

// handleNaturalLanguageEdit returns true when it handled the message
// itself (verb detected) — even if nothing ended up resolvable to an
// existing file, so the caller does not also send it as an ordinary,
// confusing chat turn.
func handleNaturalLanguageEdit(adapter core.Adapter, root string, proj *core.Project, sess *models.Session, line string, state *chatFileState, scanner *bufio.Scanner) bool {
	intent := documents.DetectEditIntent(line)
	if !intent.VerbDetected {
		return false
	}

	targets := intent.Files
	if len(targets) == 0 {
		if state.lastFile == "" {
			return false // no path mentioned and nothing to fall back to — let it go through as ordinary chat, it may not be about a file at all
		}
		targets = []string{state.lastFile}
	}

	repo := ""
	if proj != nil {
		repo = proj.Repo
	}

	type editable struct {
		full, content string
	}
	var files []editable
	for _, ref := range targets {
		full, ambiguous, exists, err := documents.ResolveExistingFile(root, repo, ref)
		if err != nil {
			consolePrint(fmt.Sprintf("[Edit] Error resolving %q: %s\n", ref, err.Error()))
			continue
		}
		if len(ambiguous) > 0 {
			consolePrint(fmt.Sprintf("[Edit] %q matches more than one file — be more specific (%s).\n", ref, strings.Join(ambiguous, ", ")))
			continue
		}
		if !exists {
			consolePrint(fmt.Sprintf("[Edit] %q does not exist yet — use a creation phrase instead (e.g. \"Generate %s\") if you meant to create it.\n", ref, ref))
			continue
		}
		content, err := documents.ReadEditableContent(full)
		if err != nil {
			consolePrint(fmt.Sprintf("[Edit] Could not read %q: %s\n", full, err.Error()))
			continue
		}
		files = append(files, editable{full: full, content: content})
	}
	if len(files) == 0 {
		return true
	}

	applyAll, askedApplyAll := false, false
	for _, f := range files {
		consolePrint(fmt.Sprintf("[Edit] Asking the model to update %s...\n", f.full))
		reply, err := sess.Send(documents.BuildEditPrompt(f.full, f.content, line))
		if err != nil {
			consolePrint("[Edit] Error: " + err.Error() + "\n")
			continue
		}
		newContent := documents.ExtractEditedContent(reply)

		diff := documents.DiffLines(f.content, newContent)
		if !diff.Changed {
			consolePrint("[Edit] No changes proposed for " + f.full + ".\n")
			continue
		}
		consolePrint(fmt.Sprintf("[Edit] Proposed changes for %s (+%d/-%d lines):\n%s", f.full, diff.LinesAdded, diff.LinesRemoved, diff.Text))
		if documents.IsRegeneratedFormat(f.full) {
			consolePrint("[Edit] Note: this format is regenerated from its text content, not byte-patched — layout/formatting will be reapplied.\n")
		}

		apply := applyAll
		if !apply {
			if len(files) > 1 && !askedApplyAll {
				askedApplyAll = true
				consolePrint(fmt.Sprintf("Apply this change to ALL %d files without asking again? (y/n): ", len(files)))
				if readYesNo(scanner) {
					applyAll, apply = true, true
				}
			}
			if !apply {
				consolePrint("Apply this change? (y/n): ")
				apply = readYesNo(scanner)
			}
		}
		if !apply {
			consolePrint("[Edit] Skipped " + f.full + ".\n")
			continue
		}

		result, err := documents.Save(root, documents.SaveRequest{Path: f.full, Content: newContent})
		if err != nil {
			consolePrint("[Edit] Error saving " + f.full + ": " + err.Error() + "\n")
			continue
		}
		consolePrint("[Edit] " + result.Message + "\n")
		state.lastFile = f.full
	}
	return true
}

// readYesNo reads one line from scanner and reports whether it means
// "yes" — accepts y/yes/s/si/sí (the person's input, not a CLI message,
// so both languages are accepted regardless of the English-only output).
func readYesNo(scanner *bufio.Scanner) bool {
	if !scanner.Scan() {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes" || answer == "s" || answer == "si" || answer == "sí"
}
