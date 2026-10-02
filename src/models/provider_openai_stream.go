// provider_openai_stream.go — streaming SSE de /v1/chat/completions.
// Robusto frente a lo que OpenRouter y compatibles realmente envían:
// comentarios ": OPENROUTER PROCESSING", "data:" con o sin espacio,
// deltas de razonamiento (delta.reasoning / reasoning_content) que no son
// respuesta, errores a mitad de stream y cuerpos JSON de error con 200.
package models

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type openAIChunk struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			Reasoning        string `json:"reasoning"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p *openAIProvider) ChatStream(ctx context.Context, model string, mc *ModelConfig, messages []ChatMessage, onToken func(string)) (string, Usage, error) {
	path, body := p.openAIRequest(model, mc, messages, true)
	var content, reasoning, stray strings.Builder
	var usage Usage
	var serverErr string

	err := postJSONStream(ctx, p.cfg, path, body, func(line []byte) error {
		raw := strings.TrimSpace(string(line))
		if raw == "" || strings.HasPrefix(raw, ":") || strings.HasPrefix(raw, "event:") {
			return nil // comentario keep-alive / evento SSE
		}
		if !strings.HasPrefix(raw, "data:") {
			if stray.Len() < 4096 { // cuerpo JSON de error devuelto con 200
				stray.WriteString(raw)
			}
			return nil
		}
		data := strings.TrimSpace(strings.TrimPrefix(raw, "data:"))
		if data == "" || data == "[DONE]" {
			return nil
		}
		var c openAIChunk
		if json.Unmarshal([]byte(data), &c) != nil {
			return nil
		}
		if c.Error != nil && c.Error.Message != "" {
			serverErr = c.Error.Message
			return nil
		}
		if len(c.Choices) > 0 {
			ch := c.Choices[0]
			if ch.Delta.Content != "" {
				content.WriteString(ch.Delta.Content)
				if onToken != nil {
					onToken(ch.Delta.Content)
				}
			}
			reasoning.WriteString(ch.Delta.Reasoning + ch.Delta.ReasoningContent)
			if ch.FinishReason != "" {
				usage.FinishReason = ch.FinishReason
			}
		}
		if c.Usage.PromptTokens > 0 || c.Usage.CompletionTokens > 0 {
			usage.PromptTokens, usage.CompletionTokens = c.Usage.PromptTokens, c.Usage.CompletionTokens
		}
		return nil
	})
	if err != nil {
		return "", usage, err
	}
	if serverErr != "" {
		return "", usage, fmt.Errorf("%s: %s", p.cfg.Provider, serverErr)
	}
	if content.Len() == 0 && reasoning.Len() == 0 && stray.Len() > 0 {
		return "", usage, fmt.Errorf("%s: respuesta inesperada del servidor: %s", p.cfg.Provider, sanitizeResponseBody([]byte(stray.String())))
	}
	return p.openAIReply(model, mc, content.String(), reasoning.String(), usage)
}
