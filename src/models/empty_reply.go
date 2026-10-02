// empty_reply.go — una respuesta vacía NUNCA debe pasar como éxito. Caso
// real: DeepSeek v4 vía OpenRouter con el límite de salida demasiado bajo
// gasta todos los tokens razonando, devuelve content "" y finish_reason
// "length"; el chat mostraba solo "[Tokens] ..." y el usuario pagaba sin
// recibir nada. Ahora Session convierte ese caso en un error explícito
// (con los tokens ya consumidos) en chat, MCP y HTTP por igual.
package models

import (
	"fmt"
	"strings"
)

// EmptyReplyError: el proveedor respondió 200 pero sin texto útil.
type EmptyReplyError struct {
	Provider, Model, Finish string
	MaxTokens               int
	Usage                   Usage
	ReasoningChars          int
}

func (e *EmptyReplyError) Error() string {
	var b strings.Builder
	finish := e.Finish
	if finish == "" {
		finish = "desconocido"
	}
	fmt.Fprintf(&b, "%s devolvió una respuesta VACÍA (modelo %s, finish_reason=%s). Tokens ya consumidos: %d entrada / %d salida.",
		e.Provider, e.Model, finish, e.Usage.PromptTokens, e.Usage.CompletionTokens)
	switch {
	case e.Finish == "length" || e.Finish == "max_tokens" || e.Finish == "MAX_TOKENS":
		fmt.Fprintf(&b, " El límite de salida (max_tokens=%d) se agotó antes de producir texto: los modelos con razonamiento gastan ese cupo pensando.", e.MaxTokens)
		b.WriteString(` Sube "max_tokens" (≥16384 recomendado) en config/models/<proveedor>/<modelo>.json o limita el razonamiento con "reasoning": {"effort":"low"}.`)
	case e.ReasoningChars > 0:
		fmt.Fprintf(&b, " El modelo solo emitió razonamiento (%d caracteres) y ningún contenido final. Reintenta o sube max_tokens.", e.ReasoningChars)
	default:
		b.WriteString(" Reintenta; si se repite revisa el modelo/proveedor y el filtro de contenido.")
	}
	return b.String()
}

// IsTruncated: la respuesta llegó pero se cortó por el límite de salida.
func (u Usage) IsTruncated() bool {
	switch u.FinishReason {
	case "length", "max_tokens", "MAX_TOKENS":
		return true
	}
	return false
}

// finishTurn es el cierre común de Send y SendStream: registra el uso
// real SIEMPRE (también cuando falla o viene vacío, para que se vea lo
// consumido), valida que haya texto y actualiza el historial. Asume que el
// último mensaje de s.History es el turno del usuario recién enviado.
func (s *Session) finishTurn(mc *ModelConfig, model, reply string, usage Usage, err error) (string, error) {
	s.LastUsage = usage
	s.LastTruncated = err == nil && usage.IsTruncated() && strings.TrimSpace(reply) != ""
	if err == nil && strings.TrimSpace(reply) == "" {
		err = &EmptyReplyError{Provider: s.Provider, Model: model, Finish: usage.FinishReason, MaxTokens: mc.ResponseMax(0), Usage: usage}
	}
	if err != nil {
		s.History = s.History[:len(s.History)-1] // no dejar el turno colgado sin respuesta
		return "", err
	}
	s.History = append(s.History, ChatMessage{Role: "assistant", Content: reply})
	return reply, nil
}
