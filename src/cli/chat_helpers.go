// chat_helpers.go — provider resolution, tool-calling loop, and
// token-usage/context-summary plumbing for `mova chat` (see
// chat_cmd.go). Split into its own file so no single file in cli/ grows
// past 300 lines.
package main

import (
	"bufio"
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

// sendWithTools handles streaming or buffered tool loops. emit receives
// every piece of output this function would otherwise print directly
// (token chunks, "[Tool] ..." trace lines) — pass nil to get the exact
// original behavior (writes straight to the terminal via consolePrint,
// as `mova chat`'s REPL always has). The TUI (see tui_chat.go) passes
// its own emit that appends to an in-memory transcript instead, since
// writing to stdout mid-render would corrupt a Bubble Tea screen; the
// model-calling/tool-loop logic itself is untouched either way — one
// implementation, two output sinks.
func sendWithTools(sess *models.Session, adapter core.Adapter, proj *core.Project, root, userText string, readLine readLineFunc, emit func(string)) (reply string, streamed bool, err error) {
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
		var result string
		var terr error
		if name == "apply_file_changes" {
			if changes, ok := mcp.ParseApplyFileChanges(args); ok {
				result = confirmAndApplyFileChanges(adapter, root, changes, readLine, emit)
			} else {
				terr = fmt.Errorf(`"changes" must be a non-empty array of {"action","path","content"}`)
			}
		} else {
			result, terr = mcp.RunAgentTool(adapter, root, name, args, proj.Tools)
		}
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

// sendWithForcedFileChanges is the variant of sendWithTools used
// exclusively by nl_save.go's natural-language file-creation flow
// (handleNaturalLanguageSave) and its TUI equivalent
// (tui_chat.go's startNaturalLanguageSave). Unlike sendWithTools, it
// ALWAYS parses the reply for an apply_file_changes call — regardless
// of project.json's "tools": {"enabled": ...} — because this
// capability must work exactly like /save already does, independent of
// that project-wide flag.
//
// This fixes a real, reported bug: when tools.enabled was false (the
// default for a brand-new project), sendWithTools took its "no tools"
// branch above, so the apply_file_changes JSON the model produced (see
// mcp.BuildApplyFileChangesInstruction) was streamed back as plain text
// and NEVER parsed — the person saw the raw JSON payload printed to the
// terminal and no file was ever written, even though the model did
// exactly what it was told. Detecting "create hola.txt with X" is
// already a deterministic, pre-confirmed decision by the person (see
// documents.DetectSaveIntent) — it must not silently depend on a
// separate, unrelated opt-in flag meant for the model calling tools
// *on its own initiative* during ordinary conversation.
func sendWithForcedFileChanges(sess *models.Session, adapter core.Adapter, root, userText string, readLine readLineFunc, emit func(string)) (reply string, err error) {
	if emit == nil {
		emit = consolePrint
	}
	reply, err = sess.Send(userText)
	if err != nil {
		return "", err
	}
	for i := 0; i < mcp.MaxAgentToolTurns; i++ {
		name, args, ok := mcp.ParseAgentToolCall(reply)
		if !ok || name != "apply_file_changes" {
			break
		}
		emit(fmt.Sprintf("[Tool] %s %v\n", name, args))
		var result string
		if changes, ok := mcp.ParseApplyFileChanges(args); ok {
			result = confirmAndApplyFileChanges(adapter, root, changes, readLine, emit)
		} else {
			result = `ERROR: "changes" must be a non-empty array of {"action","path","content"}`
		}
		emit("[Tool] " + result + "\n")
		reply, err = sess.Send(fmt.Sprintf(
			"TOOL_RESULT(%s): %s\n\nContinue the reply for the user using this real result. If you need another tool, emit another block; if you are done, answer in plain text.",
			name, result))
		if err != nil {
			return "", err
		}
	}
	return formatTerminalOutput(reply), nil
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

// contextSignature identifica el estado completo que puede afectar el
// contexto/gate en un solo hash: el struct project.json completo (foco,
// memory, agents, skills, budget, tools, llm_profile — más barato
// re-hashear todo que mantener a mano una lista de "campos que
// importan") MÁS el contenido YA RESUELTO de agents/skills/prompt.
// Esa segunda parte es la que faltaba: sin ella, editar un archivo en
// agents/base/i18n/{en,es}, prompts/base/i18n/{en,es} o
// skills/base/i18n/{en,es} — sin tocar project.json — no cambiaba la
// firma y el hot-reload nunca se disparaba en chat/mova ui chat/CLI/
// HTTP API/MCP, aunque el archivo en disco ya fuera otro. sections
// puede ser nil (p.ej. si BuildContextSections falló) sin que la firma
// dependa solo de eso.
func contextSignature(proj *core.Project, sections *core.ContextSections) string {
	if proj == nil {
		return ""
	}
	data, err := json.Marshal(proj)
	if err != nil {
		return ""
	}
	h := sha256.New()
	h.Write(data)
	if sections != nil {
		h.Write([]byte(sections.Agents))
		h.Write([]byte(sections.Skills))
		h.Write([]byte(sections.Prompt))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// printDebugLog imprime ContextSections.DebugLog (ver core.Project.Debug
// y core/engine.go) — no hace nada cuando está vacío, que es el caso
// por defecto (debug:false). Único punto compartido por chat, mova ui
// chat y MCP/HTTP (mcp/chat_tool.go llama a esta misma función) para
// que las 5 puertas muestren exactamente la misma traza, nunca una
// versión distinta cada una.
func printDebugLog(sections *core.ContextSections, emit func(string)) {
	if sections == nil || sections.DebugLog == "" {
		return
	}
	if emit == nil {
		emit = consolePrint
	}
	emit(sections.DebugLog)
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

	// BuildContextSections en sí es barato (lee unos pocos .md) — se
	// llama SIEMPRE para poder detectar cambios de contenido en
	// agents/skills/prompt (ver contextSignature), y si la firma
	// resulta igual, este mismo `sections` simplemente se descarta sin
	// pasar por el resto del pipeline (Sanitizer/PII/tokenizer), que
	// es la parte realmente costosa.
	precheckSections, _ := core.BuildContextSections(fa, root, project, task)
	signature := contextSignature(freshProj, precheckSections)
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
	sess.EgressAuditDryRun, sess.EgressAuditOutputFile = core.ResolveEgressAudit(root, project, freshProj)

	emit("[Project] project.json cambió — contexto recargado.\n")
	printDebugLog(gated.Sections, emit)
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

// scannerReadLine adapts a *bufio.Scanner (the CLI REPL's blocking
// stdin reader) into a readLineFunc — the same abstraction
// confirmAndApplyFileChanges/sendWithForcedFileChanges already use, so
// runChatDelete/handleNaturalLanguageDelete/Rename/Edit (and their
// callers) can share ONE confirmation-flow implementation between
// `mova chat` (this adapter) and `mova ui chat` (tui_chat.go's own
// menuChan-backed readLineFunc) instead of keeping two copies of the
// same y/n logic that could quietly drift apart.
func scannerReadLine(scanner *bufio.Scanner) readLineFunc {
	return func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return strings.TrimSpace(scanner.Text()), true
	}
}
