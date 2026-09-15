// nl_delete.go — natural-language file/directory DELETION for the
// MCP/HTTP "chat_completion" tool, mirroring cli/nl_delete.go's fix
// for the same real bug: "elimina"/"borra"/"delete" verbs used to also
// match editVerbRe (documents.DetectEditIntent), so a delete request
// went through the EDIT flow and just emptied the file instead of
// removing it. documents.DetectDeleteIntent is checked BEFORE edit
// intent (see chat_tool.go's dispatch order) and reuses
// documents.Delete directly — the same function the "delete_path" MCP
// tool and CLI's /delete already use.
//
// Like applyNaturalLanguageEdits (nl_edit.go), MCP/HTTP calls are
// stateless: there is no interactive y/n. An explicit "apply_delete"
// argument (default false) decides whether Delete actually removes
// anything (true) or only returns what WOULD be deleted (false/
// omitted) — call again with apply_delete: true to confirm.
package mcp

import (
	"fmt"
	"strings"

	"mova.local/core"
	"mova.local/documents"
)

// applyNaturalLanguageDelete detects and processes delete intent in
// message. handled reports whether a delete verb was found at all —
// the caller must NOT also send message through the ordinary chat
// turn, nor through applyNaturalLanguageEdits, when handled is true.
func applyNaturalLanguageDelete(statusLog *strings.Builder, root string, proj *core.Project, message string, applyDelete bool) (reply string, handled bool) {
	intent := documents.DetectDeleteIntent(message)
	if !intent.VerbDetected || len(intent.Targets) == 0 {
		return "", false
	}
	repo := ""
	if proj != nil {
		repo = proj.Repo
	}
	// "elimina el CONTENIDO de X" empties the file instead of removing
	// it — no confirmation phase needed, this never destroys the file
	// itself (see documents.ClearContent).
	if intent.ClearContentOnly {
		var out strings.Builder
		for _, t := range intent.Targets {
			msg, err := documents.ClearContent(root, repo, t)
			if err != nil {
				out.WriteString(fmt.Sprintf("[Clear] Error: %s\n", err.Error()))
				continue
			}
			out.WriteString("[Clear] " + msg + "\n")
		}
		return out.String(), true
	}
	result, err := documents.Delete(root, documents.DeleteRequest{Paths: intent.Targets, Repo: repo, Confirm: applyDelete})
	if err != nil {
		return fmt.Sprintf("[Delete] Error: %s\n", err.Error()), true
	}
	var out strings.Builder
	if result.Pending {
		out.WriteString("[Delete] " + result.Prompt + "\n")
		out.WriteString("[Delete] Not applied — this call had no \"apply_delete\": true, so nothing was removed. Call chat_completion again with the same message and \"apply_delete\": true to confirm (there is no interactive prompt on this door — see COMMANDS.md).\n")
	} else {
		out.WriteString("[Delete] " + result.Message + "\n")
	}
	statusLog.WriteString("[Delete] processed " + fmt.Sprint(len(intent.Targets)) + " target(s)\n")
	return out.String(), true
}
