// nl_save.go — natural-language file/directory creation for `mova chat`,
// the same experience Claude Desktop/Claude Console give: say what you
// want created in plain language and it gets created, without typing
// `/save` first. Detection lives in mova.local/documents (DetectSaveIntent
// — shared with the MCP door, see mcp/nl_save.go), so both stay in sync.
//
// Examples: "Genera carpeta/reporte.pdf", "Crea c:/proyecto/docs/manual.md",
// "Crea el directorio c:/temp/test y genera reporte.pdf".
//
// IMPORTANT (fixed a real bug found in QA): this used to send the user's
// raw sentence to the model as an ordinary chat turn and then save
// whatever prose came back VERBATIM as the file's content — a small/local
// model with no idea it has file-write access would answer "I can't
// create files, but here's the code to copy-paste" and THAT ENTIRE
// apology+markdown-fence text got written to disk as the file. With 2+
// files in one message, the SAME reply was saved into every file
// (never split per-file). Both are fixed now: detected file paths are
// turned into an explicit apply_file_changes instruction (see
// mcp.BuildApplyFileChangesInstruction, shared with the MCP door) and
// handed to sendWithForcedFileChanges, the SAME structured tool-calling
// + confirmation-menu pipeline Item 15 already built — so content is
// always exactly what's inside the JSON's "content" field (never a
// refusal preamble), each file gets its own content, and the
// interactive menu always appears before anything is written.
//
// `/save` keeps working exactly as before — this only covers messages
// where the person never typed it.
//
// IMPORTANT (bug found in QA, fixed here): this used to call
// sendWithTools, which — when project.json had no
// "tools": {"enabled": true} (the default) — took its "no tool-calling"
// branch and simply streamed the model's raw apply_file_changes JSON
// back as plain text, never parsing or writing it. Detecting file
// creation intent from plain language is a deterministic, already-
// confirmed decision (see documents.DetectSaveIntent below), so it must
// not depend on that unrelated flag — sendWithForcedFileChanges
// (chat_helpers.go) always runs the tool-parsing loop for exactly this
// one call, regardless of "tools.enabled".
package main

import (
	"bufio"
	"fmt"
	"strings"

	"mova.local/core"
	"mova.local/documents"
	"mova.local/mcp"
	"mova.local/models"
)

// handleNaturalLanguageSave inspects a chat line for creation intent. It
// returns true when it handled the message itself (so the caller must
// NOT also send it through the ordinary chat turn). Behavior:
//
//   - Directory-only intent ("Crea el directorio X"): created immediately,
//     no model call needed.
//   - File intent ("Genera reporte.pdf", possibly several in one
//     message): the model is instructed to call apply_file_changes with
//     one entry per detected path — sendWithForcedFileChanges then shows
//     the SAME confirmation menu a model-initiated apply_file_changes
//     call shows (see confirmAndApplyFileChanges).
//   - Both in one message ("Crea el directorio X y genera Y"): the
//     directory is created first, then the file flow runs as above.
func handleNaturalLanguageSave(adapter core.Adapter, root string, proj *core.Project, sess *models.Session, project, line string, state *chatFileState, scanner *bufio.Scanner) bool {
	intent := documents.DetectSaveIntent(line)
	if !intent.HasIntent() {
		return false
	}

	repo := ""
	if proj != nil {
		repo = proj.Repo
	}

	for _, dir := range intent.Directories {
		result, err := documents.Save(root, documents.SaveRequest{Directory: dir, Repo: repo})
		if err != nil {
			consolePrint("[Save] Error: " + err.Error() + "\n")
			continue
		}
		consolePrint("[Save] " + result.Message + "\n")
	}

	if len(intent.Files) == 0 {
		return true
	}

	consolePrint(fmt.Sprintf("[NL] Detected file creation intent for %d file(s) — asking the model to propose the content via apply_file_changes...\n", len(intent.Files)))

	replLine := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return strings.TrimSpace(scanner.Text()), true
	}
	instruction := mcp.BuildApplyFileChangesInstruction(line, intent.Files)
	reply, err := sendWithForcedFileChanges(sess, adapter, root, instruction, replLine, nil)
	if err != nil {
		consolePrint("Error: " + err.Error() + "\n")
		return true
	}
	if reply != "" {
		consolePrint(renderMarkdown(reply) + "\n")
	}
	if len(intent.Files) > 0 {
		state.lastFile = intent.Files[len(intent.Files)-1]
	}
	return true
}
