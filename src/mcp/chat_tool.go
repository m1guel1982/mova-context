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
	"strings"

	"mova.local/budget"
	"mova.local/core"
	"mova.local/documents"
	"mova.local/models"
)

func chatCompletionTool(adapter core.Adapter, root string, args map[string]any) (string, error) {
	message := str(args, "message")
	if message == "" {
		return "", fmt.Errorf("message is required")
	}

	sess, err := models.NewSession(root)
	if err != nil {
		return "", err
	}

	var statusLog strings.Builder
	project := str(args, "project")
	var proj *core.Project

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
		taskName := str(args, "task")
		// budget.BuildGatedContext runs the full Token Firewall
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
		if core.ToolsEnabled(proj.Tools) {
			statusLog.WriteString("[Tools] Enabled for this call — the model may create/write files and directories (see project.json's \"tools\").\n")
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
	if editReply, handled := applyNaturalLanguageEdits(&statusLog, sess, root, proj, message, boolArg(args, "apply_edits")); handled {
		writeTokenUsage(&statusLog, root, sess, proj)
		if project != "" && proj != nil {
			recordRealUsageMCP(root, project, proj, sess)
		}
		return statusLog.String() + documents.AutoTagCodeFences(editReply), nil
	}

	nlIntent := applyNaturalLanguageDirectories(&statusLog, root, proj, message)
	statusLog.WriteString("[" + label + "] Sending request...\n")
	reply, err := sendWithToolsMCP(&statusLog, sess, adapter, proj, root, message)
	if err != nil {
		return "", err
	}
	statusLog.WriteString("[" + label + "] Response received.\n")
	applyNaturalLanguageFiles(&statusLog, root, proj, nlIntent, message, reply)
	writeTokenUsage(&statusLog, root, sess, proj)
	statusLog.WriteString("\n")

	if project != "" && proj != nil {
		recordRealUsageMCP(root, project, proj, sess)
	}

	return statusLog.String() + documents.AutoTagCodeFences(reply), nil
}

// writeTokenUsage mirrors cli/chat_cmd.go's printTokenUsage for the
// MCP/HTTP door: shows how many tokens the request used and the active
// model's maximum context window (see mova.local/models.UsageFor). HTTP
// gets this for free — http/server.go is a thin wrapper over this same
// tool (see server.go's doc comment).
func writeTokenUsage(statusLog *strings.Builder, root string, sess *models.Session, proj *core.Project) {
	mc, err := models.DefaultCache.GetModel(root, sess.Provider, sess.Model)
	if err != nil {
		return
	}
	fallback := tokensOfText(sess.System, proj)
	statusLog.WriteString(models.UsageFor(sess, mc, fallback).FormatLine())
}

// tokensOfText mirrors cli/chat_cmd.go's tokensOf — kept as its own small
// copy here (not exported from cli, which can't be imported from mcp)
// rather than adding a new shared package for a three-line estimate call.
func tokensOfText(text string, proj *core.Project) int {
	modelHint := ""
	if proj != nil && proj.LLMProfile != nil {
		modelHint = proj.LLMProfile.Config
	}
	n, _, _ := budget.CountTokens(text, modelHint)
	return n
}

// sendWithToolsMCP mirrors cli/chat_cmd.go's sendWithTools for the MCP/HTTP
// door: same opt-in tool-calling loop (project.json's "tools"), same
// marker protocol (mcp.ParseAgentToolCall/RunAgentTool), just logging into
// statusLog (returned as part of the tool's text result) instead of the
// console. Kept as its own small copy — same reasoning tokensOfText's
// comment already gives — rather than a shared package for one loop.
func sendWithToolsMCP(statusLog *strings.Builder, sess *models.Session, adapter core.Adapter, proj *core.Project, root, userText string) (string, error) {
	reply, err := sess.Send(userText)
	if err != nil {
		return "", err
	}
	if proj == nil || adapter == nil || !core.ToolsEnabled(proj.Tools) {
		return reply, nil
	}
	for i := 0; i < MaxAgentToolTurns; i++ {
		name, args, ok := ParseAgentToolCall(reply)
		if !ok {
			break
		}
		statusLog.WriteString(fmt.Sprintf("[Tool] %s %v\n", name, args))
		result, terr := RunAgentTool(adapter, root, name, args, proj.Tools)
		if terr != nil {
			result = "ERROR: " + terr.Error()
		}
		statusLog.WriteString("[Tool] " + result + "\n")
		reply, err = sess.Send(fmt.Sprintf(
			"TOOL_RESULT(%s): %s\n\nContinue the reply for the user using this real result. If you need another tool, emit another block; if you are done, answer in plain text.",
			name, result))
		if err != nil {
			return "", err
		}
	}
	// Same cleanup cli/chat_helpers.go's sendWithTools applies — see
	// StripResidualToolArtifacts' doc comment for why this is needed and
	// why it's safe. HTTP gets this for free (http/server.go is a thin
	// wrapper over this same function via mcp.Process).
	return StripResidualToolArtifacts(reply), nil
}

// writeContextSummary mirrors cli/chat_cmd.go's printContextSummary,
// writing into the tool's returned text instead of the console. proj is
// used only to resolve "focus_display_limit" (core.FocusDisplayLimit) —
// pass nil for the built-in default of 2.
func writeContextSummary(b *strings.Builder, sections *core.ContextSections, proj *core.Project) {
	if sections.DuplicatesRemoved > 0 {
		approxTokens := sections.DuplicatesRemovedChars / 4
		b.WriteString(fmt.Sprintf("[Dedup] Removed %d duplicated paragraph(s) (~%d tokens saved).\n", sections.DuplicatesRemoved, approxTokens))
	}
	if line := core.FormatFocusSelection(sections.FocusItems, core.FocusDisplayLimit(proj)); line != "" {
		b.WriteString(line)
	}
}

// recordRealUsageMCP mirrors cli/chat_cmd.go's recordRealUsage — see that
// file's doc comment for what is (and, importantly, is never) stored.
func recordRealUsageMCP(root, project string, proj *core.Project, sess *models.Session) {
	if sess.LastUsage.PromptTokens <= 0 {
		return
	}
	localEstimate := tokensOfText(sess.System, proj)
	if localEstimate <= 0 {
		return
	}
	path := budget.HistoryPath(root, project, proj)
	_ = budget.RecordUsage(path, sess.Provider, localEstimate, sess.LastUsage.PromptTokens)

	if project == "" || proj == nil {
		return
	}
	totalTokens := sess.LastUsage.PromptTokens + sess.LastUsage.CompletionTokens
	usd := 0.0
	if prices, err := budget.LoadPrices(root); err == nil {
		if cost, ok := budget.EstimateCostFor(totalTokens, sess.Provider, sess.Model, prices); ok {
			usd = cost
		}
	}
	_ = budget.RecordSpend(budget.SpendPath(root, project), totalTokens, usd)
}

func providerLabelMCP(provider string) string {
	switch provider {
	case "anthropic":
		return "Claude"
	case "google":
		return "Gemini"
	case "openai":
		return "OpenAI"
	case "ollama":
		return "Ollama"
	default:
		if provider == "" {
			return "Model"
		}
		return strings.ToUpper(provider[:1]) + provider[1:]
	}
}
