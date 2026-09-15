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

	"mova.local/core"
	"mova.local/documents"
	"mova.local/models"
	"mova.local/patcher"
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

	blocks := documents.ExtractLabeledCodeBlocks(assistant)
	if len(blocks) == 0 {
		return "", false
	}
	blocks = filterBlocksByIDsMCP(blocks, ids)

	repoRoot := root
	if proj != nil && proj.Repo != "" {
		repoRoot = proj.Repo
	}

	applied, err := patcher.ApplyBlocks(repoRoot, blocks)
	var out strings.Builder
	for _, a := range applied {
		switch {
		case a.Symbol != "" && !a.WholeFile:
			out.WriteString("[APLICADO] " + a.Path + "::" + a.Symbol + "() - función actualizada\n")
		case a.Symbol != "" && a.WholeFile:
			out.WriteString("[Auto-Apply] " + a.Path + " - símbolo '" + a.Symbol + "' no encontrado con certeza, archivo completo reescrito\n")
		default:
			out.WriteString("[Auto-Apply] Archivo actualizado: " + a.Path + "\n")
		}
	}
	if err != nil {
		out.WriteString("[Auto-Apply] Advertencia: " + err.Error() + "\n")
	}

	if project != "" {
		if block, mErr := core.ExtractMemoryBlock(assistant); mErr == nil && block != "" {
			if err := adapter.AppendMemory(project, block); err == nil {
				out.WriteString("[Memory] Actualizado automáticamente tras Auto-Apply (" + project + ")\n")
			}
		}
	}

	statusLog.WriteString("[Auto-Apply] Confirmation detected - applying previous proposal directly.\n")
	return out.String(), true
}

func filterBlocksByIDsMCP(blocks []documents.LabeledCodeBlock, ids []string) []documents.LabeledCodeBlock {
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
