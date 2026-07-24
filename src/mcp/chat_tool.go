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
		sections, err := core.BuildContextSections(adapter, root, project, taskName)
		if err != nil {
			return "", fmt.Errorf("building project context: %w", err)
		}
		writeContextSummary(&statusLog, sections)

		if taskName == "" {
			taskName = proj.DefaultTask
		}
		if t, ok := proj.Tasks[taskName]; ok {
			if gateErr := budget.EnforceLimit(proj, &t, tokensOfText(sections.Full(), proj)); gateErr != nil {
				return "", gateErr
			}
		}
		sess.SetSystem(sections.Full() + ToolsSystemPrompt(proj.Tools))
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
		return statusLog.String() + editReply, nil
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

	return statusLog.String() + reply, nil
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
	return reply, nil
}

// writeContextSummary mirrors cli/chat_cmd.go's printContextSummary,
// writing into the tool's returned text instead of the console.
func writeContextSummary(b *strings.Builder, sections *core.ContextSections) {
	if sections.DuplicatesRemoved > 0 {
		approxTokens := sections.DuplicatesRemovedChars / 4
		b.WriteString(fmt.Sprintf("[Dedup] Removed %d duplicated paragraph(s) (~%d tokens saved).\n", sections.DuplicatesRemoved, approxTokens))
	}
	if sections.Focus != "" {
		fileCount := strings.Count(sections.Focus, "FOCUS:")
		b.WriteString(fmt.Sprintf("[Focus] Selected %d file(s).\n", fileCount))
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