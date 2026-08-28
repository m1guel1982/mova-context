// chat_helpers.go — provider resolution, tool-calling loop, and
// token-usage/context-summary plumbing for `mova chat` (see
// chat_cmd.go). Split into its own file so no single file in cli/ grows
// past 300 lines.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// sendWithTools handles streaming or buffered tool loops. emit receives
// every piece of output this function would otherwise print directly
// (token chunks, "[Tool] ..." trace lines) — pass nil to get the exact
// original behavior (writes straight to the terminal via consolePrint,
// as `mova chat`'s REPL always has). The TUI (see tui_chat.go) passes
// its own emit that appends to an in-memory transcript instead, since
// writing to stdout mid-render would corrupt a Bubble Tea screen; the
// model-calling/tool-loop logic itself is untouched either way — one
// implementation, two output sinks.
func sendWithTools(sess *models.Session, adapter core.Adapter, proj *core.Project, root, userText string, emit func(string)) (reply string, streamed bool, err error) {
	if emit == nil {
		emit = consolePrint
	}

	// Caso sin herramientas habilitadas / adapter nulo
	if proj == nil || adapter == nil || !core.ToolsEnabled(proj.Tools) {
		var rawReply strings.Builder

		// Interceptamos los tokens en buffer para evitar imprimir el prefijo [modelo]
		// y procesar el formateo limpio al finalizar la respuesta.
		reply, err = sess.SendStream(userText, func(tok string) {
			rawReply.WriteString(tok)
		})
		if err != nil {
			return "", true, err
		}

		// Aplicamos limpieza y estilo estilizado a la salida
		formattedReply := formatTerminalOutput(rawReply.String())
		emit(formattedReply + "\n")

		return formattedReply, true, nil
	}

	// Bucle normal de herramientas cuando sí están habilitadas
	reply, err = sess.Send(userText)
	if err != nil {
		return "", false, err
	}
	for i := 0; i < mcp.MaxAgentToolTurns; i++ {
		name, args, ok := mcp.ParseAgentToolCall(reply)
		if !ok {
			break
		}
		emit(fmt.Sprintf("[Tool] %s %v\n", name, args))
		result, terr := mcp.RunAgentTool(adapter, root, name, args, proj.Tools)
		if terr != nil {
			result = "ERROR: " + terr.Error()
		}
		emit("[Tool] " + result + "\n")
		reply, err = sess.Send(fmt.Sprintf(
			"TOOL_RESULT(%s): %s\n\nContinue the reply for the user using this real result. If you need another tool, emit another block; if you are done, answer in plain text.",
			name, result))
		if err != nil {
			return "", false, err
		}
	}

	formattedReply := formatTerminalOutput(reply)
	return formattedReply, false, nil
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

// ── Token Firewall display/layout helpers — shared by CLI chat
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

// ── Hot reload de project.json durante un chat ya abierto ───────────
//
// runChat (chat_cmd.go) y newChatScreen (tui_chat.go) arman el contexto
// UNA sola vez, al arrancar. Si mientras el chat sigue abierto alguien
// edita project.json (cambia "focus", agrega/saca archivos del repo,
// cambia "budget", etc.), ese cambio nunca se volvía a leer hasta
// reiniciar `mova chat`. refreshProjectContext cierra ese hueco:
// se llama antes de procesar cada mensaje, y si algo relevante cambió
// reconstruye el contexto completo (mismo pipeline que
// budget.BuildGatedContext: Sanitizer → PII → Circuit Breaker → Budget
// gate) y reemplaza el system prompt de la sesión en caliente. Esto
// también dispara de nuevo SanitizeCached (contextcache.go), que ya
// escribe mova-context-cache.json apenas detecta el hash nuevo — así
// que el cache queda al día sin necesidad de reiniciar el chat.

// projectSignature identifica el estado completo de project.json en un
// solo hash. Usamos el struct entero (no solo "focus") a propósito:
// cualquier cambio que afecte el contexto o el gate — focus, memory,
// agents, skills, budget, tools, llm_profile — tiene que disparar una
// reconstrucción; es más barato re-hashear el struct completo que
// mantener una lista de "campos que importan" sincronizada a mano cada
// vez que core.Project gane un campo nuevo.
func projectSignature(proj *core.Project) string {
	if proj == nil {
		return ""
	}
	data, err := json.Marshal(proj)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// refreshProjectContext re-lee project.json y, si su firma cambió desde
// la última vez que este chat la cargó, reconstruye el contexto gated
// y reemplaza sess.System/sess.CacheBoundary in place. Devuelve el
// (posiblemente nuevo) proj/adapter y la firma a comparar la próxima
// vez. emit sigue la misma convención que sendWithTools: nil imprime
// directo a la terminal (REPL de `mova chat`); la TUI pasa su propio
// emit que apenda al transcript en memoria.
//
// Si project.json quedó momentáneamente inválido (error de sintaxis
// mientras alguien lo edita a mano, guardado a medias, etc.) o el
// nuevo contexto no pasa el Budget/Circuit Breaker gate, se conserva el
// contexto/adapter/proj VIEJOS — todavía válidos — en vez de dejar la
// sesión sin system prompt.
func refreshProjectContext(root, project, task string, sess *models.Session, proj *core.Project, adapter core.Adapter, lastSignature string, emit func(string)) (*core.Project, core.Adapter, string) {
	if emit == nil {
		emit = consolePrint
	}
	if project == "" || sess == nil {
		return proj, adapter, lastSignature
	}

	fa := core.NewFileAdapter(root)
	freshProj, err := fa.GetProject(project)
	if err != nil {
		return proj, adapter, lastSignature
	}

	signature := projectSignature(freshProj)
	if signature == lastSignature {
		return proj, adapter, lastSignature // nada relevante cambió
	}

	freshAdapter := newAdapter(root, freshProj)
	applyProjectLLMProfile(sess, root, freshProj)

	gated := budget.BuildGatedContext(freshAdapter, root, project, task)
	if gated.Err != nil {
		emit("[Project] project.json cambió, pero el nuevo contexto no pasó el gate: " + gated.Err.Error() + "\n")
		return proj, adapter, lastSignature
	}

	systemText, boundary, _ := applyCacheLayoutQuiet(gated.Sections, freshProj)
	sess.SetSystem(systemText + mcp.ToolsSystemPrompt(freshProj.Tools))
	sess.CacheBoundary = boundary

	emit("[Project] project.json cambió — contexto recargado.\n")
	if gated.Sections != nil {
		if line := core.FormatFocusSelection(gated.Sections.FocusItems, core.FocusDisplayLimit(freshProj)); line != "" {
			emit(line)
		}
	}
	if gated.Sanitize.LinesRemoved > 0 || gated.Sanitize.BlankRemoved > 0 || gated.Sanitize.CommentsRemoved > 0 {
		emit(fmt.Sprintf("[Sanitizer] Cleaned %d repeated line(s), %d blank-line run(s).\n", gated.Sanitize.LinesRemoved, gated.Sanitize.BlankRemoved))
	}
	if gated.CircuitBreaker.Checked && gated.CircuitBreaker.Message != "" {
		emit("[Circuit Breaker] " + gated.CircuitBreaker.Message + "\n")
	}

	return freshProj, freshAdapter, signature
}
