// provider_gemini.go — Google Gemini's native REST API
// (/v1beta/models/<model>:generateContent), used when a model config's
// "type" is "google"/"gemini" (see provider.go's NewProvider).
package models

import (
	"bufio"
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

	// Normalización de BaseURL (respetando la configuración del JSON o usando el valor por defecto)
	baseURL := strings.TrimSuffix(orDefault(p.cfg.BaseURL, "https://generativelanguage.googleapis.com"), "/")
	baseURL = strings.TrimSuffix(baseURL, "/v1beta")

	cleanModel := strings.TrimPrefix(model, "models/")
	endpoint := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", baseURL, cleanModel, p.cfg.APIKey)

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

func (p *geminiNativeProvider) ChatStream(ctx context.Context, model string, mc *ModelConfig, messages []ChatMessage, onToken func(string)) (string, Usage, error) {
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

	baseURL := strings.TrimSuffix(orDefault(p.cfg.BaseURL, "https://generativelanguage.googleapis.com"), "/")
	baseURL = strings.TrimSuffix(baseURL, "/v1beta")

	cleanModel := strings.TrimPrefix(model, "models/")
	endpoint := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", baseURL, cleanModel, p.cfg.APIKey)

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

	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return "", Usage{}, fmt.Errorf("google responded %d: %s", resp.StatusCode, sanitizeResponseBody(data))
	}

	var full string
	var usage Usage
	var serverErr string

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		var chunk struct {
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

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if chunk.Error.Message != "" {
			serverErr = chunk.Error.Message
			break
		}

		if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
			text := chunk.Candidates[0].Content.Parts[0].Text
			if text != "" {
				full += text
				if onToken != nil {
					onToken(text)
				}
			}
		}

		if chunk.UsageMetadata.PromptTokenCount > 0 || chunk.UsageMetadata.CandidatesTokenCount > 0 {
			usage = Usage{
				PromptTokens:     chunk.UsageMetadata.PromptTokenCount,
				CompletionTokens: chunk.UsageMetadata.CandidatesTokenCount,
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", Usage{}, err
	}
	if serverErr != "" {
		return "", Usage{}, fmt.Errorf("google: %s", serverErr)
	}

	return full, usage, nil
}