package models

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// srvCapturing devuelve un servidor openai-compatible que guarda el último
// body recibido y responde con handler.
func srvCapturing(t *testing.T, body *map[string]any, handler func(w http.ResponseWriter)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, body)
		handler(w)
	}))
}

func TestResponseMaxPrecedence(t *testing.T) {
	cases := []struct {
		mc   ModelConfig
		def  int
		want int
	}{
		{ModelConfig{}, 512, 512},
		{ModelConfig{NumPredict: 700}, 512, 700},
		{ModelConfig{MaxTokens: 900}, 512, 900},
		{ModelConfig{MaxTokens: 900, NumPredict: 700}, 512, 900},
	}
	for i, c := range cases {
		if got := c.mc.ResponseMax(c.def); got != c.want {
			t.Fatalf("case %d: got %d want %d", i, got, c.want)
		}
	}
}

// El bug original: max_tokens del .json no llegaba al proveedor.
func TestOpenAIHonorsMaxTokensAndOmitsZeroTopP(t *testing.T) {
	var got map[string]any
	srv := srvCapturing(t, &got, func(w http.ResponseWriter) {
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "ok"}, "finish_reason": "stop"}},
		})
	})
	defer srv.Close()
	p := &openAIProvider{cfg: &ModelConfig{Provider: "openRouter", BaseURL: srv.URL}}
	mc := &ModelConfig{MaxTokens: 32768, Reasoning: json.RawMessage(`{"effort":"low"}`)}
	if _, _, err := p.Chat(context.Background(), "m", mc, []ChatMessage{{Role: "user", Content: "x"}}); err != nil {
		t.Fatal(err)
	}
	if got["max_tokens"] != float64(32768) {
		t.Fatalf("max_tokens = %v, want 32768", got["max_tokens"])
	}
	if _, has := got["top_p"]; has {
		t.Fatalf("top_p must not be sent when unset: %v", got["top_p"])
	}
	if r, _ := got["reasoning"].(map[string]any); r["effort"] != "low" {
		t.Fatalf("reasoning not forwarded: %v", got["reasoning"])
	}
}

// Caso real: DeepSeek gasta el cupo razonando → content "" + length.
func TestSessionEmptyReplyIsAnErrorWithUsage(t *testing.T) {
	var got map[string]any
	srv := srvCapturing(t, &got, func(w http.ResponseWriter) {
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "", "reasoning": "pensando..."}, "finish_reason": "length"}},
			"usage":   map[string]int{"prompt_tokens": 31633, "completion_tokens": 512},
		})
	})
	defer srv.Close()
	root := setupOpenAIProject(t, srv.URL, `"max_tokens": 512`)
	sess := newOpenAISession(t, root)
	n := len(sess.History)
	reply, err := sess.Send("revisar")
	var ee *EmptyReplyError
	if !errors.As(err, &ee) || reply != "" {
		t.Fatalf("expected EmptyReplyError, got reply=%q err=%v", reply, err)
	}
	if !strings.Contains(err.Error(), "max_tokens") || !strings.Contains(err.Error(), "31633") {
		t.Fatalf("error must explain the cause and show consumed tokens: %v", err)
	}
	if sess.LastUsage.PromptTokens != 31633 || sess.LastUsage.CompletionTokens != 512 {
		t.Fatalf("usage must be recorded even on failure: %+v", sess.LastUsage)
	}
	if len(sess.History) != n {
		t.Fatalf("failed turn must not stay in history")
	}
}

func TestSessionTruncatedReplyIsFlagged(t *testing.T) {
	var got map[string]any
	srv := srvCapturing(t, &got, func(w http.ResponseWriter) {
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "respuesta parcial"}, "finish_reason": "length"}},
			"usage":   map[string]int{"prompt_tokens": 10, "completion_tokens": 512},
		})
	})
	defer srv.Close()
	sess := newOpenAISession(t, setupOpenAIProject(t, srv.URL, `"max_tokens": 512`))
	reply, err := sess.Send("x")
	if err != nil || reply != "respuesta parcial" || !sess.LastTruncated {
		t.Fatalf("reply=%q err=%v truncated=%v", reply, err, sess.LastTruncated)
	}
}

func sseServer(t *testing.T, body *map[string]any, lines ...string) *httptest.Server {
	return srvCapturing(t, body, func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, l := range lines {
			io.WriteString(w, l+"\n\n")
		}
	})
}

func TestStreamEmptyContentReportsError(t *testing.T) {
	var got map[string]any
	srv := sseServer(t, &got,
		`: OPENROUTER PROCESSING`,
		`data: {"choices":[{"delta":{"reasoning":"hmm"}}]}`,
		`data:{"choices":[{"delta":{},"finish_reason":"length"}]}`, // sin espacio tras "data:"
		`data: {"choices":[],"usage":{"prompt_tokens":99,"completion_tokens":512}}`,
		`data: [DONE]`)
	defer srv.Close()
	p := &openAIProvider{cfg: &ModelConfig{Provider: "openRouter", BaseURL: srv.URL}}
	_, usage, err := p.ChatStream(context.Background(), "m", &ModelConfig{MaxTokens: 512}, []ChatMessage{{Role: "user", Content: "x"}}, nil)
	var ee *EmptyReplyError
	if !errors.As(err, &ee) {
		t.Fatalf("want EmptyReplyError, got %v", err)
	}
	if usage.PromptTokens != 99 || usage.FinishReason != "length" || ee.ReasoningChars == 0 {
		t.Fatalf("usage/finish not captured: %+v %+v", usage, ee)
	}
	if got["stream"] != true || got["max_tokens"] != float64(512) {
		t.Fatalf("bad request: %v", got)
	}
}

func TestStreamDeliversContentAndSurvivesQuirks(t *testing.T) {
	var got map[string]any
	srv := sseServer(t, &got,
		`: keep-alive`,
		`data: {"choices":[{"delta":{"reasoning":"pienso"}}]}`,
		`data:{"choices":[{"delta":{"content":"Hola "}}]}`,
		`data: {"choices":[{"delta":{"content":"mundo"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":3}}`,
		`data: [DONE]`)
	defer srv.Close()
	p := &openAIProvider{cfg: &ModelConfig{Provider: "openRouter", BaseURL: srv.URL}}
	var streamed strings.Builder
	reply, usage, err := p.ChatStream(context.Background(), "m", &ModelConfig{}, []ChatMessage{{Role: "user", Content: "x"}}, func(s string) { streamed.WriteString(s) })
	if err != nil || reply != "Hola mundo" || streamed.String() != "Hola mundo" {
		t.Fatalf("reply=%q streamed=%q err=%v", reply, streamed.String(), err)
	}
	if usage.CompletionTokens != 3 || usage.FinishReason != "stop" {
		t.Fatalf("usage %+v", usage)
	}
}

func TestStreamMidStreamErrorSurfaces(t *testing.T) {
	var got map[string]any
	srv := sseServer(t, &got, `data: {"error":{"message":"upstream exploded"}}`)
	defer srv.Close()
	p := &openAIProvider{cfg: &ModelConfig{Provider: "openRouter", BaseURL: srv.URL}}
	_, _, err := p.ChatStream(context.Background(), "m", &ModelConfig{}, []ChatMessage{{Role: "user", Content: "x"}}, nil)
	if err == nil || !strings.Contains(err.Error(), "upstream exploded") {
		t.Fatalf("got %v", err)
	}
}

func TestAnthropicAndOllamaUseMaxTokens(t *testing.T) {
	var got map[string]any
	srv := srvCapturing(t, &got, func(w http.ResponseWriter) {
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{{"text": "a"}, {"text": "b"}}, "stop_reason": "end_turn",
			"usage": map[string]int{"input_tokens": 1, "output_tokens": 2},
		})
	})
	defer srv.Close()
	ap := &anthropicProvider{cfg: &ModelConfig{Provider: "anthropic", BaseURL: srv.URL}}
	reply, usage, err := ap.Chat(context.Background(), "c", &ModelConfig{MaxTokens: 4096}, []ChatMessage{{Role: "user", Content: "x"}})
	if err != nil || reply != "ab" || usage.FinishReason != "end_turn" {
		t.Fatalf("reply=%q usage=%+v err=%v", reply, usage, err)
	}
	if got["max_tokens"] != float64(4096) {
		t.Fatalf("anthropic max_tokens = %v", got["max_tokens"])
	}
}

// setupOpenAIProject crea config/models/openRouter/m.json (type openai).
func setupOpenAIProject(t *testing.T, baseURL, extra string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "config", "models", "openRouter")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"type":"openai","base_url":"` + baseURL + `","model":"m","temperature":0.1,` + extra + `}`
	if err := os.WriteFile(filepath.Join(dir, "m.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func newOpenAISession(t *testing.T, root string) *Session {
	t.Helper()
	if err := SetActiveProvider(root, "openRouter"); err != nil {
		t.Fatal(err)
	}
	sess, err := NewSession(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.SetModel("m"); err != nil {
		t.Fatal(err)
	}
	return sess
}

func TestSessionSendStreamEmptyIsAnError(t *testing.T) {
	var got map[string]any
	srv := sseServer(t, &got,
		`data: {"choices":[{"delta":{},"finish_reason":"length"}],"usage":{"prompt_tokens":7,"completion_tokens":512}}`,
		`data: [DONE]`)
	defer srv.Close()
	sess := newOpenAISession(t, setupOpenAIProject(t, srv.URL, `"max_tokens": 512, "stream": true`))
	_, err := sess.SendStream("x", nil)
	var ee *EmptyReplyError
	if !errors.As(err, &ee) || sess.LastUsage.PromptTokens != 7 {
		t.Fatalf("err=%v usage=%+v", err, sess.LastUsage)
	}
}
