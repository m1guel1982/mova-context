// chat_banner.go — banner de arranque de `mova chat`. Movido tal cual desde
// chat_cmd.go (≤300 líneas por archivo).
package main

import (
	"strings"

	"mova.local/models"
)

func chatBanner(sess *models.Session) string {
	var b strings.Builder
	b.WriteString("mova chat — provider: " + sess.Provider)
	if sess.Model != "" {
		b.WriteString(", model: " + sess.Model)
	} else {
		b.WriteString(" (no model set — use `set -model <name>`)")
	}
	b.WriteString("\ntype `exit` to quit.\n\n")
	return b.String()
}
