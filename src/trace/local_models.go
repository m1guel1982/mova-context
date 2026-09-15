// local_models.go — reads the REAL, currently-configured context
// window for local inference providers from config/models/<provider>/
// *.json (e.g. config/models/ollama/llama3.2.3b.json's own
// "context_window": 131072) instead of ever showing a hardcoded
// guess. Fixes the reported "fits: unknown (context_window not
// configured)" for every local provider even when the person has
// real, configured local models on disk.
package trace

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// localProviderDirs maps a prices.json provider key to the
// config/models/ subdirectory that actually holds its model files —
// the mapping is explicit (not derived from the name) because
// prices.json is free to use whatever key it wants ("lm-studio" vs
// the directory's own "lmstudio").
var localProviderDirs = map[string]string{
	"ollama":    "ollama",
	"lm-studio": "lmstudio",
	"vllm":      "vllm",
}

type localModelFile struct {
	Model         string `json:"model"`
	ContextWindow int    `json:"context_window"`
}

// localModelContextWindows scans config/models/<dir>/*.json for every
// known local provider and returns the LARGEST context_window found
// per provider — i.e. "the biggest window any locally configured
// model for this provider currently supports". A provider with no
// directory, or no readable model file with a positive
// context_window, is simply absent from the returned map, so
// modelCompatRows keeps rendering "context_window not configured" for
// it — never a fabricated number.
func localModelContextWindows(root string) map[string]int {
	result := map[string]int{}
	if root == "" {
		return result
	}
	for providerKey, dirName := range localProviderDirs {
		dir := filepath.Join(root, "config", "models", dirName)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		best := 0
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			var mf localModelFile
			if json.Unmarshal(data, &mf) != nil {
				continue
			}
			if mf.ContextWindow > best {
				best = mf.ContextWindow
			}
		}
		if best > 0 {
			result[providerKey] = best
		}
	}
	return result
}
