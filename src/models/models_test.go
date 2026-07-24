package models

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeOllama simula lo mínimo indispensable de la API de Ollama para poder
// probar el paquete models sin depender de un servidor real.
func fakeOllama(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		if streamFlag, _ := req["stream"].(bool); streamFlag {
			w.Header().Set("Content-Type", "application/x-ndjson")
			fw := bufio.NewWriter(w)
			words := []string{"hola", " desde", " " + req["model"].(string)}
			for _, word := range words {
				line, _ := json.Marshal(map[string]any{
					"message": map[string]string{"role": "assistant", "content": word},
					"done":    false,
				})
				fw.Write(line)
				fw.WriteString("\n")
			}
			last, _ := json.Marshal(map[string]any{
				"message":           map[string]string{"role": "assistant", "content": ""},
				"done":              true,
				"prompt_eval_count": 42,
				"eval_count":        7,
			})
			fw.Write(last)
			fw.WriteString("\n")
			fw.Flush()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"message":           map[string]string{"role": "assistant", "content": "hola desde " + req["model"].(string)},
			"done":              true,
			"prompt_eval_count": 42,
			"eval_count":        7,
		})
	})

	mux.HandleFunc("/api/pull", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		fw := bufio.NewWriter(w)
		lines := []string{
			`{"status":"pulling manifest"}`,
			`{"status":"downloading","completed":50,"total":100}`,
			`{"status":"downloading","completed":100,"total":100}`,
			`{"status":"success"}`,
		}
		for _, l := range lines {
			fw.WriteString(l + "\n")
		}
		fw.Flush()
	})

	mux.HandleFunc("/api/delete", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{})
	})

	mux.HandleFunc("/api/tags", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{
				{"name": "llama3.1:8b", "size": 4700000000, "modified_at": "2026-01-01T00:00:00Z"},
			},
		})
	})

	return httptest.NewServer(mux)
}

// setupProject crea un árbol mínimo config/models/ollama/llama3.1.json (UN
// solo archivo, conexión + parámetros — ver types.go) dentro de un
// directorio temporal que actúa como raíz del proyecto Mova.
func setupProject(t *testing.T, baseURL string) string {
	t.Helper()
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "workflow.md"), []byte("# stub"), 0644)

	dir := filepath.Join(root, "config", "models", "ollama")
	os.MkdirAll(dir, 0755)

	mc := DefaultModelConfig(&ModelConfig{Provider: "ollama", Type: "ollama", BaseURL: baseURL})
	mc.Version = "3.1"
	if err := SaveModelConfig(root, "ollama", "llama3.1", mc); err != nil {
		t.Fatalf("SaveModelConfig: %v", err)
	}

	return root
}

func TestGetModelCarriesConnectionAndInference(t *testing.T) {
	srv := fakeOllama(t)
	defer srv.Close()
	root := setupProject(t, srv.URL)

	mc, err := DefaultCache.GetModel(root, "ollama", "llama3.1")
	if err != nil {
		t.Fatalf("GetModel: %v", err)
	}
	// un solo archivo trae AMBAS cosas: conexión (base_url) y parámetros
	// de inferencia (version) — ya no hay un config.json aparte.
	if mc.ResolvedBaseURL() != srv.URL {
		t.Fatalf("base url = %q, want %q", mc.ResolvedBaseURL(), srv.URL)
	}
	if mc.Version != "3.1" {
		t.Fatalf("version = %q, want %q", mc.Version, "3.1")
	}
}

func TestSetActiveProviderAndModel(t *testing.T) {
	srv := fakeOllama(t)
	defer srv.Close()
	root := setupProject(t, srv.URL)

	if err := SetActiveProvider(root, "ollama"); err != nil {
		t.Fatalf("SetActiveProvider: %v", err)
	}
	if err := SetActiveModel(root, "llama3.1"); err != nil {
		t.Fatalf("SetActiveModel: %v", err)
	}
	state, err := GetActiveState(root)
	if err != nil {
		t.Fatalf("GetActiveState: %v", err)
	}
	if state.Provider != "ollama" || state.Config != "llama3.1" {
		t.Fatalf("estado activo inesperado: %+v", state)
	}

	if err := SetActiveProvider(root, "no-existe"); err == nil {
		t.Fatalf("se esperaba error para proveedor inexistente (directorio config/models/no-existe/ no existe)")
	}
}

// TestSetActiveProviderWithoutAnyModelYet confirma que ahora SÍ se puede
// elegir un proveedor antes de tener ningún modelo configurado (ya no
// depende de un config.json separado) — solo hace falta que la carpeta
// config/models/<provider>/ exista.
func TestSetActiveProviderWithoutAnyModelYet(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "config", "models", "lmstudio"), 0755)

	if err := SetActiveProvider(root, "lmstudio"); err != nil {
		t.Fatalf("SetActiveProvider sin modelos todavía: %v", err)
	}
	names, err := ListModelConfigs(root, "lmstudio")
	if err != nil {
		t.Fatalf("ListModelConfigs: %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("se esperaba 0 modelos, got %v", names)
	}
}

func TestSessionChatAndModelSwitch(t *testing.T) {
	srv := fakeOllama(t)
	defer srv.Close()
	root := setupProject(t, srv.URL)

	// segundo modelo para probar el cambio "set -model" dentro del chat
	mistral := DefaultModelConfig(&ModelConfig{Provider: "ollama", Type: "ollama", BaseURL: srv.URL})
	if err := SaveModelConfig(root, "ollama", "mistral", mistral); err != nil {
		t.Fatal(err)
	}

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
	reply, err := sess.Send("hola")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if reply != "hola desde llama3.1" {
		t.Fatalf("reply = %q", reply)
	}

	// cambiar de modelo a mitad de la conversación: el historial se conserva
	if err := sess.SetModel("mistral"); err != nil {
		t.Fatalf("SetModel(mistral): %v", err)
	}
	if len(sess.History) != 2 {
		t.Fatalf("se esperaba conservar el historial tras cambiar de modelo, got %d mensajes", len(sess.History))
	}
	reply2, err := sess.Send("segunda pregunta")
	if err != nil {
		t.Fatalf("Send tras cambio de modelo: %v", err)
	}
	if reply2 != "hola desde mistral" {
		t.Fatalf("reply2 = %q", reply2)
	}

	user, assistant, ok := sess.LastExchange()
	if !ok || user != "segunda pregunta" || assistant != "hola desde mistral" {
		t.Fatalf("LastExchange inesperado: %q / %q / %v", user, assistant, ok)
	}
}

func TestHotReloadConfig(t *testing.T) {
	srv := fakeOllama(t)
	defer srv.Close()
	root := setupProject(t, srv.URL)

	mc, err := DefaultCache.GetModel(root, "ollama", "llama3.1")
	if err != nil {
		t.Fatalf("GetModel: %v", err)
	}
	if mc.Temperature != 0 {
		t.Fatalf("temperature inicial = %v, want 0", mc.Temperature)
	}

	// editamos el .json "a mano" (simulando al usuario) y forzamos un
	// mtime distinto para que el cambio sea detectable en sistemas de
	// archivos con resolución de tiempo baja.
	time.Sleep(10 * time.Millisecond)
	mc2 := *mc
	mc2.Temperature = 0.8
	if err := SaveModelConfig(root, "ollama", "llama3.1", mc2); err != nil {
		t.Fatalf("SaveModelConfig: %v", err)
	}

	reloaded, err := DefaultCache.GetModel(root, "ollama", "llama3.1")
	if err != nil {
		t.Fatalf("GetModel (reload): %v", err)
	}
	if reloaded.Temperature != 0.8 {
		t.Fatalf("recarga en caliente falló: temperature = %v, want 0.8", reloaded.Temperature)
	}
	// la conexión (base_url), que ahora vive en el MISMO archivo, también
	// se releyó sin perderse en el camino.
	if reloaded.ResolvedBaseURL() != srv.URL {
		t.Fatalf("base_url se perdió tras la recarga: %q", reloaded.ResolvedBaseURL())
	}
}

func TestInstallRemoveListInstalled(t *testing.T) {
	srv := fakeOllama(t)
	defer srv.Close()
	root := setupProject(t, srv.URL)

	var updates []string
	err := Install(root, "ollama", []string{"phi3"}, func(model, status string, percent int) {
		updates = append(updates, status)
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(updates) == 0 {
		t.Fatalf("se esperaban actualizaciones de progreso")
	}
	if _, err := FindModelFile(root, "ollama", "phi3"); err != nil {
		t.Fatalf("se esperaba que Install creara config/models/ollama/phi3.json: %v", err)
	}
	// Install tomó prestada la conexión de llama3.1.json (el modelo
	// hermano ya configurado) — sin eso, phi3.json no sabría a qué
	// servidor pegarle.
	phi3, err := DefaultCache.GetModel(root, "ollama", "phi3")
	if err != nil {
		t.Fatalf("GetModel(phi3): %v", err)
	}
	if phi3.ResolvedBaseURL() != srv.URL {
		t.Fatalf("phi3 no heredó la conexión del modelo hermano: %q", phi3.ResolvedBaseURL())
	}

	models, err := ListInstalled(root, "ollama")
	if err != nil {
		t.Fatalf("ListInstalled: %v", err)
	}
	if len(models) != 1 || models[0].Name != "llama3.1:8b" {
		t.Fatalf("ListInstalled inesperado: %+v", models)
	}

	if err := Remove(root, "ollama", []string{"llama3.1:8b"}); err != nil {
		t.Fatalf("Remove: %v", err)
	}
}

func TestFindModelFileAmbiguous(t *testing.T) {
	srv := fakeOllama(t)
	defer srv.Close()
	root := setupProject(t, srv.URL)
	os.WriteFile(filepath.Join(root, "config", "models", "ollama", "llama3.2.json"), []byte(`{}`), 0644)

	if _, err := FindModelFile(root, "ollama", "llama"); err == nil {
		t.Fatalf("se esperaba error de ambigüedad entre llama3.1 y llama3.2")
	}
	if name, err := FindModelFile(root, "ollama", "llama3.1"); err != nil || name != "llama3.1" {
		t.Fatalf("match exacto falló: %q, %v", name, err)
	}
}

func TestSendStreamDeliversIncrementalTokens(t *testing.T) {
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

	var chunks []string
	reply, err := sess.SendStream("hola", func(tok string) {
		chunks = append(chunks, tok)
	})
	if err != nil {
		t.Fatalf("SendStream: %v", err)
	}
	if reply != "hola desde llama3.1" {
		t.Fatalf("reply = %q", reply)
	}
	if len(chunks) < 2 {
		t.Fatalf("se esperaban múltiples fragmentos incrementales, got %d: %v", len(chunks), chunks)
	}
	joined := strings.Join(chunks, "")
	if joined != reply {
		t.Fatalf("los fragmentos concatenados (%q) no arman la respuesta final (%q)", joined, reply)
	}
	if sess.LastUsage.PromptTokens != 42 || sess.LastUsage.CompletionTokens != 7 {
		t.Fatalf("Usage real no se propagó desde el streaming: %+v", sess.LastUsage)
	}

	// el historial queda igual que con Send() normal, para que /memory y el
	// resto del chat no distingan si la última respuesta vino en streaming.
	user, assistant, ok := sess.LastExchange()
	if !ok || user != "hola" || assistant != reply {
		t.Fatalf("LastExchange inesperado tras SendStream: %q / %q / %v", user, assistant, ok)
	}
}
