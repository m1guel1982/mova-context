// agent_tools.go — lets `mova chat` (cli/chat_cmd.go) and the MCP
// "chat_completion" tool (chat_tool.go) call a whitelisted subset of
// Mova's own file/document tools DURING the conversation — so asking the
// model in plain Spanish/English to "create a directory", "write the
// corrected checkout.html", "generate a PDF" actually happens, instead of
// the model just describing in text what it would do (which is what
// happened before: mova chat never executed anything, regardless of
// provider — see project.json's "tools": {"enabled": true}, core/types.go).
//
// Deliberately NOT wired through each provider's native function-calling
// API (OpenAI "tools", Anthropic "tools", Ollama "tools"): those differ
// per provider, and small local models (like llama3.2:3b) frequently
// don't support them reliably at all. Instead this uses ONE plain-text,
// provider-agnostic protocol that works identically for Ollama, Gemini,
// Claude, GPT, or anything else Mova talks to — same "simplicidad"
// principle as the rest of the project: one protocol, three doors
// (CLI/MCP/HTTP), same as core.BuildContext.
package mcp

import (
	"encoding/json"
	"fmt"
	"strings"

	"mova.local/core"
	"mova.local/documents"
)

// toolCallStart/End delimit a tool call inside a model's plain-text
// reply. Chosen instead of bare JSON-object detection because small
// local models often wrap JSON in explanations or markdown code fences —
// a literal, unambiguous marker is far easier to find reliably than "the
// first balanced brace that looks like a tool call".
const (
	toolCallStart = "<<<MOVA_TOOL_CALL>>>"
	toolCallEnd   = "<<<END_MOVA_TOOL_CALL>>>"
)

// MaxAgentToolTurns caps how many tool round-trips a single user message
// can trigger in one go, so a confused model can't loop forever.
const MaxAgentToolTurns = 4

// AgentToolNames — the whitelist of MCP tools a chat session may call
// mid-conversation. Deliberately excludes read-only/context tools
// (get_full_context, search_context, chat_completion itself, estimate_budget...) —
// those don't need this protocol, and memory/budget are already reachable
// through /memory and /budget inside `mova chat`.
func AgentToolNames() []string {
	return []string{
		"save",
		"read_file",
		"patch_file",
		"read_document_layer",
	}
}

// allowedSet resolves project.json's optional "tools.allow" whitelist
// against the built-in one — empty/absent "allow" means "every tool in
// AgentToolNames() is allowed", the common case.
func allowedSet(allow []string) map[string]bool {
	base := AgentToolNames()
	set := map[string]bool{}
	if len(allow) == 0 {
		for _, n := range base {
			set[n] = true
		}
		return set
	}
	baseSet := map[string]bool{}
	for _, n := range base {
		baseSet[n] = true
	}
	for _, n := range allow {
		if baseSet[n] {
			set[n] = true
		}
	}
	return set
}

// ToolsSystemPrompt returns the block to append to a chat's system
// prompt when project.json's "tools.enabled" is true — describes the
// wire protocol and exactly which tools this session may call. Returns
// "" when tools are disabled (or the allow-list resolves to nothing),
// so callers can always just append the result unconditionally.
func ToolsSystemPrompt(cfg *core.ToolsConfig) string {
	if !core.ToolsEnabled(cfg) {
		return ""
	}
	set := allowedSet(cfg.Allow)
	var names []string
	for _, n := range AgentToolNames() {
		if set[n] {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n\n---\nHERRAMIENTAS DISPONIBLES (crear/escribir archivos y directorios)\n")
	b.WriteString("Podés pedirle a Mova que ejecute una acción real sobre el sistema de archivos del proyecto. Para eso, tu respuesta debe contener EXACTAMENTE este bloque, en cualquier punto de tu respuesta (no le agregues nada de texto adentro):\n\n")
	b.WriteString(toolCallStart + "\n")
	b.WriteString(`{"name": "<una_de_las_herramientas_de_abajo>", "arguments": {...}}` + "\n")
	b.WriteString(toolCallEnd + "\n\n")
	b.WriteString("Herramientas permitidas en esta sesión y sus argumentos:\n")
	for _, name := range names {
		b.WriteString("- " + name + ": " + argsHintFor(name) + "\n")
	}
	b.WriteString("\nDespués de emitir el bloque, dejá de escribir — Mova ejecuta la acción y te devuelve el resultado real como un nuevo turno para que sigas la respuesta con eso. Si no necesitás ninguna herramienta, respondé normalmente en texto, sin ningún bloque.\n---\n")
	return b.String()
}

// FileToolsHelp returns a human-readable list of every file/directory
// capability reachable from `mova chat` — both the deterministic /save
// slash command (always available once a project is loaded, regardless
// of "tools.enabled") and, when a project turns tools on, the tools the
// model itself may call autonomously. Used by the chat's `/tools`
// command — see cli/chat_cmd.go.
func FileToolsHelp() string {
	var b strings.Builder
	b.WriteString("Comandos de archivos/directorios en este chat:\n\n")
	b.WriteString("  /save \"ruta/archivo.ext\"     crea o edita un archivo con la ÚLTIMA respuesta del modelo; el formato (.md/.docx/.pdf/.xlsx/.svg/.py/...) se detecta por la extensión\n")
	b.WriteString("  /save -d \"ruta/carpeta\"       crea solo un directorio (y sus padres, si faltan)\n")
	b.WriteString("  /tools                       muestra esta ayuda\n\n")
	b.WriteString(fmt.Sprintf("Formatos soportados por /save: %s\n\n", strings.Join(documents.RegisteredExtensions(), ", ")))
	if len(AgentToolNames()) > 0 {
		b.WriteString("Si el proyecto tiene \"tools\": {\"enabled\": true} en su project.json, el propio modelo también puede invocar estas acciones dentro de su respuesta:\n")
		for _, name := range AgentToolNames() {
			b.WriteString("- " + name + ": " + argsHintFor(name) + "\n")
		}
	}
	return b.String()
}

func argsHintFor(name string) string {
	switch name {
	case "save":
		return `{"path": "carpeta/archivo.ext", "content": "..."} — el formato (.md/.txt/.docx/.pdf/.xlsx/.svg/.py/.json/...) se elige SOLO por la extensión de "path", nunca lo elegís vos a mano. Para crear solo un directorio: {"directory": "carpeta/subcarpeta"} (sin "path" ni "content"). Si el archivo ya existe, por default se sobreescribe — agregá "append": true para sumar en vez de reemplazar.`
	case "read_file":
		return `{"filename": "ruta/archivo.ext"}`
	case "patch_file":
		return `{"filename": "ruta/archivo.ext", "search": "texto exacto único", "replace": "texto nuevo"}`
	case "read_document_layer":
		return `{"filename": "ruta/archivo.docx|.xlsx|.pdf"}`
	default:
		return "{}"
	}
}

// ParseAgentToolCall looks for the FIRST marker-delimited tool call in a
// model's reply. ok=false (no error) means the reply has none — the
// normal case for most turns, including every turn once the model has
// already produced its final answer.
func ParseAgentToolCall(reply string) (name string, arguments map[string]any, ok bool) {
	start := strings.Index(reply, toolCallStart)
	if start == -1 {
		return "", nil, false
	}
	rest := reply[start+len(toolCallStart):]
	end := strings.Index(rest, toolCallEnd)
	if end == -1 {
		return "", nil, false
	}
	raw := strings.TrimSpace(rest[:end])

	var call struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal([]byte(raw), &call); err != nil || call.Name == "" {
		return "", nil, false
	}
	return call.Name, call.Arguments, true
}

// RunAgentTool executes a single whitelisted tool call from inside a chat
// session — reuses the exact same documentTool dispatcher (and therefore
// the exact same file-resolution/safety rules) as the MCP tools/call
// transport, just without going through JSON-RPC. Safe to call even if
// the caller forgot to check core.ToolsEnabled first: refuses on its own
// when tools aren't enabled, or when the tool isn't in the allow-list.
func RunAgentTool(adapter core.Adapter, root, name string, arguments map[string]any, cfg *core.ToolsConfig) (string, error) {
	if !core.ToolsEnabled(cfg) {
		return "", fmt.Errorf(`tool-calling no está habilitado para este proyecto — agregá "tools": {"enabled": true} en project.json`)
	}
	set := allowedSet(cfg.Allow)
	if !set[name] {
		return "", fmt.Errorf("la herramienta %q no está permitida para este proyecto (ver project.json's \"tools.allow\", o la lista completa en AgentToolNames)", name)
	}
	return documentTool(adapter, root, name, arguments)
}

// RunFileTool executes a whitelisted file/document tool directly, WITHOUT
// requiring project.json's "tools.enabled" — used for slash commands the
// USER types explicitly inside `mova chat` (currently just /save; see
// cli/chat_cmd.go), as opposed to RunAgentTool's autonomous, model-
// triggered path (which stays gated behind "tools.enabled", since there
// nobody but the model decided to run it). A human typing `/save` is
// already the deliberate action /memory and /budget don't gate either.
// Still restricted to AgentToolNames() — never an arbitrary tool name.
func RunFileTool(adapter core.Adapter, root, name string, arguments map[string]any) (string, error) {
	set := allowedSet(nil)
	if !set[name] {
		return "", fmt.Errorf("la herramienta %q no es un tool de archivos/directorios reconocido", name)
	}
	return documentTool(adapter, root, name, arguments)
}
