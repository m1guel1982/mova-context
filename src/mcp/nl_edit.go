// nl_edit.go — natural-language file EDITING for the MCP/HTTP
// "chat_completion" tool, mirroring cli/nl_edit.go's Claude Console-style
// review flow. The one real difference: MCP/HTTP calls are stateless (a
// fresh models.Session per call, no terminal to prompt), so there is no
// interactive "Apply this change? (y/n)". Instead, an explicit
// "apply_edits" argument (default false) decides:
//
//   - apply_edits omitted/false: the change is only PROPOSED — the diff is
//     returned in the reply, nothing is written. Call again with
//     apply_edits: true (same message) to write it.
//   - apply_edits: true: every proposed change for this message is applied
//     immediately — there's no separate "apply to all" round trip to make
//     statelessly, so passing it up front IS the "apply to all".
//
// A message with edit intent but no resolvable existing file (nothing
// found on disk, or ambiguous) is still "handled" — it returns an
// explanation instead of silently falling through to an ordinary,
// confusing chat turn about a file that isn't there.
package mcp

import (
	"fmt"
	"strings"

	"mova.local/core"
	"mova.local/documents"
	"mova.local/models"
)

// applyNaturalLanguageEdits detects and processes edit intent in message.
// handled reports whether an edit verb was found at all (with or without
// a resolvable file) — the caller must NOT also send message through the
// ordinary chat turn when handled is true.
func applyNaturalLanguageEdits(statusLog *strings.Builder, sess *models.Session, root string, proj *core.Project, message string, applyEdits bool) (reply string, handled bool) {
	intent := documents.DetectEditIntent(message)
	if !intent.VerbDetected || len(intent.Files) == 0 {
		return "", false
	}

	repo := ""
	if proj != nil {
		repo = proj.Repo
	}

	var out strings.Builder
	for _, ref := range intent.Files {
		full, ambiguous, exists, err := documents.ResolveExistingFile(root, repo, ref)
		if err != nil {
			out.WriteString(fmt.Sprintf("[Edit] Error resolving %q: %s\n", ref, err.Error()))
			continue
		}
		if len(ambiguous) > 0 {
			out.WriteString(fmt.Sprintf("[Edit] %q matches more than one file — be more specific (%s).\n", ref, strings.Join(ambiguous, ", ")))
			continue
		}
		if !exists {
			out.WriteString(fmt.Sprintf("[Edit] %q does not exist yet — use a creation phrase instead (e.g. \"Generate %s\") if you meant to create it.\n", ref, ref))
			continue
		}
		content, err := documents.ReadEditableContent(full)
		if err != nil {
			out.WriteString(fmt.Sprintf("[Edit] Could not read %q: %s\n", full, err.Error()))
			continue
		}

		statusLog.WriteString(fmt.Sprintf("[Edit] Asking the model to update %s...\n", full))
		editReply, err := sess.Send(documents.BuildEditPrompt(full, content, message))
		if err != nil {
			out.WriteString("[Edit] Error: " + err.Error() + "\n")
			continue
		}
		newContent := documents.ExtractEditedContent(editReply)

		diff := documents.DiffLines(content, newContent)
		if !diff.Changed {
			out.WriteString("[Edit] No changes proposed for " + full + ".\n")
			continue
		}
		out.WriteString(fmt.Sprintf("[Edit] Proposed changes for %s (+%d/-%d lines):\n%s", full, diff.LinesAdded, diff.LinesRemoved, diff.Text))
		if documents.IsRegeneratedFormat(full) {
			out.WriteString("[Edit] Note: this format is regenerated from its text content, not byte-patched — layout/formatting will be reapplied.\n")
		}

		if !applyEdits {
			out.WriteString("[Edit] Not applied — this call had no \"apply_edits\": true, so nothing was written. Call chat_completion again with the same message and \"apply_edits\": true to apply it (there is no interactive prompt on this door — see COMMANDS.md).\n")
			continue
		}
		result, err := documents.Save(root, documents.SaveRequest{Path: full, Content: newContent})
		if err != nil {
			out.WriteString("[Edit] Error saving " + full + ": " + err.Error() + "\n")
			continue
		}
		out.WriteString("[Edit] " + result.Message + "\n")
	}
	return out.String(), true
}
