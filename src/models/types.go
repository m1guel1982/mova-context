// types.go — estructuras compartidas para proveedores de modelos locales
// y Cloud (Ollama, LM Studio, vLLM, OpenAI, Google Gemini, Anthropic
// Claude, o cualquier otro compatible).
//
// UNA SOLA FUENTE DE VERDAD POR MODELO: hasta la versión anterior, un
// modelo necesitaba DOS archivos — config/models/<provider>/config.json
// (conexión: host/puerto/api_key, compartido por todo el proveedor) y
// config/models/<provider>/<modelo>.json (parámetros de inferencia) — y
// project.json's llm_profile repetía provider/model encima. Eso son tres
// lugares para el mismo dato, y en Windows además rompía cuando el tag
// real del modelo llevaba ":" (no es un carácter válido en nombres de
// archivo ahí).
//
// Ahora config/models/<provider>/<config>.json es el ÚNICO archivo: trae
// tanto la conexión (base_url, api_key, timeout...) como los parámetros
// del modelo (temperature, num_ctx...) en un mismo JSON. project.json's
// llm_profile ya no repite nada — solo apunta con {"provider", "config"}
// a ese archivo (ver core/types.go). config.json por proveedor queda
// eliminado.
package models

import (
	"fmt"
	"time"
)

// ModelConfig — config/models/<provider>/<config>.json. Fuente de verdad
// única para un modelo: conexión + parámetros de inferencia en un mismo
// archivo. Los nombres de campo respetan exactamente el formato pedido
// (algunos en español, el resto tal como los usa la API de inferencia —
// no se traducen para no romper compatibilidad).
type ModelConfig struct {
	// ── Conexión (antes vivía en config.json, uno por proveedor) ──────
	Provider       string `json:"provider,omitempty"`        // "ollama" | "google" | "anthropic" | "openai" | "lmstudio" | ...
	Type           string `json:"type,omitempty"`            // "ollama" (default) | "openai-compatible" | "anthropic"
	Host           string `json:"host,omitempty"`            // p.ej. "mova_ollama" (nombre del contenedor)
	Port           int    `json:"port,omitempty"`            // p.ej. 11434
	BaseURL        string `json:"base_url,omitempty"`        // si está presente, tiene prioridad sobre host+port
	APIKey         string `json:"api_key,omitempty"`         // Cloud (Gemini/OpenAI/Claude) lo exige
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"` // 0 = usa el default (120s)

	// ── Modelo (antes vivía en <modelo>.json aparte) ───────────────────
	ModelName     string  `json:"model,omitempty"` // tag real enviado a la API, p.ej. "llama3.2:3b" o "gemini-2.5-flash". Si se omite, se usa el nombre del archivo.
	Tipo          string  `json:"tipo,omitempty"`  // "llm" | "embedding" | "reranker"
	TopK          int     `json:"top_k,omitempty"`
	TopP          float64 `json:"top_p,omitempty"`
	NumCtx        int     `json:"num_ctx,omitempty"`
	Threads       int     `json:"threads,omitempty"`
	Version       string  `json:"version,omitempty"`
	Mirostat      int     `json:"mirostat,omitempty"`
	KeepAlive     string  `json:"keep_alive,omitempty"`
	NumPredict    int     `json:"num_predict,omitempty"` // también funciona como el máximo de tokens de RESPUESTA en providers openai-compatible/anthropic (ver provider.go)
	Temperature   float64 `json:"temperature"`
	ContextWindow int     `json:"context_window,omitempty"`
	RepeatPenalty float64 `json:"repeat_penalty,omitempty"`
}

// ResolvedBaseURL arma la URL final del servidor: base_url explícito gana,
// si no se construye desde host+port (default: localhost:11434, Ollama).
func (m *ModelConfig) ResolvedBaseURL() string {
	if m.BaseURL != "" {
		return m.BaseURL
	}
	host := m.Host
	if host == "" {
		host = "localhost"
	}
	port := m.Port
	if port == 0 {
		port = 11434
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}

// Timeout por request — configurable por modelo.
func (m *ModelConfig) Timeout() time.Duration {
	if m.TimeoutSeconds <= 0 {
		return 120 * time.Second
	}
	return time.Duration(m.TimeoutSeconds) * time.Second
}

// DefaultModelConfig — plantilla usada al instalar un modelo nuevo que
// todavía no tiene su .json propio (ver manage.go). seed, si no es nil,
// aporta los campos de conexión (base_url/host/port/type/api_key/timeout)
// de un modelo hermano ya configurado en el mismo proveedor — así
// `mova install` no necesita ningún config.json separado para saber a
// qué servidor pegarle.
func DefaultModelConfig(seed *ModelConfig) ModelConfig {
	cfg := ModelConfig{
		Tipo: "llm", TopK: 40, TopP: 0.9, NumCtx: 4096, Threads: 4,
		Mirostat: 0, KeepAlive: "24h", NumPredict: 512,
		Temperature: 0, ContextWindow: 131072, RepeatPenalty: 1.1,
	}
	if seed != nil {
		cfg.Provider = seed.Provider
		cfg.Type = seed.Type
		cfg.Host = seed.Host
		cfg.Port = seed.Port
		cfg.BaseURL = seed.BaseURL
		cfg.APIKey = seed.APIKey
		cfg.TimeoutSeconds = seed.TimeoutSeconds
	}
	return cfg
}

// ChatMessage — un turno de la conversación (compatible con la forma que
// usan tanto Ollama como cualquier servidor openai-compatible).
type ChatMessage struct {
	Role    string `json:"role"` // "system" | "user" | "assistant"
	Content string `json:"content"`

	// CacheBoundary: byte offset in Content where a stable, cacheable
	// prefix ends (0 = not applicable). Only meaningful when Role is
	// "system" and only read by the Anthropic provider (see
	// provider_anthropic.go) to mark that prefix with Anthropic's own
	// prompt-caching "cache_control" field — every other provider
	// (OpenAI, Gemini, Ollama) ignores this field entirely, so adding
	// it changes nothing for them. Set by mova.local/budget's
	// LayoutForCache (see cli/chat_helpers.go, cli/tui_chat.go).
	CacheBoundary int `json:"-"`
}

// ActiveState — config/models/active.json. Recuerda qué proveedor y qué
// archivo de configuración se usa por default en `mova chat` y en las
// tools de MCP/HTTP cuando no se especifica ninguno explícitamente. Es
// solo un PUNTERO (dos strings) — nunca duplica los datos de conexión o
// del modelo, esos siguen viviendo únicamente en su .json bajo
// config/models/<provider>/.
type ActiveState struct {
	Provider string `json:"provider"`
	Config   string `json:"config,omitempty"`
}

// InstalledModel — una entrada de `mova model-list` (GET /api/tags).
type InstalledModel struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modified_at"`
}
