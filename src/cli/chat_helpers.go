// chat_helpers.go — provider resolution, tool-calling loop, and
// token-usage/context-summary plumbing for `mova chat` (see
// chat_cmd.go). Split into its own file so no single file in cli/ grows
// past 300 lines.
package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"mova.local/budget"
	"mova.local/core"
	"mova.local/mcp"
	"mova.local/models"
	"mova.local/sanitize"
)

// Estilos de Lip Gloss para formatear la salida en la terminal.
var (
	codeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			PaddingLeft(2)

	textStyle = lipgloss.NewStyle()
)

// Regex para detectar si toda la respuesta viene envuelta en un bloque de código markdown.
// Se usa concatenación para evitar romper las comillas invertidas en Go.
var outerCodeBlockRegex = regexp.MustCompile(`(?s)^` + "`{3}" + `[a-zA-Z0-9_-]*\r?\n?(.*?)\r?\n?` + "`{3}" + `$`)

// formatTerminalOutput limpia artefactos, quita las etiquetas ```lenguaje
// y aplica resaltado con Lip Gloss si el contenido es un bloque de código.
func formatTerminalOutput(rawText string) string {
	// 1. Limpieza estándar del protocolo MCP / mova
	text := mcp.StripResidualToolArtifacts(rawText)
	text = strings.TrimSpace(text)

	// 2. Si el modelo envolvió la respuesta completa en ```lenguaje ... ```
	if matches := outerCodeBlockRegex.FindStringSubmatch(text); len(matches) > 1 {
		codeContent := strings.TrimSpace(matches[1])
		// Renderizamos el código con color usando Lip Gloss sin mostrar las marcas ```
		return codeStyle.Render(codeContent)
	}

	// 3. Si es texto plano o mixto, aplicamos estilo de texto
	return textStyle.Render(text)
}

// applyProjectLLMProfile makes project.json's "llm_profile" the single
// source of truth. "provider" is optional: when absent, it is resolved
// from "config" alone (see models.ResolveConfigProvider) — the provider
// folder that owns that config file, whose own "type" field is what
// actually decides provider behavior. This is what lets llm_profile
// unify everything under "config" + the target file's "type", instead
// of repeating a provider name in project.json too.
func applyProjectLLMProfile(sess *models.Session, root string, proj *core.Project) {
	if proj == nil || proj.LLMProfile == nil || proj.LLMProfile.Config == "" {
		// Host-delegated inference (see PROJECT_JSON.md § llm_profile):
		// Mova never silently talks to whatever the global default
		// provider happens to be for a project that declares no
		// llm_profile of its own. In an interactive REPL there's no
		// MCP host to delegate to — a human is typing directly — so
		// the honest move is a visible notice, not a silent fallback,
		// so nobody discovers only after the fact which model actually
		// answered.
		if proj != nil {
			consolePrint(fmt.Sprintf("[Project] No llm_profile configured — using the global active model: %s/%s. Set \"llm_profile\" in project.json to pin one.\n", sess.Provider, sess.Model))
		}
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
	fallback := totalSessionTokens(sess, proj)
	consolePrint(models.UsageFor(sess, mc, fallback).FormatLine())
}

// totalSessionTokens calcula los tokens exactos acumulados en la sesión:
// System prompt + todo el historial de la conversación en sess.History.
func totalSessionTokens(sess *models.Session, proj *core.Project) int {
	if sess == nil {
		return 0
	}
	total := tokensOf(sess.System, proj)
	for _, msg := range sess.History {
		total += tokensOf(msg.Content, proj)
	}
	return total
}

// tokensOf is a tiny local estimate helper for the pre-send budget gate.
func tokensOf(text string, proj *core.Project) int {
	if text == "" {
		return 0
	}
	modelHint := ""
	if proj != nil && proj.LLMProfile != nil {
		modelHint = proj.LLMProfile.Config
	}
	n, _, _ := budget.CountTokens(text, modelHint)
	return n
}

// printContextSummary prints the [Dedup]/[Focus] status lines. proj is
// used only to resolve "focus_display_limit" (core.FocusDisplayLimit) —
// pass nil for the built-in default of 2.
func printContextSummary(sections *core.ContextSections, proj *core.Project) {
	if sections.DuplicatesRemoved > 0 {
		approxTokens := sections.DuplicatesRemovedChars / 4
		if approxTokens == 0 && sections.DuplicatesRemovedChars > 0 {
			approxTokens = 1
		}
		consolePrint(fmt.Sprintf("[Dedup] Removed %d duplicated paragraph(s) (~%d tokens saved).\n", sections.DuplicatesRemoved, approxTokens))
	}
	if line := core.FormatFocusSelection(sections.FocusItems, core.FocusDisplayLimit(proj)); line != "" {
		consolePrint(line)
	}
	if sections.GraphStatus != "" {
		consolePrint(sections.GraphStatus + "\n")
	}
}

// recordRealUsage closes the Feedback Loop.
func recordRealUsage(root, project string, proj *core.Project, sess *models.Session) {
	if project == "" || proj == nil || sess.LastUsage.PromptTokens <= 0 {
		return
	}
	localEstimate := totalSessionTokens(sess, proj)
	if localEstimate <= 0 {
		return
	}
	path := budget.HistoryPath(root, project, proj)
	_ = budget.RecordUsage(path, sess.Provider, localEstimate, sess.LastUsage.PromptTokens)

	// Circuit Breaker spend tracking (see budget/spend.go) — records
	// this real call's tokens/USD toward the project's monthly total,
	// regardless of which provider answered (Claude, GPT, Gemini, a
	// local Ollama model...). An unpriced provider/model still records
	// tokens with $0 — the per-run token gate still works even without
	// pricing data, only the monthly USD gate needs it.
	totalTokens := sess.LastUsage.PromptTokens + sess.LastUsage.CompletionTokens
	usd := 0.0
	if prices, err := budget.LoadPrices(root); err == nil {
		if cost, ok := budget.EstimateCostFor(totalTokens, sess.Provider, sess.Model, prices); ok {
			usd = cost
		}
	}
	_ = budget.RecordSpend(budget.SpendPath(root, project), totalTokens, usd)
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

// ── Context Governance display/layout helpers — shared by CLI chat
// (chat_cmd.go) and the TUI chat screen (tui_chat.go). Provider-agnostic
// on purpose: the Sanitizer and Circuit Breaker stages run identically
// no matter which model answers (Claude, GPT, Gemini, a local Ollama
// model...); only the Cache Layout Guard's cache_control marker is
// Anthropic-specific (see models/provider_anthropic.go), and every other
// provider simply ignores the unused CacheBoundary field. ──────────────

// printSanitizeStatus shows a one-line summary of what the Sanitizer
// stage removed (see mova.local/sanitize) — silent when it removed
// nothing.
func printSanitizeStatus(stats sanitize.Stats) {
	if stats.LinesRemoved == 0 && stats.BlankRemoved == 0 && stats.CommentsRemoved == 0 {
		return
	}
	consolePrint(fmt.Sprintf("[Sanitizer] Cleaned %d repeated line(s), %d blank-line run(s).\n", stats.LinesRemoved, stats.BlankRemoved))
}

// printCircuitBreakerStatus shows the spend-governance gate's result —
// silent when the project never configured a ceiling.
func printCircuitBreakerStatus(cb budget.CircuitBreakerResult) {
	if !cb.Checked || cb.Message == "" {
		return
	}
	consolePrint("[Circuit Breaker] " + cb.Message + "\n")
}

// applyCacheLayout reorders the assembled context into a cache-aware
// static-prefix + dynamic-tail layout (see budget.LayoutForCache) when
// the project has "budget": {"cache_hint": true}, printing a status
// line — used by the plain-terminal CLI chat (chat_cmd.go), which is
// free to write straight to stdout mid-conversation.
func applyCacheLayout(sections *core.ContextSections, proj *core.Project) (text string, boundary int) {
	text, boundary, layout := applyCacheLayoutQuiet(sections, proj)
	if layout != nil {
		consolePrint(fmt.Sprintf("[Cache] Static prefix: %d tokens (fingerprint %s) — see mova-budget-report.md for details.\n", layout.StaticTokens, layout.Hash))
	}
	return text, boundary
}

// applyCacheLayoutQuiet is the same computation with no printing at
// all — used by the TUI's chat screen (tui_chat.go), where writing
// straight to stdout mid-render would corrupt the Bubble Tea screen
// (see sendWithTools' own doc comment for the same concern). Returns
// the layout too, in case a caller wants to display it through its own
// (non-stdout) rendering instead.
func applyCacheLayoutQuiet(sections *core.ContextSections, proj *core.Project) (text string, boundary int, layout *budget.CacheLayout) {
	if sections == nil || proj == nil {
		return "", 0, nil
	}
	cfg := core.ResolveBudget(proj, budget.ResolveTask(proj, ""))
	if !core.CacheGuardEnabled(cfg) {
		return sections.Full(), 0, nil
	}
	l := budget.LayoutForCache(sections, modelHintOfProj(proj))
	return l.Text, l.StaticBoundary, &l
}

func modelHintOfProj(proj *core.Project) string {
	if proj != nil && proj.LLMProfile != nil {
		return proj.LLMProfile.Config
	}
	return ""
}
