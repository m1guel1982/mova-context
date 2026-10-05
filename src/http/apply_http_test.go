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
	"testing"
	"time"

	"mova.local/core"
	"mova.local/models"
)

const hBlocks = "Propuesta.\n\n```javascript:src/plan.js::normalizarPedido()\nfunction normalizarPedido(p) {\n  return { id: p.id, etaReal: p.etaReal };\n}\n```\n\n```javascript:src/cobros.js::generarCobro()\nfunction generarCobro(plan) {\n  return { llegada: plan.pedido.etaReal || plan.pedido.etaProgramada };\n}\n```\n"

// Puerta HTTP real (StartServer + POST /mcp): pregunta → «sí» en otra petición → archivos modificados + respaldo.
func TestHTTPDoor_ProposalThenYesModifiesFiles(t *testing.T) {
	llm := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		json.NewEncoder(w).Encode(map[string]any{"message": map[string]string{"role": "assistant", "content": hBlocks}, "done": true, "prompt_eval_count": 5, "eval_count": 5})
	}))
	defer llm.Close()
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	_ = os.MkdirAll(filepath.Join(repo, "src"), 0o755)
	plan := "'use strict';\nfunction normalizarPedido(p) {\n  return { id: p.id };\n}\n"
	cobros := "'use strict';\nfunction generarCobro(plan) {\n  return { llegada: plan.pedido.etaProgramada };\n}\n"
	_ = os.WriteFile(filepath.Join(repo, "src/plan.js"), []byte(plan), 0o644)
	_ = os.WriteFile(filepath.Join(repo, "src/cobros.js"), []byte(cobros), 0o644)
	_ = os.WriteFile(filepath.Join(root, "workflow.md"), []byte("# stub"), 0o644)
	_ = os.MkdirAll(filepath.Join(root, "config", "models", "ollama"), 0o755)
	mc := models.DefaultModelConfig(&models.ModelConfig{Provider: "ollama", Type: "ollama", BaseURL: llm.URL})
	_ = models.SaveModelConfig(root, "ollama", "llama3.1", mc)
	_ = models.SetActiveProvider(root, "ollama")
	_ = models.SetActiveModel(root, "llama3.1")
	dir := filepath.Join(root, "projects", "p")
	_ = os.MkdirAll(dir, 0o755)
	repoJSON, _ := json.Marshal(repo)
	pj := fmt.Sprintf(`{"project":"p","repo":%s,"lang":"es","adapter":"file","default_task":"t","apply":{"enabled":true,"backup":true},
	  "llm_profile":{"type":"local","config":"llama3.1"},"agents":{"domain":"software","use":[]},"skills":{"domain":"software","use":[]},"tasks":{"t":{"prompt":"x"}}}`, repoJSON)
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
	call := func(args map[string]any) string {
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "chat_completion", "arguments": args}})
		resp, err := nethttp.Post(base+"/mcp", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return string(b)
	}
	out := call(map[string]any{"project": "p", "task": "t", "message": "agrega las columnas"})
	if !strings.Contains(out, "2 cambio(s)") || !strings.Contains(out, "Todavía NO") {
		t.Fatalf("HTTP reply must carry the question:\n%s", out)
	}
	if b, _ := os.ReadFile(filepath.Join(repo, "src/plan.js")); string(b) != plan {
		t.Fatal("asking must not write")
	}
	out = call(map[string]any{"project": "p", "task": "t", "message": "sí"})
	if !strings.Contains(out, "2 aplicado(s), 0 omitido(s)") {
		t.Fatalf("HTTP answer:\n%s", out)
	}
	if b, _ := os.ReadFile(filepath.Join(repo, "src/cobros.js")); !strings.Contains(string(b), "etaReal ||") {
		t.Fatal("HTTP «sí» must modify the files")
	}
	if baks, _ := filepath.Glob(filepath.Join(repo, "src", "*.mova-*.bak")); len(baks) != 2 {
		t.Fatalf("2 backups next to originals: %v", baks)
	}
}
