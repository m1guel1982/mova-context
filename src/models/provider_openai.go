// provider_openai.go — any /v1/chat/completions-shaped API: OpenAI
// itself, LM Studio, vLLM, TGI... used when a model config's "type" is
// "openai"/"openai-compatible" (see provider.go's NewProvider).
package models

import (
	"context"
	"encoding/json"
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

	// Normalización defensiva de baseURL y path
	baseURL := strings.TrimSuffix(p.cfg.ResolvedBaseURL(), "/")
	baseURL = strings.TrimSuffix(baseURL, "/chat/completions")

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

func (p *openAIProvider) ChatStream(ctx context.Context, model string, mc *ModelConfig, messages []ChatMessage, onToken func(string)) (string, Usage, error) {
	body := map[string]any{
		"model":       model,
		"messages":    messages,
		"temperature": mc.Temperature,
		"top_p":       mc.TopP,
		"max_tokens":  orDefaultInt(mc.NumPredict, 512),
		"stream":      true,
	}

	baseURL := strings.TrimSuffix(p.cfg.ResolvedBaseURL(), "/")
	baseURL = strings.TrimSuffix(baseURL, "/chat/completions")

	path := "v1/chat/completions"
	if strings.HasSuffix(baseURL, "/v1") {
		path = "chat/completions"
	}

	var full string
	var usage Usage
	var serverErr string

	err := postJSONStream(ctx, p.cfg, path, body, func(line []byte) error {
		raw := strings.TrimSpace(string(line))
		if !strings.HasPrefix(raw, "data: ") {
			return nil
		}

		data := strings.TrimPrefix(raw, "data: ")
		if data == "[DONE]" {
			return nil
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage,omitempty"`
			Error struct {
				Message string `json:"message"`
			} `json:"error,omitempty"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil
		}

		if chunk.Error.Message != "" {
			serverErr = chunk.Error.Message
			return nil
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			content := chunk.Choices[0].Delta.Content
			full += content
			if onToken != nil {
				onToken(content)
			}
		}

		if chunk.Usage.CompletionTokens > 0 || chunk.Usage.PromptTokens > 0 {
			usage = Usage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
			}
		}

		return nil
	})

	if err != nil {
		return "", Usage{}, err
	}
	if serverErr != "" {
		return "", Usage{}, fmt.Errorf("%s: %s", p.cfg.Provider, serverErr)
	}

	return full, usage, nil
}