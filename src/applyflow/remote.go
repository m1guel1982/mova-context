package applyflow

import (
	"mova.local/core"
	"mova.local/i18n"
)

// HandleRemote es la parte MCP/HTTP del flujo, llamada al INICIO de cada
// chat_completion. Si hay una propuesta pendiente en este proyecto y el
// mensaje (o el argumento explícito apply_changes: "all" | "1,3" | "none")
// es la respuesta a la pregunta, la aplica SIN llamar al modelo.
// handled=false deja que la llamada siga su camino normal.
func HandleRemote(root, project, lang, message, explicit string) (text string, handled bool) {
	p, expired := Load(root, project)
	if p == nil {
		if explicit != "" { // el cliente quiso confirmar pero no hay nada pendiente
			key := "apply.no_pending"
			if expired {
				key = "apply.expired"
			}
			return i18n.TIn(lang, key, map[string]any{"minutes": int(PendingTTL.Minutes())}), true
		}
		return "", false
	}
	answer := explicit
	if answer == "" {
		if !LooksLikeAnswer(message, len(p.Changes)) {
			return "", false // consulta nueva: la propuesta sigue pendiente
		}
		answer = message
	}
	text, _ = p.Answer(root, answer)
	if _, kind := ParseSelection(answer, len(p.Changes)); kind != AnswerInvalid {
		Clear(root, project) // contestada (sí/no/subconjunto): ya no está pendiente
	}
	return text, true
}

// Offer decide, tras una respuesta del modelo, si hay una propuesta que
// preguntar: la guarda como pendiente y devuelve el texto (lista + pregunta)
// para anexar a la respuesta. "" si no hay nada que preguntar.
func Offer(root, project, task string, proj *core.Project, reply string) string {
	p := Build(root, project, task, proj, reply)
	if p == nil {
		return ""
	}
	if err := Save(root, p); err != nil {
		return i18n.TIn(p.Lang, "apply.save_failed", map[string]any{"error": err.Error()})
	}
	return p.Render(true)
}
