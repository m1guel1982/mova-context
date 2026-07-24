// history.go implementa el "Feedback Loop": cerrar el ciclo entre la
// estimación local de tiktoken-go y los tokens reales que devuelve cada
// proveedor Cloud (OpenAI/Anthropic/Google), para que la estimación
// mejore sola con el tiempo — sin guardar NADA del contenido del
// proyecto, sin prompts, sin conversaciones, sin base de datos: solo dos
// acumuladores por proveedor, en un único archivo chico.
//
// mova-token-history.json vive en el mismo directorio que project.json
// por default, o en la ruta que project.json indique en
// "token_history_path" — ver HistoryPath, resuelta con path/filepath
// (multiplataforma: Windows, Linux, macOS).
package budget

import (
	"encoding/json"
	"os"
	"path/filepath"

	"mova.local/core"
)

// ProviderAccumulator son los dos únicos números que se guardan por
// proveedor — nunca un historial por request, nunca contenido.
type ProviderAccumulator struct {
	TotalLocalTokens int `json:"total_local_tokens"`
	TotalAPITokens   int `json:"total_api_tokens"`
}

// TokenHistory mapea mova-token-history.json exactamente: una clave por
// proveedor ("anthropic", "openai", "google", o cualquier otro que se
// agregue), cada una con sus dos acumuladores.
type TokenHistory map[string]ProviderAccumulator

// HistoryPath resuelve dónde vive mova-token-history.json: el directorio
// del propio project.json por default, o proj.TokenHistoryPath si el
// proyecto define uno — resuelto con filepath.Join, así que funciona
// igual en Windows, Linux y macOS sin ningún caso especial por SO.
func HistoryPath(root, project string, proj *core.Project) string {
	if proj != nil && proj.TokenHistoryPath != "" {
		if filepath.IsAbs(proj.TokenHistoryPath) {
			return proj.TokenHistoryPath
		}
		return filepath.Join(root, proj.TokenHistoryPath)
	}
	return filepath.Join(root, "projects", project, "mova-token-history.json")
}

// LoadHistory lee mova-token-history.json. Un archivo inexistente no es
// error — es el estado normal antes de la primera llamada real a un
// proveedor Cloud.
func LoadHistory(path string) (TokenHistory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return TokenHistory{}, nil
		}
		return nil, err
	}
	var h TokenHistory
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, err
	}
	if h == nil {
		h = TokenHistory{}
	}
	return h, nil
}

// RecordUsage suma localTokens/apiTokens al acumulador de provider y
// guarda el archivo — llamado una vez por cada llamada real a una API
// Cloud (chat_completion con un proveedor Cloud configurado). Nunca se
// guarda el prompt, la respuesta, ni ningún dato más allá de estos dos
// números.
func RecordUsage(path, provider string, localTokens, apiTokens int) error {
	history, err := LoadHistory(path)
	if err != nil {
		return err
	}
	acc := history[provider]
	acc.TotalLocalTokens += localTokens
	acc.TotalAPITokens += apiTokens
	history[provider] = acc

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// DeviationPercent calcula (API - Local) / Local * 100 para un proveedor
// — ok=false si el proveedor todavía no tiene datos ("No historical
// data").
func (h TokenHistory) DeviationPercent(provider string) (percent float64, ok bool) {
	acc, exists := h[provider]
	if !exists || acc.TotalLocalTokens == 0 {
		return 0, false
	}
	return (float64(acc.TotalAPITokens) - float64(acc.TotalLocalTokens)) / float64(acc.TotalLocalTokens) * 100, true
}
