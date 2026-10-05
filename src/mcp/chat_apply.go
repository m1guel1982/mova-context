// chat_apply.go — MCP door's twin of cli/chat_apply.go: intercepts a
// bare confirmation ("sí"/"yes"/"#1,#3") that follows a "Direct
// Application" proposal already present in the session's history
// (the MCP door receives history as part of the request - see
// chat_tool.go), and writes the labeled code blocks straight to disk
// using the exact same mova.local/patcher.ApplyBlocks the CLI uses -
// one implementation, two doors, never two copies of the logic.
package mcp

import (
	"strconv"
	"strings"

	"mova.local/applyflow"
	"mova.local/core"
	"mova.local/documents"
	"mova.local/models"
)

// applyAutoApplyConfirmationMCP mirrors cli/chat_apply.go's
// handleAutoApplyConfirmation - see its header for the full design
// rationale (why this is separate from applyNaturalLanguageEdits).
func applyAutoApplyConfirmationMCP(statusLog *strings.Builder, adapter core.Adapter, root string, proj *core.Project, project string, sess *models.Session, message string) (reply string, handled bool) {
	ok, ids := documents.DetectApplyConfirmation(message)
	if !ok {
		return "", false
	}
	_, assistant, has := sess.LastExchange()
	if !has || !documents.LooksLikeDirectApplyProposal(assistant) {
		return "", false
	}

	task := ""
	if proj != nil {
		task = core.ChatTaskName(proj, "")
	}
	p := applyflow.BuildForced(root, project, task, proj, assistant)
	if p == nil {
		return "", false
	}
	sel := map[int]bool{}
	for i := range p.Changes {
		sel[i+1] = len(ids) == 0
	}
	for _, id := range ids {
		if n, err := strconv.Atoi(strings.TrimSpace(id)); err == nil {
			sel[n] = true
		}
	}
	var out strings.Builder
	out.WriteString(p.Apply(root, sel).Text + "\n")

	if project != "" {
		if res, mErr := core.RecordMemory(adapter, root, project, "", assistant, core.RecordOptions{}); mErr == nil && res.Saved > 0 {
			out.WriteString("[Memory] Actualizado automáticamente tras Auto-Apply (" + project + ")\n")
		}
	}

	statusLog.WriteString("[Auto-Apply] Confirmation detected - applying previous proposal directly.\n")
	return out.String(), true
}
