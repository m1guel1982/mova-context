// chat_helpers.go — provider resolution, tool-calling loop, and
// token-usage/context-summary plumbing for `mova chat` (see
// chat_cmd.go). Split into its own file so no single file in cli/ grows
// past 300 lines.
package main

import (
	"fmt"
	"strings"

	"mova.local/budget"
	"mova.local/core"
	"mova.local/mcp"
	"mova.local/models"
)

// applyProjectLLMProfile makes project.json's "llm_profile" the single
// source of truth. "provider" is optional: when absent, it is resolved
// from "config" alone (see models.ResolveConfigProvider) — the provider
// folder that owns that config file, whose own "type" field is what
// actually decides provider behavior. This is what lets llm_profile
// unify everything under "config" + the target file's "type", instead
// of repeating a provider name in project.json too.
func applyProjectLLMProfile(sess *models.Session, root string, proj *core.Project) {
	if proj == nil || proj.LLMProfile == nil || proj.LLMProfile.Config == "" {
		return
	}
	provider := proj.LLMProfile.Provider
	if provider == "" {
		resolved, err := models.ResolveConfigProvider(root, proj.LLMProfile.Config)
		if err != nil {
			consolePrint(fmt.Sprintf("[Project] Warning: could not resolve provider for config %q: %s\n", proj.LLMProfile.Config, err.Error()))
			return
		}
		provider = resolved
	}
	if err := sess.SwitchProvider(provider, proj.LLMProfile.Config); err != nil {
		consolePrint(fmt.Sprintf("[Project] Warning: could not switch to configured provider %q: %s\n", provider, err.Error()))
		return
	}
	consolePrint(fmt.Sprintf("[Project] Using configured provider: %s (%s)\n", sess.Provider, sess.Model))
}

// sendWithTools handles streaming or buffered tool loops.
func sendWithTools(sess *models.Session, adapter core.Adapter, proj *core.Project, root, userText string) (reply string, streamed bool, err error) {
	if proj == nil || adapter == nil || !core.ToolsEnabled(proj.Tools) {
		printedPrefix := false
		reply, err = sess.SendStream(userText, func(tok string) {
			if !printedPrefix {
				consolePrint("[" + sess.Model + "] ")
				printedPrefix = true
			}
			consolePrint(tok)
		})
		if err == nil {
			consolePrint("\n")
		}
		return reply, true, err
	}

	reply, err = sess.Send(userText)
	if err != nil {
		return "", false, err
	}
	for i := 0; i < mcp.MaxAgentToolTurns; i++ {
		name, args, ok := mcp.ParseAgentToolCall(reply)
		if !ok {
			break
		}
		consolePrint(fmt.Sprintf("[Tool] %s %v\n", name, args))
		result, terr := mcp.RunAgentTool(adapter, root, name, args, proj.Tools)
		if terr != nil {
			result = "ERROR: " + terr.Error()
		}
		consolePrint("[Tool] " + result + "\n")
		reply, err = sess.Send(fmt.Sprintf(
			"TOOL_RESULT(%s): %s\n\nContinue the reply for the user using this real result. If you need another tool, emit another block; if you are done, answer in plain text.",
			name, result))
		if err != nil {
			return "", false, err
		}
	}
	return reply, false, nil
}

// printTokenUsage shows, after every response, how many tokens the
// request used and the active model's maximum context window — the
// same minimal usage line MCP/HTTP print (see mcp/chat_tool.go), backed
// by mova.local/models.UsageFor. Purely informational: never blocks
// anything (that's mova.local/budget.EnforceLimit's job, run before the
// request was even sent).
func printTokenUsage(root string, sess *models.Session, proj *core.Project) {
	mc, err := models.DefaultCache.GetModel(root, sess.Provider, sess.Model)
	if err != nil {
		return
	}
	fallback := tokensOf(sess.System, proj)
	consolePrint(models.UsageFor(sess, mc, fallback).FormatLine())
}

// tokensOf is a tiny local estimate helper for the pre-send budget gate.
func tokensOf(text string, proj *core.Project) int {
	modelHint := ""
	if proj != nil && proj.LLMProfile != nil {
		modelHint = proj.LLMProfile.Config
	}
	n, _, _ := budget.CountTokens(text, modelHint)
	return n
}

// printContextSummary prints the [Dedup]/[Focus] status lines.
 
func printContextSummary(sections *core.ContextSections) {
	if sections.DuplicatesRemoved > 0 {
		approxTokens := sections.DuplicatesRemovedChars / 4
		if approxTokens == 0 && sections.DuplicatesRemovedChars > 0 {
			approxTokens = 1
		}
		consolePrint(fmt.Sprintf("[Dedup] Removed %d duplicated paragraph(s) (~%d tokens saved).\n", sections.DuplicatesRemoved, approxTokens))
	}
	if sections.Focus != "" {
		fileCount := strings.Count(sections.Focus, "FOCUS:")
		consolePrint(fmt.Sprintf("[Focus] Selected %d file(s).\n", fileCount))
	}
}

// recordRealUsage closes the Feedback Loop.
func recordRealUsage(root, project string, proj *core.Project, sess *models.Session) {
	if project == "" || proj == nil || sess.LastUsage.PromptTokens <= 0 {
		return
	}
	localEstimate := tokensOf(sess.System, proj)
	if localEstimate <= 0 {
		return
	}
	path := budget.HistoryPath(root, project, proj)
	_ = budget.RecordUsage(path, sess.Provider, localEstimate, sess.LastUsage.PromptTokens)
}

// providerLabel formats provider names for display.
func providerLabel(provider string) string {
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
