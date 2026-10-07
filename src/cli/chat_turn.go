// chat_turn.go — un turno de chat ordinario (mensaje → respuesta).
// Movido desde chat_cmd.go (≤300 líneas por archivo) y ampliado:
//   - registra la síntesis de la respuesta en memory.md (core.RecordMemory)
//     cuando project.json tiene "memory" activo, para que cualquier otra
//     tarea —en este chat, en otro chat, por MCP o por HTTP— la lea;
//   - muestra los tokens consumidos TAMBIÉN cuando el turno falla o la
//     respuesta viene vacía (antes se perdía ese dato);
//   - avisa si la respuesta se cortó por el límite de salida.
package main

import (
	"bufio"
	"fmt"
	"strings"

	"mova.local/core"
	"mova.local/i18n"
	"mova.local/models"
)

// runChatTurn sends one ordinary chat message and prints the reply, the
// token-usage line, and records real usage for the Feedback Loop. Devuelve
// true si escribió memoria (el llamador recalcula la firma de contexto).
func runChatTurn(sess *models.Session, adapter core.Adapter, proj *core.Project, root, project, task, line string, scanner *bufio.Scanner) (recorded bool) {
	label := providerLabel(sess.Provider)
	consolePrint(i18n.T("chat.sending_request", map[string]any{"label": label}) + "\n")

	replLine := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return strings.TrimSpace(scanner.Text()), true
	}
	reply, streamed, err := sendWithTools(sess, adapter, proj, root, line, replLine, nil)
	if err != nil {
		consolePrint("Error: " + err.Error() + "\n")
		// Los tokens ya se gastaron aunque no llegara texto: mostrarlos.
		if sess.LastUsage.PromptTokens > 0 || sess.LastUsage.CompletionTokens > 0 {
			printTokenUsage(root, sess, proj)
			recordRealUsage(root, project, proj, sess)
		}
		return false
	}
	if !streamed {
		if sess.LastReplyWasDryRun {
			// The provider was never called — say so instead of the
			// generic "Response received.", which would otherwise
			// falsely claim a real network round-trip happened (see
			// project.json's "egress_audit": {"dry_run": true}).
			consolePrint("[egress_audit] " + i18n.T("reports.egress_dry_run_done") + "\n")
		} else {
			consolePrint(i18n.T("chat.response_received", map[string]any{"label": label}) + "\n")
		}
		consolePrint(fmt.Sprintf("[%s]\n%s\n", sess.Model, renderMarkdown(reply)))
	}
	if sess.LastTruncated {
		consolePrint(i18n.T("chat.reply_truncated") + "\n")
	}
	// «apply» activo: Mova pregunta y modifica los archivos propuestos.
	offerChatApply(root, project, task, proj, reply, scanner)
	printTokenUsage(root, sess, proj)
	recordRealUsage(root, project, proj, sess)
	if project != "" && !sess.LastReplyWasDryRun {
		res, err := core.RecordMemory(adapter, root, project, task, reply, core.RecordOptions{Source: "observed", RunID: runID(sess)})
		switch {
		case err != nil:
			consolePrint("[Memory] no se pudo guardar: " + err.Error() + "\n")
		case res.Message() != "":
			consolePrint(res.Message() + "\n")
		}
		return err == nil && res.Saved > 0
	}
	return false
}

// runID is the evidence run of the session ("" when none).
func runID(sess *models.Session) string {
	if sess == nil || sess.Run == nil {
		return ""
	}
	return sess.Run.ID
}
