package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"mova.local/core"
	"mova.local/models"
)

const (
	cliPlan    = "'use strict';\nfunction normalizarPedido(p) {\n  return { id: p.id };\n}\n"
	cliCobros  = "'use strict';\nfunction generarCobro(plan) {\n  return { llegada: plan.pedido.etaProgramada };\n}\nmodule.exports = { generarCobro };\n"
	cliBlocks  = "Propuesta.\n\n```javascript:src/plan.js::normalizarPedido()\nfunction normalizarPedido(p) {\n  return { id: p.id, etaReal: p.etaReal };\n}\n```\n\n```javascript:src/cobros.js::generarCobro()\nfunction generarCobro(plan) {\n  return { llegada: plan.pedido.etaReal || plan.pedido.etaProgramada };\n}\n```\n"
	cliClassic = "Propuesta:\n```diff\n+ etaReal\n```\n**¿Deseas que aplique estas modificaciones directamente en los archivos fuentes? (Sí/No)**"
)

func applyChatFixture(t *testing.T, applyField string, replies ...string) (root, repo string, sess *models.Session, got *[]string) {
	t.Helper()
	var mu sync.Mutex
	got = &[]string{}
	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct{ Role, Content string } `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		*got = append(*got, body.Messages[len(body.Messages)-1].Content)
		out := "sin guion"
		if len(replies) > 0 {
			out, replies = replies[0], replies[1:]
		}
		mu.Unlock()
		json.NewEncoder(w).Encode(map[string]any{"message": map[string]string{"role": "assistant", "content": out}, "done": true, "prompt_eval_count": 5, "eval_count": 5})
	}))
	t.Cleanup(llm.Close)
	root = t.TempDir()
	repo = filepath.Join(root, "repo")
	_ = os.MkdirAll(filepath.Join(repo, "src"), 0o755)
	_ = os.WriteFile(filepath.Join(repo, "src/plan.js"), []byte(cliPlan), 0o644)
	_ = os.WriteFile(filepath.Join(repo, "src/cobros.js"), []byte(cliCobros), 0o644)
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
	repoJSON, _ := json.Marshal(repo)
	pj := fmt.Sprintf(`{"project":"p","repo":%s,"lang":"es","adapter":"file","default_task":"t",%s
	  "agents":{"domain":"software","use":[]},"skills":{"domain":"software","use":[]},"tasks":{"t":{"prompt":"x"}}}`, repoJSON, applyField)
	_ = os.WriteFile(filepath.Join(dir, "project.json"), []byte(pj), 0o644)
	s, err := models.NewSession(root)
	if err != nil {
		t.Fatal(err)
	}
	s.System = "ctx"
	return root, repo, s, got
}

func readF(p string) string { b, _ := os.ReadFile(p); return string(b) }

// Puerta CHAT: la respuesta trae bloques con destino → Mova pregunta por la terminal → «s» modifica TODO.
func TestChatDoor_AsksThenModifiesAllFilesWithBackup(t *testing.T) {
	root, repo, sess, _ := applyChatFixture(t, `"apply":true,`, cliBlocks)
	ad := core.NewFileAdapter(root)
	proj, _ := ad.GetProject("p")
	runChatTurn(sess, ad, proj, root, "p", "t", "agrega las columnas", bufio.NewScanner(strings.NewReader("s\n")))
	if !strings.Contains(readF(filepath.Join(repo, "src/plan.js")), "etaReal: p.etaReal") || !strings.Contains(readF(filepath.Join(repo, "src/cobros.js")), "etaReal ||") {
		t.Fatal("answering «s» must modify BOTH files")
	}
	if baks, _ := filepath.Glob(filepath.Join(repo, "src", "*.mova-*.bak")); len(baks) != 2 {
		t.Fatalf("2 backups next to the originals expected: %v", baks)
	}
}

func TestChatDoor_NoAndSubsetAndApplyOff(t *testing.T) {
	root, repo, sess, _ := applyChatFixture(t, `"apply":{"enabled":true,"backup":false},`, cliBlocks, cliBlocks)
	ad := core.NewFileAdapter(root)
	proj, _ := ad.GetProject("p")
	runChatTurn(sess, ad, proj, root, "p", "t", "x", bufio.NewScanner(strings.NewReader("n\n")))
	if readF(filepath.Join(repo, "src/plan.js")) != cliPlan {
		t.Fatal("«n» must leave everything untouched")
	}
	runChatTurn(sess, ad, proj, root, "p", "t", "x", bufio.NewScanner(strings.NewReader("2\n")))
	if !strings.Contains(readF(filepath.Join(repo, "src/cobros.js")), "etaReal ||") || strings.Contains(readF(filepath.Join(repo, "src/plan.js")), "etaReal") {
		t.Fatal("«2» must apply only the 2nd change")
	}
	if baks, _ := filepath.Glob(filepath.Join(repo, "src", "*.bak")); len(baks) != 0 {
		t.Fatalf("backup:false → no backups: %v", baks)
	}
	// apply ausente: comportamiento de siempre, ni pregunta ni escribe
	root2, repo2, sess2, _ := applyChatFixture(t, ``, cliBlocks)
	ad2 := core.NewFileAdapter(root2)
	proj2, _ := ad2.GetProject("p")
	runChatTurn(sess2, ad2, proj2, root2, "p", "t", "x", bufio.NewScanner(strings.NewReader("s\n")))
	if readF(filepath.Join(repo2, "src/plan.js")) != cliPlan {
		t.Fatal("apply absent must never write")
	}
}

// El caso real del reporte: prompt clásico (Sí/No) sin destinos. «Sí» ya no se pierde: se pide el formato aplicable.
func TestChatDoor_ClassicYesNoPromptIsConvertedAndApplied(t *testing.T) {
	root, repo, sess, got := applyChatFixture(t, `"apply":true,`, cliClassic, cliBlocks)
	ad := core.NewFileAdapter(root)
	proj, _ := ad.GetProject("p")
	runChatTurn(sess, ad, proj, root, "p", "t", "agrega las columnas", bufio.NewScanner(strings.NewReader("")))
	line := reformatIfNeeded(sess, proj, "t", "Sí")
	if !strings.Contains(line, "bloques de código") {
		t.Fatalf("«Sí» must become the reformat request, got %q", line)
	}
	runChatTurn(sess, ad, proj, root, "p", "t", line, bufio.NewScanner(strings.NewReader("s\n")))
	if !strings.Contains(readF(filepath.Join(repo, "src/plan.js")), "etaReal: p.etaReal") {
		t.Fatalf("the approved proposal must end up written; model saw %v", *got)
	}
}
