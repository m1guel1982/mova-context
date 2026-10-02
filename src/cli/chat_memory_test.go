package main

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mova.local/core"
	"mova.local/models"
)

const chatBlock = "```memory\n**Tarea:** analizar\n**Realizado:** análisis\n**Hallazgos:** `Programacion.js::saveAtencion` — ONB_SCT — se pierde\n```"

func chatFixture(t *testing.T, memField string) (root string, sess *models.Session) {
	t.Helper()
	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"role": "assistant", "content": "Informe. " + strings.Repeat("Detalle ACT/SCT. ", 14) + "\n\n" + chatBlock},
			"done":    true, "prompt_eval_count": 10, "eval_count": 5,
		})
	}))
	t.Cleanup(llm.Close)
	root = t.TempDir()
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
	pj := `{"project":"p","repo":".","lang":"es","adapter":"file","default_task":"analizar"` + memField + `,
	  "agents":{"domain":"software","use":[]},"skills":{"domain":"software","use":[]},
	  "tasks":{"analizar":{"prompt":"x"},"agregar-columnas":{"prompt":"y"}}}`
	_ = os.WriteFile(filepath.Join(dir, "project.json"), []byte(pj), 0o644)
	s, err := models.NewSession(root)
	if err != nil {
		t.Fatal(err)
	}
	s.System = "contexto"
	return root, s
}

// Puerta Chat: el turno registra la síntesis; repetirla no duplica; con memory apagado no escribe.
func TestChatTurnRecordsMemoryAndDedupes(t *testing.T) {
	root, sess := chatFixture(t, `,"memory":true`)
	ad := core.NewFileAdapter(root)
	proj, _ := ad.GetProject("p")
	scan := bufio.NewScanner(strings.NewReader(""))
	if !runChatTurn(sess, ad, proj, root, "p", "analizar", "revisar", scan) {
		t.Fatal("first turn must record memory")
	}
	mem, _ := os.ReadFile(filepath.Join(root, "projects", "p", "memory.md"))
	if !strings.Contains(string(mem), "ONB_SCT — se pierde") || !strings.Contains(string(mem), "— tarea: analizar") {
		t.Fatalf("memory.md:\n%s", mem)
	}
	if runChatTurn(sess, ad, proj, root, "p", "analizar", "otra vez", scan) {
		t.Fatal("an identical synthesis must not be recorded twice")
	}
	if strings.Count(readMem(root), "sha=") != 1 {
		t.Fatalf("duplicated entry:\n%s", readMem(root))
	}

	rootOff, sessOff := chatFixture(t, ``)
	adOff := core.NewFileAdapter(rootOff)
	projOff, _ := adOff.GetProject("p")
	if runChatTurn(sessOff, adOff, projOff, rootOff, "p", "analizar", "revisar", scan) {
		t.Fatal("memory absent → nothing recorded")
	}
	if _, err := os.Stat(filepath.Join(rootOff, "projects", "p", "memory.md")); err == nil {
		t.Fatal("memory.md must not exist when memory is off")
	}
}

func readMem(root string) string {
	b, _ := os.ReadFile(filepath.Join(root, "projects", "p", "memory.md"))
	return string(b)
}

// Tras escribir su propia memoria, el chat NO debe recargar el contexto en el turno siguiente
// (perdería el caché del proveedor); pero una escritura de OTRA puerta sí debe detectarse.
func TestChatResignAvoidsSelfReloadButSeesForeignWrites(t *testing.T) {
	root, sess := chatFixture(t, `,"memory":true`)
	ad := core.NewFileAdapter(root)
	proj, _ := ad.GetProject("p")
	scan := bufio.NewScanner(strings.NewReader(""))
	runChatTurn(sess, ad, proj, root, "p", "analizar", "revisar", scan)
	sig := resignContext(root, "p", "analizar", proj)
	var msgs []string
	emit := func(s string) { msgs = append(msgs, s) }
	_, _, sig2 := refreshProjectContext(root, "p", "analizar", sess, proj, ad, sig, emit)
	if len(msgs) != 0 || sig2 != sig {
		t.Fatalf("own memory write must not trigger a reload: %v", msgs)
	}
	_ = ad.AppendMemory("p", "## 2026-10-02 — tarea: otra\n<!-- mova:entry task=otra sha=aaaaaaaaaaaa -->\nescrito por MCP")
	msgs = nil
	refreshProjectContext(root, "p", "analizar", sess, proj, ad, sig, emit)
	if len(msgs) == 0 {
		t.Fatal("a memory write from another door must reload the context")
	}
}
