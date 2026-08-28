// config.go — lee config/models/<provider>/*.json del disco. Un solo
// archivo por modelo (conexión + parámetros); ya no existe un
// config.json separado por proveedor (ver types.go).
//
// Recarga en caliente: cada modelo se cachea junto con el mtime del
// archivo. Antes de usar una configuración (cada mensaje del chat, cada
// llamada MCP/HTTP) se compara el mtime actual contra el cacheado — si
// cambió, se relee el JSON. Sin goroutines, sin watchers de SO: barato,
// determinista, y funciona igual en procesos de un solo comando (mova
// install) que en procesos de vida larga (mova chat, mova mcp start).
package models

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// ModelsRoot — config/models dentro de la raíz del proyecto Mova.
func ModelsRoot(root string) string {
	return filepath.Join(root, "config", "models")
}

func providerDir(root, provider string) string {
	return filepath.Join(ModelsRoot(root), provider)
}

// ListProviders — subdirectorios de config/models (uno por proveedor).
func ListProviders(root string) ([]string, error) {
	entries, err := os.ReadDir(ModelsRoot(root))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// ListModelConfigs — nombres de modelo (sin ".json") que tienen archivo
// de configuración propio bajo config/models/<provider>/. Excluye
// "active.json" por las dudas si alguna vez terminara ahí por error (vive
// un nivel arriba, en ModelsRoot, no dentro de providerDir — ver
// activePath), y cualquier archivo oculto.
func ListModelConfigs(root, provider string) ([]string, error) {
	entries, err := os.ReadDir(providerDir(root, provider))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".json") || n == "active.json" || strings.HasPrefix(n, ".") {
			continue
		}
		names = append(names, strings.TrimSuffix(n, ".json"))
	}
	sort.Strings(names)
	return names, nil
}

// FindModelFile resuelve un nombre de modelo escrito por el usuario contra
// los .json existentes: primero coincidencia exacta, luego por prefijo
// (para que "llama" encuentre "llama3.1.json" si es la única opción).
func FindModelFile(root, provider, query string) (string, error) {
	names, err := ListModelConfigs(root, provider)
	if err != nil {
		return "", fmt.Errorf("no se pudo leer config/models/%s: %w", provider, err)
	}
	for _, n := range names {
		if n == query {
			return n, nil
		}
	}
	var matches []string
	for _, n := range names {
		if strings.HasPrefix(n, query) || strings.Contains(n, query) {
			matches = append(matches, n)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("%q es ambiguo, coincide con: %s", query, strings.Join(matches, ", "))
	}
	return "", fmt.Errorf("modelo %q no encontrado en config/models/%s (instalado con `mova install %s`, o creá config/models/%s/%s.json a mano)", query, provider, query, provider, query)
}

// ResolveConfigProvider finds which provider subdirectory under
// config/models/ holds a file named "<config>.json", so callers (see
// cli/chat_cmd.go and mcp/chat_tool.go) never have to know the provider
// folder in advance. This is what lets project.json's "llm_profile" drop
// the redundant "provider" key entirely and declare only "config" — the
// provider identity is then read from that ONE resolved file's own
// "type" field (see ModelConfig.Type), never duplicated in two places.
// Returns an error if no provider folder has that config, or if more
// than one does (ambiguous — the caller should set "provider" in
// llm_profile explicitly to disambiguate in that rare case).
func ResolveConfigProvider(root, config string) (string, error) {
	providers, err := ListProviders(root)
	if err != nil {
		return "", fmt.Errorf("could not read config/models: %w", err)
	}
	var matches []string
	for _, p := range providers {
		if _, err := FindModelFile(root, p, config); err == nil {
			matches = append(matches, p)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", fmt.Errorf("config %q not found under any config/models/<provider>/ folder — check \"config\" in llm_profile, or set \"provider\" explicitly if the file uses a name shared across providers", config)
	default:
		return "", fmt.Errorf("config %q is ambiguous — found under: %s. Set \"provider\" in llm_profile to pick one", config, strings.Join(matches, ", "))
	}
}

// ── caché de configuración de modelo con recarga en caliente ───────────────

type cachedModel struct {
	cfg     ModelConfig
	modTime int64
}

// ConfigCache guarda la última versión leída de cada modelo. Segura para
// uso concurrente (el chat y los handlers de MCP/HTTP pueden compartirla).
type ConfigCache struct {
	mu      sync.RWMutex
	entries map[string]cachedModel
}

// DefaultCache — instancia compartida por todo el proceso.
var DefaultCache = &ConfigCache{entries: map[string]cachedModel{}}

// GetModel devuelve la configuración vigente de <provider>/<model>,
// releyendo el JSON solo si el archivo cambió desde la última lectura.
func (c *ConfigCache) GetModel(root, provider, model string) (*ModelConfig, error) {
	path := filepath.Join(providerDir(root, provider), model+".json")
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("configuración no encontrada: %s: %w", path, err)
	}
	key := provider + "/" + model
	mtime := info.ModTime().UnixNano()

	c.mu.RLock()
	cached, ok := c.entries[key]
	c.mu.RUnlock()
	if ok && cached.modTime == mtime {
		cfg := cached.cfg
		return &cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg ModelConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%s: JSON inválido: %w", path, err)
	}
	if cfg.Provider == "" {
		cfg.Provider = provider
	}

	c.mu.Lock()
	c.entries[key] = cachedModel{cfg: cfg, modTime: mtime}
	c.mu.Unlock()

	return &cfg, nil
}

// SaveModelConfig escribe (o sobreescribe) config/models/<provider>/<model>.json.
func SaveModelConfig(root, provider, model string, cfg ModelConfig) error {
	dir := providerDir(root, provider)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, model+".json"), data, 0644)
}

// ── proveedor / modelo activo (config/models/active.json) ──────────────────

func activePath(root string) string {
	return filepath.Join(ModelsRoot(root), "active.json")
}

// GetActiveState lee el proveedor/modelo activo. Si no existe el archivo,
// devuelve un estado vacío (sin error) — es responsabilidad del llamador
// pedir que se configure uno con `mova config <provider>`.
func GetActiveState(root string) (*ActiveState, error) {
	data, err := os.ReadFile(activePath(root))
	if os.IsNotExist(err) {
		return &ActiveState{}, nil
	}
	if err != nil {
		return nil, err
	}
	var s ActiveState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("active.json inválido: %w", err)
	}
	return &s, nil
}

func saveActiveState(root string, s *ActiveState) error {
	if err := os.MkdirAll(ModelsRoot(root), 0755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(activePath(root), data, 0644)
}

// SetActiveProvider — `mova config <provider>`. Solo valida que exista el
// directorio config/models/<provider>/ — ya no depende de un config.json
// separado, así que podés elegir un proveedor antes de tener ningún
// modelo configurado ahí adentro (por ejemplo, justo antes de crear el
// primer config/models/<provider>/<modelo>.json a mano).
func SetActiveProvider(root, provider string) error {
	info, err := os.Stat(providerDir(root, provider))
	if err != nil || !info.IsDir() {
		return fmt.Errorf("no existe config/models/%s/ — creá la carpeta y al menos un archivo <modelo>.json (ver docs/i18n/es/COMMANDS.md)", provider)
	}
	s, err := GetActiveState(root)
	if err != nil {
		return err
	}
	s.Provider = provider
	s.Config = "" // cambiar de proveedor invalida el modelo elegido antes
	return saveActiveState(root, s)
}

// SetActiveModel — usado por `set -model <nombre>` dentro del chat.
func SetActiveModel(root, model string) error {
	s, err := GetActiveState(root)
	if err != nil {
		return err
	}
	if s.Provider == "" {
		return fmt.Errorf("no hay proveedor activo — corré `mova config <provider>` primero")
	}
	resolved, err := FindModelFile(root, s.Provider, model)
	if err != nil {
		return err
	}
	s.Config = resolved
	return saveActiveState(root, s)
}