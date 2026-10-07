// chat_tool.go — MCP tool "chat_completion". Same door as every other
// tool (executeTool in server.go), automatically exposed over HTTP too
// because http/server.go is a thin wrapper over Process().
//
// Unlike `mova chat` (which keeps a session with history across
// messages), this tool is stateless per call: each MCP/HTTP invocation
// builds a fresh session, optionally injects [project]/[task]'s context
// as the system prompt, sends the message, and returns the reply. A
// client that wants a multi-turn conversation can pass its own history
// in "history" (optional).
//
// Before sending anything to a model, the configured "budget":
// {"max_tokens": N} limit (if any) is enforced — identical to `mova chat`
// (see cli/chat_cmd.go and mova.local/budget.EnforceLimit), so CLI and
// MCP/HTTP behave exactly the same way.
package mcp

import (
	"encoding/json"
	"fmt"
	"mova.local/evidence"
	"strings"

	"mova.local/budget"
	"mova.local/core"
	"mova.local/documents"
	"mova.local/i18n"
	"mova.local/models"
)

func chatCompletionTool(adapter core.Adapter, root string, args map[string]any) (string, error) {
	message := str(args, "message")
	// «apply_changes» responde a una propuesta pendiente y no necesita mensaje.
	if message == "" && str(args, "apply_changes") == "" {
		return "", fmt.Errorf("message is required")
	}

	sess, err := models.NewSession(root)
	if err != nil {
		return "", err
	}

	var statusLog strings.Builder
	project := str(args, "project")
	var proj *core.Project
	savedTask := "" // tarea efectiva de esta llamada (para guardar su resultado)

	if project != "" {
		statusLog.WriteString("[Project] Loading project configuration...\n")
		proj, err = adapter.GetProject(project)
		if err != nil {
			return "", fmt.Errorf("loading project %q: %w", project, err)
		}
		if proj.LLMProfile != nil && proj.LLMProfile.Config != "" {
			provider := proj.LLMProfile.Provider
			if provider == "" {
				resolved, rerr := models.ResolveConfigProvider(root, proj.LLMProfile.Config)
				if rerr != nil {
					statusLog.WriteString(fmt.Sprintf("[Project] Warning: could not resolve provider for config %q: %s\n", proj.LLMProfile.Config, rerr.Error()))
					provider = ""
				} else {
					provider = resolved
				}
			}
			if provider != "" {
				if err := sess.SwitchProvider(provider, proj.LLMProfile.Config); err != nil {
					statusLog.WriteString(fmt.Sprintf("[Project] Warning: could not switch to configured provider %q: %s\n", provider, err.Error()))
				} else {
					statusLog.WriteString(fmt.Sprintf("[Project] Using configured provider: %s (%s)\n", sess.Provider, sess.Model))
				}
			}
		}
	}

	if modelName := str(args, "model"); modelName != "" {
		if err := sess.SetModel(modelName); err != nil {
			return "", err
		}
	}

	if project != "" {
		statusLog.WriteString("[Context] Building context...\n")
		// Tarea nombrada → solo su prompt/focus/grafo; sin tarea y con
		// varias declaradas → todas (core.ChatTaskName), igual que
		// `mova chat`. Cada llamada MCP/HTTP es una sesión nueva, así que
		// memory.md (core.RecordMemory) es lo que une
		// una tarea con la siguiente.
		taskName := core.ChatTaskName(proj, str(args, "task"))
		savedTask = taskName
		// budget.BuildGatedContext runs the full Context Governance
		// (Sanitizer → Circuit Breaker → the existing max_tokens gate)
		// — the exact same pipeline `mova chat`/the TUI/`mova run`
		// already go through, so MCP never has its own copy of
		// "build then gate".
		gated := budget.BuildGatedContext(adapter, root, project, taskName)
		if gated.Sections != nil {
			writeContextSummary(&statusLog, gated.Sections, proj)
		}
		if gated.Sanitize.LinesRemoved > 0 || gated.Sanitize.BlankRemoved > 0 {
			statusLog.WriteString(fmt.Sprintf("[Sanitizer] Cleaned %d repeated line(s), %d blank-line run(s).\n", gated.Sanitize.LinesRemoved, gated.Sanitize.BlankRemoved))
		}
		if gated.CircuitBreaker.Message != "" {
			statusLog.WriteString("[Circuit Breaker] " + gated.CircuitBreaker.Message + "\n")
		}
		dryRunCfg, _ := core.ResolveEgressAudit(root, project, proj)
		door, model := "chat:provider", evidence.Attr{Value: sess.Provider + "/" + sess.Model, Source: "observed:sesión de Mova (Mova llama al proveedor)"}
		if proj.LLMProfile == nil || proj.LLMProfile.Config == "" {
			door, model = "mcp:chat_completion(host)", hostModelAttr(proj)
		}
		run, rerr := budget.RecordRun(root, project, taskName, proj, gated, budget.RunInfo{Door: door, Agent: agentAttr(), Model: model}, dryRunCfg)
		if rerr != nil {
			return "", fmt.Errorf("evidence: no se pudo escribir la evidencia, no se libera contexto: %w", rerr)
		}
		sess.Run, sess.ProjectName, sess.TaskName = run, project, taskName
		statusLog.WriteString("[Evidence] run " + run.ID + "\n")
		if gated.Err != nil {
			return "", gated.Err
		}

		systemText, boundary := gated.Text, 0
		if cfg := core.ResolveBudget(proj, budget.ResolveTask(proj, taskName)); core.CacheGuardEnabled(cfg) {
			modelHint := ""
			if proj.LLMProfile != nil {
				modelHint = proj.LLMProfile.Config
			}
			layout := budget.LayoutForCache(gated.Sections, modelHint)
			systemText, boundary = layout.Text, layout.StaticBoundary
			statusLog.WriteString(fmt.Sprintf("[Cache] Static prefix: %d tokens (fingerprint %s).\n", layout.StaticTokens, layout.Hash))
		}
		sess.SetSystem(systemText + ToolsSystemPrompt(proj.Tools))
		sess.CacheBoundary = boundary
		sess.EgressAuditDryRun = dryRunCfg
		if core.ToolsEnabled(proj.Tools) {
			statusLog.WriteString("[Tools] Enabled for this call — the model may create/write files and directories (see project.json's \"tools\").\n")
		}
		if gated.Sections != nil && gated.Sections.GraphStatus != "" {
			statusLog.WriteString(gated.Sections.GraphStatus + "\n")
		}
		if gated.Sections != nil && gated.Sections.DebugLog != "" {
			statusLog.WriteString(gated.Sections.DebugLog)
		}

		// Air-gap (egress_audit.dry_run) — the ONLY condition that
		// activates it; there is no separate "enabled" switch (see
		// models/egress_audit.go's package comment). Checked here,
		// before ANY of the tool-calling/NL-edit machinery below, so
		// dry_run blocks the ENTIRE turn — not just the plain chat
		// reply path — and nothing this project's context touches can
		// leave through a side door (file edits, deletes, natural-
		// language intents all need a model call too, and none of
		// them run once this returns).
		if sess.EgressAuditDryRun {
			gateResult, gerr := models.EgressGate(true, sess.System, sess.Model)
			if gerr != nil {
				return "", gerr
			}
			return statusLog.String() + gateResult.Message, nil
		}

		// Host-delegated inference (no llm_profile): Mova never calls a
		// local/cloud provider on its own — the governed, sanitized
		// context is returned in the tool result so the MCP HOST
		// (Cursor, Claude Code, Grok, whichever client invoked this
		// tool) can run inference itself. Same tool-calling/NL-edit
		// machinery below is skipped for the same reason as the
		// dry_run branch above: it all requires a model Mova isn't
		// configured to call.
		if proj.LLMProfile == nil || proj.LLMProfile.Config == "" {
			// Also an egress event: the governed context leaves Mova in the
			// tool result for the MCP host. Its evidence is the run
			// recorded above (door "mcp:chat_completion(host)").
			hint := ""
			if core.MemoryEnabled(proj) { // Mova no ve la respuesta del anfitrión: que registre él su síntesis
				hint = "\n\n---\n[Memory] memory está activo en este proyecto: al terminar, llama a la herramienta save_memory con project=\"" + project + "\", task=\"" + taskName + "\" y tu bloque ```memory como entry, para que las demás tareas lo lean."
			}
			return statusLog.String() + i18n.T("reports.egress_delegated_header") + "\n\n" + sess.System + "\n\n---\n" + message + hint, nil
		}
	}

	if history, ok := args["history"].([]any); ok {
		for _, h := range history {
			raw, _ := json.Marshal(h)
			var msg models.ChatMessage
			if json.Unmarshal(raw, &msg) == nil && msg.Role != "" {
				sess.History = append(sess.History, msg)
			}
		}
	}

	label := providerLabelMCP(sess.Provider)
	// Cambios pendientes de una llamada anterior y «Sí» a un prompt (Sí/No):
	// ver chat_tool_apply.go (mismo flujo que el chat, mova.local/applyflow).
	nlMessage := message
	if txt, handled := remoteApplyAnswer(root, project, proj, args, message); handled {
		return statusLog.String() + txt, nil
	}
	if rq, ok := reformatForApply(&statusLog, sess, proj, savedTask, message); ok {
		message, nlMessage = rq, ""
	}
	if applyReply, handled := applyAutoApplyConfirmationMCP(&statusLog, adapter, root, proj, project, sess, message); handled {
		writeTokenUsage(&statusLog, root, sess, proj)
		if project != "" && proj != nil {
			recordRealUsageMCP(root, project, proj, sess)
		}
		return statusLog.String() + applyReply, nil
	}
	// Order matters: DELETE and READ are checked before EDIT — a real
	// bug found in QA had "elimina el archivo X" going through the
	// edit flow (which just emptied the file's content instead of
	// removing it), because delete verbs used to also match
	// editVerbRe. See nl_delete.go/nl_read.go.
	if deleteReply, handled := applyNaturalLanguageDelete(&statusLog, root, proj, nlMessage, boolArg(args, "apply_delete")); handled {
		writeTokenUsage(&statusLog, root, sess, proj)
		return statusLog.String() + deleteReply, nil
	}
	if renameReply, handled := applyNaturalLanguageRename(&statusLog, root, proj, nlMessage, boolArg(args, "apply_rename")); handled {
		writeTokenUsage(&statusLog, root, sess, proj)
		return statusLog.String() + renameReply, nil
	}
	if readReply, handled := applyNaturalLanguageRead(root, proj, nlMessage); handled {
		return statusLog.String() + readReply, nil
	}
	if editReply, handled := applyNaturalLanguageEdits(&statusLog, sess, root, proj, nlMessage, boolArg(args, "apply_edits")); handled {
		writeTokenUsage(&statusLog, root, sess, proj)
		if project != "" && proj != nil {
			recordRealUsageMCP(root, project, proj, sess)
		}
		return statusLog.String() + documents.AutoTagCodeFences(editReply), nil
	}

	nlIntent := applyNaturalLanguageDirectories(&statusLog, root, proj, nlMessage)
	var reply string
	if len(nlIntent.Files) > 0 {
		// File-creation intent detected directly in the person's own
		// message: handled entirely here (own forced apply_file_changes
		// instruction, see applyNaturalLanguageFilesViaChanges), NOT
		// through the ordinary sendWithToolsMCP turn below — mirrors
		// cli/nl_save.go's handleNaturalLanguageSave, which also fully
		// takes over instead of falling through to a normal chat turn.
		reply, err = applyNaturalLanguageFilesViaChanges(&statusLog, sess, adapter, root, nlIntent, message)
		if err != nil {
			return "", err
		}
	} else {
		statusLog.WriteString(i18n.T("chat.sending_request", map[string]any{"label": label}) + "\n")
		reply, err = sendWithToolsMCP(&statusLog, sess, adapter, proj, root, message)
		if err != nil {
			return "", err
		}
		if sess.LastReplyWasDryRun {
			// The provider was never called — say so instead of the
			// generic "Response received.", which would otherwise
			// falsely claim a real network round-trip happened (see
			// project.json's "egress_audit": {"dry_run": true}).
			statusLog.WriteString("[egress_audit] " + i18n.T("reports.egress_dry_run_done") + "\n")
		} else {
			statusLog.WriteString(i18n.T("chat.response_received", map[string]any{"label": label}) + "\n")
		}
	}
	if sess.LastTruncated {
		statusLog.WriteString(i18n.T("chat.reply_truncated") + "\n")
	}
	writeTokenUsage(&statusLog, root, sess, proj)
	statusLog.WriteString("\n")

	if project != "" && proj != nil {
		recordRealUsageMCP(root, project, proj, sess)
		if !sess.LastReplyWasDryRun {
			// Misma regla que el chat: solo si project.json tiene "memory"
			// activo. Cada llamada MCP/HTTP es una sesión nueva; memory.md
			// es lo que une una tarea con la siguiente.
			res, merr := core.RecordMemory(adapter, root, project, savedTask, reply, core.RecordOptions{Source: "observed", RunID: runIDOf(sess)})
			switch {
			case merr != nil:
				statusLog.WriteString("[Memory] no se pudo guardar: " + merr.Error() + "\n")
			case res.Message() != "":
				statusLog.WriteString(res.Message() + "\n")
			}
		}
	}

	return statusLog.String() + documents.AutoTagCodeFences(reply) + offerApply(root, project, savedTask, proj, sess, reply), nil
}
