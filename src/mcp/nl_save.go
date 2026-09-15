// nl_save.go — natural-language file/directory creation for the MCP/HTTP
// "chat_completion" tool, mirroring cli/nl_save.go so a plain message
// like "Genera reporte.pdf" creates a file the same way whether it comes
// from `mova chat`, an MCP client, or a raw HTTP POST — no special
// command needed on any door. Detection itself lives in
// mova.local/documents.DetectSaveIntent, shared by both. Append/overwrite
// modifiers ("agrega al final", "sobreescribe", "no lo sobreescribas" —
// mova.local/documents.DetectSaveModifiers) work here too.
//
// IMPORTANT (bug found in QA, same family as the one cli/nl_save.go's
// header already documents): this file used to send the person's raw
// message to the model as an ordinary turn (sendWithToolsMCP) and save
// WHATEVER TEXT came back verbatim into every detected path — with 2+
// files in one message, the exact same reply landed in every file
// (never split per-file), and a model with no idea it can write files
// would answer "I can't create files" and THAT got saved instead. Fixed
// the same way the CLI door was: applyNaturalLanguageFilesViaChanges
// below turns the detected paths into an explicit apply_file_changes
// instruction (BuildApplyFileChangesInstruction) and runs it through
// sendForcedFileChangesMCP, which parses that ONE structured tool call
// regardless of project.json's "tools.enabled" — same reasoning as
// cli/chat_helpers.go's sendWithForcedFileChanges. MCP/HTTP have no
// terminal for an interactive confirmation menu, so — exactly like this
// door always did for nl_save — every change is applied immediately;
// the difference is the content is now correct per file instead of one
// shared blob.
package mcp

import (
	"fmt"
	"strings"

	"mova.local/core"
	"mova.local/documents"
	"mova.local/i18n"
	"mova.local/models"
)

// applyNaturalLanguageDirectories creates any directory-creation intent
// found in message BEFORE the message is sent to the model — directories
// never depend on the model's reply, so there's no reason to wait.
func applyNaturalLanguageDirectories(statusLog *strings.Builder, root string, proj *core.Project, message string) documents.SaveIntent {
	intent := documents.DetectSaveIntent(message)
	if len(intent.Directories) == 0 {
		return intent
	}
	repo := ""
	if proj != nil {
		repo = proj.Repo
	}
	for _, dir := range intent.Directories {
		result, err := documents.Save(root, documents.SaveRequest{Directory: dir, Repo: repo})
		if err != nil {
			statusLog.WriteString("[Save] Error: " + err.Error() + "\n")
			continue
		}
		statusLog.WriteString("[Save] " + result.Message + "\n")
	}
	return intent
}

// BuildApplyFileChangesInstruction turns the person's own file-creation
// request (already detected via documents.DetectSaveIntent by both
// cli/nl_save.go and this file's applyNaturalLanguageDirectories) plus
// the paths already found by regex into an imperative instruction a
// model has no real room to refuse or misinterpret: it does NOT ask the
// model to decide whether a tool is appropriate (that is exactly where
// weak/local models fail — they answer "I can't create files" because
// they don't know this tool exists), it spells out the EXACT
// <<<MOVA_TOOL_CALL>>> block to answer with — including the wrapper
// ToolsSystemPrompt teaches, repeated here so this works even when
// project.json has "tools": {"enabled": false} (the whole point of
// nl_save: file creation from plain language must not depend on that
// flag, same as /save never has) — leaving only the actual file CONTENT
// for the model to produce. Shared by cli/nl_save.go (CLI door) and
// this package's own applyNaturalLanguageFilesViaChanges (MCP/HTTP
// door) so both doors teach the model the identical wire format.
func BuildApplyFileChangesInstruction(original string, paths []string) string {
	var b strings.Builder
	b.WriteString(original)
	b.WriteString("\n\n---\nSYSTEM NOTE: you have local file-system write access through the apply_file_changes tool — you are NOT a sandboxed assistant without file access, and must NEVER say you can't create/write files, and must NEVER just print the JSON as plain text. ")
	fmt.Fprintf(&b, "Respond with EXACTLY the block below and nothing else (no prose before or after, no apology, no markdown fences) — %d entr%s in \"changes\", one per path, action \"create\", real content for each (never leave \"content\" empty or as a placeholder):\n\n", len(paths), plural(len(paths)))
	b.WriteString(ToolCallStart + "\n")
	b.WriteString(`{"name": "apply_file_changes", "arguments": {"changes": [` + "\n")
	for _, p := range paths {
		fmt.Fprintf(&b, "  {\"action\": \"create\", \"path\": %q, \"content\": \"...\"},\n", p)
	}
	b.WriteString("]}}\n")
	b.WriteString(ToolCallEnd + "\n")
	return b.String()
}

func plural(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

// applyNaturalLanguageFilesViaChanges replaces the old
// applyNaturalLanguageFiles (see this file's header for the bug that
// approach had): instead of reusing whatever reply the ordinary chat
// turn already produced, it sends its OWN apply_file_changes-forced
// instruction (BuildApplyFileChangesInstruction) via
// sendForcedFileChangesMCP and applies every resulting change
// immediately — MCP/HTTP have no interactive terminal for a
// confirmation menu (see mcp.DescribePendingChanges), and the paths
// here came from the person's own message, not the model's initiative,
// so immediate application preserves this door's existing nl_save
// contract. Deliberately does not apply append/overwrite modifiers
// (documents.DetectSaveModifiers) — cli/nl_save.go's own creation flow
// never has either, so both doors now match exactly; a modifier only
// makes sense against an ALREADY EXISTING file, which is nl_edit.go's
// territory, not nl_save's.
func applyNaturalLanguageFilesViaChanges(statusLog *strings.Builder, sess *models.Session, adapter core.Adapter, root string, intent documents.SaveIntent, message string) (string, error) {
	if len(intent.Files) == 0 {
		return "", nil
	}
	statusLog.WriteString(fmt.Sprintf("[NL] Detected file creation intent for %d file(s) — asking the model to propose the content via apply_file_changes...\n", len(intent.Files)))
	instruction := BuildApplyFileChangesInstruction(message, intent.Files)
	return sendForcedFileChangesMCP(statusLog, sess, adapter, root, instruction)
}

// sendForcedFileChangesMCP sends an apply_file_changes-forced
// instruction and, unlike sendWithToolsMCP, ALWAYS parses the reply for
// that one call regardless of project.json's "tools.enabled" — same
// reasoning as cli/chat_helpers.go's sendWithForcedFileChanges. Every
// change is applied immediately (no interactive menu exists on this
// door — see mcp.DescribePendingChanges' doc comment for why a
// MODEL-initiated apply_file_changes call never auto-applies here; this
// case is different because the paths were already decided by the
// person's own message, not the model, before this function is even
// called).
func sendForcedFileChangesMCP(statusLog *strings.Builder, sess *models.Session, adapter core.Adapter, root, instruction string) (string, error) {
	reply, err := sess.Send(instruction)
	if err != nil {
		return "", err
	}
	for i := 0; i < MaxAgentToolTurns; i++ {
		name, args, ok := ParseAgentToolCall(reply)
		if !ok || name != "apply_file_changes" {
			break
		}
		statusLog.WriteString(fmt.Sprintf("[Tool] %s %v\n", name, args))
		changes, ok := ParseApplyFileChanges(args)
		if !ok {
			statusLog.WriteString("[Tool] ERROR: \"changes\" must be a non-empty array of {\"action\",\"path\",\"content\"}\n")
			break
		}
		var results []string
		for _, c := range changes {
			res, aerr := ApplyFileChange(adapter, root, c)
			if aerr != nil {
				line := i18n.T("tools.file_changes.status_error", map[string]any{"path": c.Path, "error": aerr.Error()})
				statusLog.WriteString(line + "\n")
				results = append(results, line)
				continue
			}
			_ = res
			line := i18n.T("tools.file_changes.status_applied", map[string]any{"path": c.Path})
			statusLog.WriteString(line + "\n")
			results = append(results, line)
		}
		result := strings.Join(results, "\n")
		reply, err = sess.Send(fmt.Sprintf(
			"TOOL_RESULT(%s): %s\n\nContinue the reply for the user using this real result. If you need another tool, emit another block; if you are done, answer in plain text.",
			name, result))
		if err != nil {
			return "", err
		}
	}
	return reply, nil
}
