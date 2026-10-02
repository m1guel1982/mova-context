package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"mova.local/core"
	"mova.local/models"
)

const httpBlock = "```memory\n**Tarea:** analizar\n**Realizado:** análisis\n**Hallazgos:** `FPVE.js::_parseaVuelo` — TDN_SCT — se pierde\n```"

// Puerta HTTP real: servidor levantado con StartServer, POST /mcp (JSON-RPC) → chat_completion.
func TestHTTPDoor_MemoryCarriesAnalysisToNextTask(t *testing.T) {
	var mu sync.Mutex
	var lastSent string
	llm := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		raw, _ := io.ReadAll(r.Body)
		mu.Lock()
		lastSent = string(raw)
		mu.Unlock()
		json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"role": "assistant", "content": "Informe. " + strings.Repeat("Detalle ACT/SCT. ", 14) + "\n\n" + httpBlock},
			"done":    true, "prompt_eval_count": 10, "eval_count": 5,
		})
	}))
	defer llm.Close()

	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "workflow.md"), []byte("# stub"), 0o644)
	_ = os.MkdirAll(filepath.Join(root, "config", "models", "ollama"), 0o755)
	mc := models.DefaultModelConfig(&models.ModelConfig{Provider: "ollama", Type: "ollama", BaseURL: llm.URL})
	if err := models.SaveModelConfig(root, "ollama", "llama3.1", mc); err != nil {
		t.Fatal(err)
	}
	_ = models.SetActiveProvider(root, "ollama")
	_ = models.SetActiveModel(root, "llama3.1")
	dir := filepath.Join(root, "projects", "p")
	_ = os.MkdirAll(dir, 0o755)
	pj := `{"project":"p","repo":".","lang":"es","adapter":"file","default_task":"analizar","memory":true,
	  "llm_profile":{"type":"local","config":"llama3.1"},"agents":{"domain":"software","use":[]},"skills":{"domain":"software","use":[]},
	  "tasks":{"analizar":{"prompt":"x"},"agregar-columnas":{"prompt":"y"}}}`
	_ = os.WriteFile(filepath.Join(dir, "project.json"), []byte(pj), 0o644)

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	go func() { _ = StartServer(core.NewFileAdapter(root), root, port) }()
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	for i := 0; i < 100; i++ {
		if r, err := nethttp.Get(base + "/health"); err == nil {
			r.Body.Close()
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	call := func(task string) string {
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call",
			"params": map[string]any{"name": "chat_completion", "arguments": map[string]any{"project": "p", "task": task, "message": "trabaja"}}})
		resp, err := nethttp.Post(base+"/mcp", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return string(b)
	}
	if out := call("analizar"); !strings.Contains(out, "entrada(s) guardada(s)") {
		t.Fatalf("HTTP reply lacks the memory status:\n%s", out)
	}
	mem, err := os.ReadFile(filepath.Join(dir, "memory.md"))
	if err != nil || !strings.Contains(string(mem), "TDN_SCT — se pierde") {
		t.Fatalf("memory.md via HTTP: %v\n%s", err, mem)
	}
	call("agregar-columnas") // otra petición HTTP = sesión nueva
	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(lastSent, "MEMORY") || !strings.Contains(lastSent, "TDN_SCT") {
		t.Fatalf("2nd HTTP request's LLM call does not carry the 1st task's synthesis:\n%s", lastSent)
	}
}
