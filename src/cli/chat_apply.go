// chat_apply.go — "Auto-Apply": intercepts a bare confirmation
// ("sí"/"si"/"yes"/"s"/"y", or "#1,#3") that follows a "Direct
// Application" proposal in the PREVIOUS assistant turn, and writes
// the labeled code blocks from that proposal straight to disk without
// a second model round-trip. This is deliberately a SEPARATE, simpler
// flow from nl_edit.go's ("modify the login function" -> model call ->
// diff -> y/n, all within one message): nl_edit.go handles a person
// explicitly asking for an edit; this handles a person confirming a
// proposal the model already made unprompted, in its own words,
// earlier in the conversation. Detection (mova.local/documents) and
// application (mova.local/patcher) are shared with the MCP door (see
// mcp/chat_tool.go) - this file only wires them into the CLI REPL.
package main

import (
	"strconv"
	"strings"

	"mova.local/applyflow"
	"mova.local/core"
	"mova.local/documents"
	"mova.local/i18n"
	"mova.local/models"
)

// handleAutoApplyConfirmation returns true when it handled line itself
// (the caller must not also send it through an ordinary chat turn).
// It deliberately does nothing (returns false) unless BOTH are true:
// the line is a bare confirmation, AND the previous assistant message
// actually looks like a Direct Application proposal — an ordinary
// "sí" answering an unrelated question is never hijacked.
func handleAutoApplyConfirmation(adapter core.Adapter, root string, proj *core.Project, sess *models.Session, project, line string, emit func(string)) bool {
	if emit == nil {
		emit = consolePrint
	}
	ok, ids := documents.DetectApplyConfirmation(line)
	if !ok {
		return false
	}
	_, assistant, has := sess.LastExchange()
	if !has || !documents.LooksLikeDirectApplyProposal(assistant) {
		return false
	}

	task := ""
	if proj != nil {
		task = core.ChatTaskName(proj, "")
	}
	p := applyflow.BuildForced(root, project, task, proj, assistant)
	if p == nil {
		return false
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
	emit(p.Apply(root, sel).Text + "\n")

	if project != "" {
		if res, mErr := core.RecordMemory(adapter, root, project, "", assistant, core.RecordOptions{}); mErr == nil && res.Saved > 0 {
			emit(i18n.T("chat.memory_updated", map[string]any{"project": project}) + "\n")
		}
	}

	return true
}
