// chat_cmd.go — `mova chat [project] [task]`.
//
// A simple REPL that talks to the active local/Cloud provider
// (config/models/active.json) through mova.local/models.Session.
// If [project] is given (and optionally [task]), the full Mova context
// (agents+skills+prompt+memory+focus — the same one `mova run` builds)
// is injected as the system message: the model reasons under the same
// rules a powerful model would, without copy-pasting anything by hand.
//
// Before sending anything to a model, the configured "budget":
// {"max_tokens": N} limit (if any) is enforced — see
// mova.local/budget.EnforceLimit. This is the same hard gate the MCP
// chat_completion tool applies, so CLI and MCP/HTTP behave identically.
//
// Inside the chat:
//
//	set -model <name>        switch models, keeps history
//	/memory                  saves the last exchange to memory.md (requires [project])
//	/budget                  generates mova-budget-report.md for the active project (requires [project])
//	/save "path"             saves the model's last reply to path — format auto-picked from extension
//	/save -c "path"          saves ONLY the source code blocks from the model's last reply
//	/save -d "path"          creates only that directory (requires [project])
//	/save -append "path"     appends the model's last reply to an existing file instead of overwriting it
//	/save -overwrite "path"  forces overwriting an existing file
//	/save -no-overwrite "path" fails instead of overwriting an existing file
//	/delete "path" ["path2" ...]  deletes files/directories, confirming each one (Y/N) — see delete_cmd.go
//	/tools                   lists every file/directory capability available in this chat
//	/clear                   clears the terminal screen
//	exit | quit              ends the session
//
// workflow.md: "lee workflow.md", "leer workflow.md", "ejecuta
// workflow.md", "run workflow.md", "execute workflow.md",
// "workflow.md <project>", "workflow.md <project> <task>" all resolve
// the project and validate its Budget BEFORE ever reading workflow.md —
// see workflow_cmd.go and mova.local/budget.LoadWorkflow. If the
// resulting context exceeds the configured limit, workflow.md is never
// loaded and the same ERROR/Suggestion block `mova budget` shows is
// printed instead.
//
// Natural-language file/directory creation (no /save needed): plain
// messages like "Genera reporte.pdf" or "Crea el directorio docs/out"
// are detected automatically — see nl_save.go.
//
// Natural-language file EDITING (Claude Console-style): plain messages
// like "Modifica la función login en auth.go" or "Cambia el texto de
// report.md" propose a reviewable change and ask "Apply this change?
// (y/n)" before writing anything — see nl_edit.go.
//
// See also: chat_helpers.go (session/provider/token-usage plumbing),
// chat_save.go (/memory, /budget, /save), nl_save.go (natural-language
// file creation), nl_edit.go (natural-language file editing).
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"mova.local/budget"
	"mova.local/core"
	"mova.local/mcp"
	"mova.local/models"
)

func runChat(root, project, task string) {
	sess, err := models.NewSession(root)
	must(err)

	var adapter core.Adapter
	var proj *core.Project
	if project != "" {
		consolePrint("[Project] Loading project configuration...\n")
		fa := core.NewFileAdapter(root)
		proj, _ = fa.GetProject(project)
		adapter = newAdapter(root, proj)
		applyProjectLLMProfile(sess, root, proj)

		consolePrint("[Context] Building context...\n")
		// budget.BuildGatedContext runs the full Token Firewall
		// (Sanitizer → Circuit Breaker → the existing max_tokens gate)
		// — the exact same pipeline `mova run`/`mova jobs run`/
		// `mova agents run` already go through, so chat never has its
		// own copy of "build then gate".
		gated := budget.BuildGatedContext(adapter, root, project, task)
		if gated.Sections != nil {
			printContextSummary(gated.Sections)
		}
		printSanitizeStatus(gated.Sanitize)
		printCircuitBreakerStatus(gated.CircuitBreaker)
		if gated.Err != nil {
			consolePrint("\n" + gated.Err.Error() + "\n\n")
			return
		}

		systemText, boundary := applyCacheLayout(gated.Sections, proj)
		sess.SetSystem(systemText + mcp.ToolsSystemPrompt(proj.Tools))
		sess.CacheBoundary = boundary
		if core.ToolsEnabled(proj.Tools) {
			consolePrint("[Tools] Enabled for this chat — the model may create/write files and directories (see project.json's \"tools\").\n")
		}
		consolePrint("[Context] Project loaded: " + project + "\n")
	}

	consolePrint(chatBanner(sess))

	fileState := &chatFileState{}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for {
		consolePrint("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		switch {
		case line == "exit" || line == "quit" || line == "salir":
			consolePrint("bye!\n")
			return

		case strings.HasPrefix(line, "set -model"):
			name := strings.TrimSpace(strings.TrimPrefix(line, "set -model"))
			if name == "" {
				consolePrint("Usage: set -model <name>\n")
				continue
			}
			if err := sess.SetModel(name); err != nil {
				consolePrint("Error: " + err.Error() + "\n")
				continue
			}
			consolePrint(fmt.Sprintf("[Model] Switched to: %s (provider: %s)\n", sess.Model, sess.Provider))

		case line == "/memory":
			runChatMemory(adapter, project, sess, nil)

		case line == "/budget":
			runChatBudget(root, adapter, project, task, nil)

		case line == "/tools":
			consolePrint(mcp.FileToolsHelp())

		case line == "/clear":
			clearScreen()

		case strings.HasPrefix(line, "/save"):
			runChatSave(adapter, root, proj, sess, strings.TrimSpace(strings.TrimPrefix(line, "/save")), fileState, nil)

		case strings.HasPrefix(line, "/delete"):
			runChatDelete(root, proj, strings.TrimSpace(strings.TrimPrefix(line, "/delete")), scanner)

		// No explicit command matched: try workflow.md first ("lee
		// workflow.md", "ejecuta workflow.md", "workflow.md <project>
		// [task]" — see workflow_cmd.go), then natural-language EDIT
		// intent (modify an EXISTING file — see nl_edit.go), then
		// natural-language CREATE intent (a NEW file/directory — see
		// nl_save.go). Only falls through to an ordinary chat turn if
		// the message carries none of these.
		default:
			if handleWorkflowCommand(project, task, sess, root, line) {
				continue
			}
			if handleNaturalLanguageEdit(adapter, root, proj, sess, line, fileState, scanner) {
				continue
			}
			if handleNaturalLanguageSave(adapter, root, proj, sess, project, line, fileState) {
				continue
			}
			runChatTurn(sess, adapter, proj, root, project, line)
		}
	}
}

// runChatTurn sends one ordinary chat message and prints the reply, the
// token-usage line, and records real usage for the Feedback Loop.
func runChatTurn(sess *models.Session, adapter core.Adapter, proj *core.Project, root, project, line string) {
	label := providerLabel(sess.Provider)
	consolePrint("[" + label + "] Sending request...\n")

	reply, streamed, err := sendWithTools(sess, adapter, proj, root, line, nil)
	if err != nil {
		consolePrint("Error: " + err.Error() + "\n")
		return
	}
	if !streamed {
		consolePrint("[" + label + "] Response received.\n")
		consolePrint(fmt.Sprintf("[%s]\n%s\n", sess.Model, renderMarkdown(reply)))
	}
	printTokenUsage(root, sess, proj)
	recordRealUsage(root, project, proj, sess)
}

func chatBanner(sess *models.Session) string {
	var b strings.Builder
	b.WriteString("mova chat — provider: " + sess.Provider)
	if sess.Model != "" {
		b.WriteString(", model: " + sess.Model)
	} else {
		b.WriteString(" (no model set — use `set -model <name>`)")
	}
	b.WriteString("\ntype `exit` to quit.\n\n")
	return b.String()
}
