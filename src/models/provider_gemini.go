// provider_gemini.go — Google Gemini's native REST API
// (/v1beta/models/<model>:generateContent), used when a model config's
// "type" is "google"/"gemini" (see provider.go's NewProvider).
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

type geminiNativeProvider struct{ cfg *ModelConfig }

func (p *geminiNativeProvider) Chat(ctx context.Context, model string, mc *ModelConfig, messages []ChatMessage) (string, Usage, error) {
	type geminiPart struct {
		Text string `json:"text"`
	}
	type geminiContent struct {
		Role  string       `json:"role"`
		Parts []geminiPart `json:"parts"`
	}

	var contents []geminiContent
	var systemInstruction *geminiContent

	for _, m := range messages {
		role := m.Role
		if role == "system" {
			systemInstruction = &geminiContent{
				Parts: []geminiPart{{Text: m.Content}},
			}
			continue
		}
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}

	body := map[string]any{
		"contents": contents,
		"generationConfig": map[string]any{
			"temperature":     mc.Temperature,
			"topP":            mc.TopP,
			"maxOutputTokens": orDefaultInt(mc.NumPredict, 2048),
		},
	}
	if systemInstruction != nil {
		body["systemInstruction"] = systemInstruction
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", Usage{}, err
	}

	// Build the exact REST URL:
	// https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=...
	cleanModel := strings.TrimPrefix(model, "models/")
	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", cleanModel, p.cfg.APIKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", Usage{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := SharedClient.Do(req)
	if err != nil {
		return "", Usage{}, fmt.Errorf("could not reach gemini: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", Usage{}, err
	}

	if resp.StatusCode >= 300 {
		return "", Usage{}, fmt.Errorf("google responded %d: %s", resp.StatusCode, sanitizeResponseBody(data))
	}

	var out struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(data, &out); err != nil {
		return "", Usage{}, fmt.Errorf("gemini unmarshal error: %w", err)
	}
	if out.Error.Message != "" {
		return "", Usage{}, fmt.Errorf("google: %s", out.Error.Message)
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return "", Usage{}, fmt.Errorf("google: empty response (no candidates)")
	}

	text := out.Candidates[0].Content.Parts[0].Text
	usage := Usage{
		PromptTokens:     out.UsageMetadata.PromptTokenCount,
		CompletionTokens: out.UsageMetadata.CandidatesTokenCount,
	}

	return text, usage, nil
}
