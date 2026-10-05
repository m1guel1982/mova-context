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
//	set -model <name>         switch models, keeps history
//	/tasks                    lists the project's tasks
//	/task <name|all>          switches task without leaving the chat (keeps history; memory.md carries the context over)
//	/run <name>               switches to a task and sends its QUERY
//	/memory                   saves the last exchange to memory.md (requires [project])
//	/budget                   generates mova-budget-report.md for the active project (requires [project])
//	/context-trace            runs `context-trace` for the active project, or --repo <url> for a remote one
//	/save "path"              saves the model's last reply to path — format auto-picked from extension
//	/save -c "path"           saves ONLY the source code blocks from the model's last reply
//	/save -d "path"           creates only that directory (requires [project])
//	/save -append "path"      appends the model's last reply to an existing file instead of overwriting it
//	/save -overwrite "path"   forces overwriting an existing file
//	/save -no-overwrite "path" fails instead of overwriting an existing file
//	/delete "path" ["path2" ...]  deletes files/directories, confirming each one (Y/N) — see delete_cmd.go
//	/tools                    lists every file/directory capability available in this chat
//	/clear                    clears the terminal screen
//	exit | quit               ends the session
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
	"mova.local/documents"
	"mova.local/mcp"
	"mova.local/models"
)

func runChat(root, project, task string) {
	// Los grafos se generan en segundo plano: ni el arranque ni `exit`
	// esperan un render; al terminar cada uno se avisa en la terminal.
	enableBackgroundGraphs(func(msg string) { consolePrint("\n" + msg + "\n> ") })
	sess, err := models.NewSession(root)
	must(err)

	var adapter core.Adapter
	var proj *core.Project
	var gatedSections *core.ContextSections
	if project != "" {
		// Multiagent groups (projects/<group>/config.json) have no
		// project.json of their own — GetProject below fails for them.
		// resolveChatTarget mirrors exactly how `mova run --count`/
		// `mova agents run` already tell "group" from "project" apart
		// (orchestrator.IsGroup) before deciding what to do — it never
		// changes behavior for an ordinary project (GetProject already
		// resolves projects/<group>/<agent> given the literal
		// "<group>/<agent>" string, unchanged).
		resolved, resolvedTask, ok := resolveChatTarget(root, project, task)
		if !ok {
			return // resolveChatTarget already printed guidance
		}
		project, task = resolved, resolvedTask

		consolePrint("[Project] Loading project configuration...\n")
		fa := core.NewFileAdapter(root)
		proj, _ = fa.GetProject(project)
		// Con una tarea nombrada (`mova chat <proyecto> analizar`) solo se
		// carga el prompt/focus/grafo de ESA tarea; sin tarea y con varias
		// declaradas, TODAS (core.ChatTaskName). `all` fuerza el modo todas.
		task = core.ChatTaskName(proj, task)
		adapter = newAdapter(root, proj)
		applyProjectLLMProfile(sess, root, proj)

		consolePrint("[Context] Building context...\n")
		// budget.BuildGatedContext runs the full Context Governance
		// (Sanitizer → Circuit Breaker → the existing max_tokens gate)
		// — the exact same pipeline `mova run`/`mova jobs run`/
		// `mova agents run` already go through, so chat never has its
		// own copy of "build then gate".
		gated := budget.BuildGatedContext(adapter, root, project, task)
		gatedSections = gated.Sections
		if gated.Sections != nil {
			printContextSummary(gated.Sections, proj)
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
		sess.EgressAuditDryRun, sess.EgressAuditOutputFile = core.ResolveEgressAudit(root, project, proj)
		if core.ToolsEnabled(proj.Tools) {
			consolePrint("[Tools] Enabled for this chat — the model may create/write files and directories (see project.json's \"tools\").\n")
		}
		consolePrint("[Context] Project loaded: " + project + "\n")
		printDebugLog(gatedSections, nil)
	}

	consolePrint(chatBanner(sess))

	fileState := &chatFileState{}
	// signature: firma de project.json + contenido resuelto de
	// agents/skills/prompt (contextSignature) al momento de construir
	// el contexto de arriba — ver refreshProjectContext, que compara
	// contra esta antes de cada turno para detectar ediciones hechas
	// mientras esta sesión de `mova chat` sigue abierta, incluyendo
	// ediciones a los archivos base de agents/prompts/skills (no solo
	// a project.json).
	signature := contextSignature(proj, gatedSections)
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

		// Salir no necesita recargar nada: se sale ya, sin reconstruir contexto.
		if line == "exit" || line == "quit" || line == "salir" {
			consolePrint("bye!\n")
			return
		}

		// Hot reload: antes de interpretar la línea (comando o turno de
		// chat), releer project.json y — solo si algo relevante
		// cambió — reconstruir contexto/system prompt en caliente.
		// Nunca reconstruye si nada cambió (cheap: stat + hash).
		if project != "" {
			proj, adapter, signature = refreshProjectContext(root, project, task, sess, proj, adapter, signature, nil)
		}

		switch {
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

		case line == "/tasks":
			consolePrint(taskListing(root, project, proj, task))

		case strings.HasPrefix(line, "/task ") || line == "/task":
			if project == "" || proj == nil {
				consolePrint("[Task] /task requiere abrir el chat con un proyecto.\n")
				continue
			}
			name := strings.TrimSpace(strings.TrimPrefix(line, "/task"))
			if name == "" {
				consolePrint(taskListing(root, project, proj, task))
				continue
			}
			if t, p2, a2, sig, ok := switchChatTask(root, project, name, sess, proj, adapter, task); ok {
				task, proj, adapter, signature = t, p2, a2, sig
			}

		case strings.HasPrefix(line, "/run"):
			if project == "" || proj == nil {
				consolePrint("[Task] /run requiere abrir el chat con un proyecto.\n")
				continue
			}
			name := strings.TrimSpace(strings.TrimPrefix(line, "/run"))
			if name == "" || core.IsAllTasks(core.NormalizeTaskArg(proj, name)) {
				consolePrint("[Task] Uso: /run <tarea>  (una tarea concreta; ver /tasks)\n")
				continue
			}
			t, p2, a2, sig, ok := switchChatTask(root, project, name, sess, proj, adapter, task)
			if !ok {
				continue
			}
			task, proj, adapter, signature = t, p2, a2, sig
			if runChatTurn(sess, adapter, proj, root, project, task, taskQuery(proj, task), scanner) {
				signature = resignContext(root, project, task, proj)
			}

		case line == "/memory":
			runChatMemory(adapter, project, sess, nil)

		case line == "/budget":
			runChatBudget(root, adapter, project, task, nil)

		case strings.HasPrefix(line, "/diagram"):
			runChatDiagram(root, adapter, project, task, strings.TrimSpace(strings.TrimPrefix(line, "/diagram")))

		case strings.HasPrefix(line, "/context-trace"):
			runChatTrace(root, adapter, project, task, strings.TrimSpace(strings.TrimPrefix(line, "/context-trace")), scanner)

		case line == "/tools":
			consolePrint(mcp.FileToolsHelp())

		case line == "/clear":
			clearScreen()

		case strings.HasPrefix(line, "/save"):
			runChatSave(adapter, root, proj, sess, strings.TrimSpace(strings.TrimPrefix(line, "/save")), fileState, nil)

		case strings.HasPrefix(line, "/delete"):
			runChatDelete(root, proj, strings.TrimSpace(strings.TrimPrefix(line, "/delete")), scannerReadLine(scanner), nil)

		// No explicit command matched: try workflow.md first ("lee
		// workflow.md", "ejecuta workflow.md", "workflow.md <project>
		// [task]" — see workflow_cmd.go), then natural-language EDIT
		// intent (modify an EXISTING file — see nl_edit.go), then
		// natural-language CREATE intent (a NEW file/directory — see
		// nl_save.go). Only falls through to an ordinary chat turn if
		// the message carries none of these.
		//
		// Order matters: DELETE is checked before EDIT (both share
		// "elimina"/"borra"-style phrasing in Spanish — a bug found in
		// QA had "elimina el archivo X" going through the edit flow,
		// which just emptied the file's content instead of actually
		// removing it). READ needs no model call and no confirmation,
		// so it's cheap to check early too.
		default:
			if handleWorkflowCommand(project, task, sess, root, line, nil) {
				continue
			}
			if handleAutoApplyConfirmation(adapter, root, proj, sess, project, line, nil) {
				continue
			}
			if handleNaturalLanguageDelete(root, proj, line, fileState, scannerReadLine(scanner), nil) {
				continue
			}
			if handleNaturalLanguageRename(root, proj, line, scannerReadLine(scanner), nil) {
				continue
			}
			if handleNaturalLanguageRead(root, proj, line, fileState, nil) {
				continue
			}
			if handleNaturalLanguageEdit(adapter, root, proj, sess, line, fileState, scannerReadLine(scanner), nil) {
				continue
			}
			if handleNaturalLanguageSave(adapter, root, proj, sess, project, line, fileState, scanner) {
				continue
			}
			if documents.DetectMemoryIntent(line) {
				runChatMemory(adapter, project, sess, nil)
				continue
			}
			line = reformatIfNeeded(sess, proj, task, line)
			if runChatTurn(sess, adapter, proj, root, project, task, line, scanner) {
				signature = resignContext(root, project, task, proj)
			}
		}
	}
}
