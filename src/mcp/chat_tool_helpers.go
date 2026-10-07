// chat_tool_helpers.go — utilidades de chat_completion (uso de tokens,
// tool-calling, resumen de contexto, feedback loop). Movidas tal cual desde
// chat_tool.go para mantener cada archivo ≤300 líneas.
package mcp

import (
	"fmt"
	"strings"

	"mova.local/budget"
	"mova.local/core"
	"mova.local/models"
)

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
		var result string
		var terr error
		if name == "apply_file_changes" {
			// MCP/HTTP have no terminal for the interactive menu
			// cli/apply_file_changes.go shows — see
			// DescribePendingChanges' doc comment. Never auto-writes.
			if changes, ok := ParseApplyFileChanges(args); ok {
				result = DescribePendingChanges(changes)
			} else {
				terr = fmt.Errorf(`"changes" must be a non-empty array of {"action","path","content"}`)
			}
		} else {
			result, terr = RunLoopTool(adapter, root, sess, proj, name, args)
		}
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
