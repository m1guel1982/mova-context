// chat.go — sesión conversacional reutilizable desde el CLI (`mova chat`),
// MCP (tool chat_completion) y HTTP (mismo tool, vía /mcp). Una sola
// implementación, tres puertas — mismo principio que core.BuildContext.
package models

import (
	"context"
	"fmt"
)

// maxHistoryMessages acota cuántos turnos previos se reenvían al modelo.
// Es una salvaguarda simple (no un tokenizer real) para que sesiones
// largas no exploten num_ctx en modelos chicos; cada modelo igual define
// su propio num_ctx real en su .json.
const maxHistoryMessages = 40

// Session mantiene el estado de una conversación con un modelo. No cachea
// el ModelConfig ni el Provider resuelto: cada Send() los relee a través
// de DefaultCache (ver config.go), así que si alguien edita el .json del
// modelo a mano mientras el chat está abierto, el próximo mensaje ya usa
// los valores nuevos — incluida la conexión (base_url/api_key/timeout),
// que ahora vive en el mismo archivo que los parámetros de inferencia.
type Session struct {
	Root      string
	Provider  string
	Model     string // nombre de archivo (sin .json) del config activo
	System    string // system prompt fijo (p.ej. el contexto de mova run)
	History   []ChatMessage
	LastUsage Usage // tokens reales del último turno (0,0 si el proveedor no los reporta)

	// CacheBoundary: byte offset in System where a stable, cacheable
	// prefix ends (0 = disabled). Set by mova.local/budget's
	// LayoutForCache when a project has "budget": {"cache_hint": true}
	// — see cli/chat_helpers.go, cli/tui_chat.go. Threaded into the
	// system ChatMessage built below, read only by provider_anthropic.go.
	CacheBoundary int
}

// NewSession arranca una sesión usando el proveedor/modelo activo
// (config/models/active.json).
func NewSession(root string) (*Session, error) {
	state, err := GetActiveState(root)
	if err != nil {
		return nil, err
	}
	if state.Provider == "" {
		return nil, fmt.Errorf("no hay proveedor activo — corré `mova config <provider>` (p.ej. `mova config ollama`)")
	}
	s := &Session{Root: root, Provider: state.Provider}
	if state.Config != "" {
		_ = s.SetModel(state.Config) // best-effort: si ya no existe, se pide de nuevo
	}
	return s, nil
}

// SwitchProvider changes BOTH the provider and model of an active session
// — used so a project's own "llm_profile" (provider/config in
// project.json) becomes the single source of truth for which provider a
// project's chat uses, instead of always falling back to the global
// config/models/active.json default. See cli/chat_cmd.go and
// mcp/chat_tool.go for the call site (right after a project is loaded).
//
// This is a SESSION-LOCAL override: unlike SetModel (used by `set -model`
// inside the chat, which persists to the global active.json on purpose),
// SwitchProvider never touches the global default — a project's
// llm_profile shouldn't leak into other unrelated chats.
func (s *Session) SwitchProvider(provider, model string) error {
	s.Provider = provider
	s.Model = ""
	if model == "" {
		return nil
	}
	resolved, err := FindModelFile(s.Root, provider, model)
	if err != nil {
		return err
	}
	s.Model = resolved
	return nil
}

// SetModel cambia el modelo activo de la sesión sin perder el historial de
// conversación. Busca primero dentro del proveedor activo y, si no lo encuentra,
// escanea agnósticamente los otros proveedores para conmutar de proveedor
// en caliente sin tocar la configuración global (active.json).
func (s *Session) SetModel(name string) error {
	// 1. Intenta resolver en el proveedor activo de la sesión
	if err := s.SwitchProvider(s.Provider, name); err == nil {
		return nil
	}

	// 2. Si no existe en el proveedor actual, busca dinámicamente en otros proveedores
	providers := []string{"google", "ollama", "openai", "anthropic"}
	for _, p := range providers {
		if p == s.Provider {
			continue
		}
		if err := s.SwitchProvider(p, name); err == nil {
			return nil
		}
	}

	return fmt.Errorf("modelo %q no encontrado bajo config/models/ en ningún proveedor", name)
}

// SetSystem fija (o reemplaza) el mensaje de sistema — típicamente el
// contexto completo de `mova run [project] [task]`, opcionalmente con el
// protocolo de tool-calling de mova.local/mcp agregado atrás (ver
// cli/chat_cmd.go y mcp/agent_tools.go).
func (s *Session) SetSystem(text string) {
	s.System = text
}

// Send manda un mensaje del usuario y devuelve la respuesta del modelo.
// Relee la configuración del modelo en cada llamada (ver config.go:
// ConfigCache) — si alguien edita el .json a mano mientras el chat está
// abierto, el próximo mensaje ya usa los valores nuevos.
func (s *Session) Send(userText string) (string, error) {
	if s.Model == "" {
		return "", fmt.Errorf("no hay modelo activo — usá `set -model <nombre>` primero")
	}
	mc, err := DefaultCache.GetModel(s.Root, s.Provider, s.Model)
	if err != nil {
		return "", err
	}
	pv := NewProvider(mc)

	s.History = append(s.History, ChatMessage{Role: "user", Content: userText})
	trimHistory(&s.History)

	messages := make([]ChatMessage, 0, len(s.History)+1)
	if s.System != "" {
		messages = append(messages, ChatMessage{Role: "system", Content: s.System, CacheBoundary: s.CacheBoundary})
	}
	messages = append(messages, s.History...)

	ctx, cancel := context.WithTimeout(context.Background(), mc.Timeout())
	defer cancel()

	// Si el JSON (mc) trae un "model" explícito (el tag real, como
	// "llama3.2:3b" o "gemini-2.5-flash"), usamos ese. Si no viene (está
	// vacío), fallback al nombre del archivo (s.Model) — útil cuando el
	// nombre del archivo ya ES el tag real (sin caracteres inválidos en
	// Windows), como "gemini-2.5-flash" o "mistral".
	modelTag := s.Model
	if mc.ModelName != "" {
		modelTag = mc.ModelName
	}

	reply, usage, err := pv.Chat(ctx, modelTag, mc, messages)
	if err != nil {
		// no dejamos el turno del usuario "colgado" sin respuesta en el historial
		s.History = s.History[:len(s.History)-1]
		return "", err
	}
	s.LastUsage = usage
	s.History = append(s.History, ChatMessage{Role: "assistant", Content: reply})
	return reply, nil
}

// SendStream es igual que Send, pero si el proveedor activo implementa
// StreamProvider (hoy: solo Ollama), invoca onToken con cada fragmento
// de texto a medida que el modelo lo genera — útil para que `mova chat`
// muestre la respuesta apareciendo palabra por palabra en vez de una
// pantalla congelada hasta que termine (la latencia total no cambia,
// pero la percibida sí, sobre todo corriendo en CPU). Si el proveedor
// activo no soporta streaming, cae a Send() sin romper nada: onToken se
// llama una sola vez con la respuesta completa al final.
func (s *Session) SendStream(userText string, onToken func(string)) (string, error) {
	if s.Model == "" {
		return "", fmt.Errorf("no hay modelo activo — usá `set -model <nombre>` primero")
	}
	mc, err := DefaultCache.GetModel(s.Root, s.Provider, s.Model)
	if err != nil {
		return "", err
	}
	pv := NewProvider(mc)

	sp, ok := pv.(StreamProvider)
	if !ok {
		reply, err := s.Send(userText)
		if err == nil && onToken != nil {
			onToken(reply)
		}
		return reply, err
	}

	s.History = append(s.History, ChatMessage{Role: "user", Content: userText})
	trimHistory(&s.History)

	messages := make([]ChatMessage, 0, len(s.History)+1)
	if s.System != "" {
		messages = append(messages, ChatMessage{Role: "system", Content: s.System, CacheBoundary: s.CacheBoundary})
	}
	messages = append(messages, s.History...)

	ctx, cancel := context.WithTimeout(context.Background(), mc.Timeout())
	defer cancel()

	modelTag := s.Model
	if mc.ModelName != "" {
		modelTag = mc.ModelName
	}

	reply, usage, err := sp.ChatStream(ctx, modelTag, mc, messages, onToken)
	if err != nil {
		s.History = s.History[:len(s.History)-1]
		return "", err
	}
	s.LastUsage = usage
	s.History = append(s.History, ChatMessage{Role: "assistant", Content: reply})
	return reply, nil
}

// LastExchange — último par (user, assistant), usado por `/memory` en el
// chat para guardar la sesión sin tener que copiar y pegar a mano.
func (s *Session) LastExchange() (user, assistant string, ok bool) {
	if len(s.History) < 2 {
		return "", "", false
	}
	last := s.History[len(s.History)-2:]
	if last[0].Role == "user" && last[1].Role == "assistant" {
		return last[0].Content, last[1].Content, true
	}
	return "", "", false
}

func trimHistory(h *[]ChatMessage) {
	if len(*h) > maxHistoryMessages {
		*h = (*h)[len(*h)-maxHistoryMessages:]
	}
}
