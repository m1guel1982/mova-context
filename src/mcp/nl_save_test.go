// nl_save_test.go — regresión del Bug A reportado en QA: "crea un
// archivo hola.txt con el texto hello y otro chat.txt con chucha" hacía
// que Mova imprimiera el JSON de apply_file_changes ("[NL] Detected
// file creation intent...") pero NUNCA escribiera nada en disco. La
// causa real era que el bucle de tool-calling sólo corría cuando
// project.json tenía "tools": {"enabled": true} — ver
// sendForcedFileChangesMCP (nl_save.go) y su equivalente CLI
// (cli/chat_helpers.go's sendWithForcedFileChanges). Este test simula
// un modelo (vía un servidor Ollama falso) que responde exactamente
// como el bug reportaba — un objeto JSON "changes" sin ningún envoltorio
// <<<MOVA_TOOL_CALL>>> (la forma que un modelo chico/local suele
// producir, ver parseBareToolShape) — y comprueba que los archivos
// terminan realmente en disco, con el contenido correcto POR ARCHIVO
// (no el mismo texto repetido en todos, el otro bug histórico que este
// mismo camino tenía).
package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mova.local/core"
	"mova.local/documents"
	"mova.local/models"
)

// fakeToolCallingOllama simula un modelo que en su PRIMER turno responde
// con el JSON crudo de apply_file_changes (tal como el bug reportado) y,
// en cualquier turno posterior (la continuación TOOL_RESULT que
// sendForcedFileChangesMCP/sendWithForcedFileChanges envían después de
// aplicar los cambios), responde con texto plano — así el bucle de
// herramientas termina en la segunda vuelta, como con un modelo real.
func fakeToolCallingOllama(t *testing.T, firstReply string) *httptest.Server {
	t.Helper()
	calls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		calls++
		reply := "Listo, los archivos fueron creados."
		if calls == 1 {
			reply = firstReply
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"message":           map[string]string{"role": "assistant", "content": reply},
			"done":              true,
			"prompt_eval_count": 10,
			"eval_count":        5,
		})
	})
	return httptest.NewServer(mux)
}

// setupMCPProject arma una raíz mínima de proyecto Mova con "ollama"
// como proveedor activo apuntando al servidor falso — igual que
// models.setupProject (models_test.go), copiado aquí porque ese helper
// no está exportado desde el paquete models.
func setupMCPProject(t *testing.T, baseURL string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "workflow.md"), []byte("# stub"), 0644); err != nil {
		t.Fatalf("WriteFile workflow.md: %v", err)
	}
	dir := filepath.Join(root, "config", "models", "ollama")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	mc := models.DefaultModelConfig(&models.ModelConfig{Provider: "ollama", Type: "ollama", BaseURL: baseURL})
	if err := models.SaveModelConfig(root, "ollama", "llama3.1", mc); err != nil {
		t.Fatalf("SaveModelConfig: %v", err)
	}
	if err := models.SetActiveProvider(root, "ollama"); err != nil {
		t.Fatalf("SetActiveProvider: %v", err)
	}
	if err := models.SetActiveModel(root, "llama3.1"); err != nil {
		t.Fatalf("SetActiveModel: %v", err)
	}
	return root
}

func TestApplyNaturalLanguageFilesViaChanges_WritesRealFiles(t *testing.T) {
	firstReply := `{"changes": [
		{"path": "hola.txt", "action": "create", "content": "hello"},
		{"path": "chat.txt", "action": "create", "content": "chucha"}
	]}`
	srv := fakeToolCallingOllama(t, firstReply)
	defer srv.Close()

	root := setupMCPProject(t, srv.URL)
	sess, err := models.NewSession(root)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	adapter := core.NewFileAdapter(root)

	var statusLog strings.Builder
	intent := documents.SaveIntent{Files: []string{"hola.txt", "chat.txt"}}
	if _, err := applyNaturalLanguageFilesViaChanges(&statusLog, sess, adapter, root, intent,
		"crea un archivo hola.txt con el texto hello y otro chat.txt con chucha"); err != nil {
		t.Fatalf("applyNaturalLanguageFilesViaChanges: %v (statusLog: %s)", err, statusLog.String())
	}

	holaBytes, err := os.ReadFile(filepath.Join(root, "hola.txt"))
	if err != nil {
		t.Fatalf("hola.txt nunca se creó en disco: %v (statusLog: %s)", err, statusLog.String())
	}
	if string(holaBytes) != "hello" {
		t.Fatalf("contenido de hola.txt = %q, se esperaba %q", string(holaBytes), "hello")
	}

	chatBytes, err := os.ReadFile(filepath.Join(root, "chat.txt"))
	if err != nil {
		t.Fatalf("chat.txt nunca se creó en disco: %v (statusLog: %s)", err, statusLog.String())
	}
	if string(chatBytes) != "chucha" {
		t.Fatalf("contenido de chat.txt = %q, se esperaba %q", string(chatBytes), "chucha")
	}

	if !strings.Contains(statusLog.String(), "hola.txt") {
		t.Fatalf("se esperaba que el statusLog mencionara hola.txt, got %q", statusLog.String())
	}
}

// TestApplyNaturalLanguageFilesViaChanges_SingleFile cubre el caso más
// simple del reporte original ("crea un archivo hola.txt con el texto
// hello en ...") — un solo archivo, para descartar que el bug dependiera
// de tener 2+ archivos en el mismo mensaje.
func TestApplyNaturalLanguageFilesViaChanges_SingleFile(t *testing.T) {
	firstReply := `{"changes": [{"path": "hola.txt", "action": "create", "content": "hello"}]}`
	srv := fakeToolCallingOllama(t, firstReply)
	defer srv.Close()

	root := setupMCPProject(t, srv.URL)
	sess, err := models.NewSession(root)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	adapter := core.NewFileAdapter(root)

	var statusLog strings.Builder
	intent := documents.SaveIntent{Files: []string{"hola.txt"}}
	if _, err := applyNaturalLanguageFilesViaChanges(&statusLog, sess, adapter, root, intent,
		"crea un archivo hola.txt con el texto hello"); err != nil {
		t.Fatalf("applyNaturalLanguageFilesViaChanges: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(root, "hola.txt"))
	if err != nil {
		t.Fatalf("hola.txt nunca se creó en disco: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("contenido = %q, se esperaba %q", string(got), "hello")
	}
}

// TestBuildApplyFileChangesInstruction_TeachesWireProtocol comprueba
// que la instrucción compartida (usada por CLI y MCP) siempre incluye
// el envoltorio <<<MOVA_TOOL_CALL>>> exacto — necesario porque, con
// "tools.enabled": false, el modelo NUNCA ve ese protocolo en su system
// prompt (ToolsSystemPrompt sólo lo agrega cuando tools está prendido),
// así que esta instrucción es la ÚNICA fuente que se lo enseña.
func TestBuildApplyFileChangesInstruction_TeachesWireProtocol(t *testing.T) {
	got := BuildApplyFileChangesInstruction("crea hola.txt con 'hola'", []string{"hola.txt"})
	if !strings.Contains(got, ToolCallStart) || !strings.Contains(got, ToolCallEnd) {
		t.Fatalf("expected the instruction to teach the %s/%s wrapper, got %q", ToolCallStart, ToolCallEnd, got)
	}
	if !strings.Contains(got, `"apply_file_changes"`) {
		t.Fatalf("expected the instruction to name apply_file_changes explicitly, got %q", got)
	}
	if !strings.Contains(got, "hola.txt") {
		t.Fatalf("expected the instruction to list the exact path, got %q", got)
	}
}
