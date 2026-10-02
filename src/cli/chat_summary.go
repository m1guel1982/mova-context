// chat_summary.go — resumen compacto de un intercambio (respaldo histórico
// de /memory). Movido tal cual desde chat_save.go (≤300 líneas).
package main

import (
	"fmt"
	"strings"
)

// summarizeContent extracts the core task and main lines of the assistant response.
func summarizeContent(user, assistant string) string {
	// 1. Sanitizar y acortar la tarea del usuario
	userTask := strings.TrimSpace(user)
	if idx := strings.Index(userTask, "\n"); idx != -1 {
		userTask = userTask[:idx]
	}
	if len(userTask) > 80 {
		userTask = userTask[:80] + "..."
	}

	// 2. Extraer las primeras 3 líneas útiles de la respuesta
	lines := strings.Split(assistant, "\n")
	var keyLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Ignorar líneas vacías, tablas, separadores y headers pesados
		if trimmed == "" || strings.HasPrefix(trimmed, "|") || strings.HasPrefix(trimmed, "---") || strings.HasPrefix(trimmed, "##") {
			continue
		}

		keyLines = append(keyLines, trimmed)
		if len(keyLines) >= 3 {
			break
		}
	}

	compactAssistant := strings.Join(keyLines, "\n")
	if compactAssistant == "" {
		compactAssistant = "Review completed."
	}

	return fmt.Sprintf("**task:** %s\n**summary:**\n%s\n", userTask, compactAssistant)
}
