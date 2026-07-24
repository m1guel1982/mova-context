package models

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionCapturesRealUsage(t *testing.T) {
	srv := fakeOllama(t)
	defer srv.Close()
	root := setupProject(t, srv.URL)

	if err := SetActiveProvider(root, "ollama"); err != nil {
		t.Fatal(err)
	}
	sess, err := NewSession(root)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if err := sess.SetModel("llama3.1"); err != nil {
		t.Fatalf("SetModel: %v", err)
	}
	if _, err := sess.Send("hola"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if sess.LastUsage.PromptTokens != 42 || sess.LastUsage.CompletionTokens != 7 {
		t.Fatalf("expected real usage {42,7} from the fake Ollama response, got %+v", sess.LastUsage)
	}
}

func TestOpenAIProviderCapturesUsage(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": "hi"}}},
			"usage":   map[string]int{"prompt_tokens": 100, "completion_tokens": 20},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	p := &openAIProvider{cfg: &ModelConfig{Provider: "openai", BaseURL: srv.URL}}
	reply, usage, err := p.Chat(context.Background(), "gpt-5", &ModelConfig{}, []ChatMessage{{Role: "user", Content: "hola"}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if reply != "hi" {
		t.Fatalf("reply = %q", reply)
	}
	if usage.PromptTokens != 100 || usage.CompletionTokens != 20 {
		t.Fatalf("expected usage {100,20}, got %+v", usage)
	}
}

func TestSessionSwitchProvider_ChangesProviderAndModel(t *testing.T) {
	ollamaSrv := fakeOllama(t)
	defer ollamaSrv.Close()
	root := setupProject(t, ollamaSrv.URL)

	// Segundo proveedor ("myopenai") — simula lo que project.json's
	// llm_profile.provider apuntaría a un Cloud/servidor distinto del
	// default global.
	openaiMux := http.NewServeMux()
	openaiMux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": "hi from myopenai"}}},
			"usage":   map[string]int{"prompt_tokens": 10, "completion_tokens": 2},
		})
	})
	openaiSrv := httptest.NewServer(openaiMux)
	defer openaiSrv.Close()

	// UN solo archivo: conexión (provider/type/base_url) + parámetros de
	// inferencia juntos — ya no hay un config.json de proveedor aparte.
	mc := DefaultModelConfig(&ModelConfig{Provider: "myopenai", Type: "openai-compatible", BaseURL: openaiSrv.URL})
	if err := SaveModelConfig(root, "myopenai", "gpt-5", mc); err != nil {
		t.Fatal(err)
	}

	if err := SetActiveProvider(root, "ollama"); err != nil {
		t.Fatal(err)
	}
	sess, err := NewSession(root)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if sess.Provider != "ollama" {
		t.Fatalf("expected the global default provider (ollama) before switching, got %q", sess.Provider)
	}

	// Esto es lo que project.json's llm_profile.provider/model dispara —
	// ver cli/chat_cmd.go's applyProjectLLMProfile / mcp/chat_tool.go.
	if err := sess.SwitchProvider("myopenai", "gpt-5"); err != nil {
		t.Fatalf("SwitchProvider: %v", err)
	}
	if sess.Provider != "myopenai" || sess.Model != "gpt-5" {
		t.Fatalf("expected provider/model to switch to myopenai/gpt-5, got %s/%s", sess.Provider, sess.Model)
	}

	reply, err := sess.Send("hola")
	if err != nil {
		t.Fatalf("Send after SwitchProvider: %v", err)
	}
	if reply != "hi from myopenai" {
		t.Fatalf("expected the reply to come from the switched provider, got %q", reply)
	}
	if sess.LastUsage.PromptTokens != 10 {
		t.Fatalf("expected usage from the switched provider, got %+v", sess.LastUsage)
	}
}

func TestAnthropicProviderSplitsSystemAndCapturesUsage(t *testing.T) {
	var capturedBody map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key header, got %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Error("expected anthropic-version header to be set")
		}
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{{"type": "text", "text": "hello from claude"}},
			"usage":   map[string]int{"input_tokens": 55, "output_tokens": 12},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	p := &anthropicProvider{cfg: &ModelConfig{Provider: "anthropic", BaseURL: srv.URL, APIKey: "test-key"}}
	messages := []ChatMessage{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "hola"},
	}
	reply, usage, err := p.Chat(context.Background(), "claude", &ModelConfig{}, messages)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if reply != "hello from claude" {
		t.Fatalf("reply = %q", reply)
	}
	if usage.PromptTokens != 55 || usage.CompletionTokens != 12 {
		t.Fatalf("expected usage {55,12}, got %+v", usage)
	}
	if capturedBody["system"] != "You are helpful." {
		t.Fatalf("expected system prompt to be sent as a top-level field, got: %v", capturedBody["system"])
	}
	msgs, _ := capturedBody["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("expected exactly 1 message (system removed from the array), got %d", len(msgs))
	}
}
