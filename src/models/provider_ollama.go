// provider_ollama.go — Ollama's native /api/chat, used when a model
// config's "type" is "ollama" or anything unrecognized (see provider.go's
// NewProvider default case) — the default assumption for local models.
package models

import (
	"context"
	"encoding/json"
	"fmt"
)

type ollamaProvider struct{ cfg *ModelConfig }

func (p *ollamaProvider) Chat(ctx context.Context, model string, mc *ModelConfig, messages []ChatMessage) (string, Usage, error) {
	body := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   false,
		"options": map[string]any{
			"top_k":          mc.TopK,
			"top_p":          mc.TopP,
			"num_ctx":        mc.NumCtx,
			"num_thread":     mc.Threads,
			"mirostat":       mc.Mirostat,
			"num_predict":    mc.NumPredict,
			"temperature":    mc.Temperature,
			"repeat_penalty": mc.RepeatPenalty,
		},
		"keep_alive": orDefault(mc.KeepAlive, "24h"),
	}

	var out struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Error           string `json:"error"`
		PromptEvalCount int    `json:"prompt_eval_count"`
		EvalCount       int    `json:"eval_count"`
	}
	if err := postJSON(ctx, p.cfg, "/api/chat", body, &out); err != nil {
		return "", Usage{}, err
	}
	if out.Error != "" {
		return "", Usage{}, fmt.Errorf("ollama: %s", out.Error)
	}
	return out.Message.Content, Usage{PromptTokens: out.PromptEvalCount, CompletionTokens: out.EvalCount}, nil
}

func (p *ollamaProvider) ChatStream(ctx context.Context, model string, mc *ModelConfig, messages []ChatMessage, onToken func(string)) (string, Usage, error) {
	body := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   true,
		"options": map[string]any{
			"top_k":          mc.TopK,
			"top_p":          mc.TopP,
			"num_ctx":        mc.NumCtx,
			"num_thread":     mc.Threads,
			"mirostat":       mc.Mirostat,
			"num_predict":    mc.NumPredict,
			"temperature":    mc.Temperature,
			"repeat_penalty": mc.RepeatPenalty,
		},
		"keep_alive": orDefault(mc.KeepAlive, "24h"),
	}

	var full string
	var usage Usage
	var serverErr string

	err := postJSONStream(ctx, p.cfg, "/api/chat", body, func(line []byte) {
		var chunk struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done            bool   `json:"done"`
			Error           string `json:"error"`
			PromptEvalCount int    `json:"prompt_eval_count"`
			EvalCount       int    `json:"eval_count"`
		}
		if err := json.Unmarshal(line, &chunk); err != nil {
			return
		}
		if chunk.Error != "" {
			serverErr = chunk.Error
			return
		}
		if chunk.Message.Content != "" {
			full += chunk.Message.Content
			if onToken != nil {
				onToken(chunk.Message.Content)
			}
		}
		if chunk.Done {
			usage = Usage{PromptTokens: chunk.PromptEvalCount, CompletionTokens: chunk.EvalCount}
		}
	})
	if err != nil {
		return "", Usage{}, err
	}
	if serverErr != "" {
		return "", Usage{}, fmt.Errorf("ollama: %s", serverErr)
	}
	return full, usage, nil
}
