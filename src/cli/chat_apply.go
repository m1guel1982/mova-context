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

	"mova.local/core"
	"mova.local/documents"
	"mova.local/i18n"
	"mova.local/models"
	"mova.local/patcher"
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

	blocks := documents.ExtractLabeledCodeBlocks(assistant)
	if len(blocks) == 0 {
		return false
	}
	blocks = filterBlocksByIDs(blocks, ids)

	repoRoot := root
	if proj != nil && proj.Repo != "" {
		repoRoot = proj.Repo
	}

	applied, err := patcher.ApplyBlocks(repoRoot, blocks)
	for _, a := range applied {
		if a.Symbol != "" && !a.WholeFile {
			emit(i18n.T("chat.auto_apply_symbol_updated", map[string]any{"path": a.Path, "symbol": a.Symbol}) + "\n")
		} else if a.WholeFile && a.Symbol != "" {
			emit("[Auto-Apply] " + a.Path + " - símbolo '" + a.Symbol + "' no encontrado con certeza, archivo completo reescrito\n")
		} else {
			emit(i18n.T("chat.auto_apply_file_updated", map[string]any{"path": a.Path}) + "\n")
		}
	}
	if err != nil {
		emit("[Auto-Apply] Advertencia: " + err.Error() + "\n")
	}

	if project != "" {
		if block, mErr := core.ExtractMemoryBlock(assistant); mErr == nil && block != "" {
			if err := adapter.AppendMemory(project, block); err == nil {
				emit(i18n.T("chat.memory_updated", map[string]any{"project": project}) + "\n")
			}
		}
	}

	return true
}

// filterBlocksByIDs keeps only the 1-indexed blocks named in ids
// (from a "#1,#3" confirmation) - an empty ids list means "apply
// everything", the plain "sí"/"yes" case.
func filterBlocksByIDs(blocks []documents.LabeledCodeBlock, ids []string) []documents.LabeledCodeBlock {
	if len(ids) == 0 {
		return blocks
	}
	wanted := map[int]bool{}
	for _, id := range ids {
		if n, err := strconv.Atoi(strings.TrimSpace(id)); err == nil {
			wanted[n] = true
		}
	}
	var out []documents.LabeledCodeBlock
	for i, b := range blocks {
		if wanted[i+1] {
			out = append(out, b)
		}
	}
	return out
}
