package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"mova.local/models"
)

const jsPlan = "'use strict';\nfunction normalizarPedido(p) {\n  return { id: p.id };\n}\n"
const jsCobros = "'use strict';\nfunction generarCobro(plan) {\n  return { llegada: plan.pedido.etaProgramada };\n}\nmodule.exports = { generarCobro };\n"

const labeled = "Propuesta de cambios mínimos.\n\n```javascript:src/plan.js::normalizarPedido()\nfunction normalizarPedido(p) {\n  return { id: p.id, etaReal: p.etaReal };\n}\n```\n\n```javascript:src/cobros.js::generarCobro()\nfunction generarCobro(plan) {\n  return { llegada: plan.pedido.etaReal || plan.pedido.etaProgramada };\n}\n```\n"

// Respuesta del modelo al prompt CLÁSICO: pregunta (Sí/No) y SIN bloques con destino.
const classic = "Propuesta:\n```diff\n- return { id: p.id };\n+ return { id: p.id, etaReal: p.etaReal };\n```\n**¿Deseas que aplique estas modificaciones directamente en los archivos fuentes? (Sí/No)**"

type scripted struct {
	mu       sync.Mutex
	replies  []string
	received []string
}

func (s *scripted) server(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body struct {
			Messages []struct{ Role, Content string } `json:"messages"`
		}
		_ = json.Unmarshal(raw, &body)
		s.mu.Lock()
		last := ""
		if n := len(body.Messages); n > 0 {
			last = body.Messages[n-1].Content
		}
		s.received = append(s.received, last)
		out := "sin guion"
		if len(s.replies) > 0 {
			out, s.replies = s.replies[0], s.replies[1:]
		}
		s.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]any{"message": map[string]string{"role": "assistant", "content": out}, "done": true, "prompt_eval_count": 5, "eval_count": 5})
	}))
}

func applyProject(t *testing.T, baseURL, applyField string) (root, repo string) {
	t.Helper()
	root = t.TempDir()
	repo = filepath.Join(root, "repo")
	_ = os.MkdirAll(filepath.Join(repo, "src"), 0o755)
	_ = os.WriteFile(filepath.Join(repo, "src/plan.js"), []byte(jsPlan), 0o644)
	_ = os.WriteFile(filepath.Join(repo, "src/cobros.js"), []byte(jsCobros), 0o644)
	_ = os.WriteFile(filepath.Join(root, "workflow.md"), []byte("# stub"), 0o644)
	_ = os.MkdirAll(filepath.Join(root, "config", "models", "ollama"), 0o755)
	mc := models.DefaultModelConfig(&models.ModelConfig{Provider: "ollama", Type: "ollama", BaseURL: baseURL})
	if err := models.SaveModelConfig(root, "ollama", "llama3.1", mc); err != nil {
		t.Fatal(err)
	}
	_ = models.SetActiveProvider(root, "ollama")
	_ = models.SetActiveModel(root, "llama3.1")
	dir := filepath.Join(root, "projects", "p")
	_ = os.MkdirAll(dir, 0o755)
	repoJSON, _ := json.Marshal(repo)
	pj := fmt.Sprintf(`{"project":"p","repo":%s,"lang":"es","adapter":"file","default_task":"agregar-columnas","llm_profile":{"type":"local","config":"llama3.1"},
	  "agents":{"domain":"software","use":[]},"skills":{"domain":"software","use":[]},%s
	  "tasks":{"agregar-columnas":{"prompt":"y"}}}`, repoJSON, applyField)
	_ = os.WriteFile(filepath.Join(dir, "project.json"), []byte(pj), 0o644)
	return root, repo
}

func rd(p string) string { b, _ := os.ReadFile(p); return string(b) }

// Puerta MCP/HTTP, prompt con bloques etiquetados: pregunta → «sí» (otra llamada) → archivos modificados + respaldo al lado.
func TestMCP_ProposalQuestionThenYesModifiesFiles(t *testing.T) {
	sc := &scripted{replies: []string{labeled}}
	srv := sc.server(t)
	defer srv.Close()
	root, repo := applyProject(t, srv.URL, `"apply":true,`)

	out := callTool(t, root, "chat_completion", map[string]any{"project": "p", "task": "agregar-columnas", "message": "agrega las columnas"})
	for _, must := range []string{"2 cambio(s)", "src/plan.js::normalizarPedido()", "src/cobros.js::generarCobro()", "Todavía NO"} {
		if !strings.Contains(out, must) {
			t.Fatalf("reply lacks %q:\n%s", must, out)
		}
	}
	if rd(filepath.Join(repo, "src/plan.js")) != jsPlan {
		t.Fatal("the question must not modify anything")
	}
	if _, err := os.Stat(filepath.Join(root, "projects", "p", "pending-changes.json")); err != nil {
		t.Fatal("proposal must be stored as pending")
	}
	n := len(sc.received)
	out = callTool(t, root, "chat_completion", map[string]any{"project": "p", "task": "agregar-columnas", "message": "sí"})
	if len(sc.received) != n {
		t.Fatal("answering must NOT call the model again")
	}
	if !strings.Contains(out, "2 aplicado(s), 0 omitido(s)") {
		t.Fatalf("result:\n%s", out)
	}
	if !strings.Contains(rd(filepath.Join(repo, "src/plan.js")), "etaReal: p.etaReal") || !strings.Contains(rd(filepath.Join(repo, "src/cobros.js")), "etaReal ||") {
		t.Fatal("files were not modified")
	}
	baks, _ := filepath.Glob(filepath.Join(repo, "src", "*.mova-*.bak"))
	if len(baks) != 2 {
		t.Fatalf("expected 2 backups next to the originals: %v", baks)
	}
}

// El caso real del reporte: prompt clásico (Sí/No) sin destinos → «sí» → Mova pide los bloques → pregunta → «1» → se modifica.
func TestMCP_ClassicYesNoPromptNowReallyModifiesFiles(t *testing.T) {
	sc := &scripted{replies: []string{classic, labeled}}
	srv := sc.server(t)
	defer srv.Close()
	root, repo := applyProject(t, srv.URL, `"apply":{"enabled":true,"backup":true},`)

	callTool(t, root, "chat_completion", map[string]any{"project": "p", "task": "agregar-columnas", "message": "agrega las columnas"})
	history := []any{ // como llega por JSON-RPC
		map[string]any{"role": "user", "content": "agrega las columnas"},
		map[string]any{"role": "assistant", "content": classic},
	}
	out := callTool(t, root, "chat_completion", map[string]any{"project": "p", "task": "agregar-columnas", "message": "Sí", "history": history})
	if got := sc.received[len(sc.received)-1]; !strings.Contains(got, "bloques de código") || !strings.Contains(got, "NO afirmes") {
		t.Fatalf("«Sí» must be turned into the reformat request, model got: %q", got)
	}
	if !strings.Contains(out, "2 cambio(s)") || rd(filepath.Join(repo, "src/plan.js")) != jsPlan {
		t.Fatalf("after reformat Mova must ASK, not write:\n%s", out)
	}
	callTool(t, root, "chat_completion", map[string]any{"project": "p", "task": "agregar-columnas", "message": "1"})
	if !strings.Contains(rd(filepath.Join(repo, "src/plan.js")), "etaReal: p.etaReal") {
		t.Fatal("#1 must be applied")
	}
	if strings.Contains(rd(filepath.Join(repo, "src/cobros.js")), "etaReal") {
		t.Fatal("#2 was not selected")
	}
}

func TestMCP_ApplyOffOrNoBlocksChangesNothing(t *testing.T) {
	sc := &scripted{replies: []string{labeled}}
	srv := sc.server(t)
	defer srv.Close()
	root, repo := applyProject(t, srv.URL, ``)
	out := callTool(t, root, "chat_completion", map[string]any{"project": "p", "task": "agregar-columnas", "message": "agrega"})
	if strings.Contains(out, "Todavía NO") || rd(filepath.Join(repo, "src/plan.js")) != jsPlan {
		t.Fatalf("apply absent must keep the old read-only behavior:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(root, "projects", "p", "pending-changes.json")); err == nil {
		t.Fatal("no pending file when apply is off")
	}
}

func TestMCP_ExplicitApplyChangesArgAndNo(t *testing.T) {
	sc := &scripted{replies: []string{labeled, labeled}}
	srv := sc.server(t)
	defer srv.Close()
	root, repo := applyProject(t, srv.URL, `"apply":{"enabled":true,"backup":false},`)
	callTool(t, root, "chat_completion", map[string]any{"project": "p", "task": "agregar-columnas", "message": "x"})
	out := callTool(t, root, "chat_completion", map[string]any{"project": "p", "message": "ignorado", "apply_changes": "none"})
	if !strings.Contains(out, "No se modificó ningún archivo") || rd(filepath.Join(repo, "src/plan.js")) != jsPlan {
		t.Fatalf("none must discard:\n%s", out)
	}
	callTool(t, root, "chat_completion", map[string]any{"project": "p", "task": "agregar-columnas", "message": "x"})
	callTool(t, root, "chat_completion", map[string]any{"project": "p", "apply_changes": "all"})
	if !strings.Contains(rd(filepath.Join(repo, "src/cobros.js")), "etaReal ||") {
		t.Fatal("apply_changes:all must apply")
	}
	if baks, _ := filepath.Glob(filepath.Join(repo, "src", "*.bak")); len(baks) != 0 {
		t.Fatalf("backup:false must not create backups: %v", baks)
	}
}
