// provider.go — the Provider interface every model backend implements,
// and NewProvider, which picks the right one purely from ModelConfig.Type
// (see models/types.go and the "type" reference in docs/i18n/en/
// COMMANDS.md). Each implementation lives in its own sibling file so none
// of them grows past 300 lines: provider_gemini.go, provider_ollama.go,
// provider_openai.go, provider_anthropic.go. provider_http.go holds the
// generic HTTP plumbing (postJSON/postJSONStream) shared by more than one
// of them.
package models

import (
	"context"
	"strings"
)

type Usage struct {
	PromptTokens     int
	CompletionTokens int
}

type Provider interface {
	Chat(ctx context.Context, model string, mc *ModelConfig, messages []ChatMessage) (string, Usage, error)
	ChatStream(ctx context.Context, model string, mc *ModelConfig, messages []ChatMessage, onToken func(string)) (string, Usage, error)
}

type StreamProvider interface {
	ChatStream(ctx context.Context, model string, mc *ModelConfig, messages []ChatMessage, onToken func(string)) (string, Usage, error)
}

// NewProvider selects a Provider purely from pc.Type — the single field
// that decides request shape (see the "type" table in COMMANDS.md).
// pc.Provider (the config/models/ folder name) never affects this choice
// except for one narrow legacy case: a file declaring
// "type": "openai-compatible" that's actually pointed at Google's
// OpenAI-compatible endpoint (by folder name or base_url) still needs the
// Gemini-native path, since Google's OpenAI-compatible surface doesn't
// support every feature this project relies on.
func NewProvider(pc *ModelConfig) Provider {
	switch pc.Type {
	case "google", "gemini":
		return &geminiNativeProvider{cfg: pc}
	case "openai-compatible", "openai":
		if strings.Contains(strings.ToLower(pc.Provider), "google") || strings.Contains(pc.BaseURL, "googleapis.com") {
			return &geminiNativeProvider{cfg: pc}
		}
		return &openAIProvider{cfg: pc}
	case "anthropic", "claude":
		return &anthropicProvider{cfg: pc}
	default:
		return &ollamaProvider{cfg: pc}
	}
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func orDefaultInt(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}

// sanitizeResponseBody trims and caps a raw HTTP error body for safe
// inclusion in an error message — never echoes more than 300 characters
// of whatever a provider's server happened to return.
func sanitizeResponseBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 300 {
		return s[:300] + "... [truncated]"
	}
	if s == "" {
		return "(empty response from server)"
	}
	return s
}