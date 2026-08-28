// provider_http.go — generic HTTP request helpers shared by
// provider_ollama.go and provider_openai.go (both hit a plain JSON
// endpoint under cfg.ResolvedBaseURL()); Gemini and Anthropic build their
// own requests directly since their URL/header shapes differ enough not
// to share this code.
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

func postJSON(ctx context.Context, cfg *ModelConfig, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	baseURL := strings.TrimSuffix(cfg.ResolvedBaseURL(), "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	rawURL := baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	resp, err := SharedClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach %s: %w", cfg.Provider, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s responded %d: %s", cfg.Provider, resp.StatusCode, sanitizeResponseBody(data))
	}
	return json.Unmarshal(data, out)
}

func postJSONStream(ctx context.Context, cfg *ModelConfig, path string, body any, onLine func(line []byte) error) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	baseURL := strings.TrimSuffix(cfg.ResolvedBaseURL(), "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	rawURL := baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	resp, err := SharedClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach %s: %w", cfg.Provider, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s responded %d: %s", cfg.Provider, resp.StatusCode, sanitizeResponseBody(data))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if err := onLine(line); err != nil {
			return err
		}
	}
	return scanner.Err()
}