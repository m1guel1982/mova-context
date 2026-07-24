// manage.go — mova install / mova remove / mova model-list.
//
// Estas tres operaciones dependen de la API nativa de Ollama
// (/api/pull, /api/delete, /api/tags): no existe un estándar
// openai-compatible para gestionar modelos, así que por ahora solo
// aplican a proveedores con Type == "ollama". Para LM Studio/vLLM/Cloud
// el modelo se configura a mano — Mova solo necesita su
// config/models/<provider>/<config>.json para poder chatear con él.
package models

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ProgressFunc recibe actualizaciones de instalación: modelo, status
// legible ("pulling manifest", "downloading", ...) y porcentaje (0-100,
// -1 si el servidor todavía no reportó tamaño total).
type ProgressFunc func(model, status string, percent int)

// seedConnection resuelve los datos de conexión (base_url/host/port/
// type/api_key/timeout) a usar para instalar/listar/eliminar modelos de
// un proveedor. Como ya no existe un config.json separado por proveedor,
// se toma prestada la conexión de cualquier modelo hermano que ya exista
// en config/models/<provider>/ (todos comparten servidor). Si el
// proveedor todavía no tiene NINGÚN modelo configurado, se asume un
// bootstrap de Ollama en localhost — razonable porque Install/Remove/
// ListInstalled de por sí solo operan contra servidores tipo "ollama".
func seedConnection(root, provider string) (*ModelConfig, error) {
	names, err := ListModelConfigs(root, provider)
	if err == nil && len(names) > 0 {
		return DefaultCache.GetModel(root, provider, names[0])
	}
	return &ModelConfig{Provider: provider, Type: "ollama", BaseURL: "http://localhost:11434"}, nil
}

// Install descarga cada modelo en el proveedor activo y, si no existe
// todavía, le crea una configuración por default en
// config/models/<provider>/<modelo>.json (recargable en caliente desde
// el primer momento, sin reiniciar nada) — reutilizando la conexión de
// cualquier modelo hermano ya configurado.
func Install(root, provider string, modelNames []string, onProgress ProgressFunc) error {
	seed, err := seedConnection(root, provider)
	if err != nil {
		return err
	}
	if seed.Type != "" && seed.Type != "ollama" {
		return fmt.Errorf("instalación automática no soportada para el tipo %q — instalá el modelo desde la propia herramienta y agregá su config/models/%s/<modelo>.json manualmente", seed.Type, provider)
	}

	for _, model := range modelNames {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if err := pullOne(context.Background(), seed, model, onProgress); err != nil {
			return fmt.Errorf("instalando %q: %w", model, err)
		}
		if _, err := FindModelFile(root, provider, model); err != nil {
			if err := SaveModelConfig(root, provider, model, DefaultModelConfig(seed)); err != nil {
				return fmt.Errorf("modelo %q instalado, pero no se pudo escribir su config: %w", model, err)
			}
		}
	}
	return nil
}

func pullOne(ctx context.Context, cfg *ModelConfig, model string, onProgress ProgressFunc) error {
	payload, _ := json.Marshal(map[string]any{"model": model, "stream": true})
	url := cfg.ResolvedBaseURL() + "/api/pull"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := SharedClient.Do(req)
	if err != nil {
		return fmt.Errorf("no se pudo contactar a %s (%s): %w", cfg.Provider, url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s respondió %d en /api/pull", cfg.Provider, resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var evt struct {
			Status    string `json:"status"`
			Error     string `json:"error"`
			Completed int64  `json:"completed"`
			Total     int64  `json:"total"`
		}
		if err := json.Unmarshal(line, &evt); err != nil {
			continue // línea no-JSON aislada; no aborta la descarga completa
		}
		if evt.Error != "" {
			return fmt.Errorf("%s", evt.Error)
		}
		percent := -1
		if evt.Total > 0 {
			percent = int(evt.Completed * 100 / evt.Total)
		}
		if onProgress != nil {
			onProgress(model, evt.Status, percent)
		}
	}
	return scanner.Err()
}

// Remove — `mova remove modelo1,modelo2`. Solo válido para proveedores
// tipo "ollama" (mismo motivo que Install).
func Remove(root, provider string, modelNames []string) error {
	cfg, err := seedConnection(root, provider)
	if err != nil {
		return err
	}
	if cfg.Type != "" && cfg.Type != "ollama" {
		return fmt.Errorf("eliminación remota no soportada para el tipo %q", cfg.Type)
	}
	for _, model := range modelNames {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		var out struct {
			Error string `json:"error"`
		}
		payload, _ := json.Marshal(map[string]string{"model": model})
		req, err := http.NewRequest(http.MethodDelete, cfg.ResolvedBaseURL()+"/api/delete", strings.NewReader(string(payload)))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := SharedClient.Do(req)
		if err != nil {
			return fmt.Errorf("eliminando %q: %w", model, err)
		}
		json.NewDecoder(resp.Body).Decode(&out)
		resp.Body.Close()
		if out.Error != "" {
			return fmt.Errorf("eliminando %q: %s", model, out.Error)
		}
	}
	return nil
}

// ListInstalled — `mova model-list` (GET /api/tags).
func ListInstalled(root, provider string) ([]InstalledModel, error) {
	cfg, err := seedConnection(root, provider)
	if err != nil {
		return nil, err
	}
	if cfg.Type != "" && cfg.Type != "ollama" {
		return nil, fmt.Errorf("listado remoto no soportado para el tipo %q", cfg.Type)
	}
	var out struct {
		Models []InstalledModel `json:"models"`
	}
	req, err := http.NewRequest(http.MethodGet, cfg.ResolvedBaseURL()+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := SharedClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("no se pudo contactar a %s: %w", cfg.Provider, err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Models, nil
}
