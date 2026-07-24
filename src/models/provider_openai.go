// provider_openai.go — any /v1/chat/completions-shaped API: OpenAI
// itself, LM Studio, vLLM, TGI... used when a model config's "type" is
// "openai"/"openai-compatible" (see provider.go's NewProvider).
package models

import (
	"context"
	"fmt"
	"strings"
)

type openAIProvider struct{ cfg *ModelConfig }

func (p *openAIProvider) Chat(ctx context.Context, model string, mc *ModelConfig, messages []ChatMessage) (string, Usage, error) {
	body := map[string]any{
		"model":       model,
		"messages":    messages,
		"temperature": mc.Temperature,
		"top_p":       mc.TopP,
		"max_tokens":  orDefaultInt(mc.NumPredict, 512),
		"stream":      false,
	}

	var out struct {
		Choices []struct {
			Message ChatMessage `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	baseURL := strings.TrimSuffix(p.cfg.ResolvedBaseURL(), "/")
	path := "v1/chat/completions"

	if strings.HasSuffix(baseURL, "/v1") {
		path = "chat/completions"
	}

	if err := postJSON(ctx, p.cfg, path, body, &out); err != nil {
		return "", Usage{}, err
	}
	if out.Error.Message != "" {
		return "", Usage{}, fmt.Errorf("%s: %s", p.cfg.Provider, out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return "", Usage{}, fmt.Errorf("%s: empty response (no choices)", p.cfg.Provider)
	}
	usage := Usage{PromptTokens: out.Usage.PromptTokens, CompletionTokens: out.Usage.CompletionTokens}
	return out.Choices[0].Message.Content, usage, nil
}
