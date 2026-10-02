// provider_openai.go — any /v1/chat/completions-shaped API: OpenAI
// itself, OpenRouter, LM Studio, vLLM, TGI... used when a model config's
// "type" is "openai"/"openai-compatible" (see provider.go's NewProvider).
// El streaming vive en provider_openai_stream.go.
package models

import (
	"context"
	"fmt"
	"strings"
)

type openAIProvider struct{ cfg *ModelConfig }

// openAIRequest arma path y body, idénticos para Chat y ChatStream.
// "max_tokens" sale de ModelConfig.ResponseMax (antes se ignoraba el
// max_tokens del .json y se mandaba 512). top_p solo viaja si está
// configurado: mandar 0 por defecto recortaba el muestreo sin que nadie lo
// hubiera pedido.
func (p *openAIProvider) openAIRequest(model string, mc *ModelConfig, messages []ChatMessage, stream bool) (string, map[string]any) {
	body := map[string]any{
		"model":       model,
		"messages":    messages,
		"temperature": mc.Temperature,
		"max_tokens":  mc.ResponseMax(512),
		"stream":      stream,
	}
	if mc.TopP > 0 {
		body["top_p"] = mc.TopP
	}
	if len(mc.Reasoning) > 0 {
		body["reasoning"] = mc.Reasoning
	}
	baseURL := strings.TrimSuffix(p.cfg.ResolvedBaseURL(), "/")
	baseURL = strings.TrimSuffix(baseURL, "/chat/completions")
	if stream && (strings.Contains(baseURL, "openrouter.ai") || strings.Contains(baseURL, "api.openai.com")) {
		body["stream_options"] = map[string]any{"include_usage": true} // uso REAL en el último chunk
	}
	path := "v1/chat/completions"
	if strings.HasSuffix(baseURL, "/v1") {
		path = "chat/completions"
	}
	return path, body
}

// openAIReply decide el resultado final a partir de lo acumulado, igual
// para stream y no-stream: contenido → éxito; sin contenido pero con
// razonamiento completo (finish=stop) → se muestra el razonamiento; en
// cualquier otro caso EmptyReplyError con el uso ya consumido.
func (p *openAIProvider) openAIReply(model string, mc *ModelConfig, content, reasoning string, usage Usage) (string, Usage, error) {
	if strings.TrimSpace(content) != "" {
		return content, usage, nil
	}
	if strings.TrimSpace(reasoning) != "" && usage.FinishReason == "stop" {
		return "[sin contenido final — se muestra el razonamiento del modelo]\n\n" + reasoning, usage, nil
	}
	return "", usage, &EmptyReplyError{Provider: p.cfg.Provider, Model: model, Finish: usage.FinishReason,
		MaxTokens: mc.ResponseMax(512), Usage: usage, ReasoningChars: len(reasoning)}
}

func (p *openAIProvider) Chat(ctx context.Context, model string, mc *ModelConfig, messages []ChatMessage) (string, Usage, error) {
	path, body := p.openAIRequest(model, mc, messages, false)
	var out struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				Reasoning        string `json:"reasoning"`
				ReasoningContent string `json:"reasoning_content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
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
	ch := out.Choices[0]
	usage := Usage{PromptTokens: out.Usage.PromptTokens, CompletionTokens: out.Usage.CompletionTokens, FinishReason: ch.FinishReason}
	reasoning := ch.Message.Reasoning + ch.Message.ReasoningContent
	return p.openAIReply(model, mc, ch.Message.Content, reasoning, usage)
}
