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

const doorBlock = "```memory\n**Tarea:** analizar\n**Realizado:** análisis ACT/SCT\n**Hallazgos:** `Programacion.js::_mapVueloHistorialCobros` — ONB_ACT — se pierde\n**Datos clave:** TDN_ACT, ONB_ACT\n```"

// recorder: LLM falso (Ollama) que guarda los mensajes recibidos y responde con una síntesis.
type recorder struct {
	mu   sync.Mutex
	last []string
}

func (r *recorder) server(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		raw, _ := io.ReadAll(req.Body)
		var body struct {
			Messages []struct{ Role, Content string } `json:"messages"`
		}
		_ = json.Unmarshal(raw, &body)
		r.mu.Lock()
		r.last = nil
		for _, m := range body.Messages {
			r.last = append(r.last, m.Role+": "+m.Content)
		}
		r.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"role": "assistant", "content": "Informe técnico. " + strings.Repeat("Detalle ACT/SCT. ", 14) + "\n\n" + doorBlock},
			"done":    true, "prompt_eval_count": 10, "eval_count": 5,
		})
	}))
}

func (r *recorder) sent() string { r.mu.Lock(); defer r.mu.Unlock(); return strings.Join(r.last, "\n") }

func memProject(t *testing.T, baseURL, memField string) (root string) {
	t.Helper()
	root = t.TempDir()
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
	pj := fmt.Sprintf(`{"project":"p","repo":".","lang":"es","adapter":"file","default_task":"analizar","llm_profile":{"type":"local","config":"llama3.1"},
	  "agents":{"domain":"software","use":[]},"skills":{"domain":"software","use":[]},
	  "tasks":{"analizar":{"prompt":"x"},"agregar-columnas":{"prompt":"y"}}%s}`, memField)
	if err := os.WriteFile(filepath.Join(dir, "project.json"), []byte(pj), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// Puerta MCP/HTTP (mcp.Process es el punto de entrada de stdio Y de http/server.go):
// tarea 1 deja su síntesis y la tarea 2, en una llamada/sesión NUEVA, la recibe.
func TestChatCompletion_MemoryCarriesAnalysisToNextTask(t *testing.T) {
	rec := &recorder{}
	srv := rec.server(t)
	defer srv.Close()
	root := memProject(t, srv.URL, `,"memory":true`)

	out := callTool(t, root, "chat_completion", map[string]any{"project": "p", "task": "analizar", "message": "revisar"})
	if !strings.Contains(out, "[Memory] 1 entrada(s) guardada(s)") {
		t.Fatalf("status line missing:\n%s", out)
	}
	mem, err := os.ReadFile(filepath.Join(root, "projects", "p", "memory.md"))
	if err != nil || !strings.Contains(string(mem), "ONB_ACT — se pierde") || !strings.Contains(string(mem), "— tarea: analizar") {
		t.Fatalf("memory.md not written as expected: %v\n%s", err, mem)
	}
	callTool(t, root, "chat_completion", map[string]any{"project": "p", "task": "agregar-columnas", "message": "agrega las columnas"})
	sent := rec.sent()
	if !strings.Contains(sent, "## MEMORY") || !strings.Contains(sent, "ONB_ACT — se pierde") {
		t.Fatalf("the 2nd task's LLM request does not carry the 1st task's synthesis:\n%s", sent)
	}
	// El mismo resultado, ahora sin tarea (todas): también la ve y no duplica la entrada idéntica.
	callTool(t, root, "chat_completion", map[string]any{"project": "p", "message": "continúa"})
	mem2, _ := os.ReadFile(filepath.Join(root, "projects", "p", "memory.md"))
	if strings.Count(string(mem2), "sha=") != 1 {
		t.Fatalf("identical synthesis must not be duplicated:\n%s", mem2)
	}
}

func TestChatCompletion_MemoryOffWritesNothing(t *testing.T) {
	for _, field := range []string{``, `,"memory":false`} {
		rec := &recorder{}
		srv := rec.server(t)
		root := memProject(t, srv.URL, field)
		out := callTool(t, root, "chat_completion", map[string]any{"project": "p", "task": "analizar", "message": "revisar"})
		srv.Close()
		if strings.Contains(out, "[Memory]") {
			t.Fatalf("%q: no memory status expected:\n%s", field, out)
		}
		if _, err := os.Stat(filepath.Join(root, "projects", "p", "memory.md")); err == nil {
			t.Fatalf("%q: memory.md must not be created", field)
		}
	}
}

func TestChatCompletion_MemoryCustomPath(t *testing.T) {
	rec := &recorder{}
	srv := rec.server(t)
	defer srv.Close()
	ext := filepath.Join(t.TempDir(), "mnt", "memoria-compartida")
	root := memProject(t, srv.URL, fmt.Sprintf(`,"memory":%q`, filepath.ToSlash(ext)+"/"))
	callTool(t, root, "chat_completion", map[string]any{"project": "p", "task": "analizar", "message": "revisar"})
	if b, err := os.ReadFile(filepath.Join(ext, "memory.md")); err != nil || !strings.Contains(string(b), "ONB_ACT") {
		t.Fatalf("memory must be written to the configured path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "projects", "p", "memory.md")); err == nil {
		t.Fatal("must not write the default location")
	}
}

// Modo delegado (sin llm_profile): Mova no ve la respuesta del anfitrión; le pide registrar su
// síntesis con save_memory, que respeta el campo "memory" y usa el mismo formato/dedupe.
func TestDelegatedMode_SaveMemoryHonorsFlagAndHint(t *testing.T) {
	rec := &recorder{}
	srv := rec.server(t)
	defer srv.Close()
	root := memProject(t, srv.URL, `,"memory":true`)
	// quitar el llm_profile → modo delegado
	pj := filepath.Join(root, "projects", "p", "project.json")
	b, _ := os.ReadFile(pj)
	_ = os.WriteFile(pj, []byte(strings.Replace(string(b), `"llm_profile":{"type":"local","config":"llama3.1"},`, "", 1)), 0o644)

	out := callTool(t, root, "chat_completion", map[string]any{"project": "p", "task": "analizar", "message": "revisar"})
	if !strings.Contains(out, "save_memory") || !strings.Contains(out, `task="analizar"`) {
		t.Fatalf("delegated reply must tell the host to call save_memory:\n%s", out)
	}
	res := callTool(t, root, "save_memory", map[string]any{"project": "p", "task": "analizar", "entry": doorBlock})
	if !strings.Contains(res, "memory saved") {
		t.Fatalf("save_memory: %s", res)
	}
	if again := callTool(t, root, "save_memory", map[string]any{"project": "p", "task": "analizar", "entry": doorBlock}); !strings.Contains(again, "sin cambios") {
		t.Fatalf("identical entry must be a no-op: %s", again)
	}
	mem, _ := os.ReadFile(filepath.Join(root, "projects", "p", "memory.md"))
	if strings.Count(string(mem), "sha=") != 1 || !strings.Contains(string(mem), "ONB_ACT") {
		t.Fatalf("memory.md:\n%s", mem)
	}

	// memory apagado → ni se guarda ni se pide guardar
	root2 := memProject(t, srv.URL, ``)
	if r := callTool(t, root2, "save_memory", map[string]any{"project": "p", "entry": doorBlock}); !strings.Contains(r, "desactivada") {
		t.Fatalf("memory off must say so: %s", r)
	}
	if _, err := os.Stat(filepath.Join(root2, "projects", "p", "memory.md")); err == nil {
		t.Fatal("memory off: nothing may be written")
	}
}
