// chat_refresh.go — hot reload del contexto durante un chat abierto.
// Movido desde chat_helpers.go (≤300 líneas por archivo).
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mova.local/evidence"
	"strings"

	"mova.local/budget"
	"mova.local/core"
	"mova.local/mcp"
	"mova.local/models"
)

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
		h.Write([]byte(sections.Memory)) // memory.md: si otra puerta la cambió, el chat recarga
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
	sess.EgressAuditDryRun, _ = core.ResolveEgressAudit(root, project, freshProj)
	// The released context changed: a new immutable run for it.
	if run, rerr := budget.RecordRun(root, project, task, freshProj, gated, budget.RunInfo{
		Door:  "chat:provider",
		Agent: evidence.Attr{Value: "mova-chat", Source: "observed:cli"},
		Model: evidence.Attr{Value: sess.Provider + "/" + sess.Model, Source: "observed:sesión de Mova (Mova llama al proveedor)"},
	}, sess.EgressAuditDryRun); rerr == nil {
		sess.Run, sess.ProjectName, sess.TaskName = run, project, task
		emit("[Evidence] run " + run.ID + "\n")
	}

	emit("[Project] contexto recargado (cambió project.json, un prompt/agent/skill o memory.md).\n")
	if gated.Sections != nil && gated.Sections.GraphStatus != "" {
		emit(gated.Sections.GraphStatus + "\n")
	}
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

// resignContext recalcula la firma del contexto DESPUÉS de que este mismo
// chat escribió en memory.md. Sin esto, la propia escritura cambiaría la
// firma y el siguiente turno reconstruiría el system prompt (perdiendo el
// caché del proveedor) para re-inyectar algo que la conversación ya
// contiene. Cambios hechos por OTRA puerta siguen detectándose.
func resignContext(root, project, task string, proj *core.Project) string {
	fa := core.NewFileAdapter(root)
	fresh, err := fa.GetProject(project)
	if err != nil {
		return ""
	}
	sections, err := core.BuildContextSections(fa, root, project, task)
	if err != nil {
		return ""
	}
	return contextSignature(fresh, sections)
}
