// chat_send.go — envío de un turno de chat (con o sin tool-calling).
// Movido tal cual desde chat_helpers.go para mantener cada archivo ≤300
// líneas; sin cambios de comportamiento.
package main

import (
	"fmt"
	"strings"

	"mova.local/core"
	"mova.local/mcp"
	"mova.local/models"
)

// sendWithTools handles streaming or buffered tool loops. emit receives
// every piece of output this function would otherwise print directly
// (token chunks, "[Tool] ..." trace lines) — pass nil to get the exact
// original behavior (writes straight to the terminal via consolePrint,
// as `mova chat`'s REPL always has). The TUI (see tui_chat.go) passes
// its own emit that appends to an in-memory transcript instead, since
// writing to stdout mid-render would corrupt a Bubble Tea screen; the
// model-calling/tool-loop logic itself is untouched either way — one
// implementation, two output sinks.
func sendWithTools(sess *models.Session, adapter core.Adapter, proj *core.Project, root, userText string, readLine readLineFunc, emit func(string)) (reply string, streamed bool, err error) {
	if emit == nil {
		emit = consolePrint
	}

	// Caso sin herramientas habilitadas / adapter nulo
	if proj == nil || adapter == nil || !core.ToolsEnabled(proj.Tools) {
		var rawReply strings.Builder

		// Interceptamos los tokens en buffer para evitar imprimir el prefijo [modelo]
		// y procesar el formateo limpio al finalizar la respuesta.
		reply, err = sess.SendStream(userText, func(tok string) {
			rawReply.WriteString(tok)
		})
		if err != nil {
			return "", true, err
		}

		// Aplicamos limpieza y estilo estilizado a la salida
		formattedReply := formatTerminalOutput(rawReply.String())
		emit(formattedReply + "\n")

		return formattedReply, true, nil
	}

	// Bucle normal de herramientas cuando sí están habilitadas
	reply, err = sess.Send(userText)
	if err != nil {
		return "", false, err
	}
	for i := 0; i < mcp.MaxAgentToolTurns; i++ {
		name, args, ok := mcp.ParseAgentToolCall(reply)
		if !ok {
			break
		}
		emit(fmt.Sprintf("[Tool] %s %v\n", name, args))
		var result string
		var terr error
		if name == "apply_file_changes" {
			if changes, ok := mcp.ParseApplyFileChanges(args); ok {
				result = confirmAndApplyFileChanges(adapter, root, changes, readLine, emit)
			} else {
				terr = fmt.Errorf(`"changes" must be a non-empty array of {"action","path","content"}`)
			}
		} else {
			result, terr = mcp.RunAgentTool(adapter, root, name, args, proj.Tools)
		}
		if terr != nil {
			result = "ERROR: " + terr.Error()
		}
		emit("[Tool] " + result + "\n")
		reply, err = sess.Send(fmt.Sprintf(
			"TOOL_RESULT(%s): %s\n\nContinue the reply for the user using this real result. If you need another tool, emit another block; if you are done, answer in plain text.",
			name, result))
		if err != nil {
			return "", false, err
		}
	}

	formattedReply := formatTerminalOutput(reply)
	return formattedReply, false, nil
}

// sendWithForcedFileChanges is the variant of sendWithTools used
// exclusively by nl_save.go's natural-language file-creation flow
// (handleNaturalLanguageSave) and its TUI equivalent
// (tui_chat.go's startNaturalLanguageSave). Unlike sendWithTools, it
// ALWAYS parses the reply for an apply_file_changes call — regardless
// of project.json's "tools": {"enabled": ...} — because this
// capability must work exactly like /save already does, independent of
// that project-wide flag.
//
// This fixes a real, reported bug: when tools.enabled was false (the
// default for a brand-new project), sendWithTools took its "no tools"
// branch above, so the apply_file_changes JSON the model produced (see
// mcp.BuildApplyFileChangesInstruction) was streamed back as plain text
// and NEVER parsed — the person saw the raw JSON payload printed to the
// terminal and no file was ever written, even though the model did
// exactly what it was told. Detecting "create hola.txt with X" is
// already a deterministic, pre-confirmed decision by the person (see
// documents.DetectSaveIntent) — it must not silently depend on a
// separate, unrelated opt-in flag meant for the model calling tools
// *on its own initiative* during ordinary conversation.
func sendWithForcedFileChanges(sess *models.Session, adapter core.Adapter, root, userText string, readLine readLineFunc, emit func(string)) (reply string, err error) {
	if emit == nil {
		emit = consolePrint
	}
	reply, err = sess.Send(userText)
	if err != nil {
		return "", err
	}
	for i := 0; i < mcp.MaxAgentToolTurns; i++ {
		name, args, ok := mcp.ParseAgentToolCall(reply)
		if !ok || name != "apply_file_changes" {
			break
		}
		emit(fmt.Sprintf("[Tool] %s %v\n", name, args))
		var result string
		if changes, ok := mcp.ParseApplyFileChanges(args); ok {
			result = confirmAndApplyFileChanges(adapter, root, changes, readLine, emit)
		} else {
			result = `ERROR: "changes" must be a non-empty array of {"action","path","content"}`
		}
		emit("[Tool] " + result + "\n")
		reply, err = sess.Send(fmt.Sprintf(
			"TOOL_RESULT(%s): %s\n\nContinue the reply for the user using this real result. If you need another tool, emit another block; if you are done, answer in plain text.",
			name, result))
		if err != nil {
			return "", err
		}
	}
	return formatTerminalOutput(reply), nil
}
