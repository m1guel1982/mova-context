// provider_anthropic.go — Anthropic's /v1/messages shape (its own
// headers and separate top-level "system" field, not OpenAI-compatible),
// used when a model config's "type" is "anthropic"/"claude" (see
// provider.go's NewProvider).
package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type anthropicProvider struct{ cfg *ModelConfig }

func (p *anthropicProvider) Chat(ctx context.Context, model string, mc *ModelConfig, messages []ChatMessage) (string, Usage, error) {
	system, boundary, rest := splitSystemMessage(messages)

	body := map[string]any{
		"model":      model,
		"max_tokens": orDefaultInt(mc.NumPredict, 1024),
		"messages":   rest,
	}
	if system != "" {
		body["system"] = systemField(system, boundary)
	}

	var out struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := postAnthropicJSON(ctx, p.cfg, body, &out); err != nil {
		return "", Usage{}, err
	}
	if out.Error.Message != "" {
		return "", Usage{}, fmt.Errorf("anthropic: %s", out.Error.Message)
	}
	if len(out.Content) == 0 {
		return "", Usage{}, fmt.Errorf("anthropic: empty response (no content)")
	}
	usage := Usage{PromptTokens: out.Usage.InputTokens, CompletionTokens: out.Usage.OutputTokens}
	return out.Content[0].Text, usage, nil
}

// splitSystemMessage pulls the first "system"-role message out of a chat
// history — Anthropic's API takes system as a separate top-level field,
// not as a message with role "system" like OpenAI/Ollama expect. Also
// returns that message's CacheBoundary (0 if unset/not applicable).
func splitSystemMessage(messages []ChatMessage) (system string, boundary int, rest []ChatMessage) {
	rest = make([]ChatMessage, 0, len(messages))
	for _, m := range messages {
		if m.Role == "system" && system == "" {
			system = m.Content
			boundary = m.CacheBoundary
			continue
		}
		rest = append(rest, m)
	}
	return system, boundary, rest
}

// systemField builds Anthropic's "system" request field. With no
// boundary (the common case — "cache_hint" not enabled, or no provider
// benefits from it), it's the same plain string Anthropic has always
// accepted. With a boundary, it becomes a two-block array: the stable
// prefix marked with "cache_control": {"type": "ephemeral"} (Anthropic's
// prompt-caching marker — see https://docs.anthropic.com, "Prompt
// caching"), and the rest as a second, uncached block. Anthropic accepts
// both forms for "system"; this never breaks a request that doesn't use it.
func systemField(system string, boundary int) any {
	if boundary <= 0 || boundary >= len(system) {
		return system
	}
	return []map[string]any{
		{"type": "text", "text": system[:boundary], "cache_control": map[string]string{"type": "ephemeral"}},
		{"type": "text", "text": system[boundary:]},
	}
}

func postAnthropicJSON(ctx context.Context, cfg *ModelConfig, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	// Normalizamos la URL eliminando barras finales y sufijos "/v1/messages" duplication-prone
	baseURL := strings.TrimSuffix(orDefault(cfg.BaseURL, "https://api.anthropic.com"), "/")
	baseURL = strings.TrimSuffix(baseURL, "/v1/messages")
	reqURL := baseURL + "/v1/messages"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("x-api-key", cfg.APIKey)

	resp, err := SharedClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach anthropic: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("anthropic responded %d: %s", resp.StatusCode, sanitizeResponseBody(data))
	}
	return json.Unmarshal(data, out)
}