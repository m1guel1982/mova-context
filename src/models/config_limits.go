// config_limits.go — resolución del tope de tokens de RESPUESTA de un
// modelo. Una sola función para los cuatro proveedores (openai-compatible,
// anthropic, gemini, ollama), así "max_tokens" del .json se respeta igual
// en chat, MCP y HTTP.
package models

// ResponseMax devuelve el máximo de tokens de respuesta a enviar al
// proveedor: "max_tokens" si el .json lo declara, si no "num_predict", si
// no def (el default propio de cada proveedor).
func (m *ModelConfig) ResponseMax(def int) int {
	if m.MaxTokens > 0 {
		return m.MaxTokens
	}
	if m.NumPredict > 0 {
		return m.NumPredict
	}
	return def
}
