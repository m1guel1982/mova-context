// chat_tool_apply.go — la parte «modificar archivos» de chat_completion
// (puertas MCP y HTTP). Extraída tal cual de chat_tool.go (≤300 líneas por
// archivo); la lógica vive en mova.local/applyflow, compartida con el chat.
package mcp

import (
	"strings"

	"mova.local/applyflow"
	"mova.local/core"
	"mova.local/i18n"
	"mova.local/models"
)

// remoteApplyAnswer: si hay una propuesta pendiente en el proyecto y este
// mensaje (o el argumento apply_changes) es la respuesta a la pregunta, la
// aplica SIN llamar al modelo.
func remoteApplyAnswer(root, project string, proj *core.Project, args map[string]any, message string) (string, bool) {
	if project == "" || proj == nil {
		return "", false
	}
	return applyflow.HandleRemote(root, project, proj.Lang, message, str(args, "apply_changes"))
}

// reformatForApply cubre el prompt que le pide al modelo «aplicar (Sí/No)»:
// si el «sí» llega y la propuesta no traía bloques con destino, devuelve la
// petición interna de bloques que reemplaza al mensaje. El llamador debe
// ocultarla a los detectores de lenguaje natural (menciona rutas y verbos
// de edición), por eso devuelve también nlMessage vacío vía el bool.
func reformatForApply(statusLog *strings.Builder, sess *models.Session, proj *core.Project, task, message string) (string, bool) {
	if proj == nil {
		return "", false
	}
	_, assistant, has := sess.LastExchange()
	if !has {
		return "", false
	}
	rq, ok := applyflow.ReformatRequest(proj, task, message, assistant)
	if ok {
		statusLog.WriteString(i18n.T("apply.reformatting") + "\n")
	}
	return rq, ok
}

// offerApply: con «apply» activo guarda la propuesta como pendiente y
// devuelve la lista + la pregunta para anexar a la respuesta ("" si no hay).
// Nada se escribe hasta que la persona responda.
func offerApply(root, project, task string, proj *core.Project, sess *models.Session, reply string) string {
	if project == "" || proj == nil || sess.LastReplyWasDryRun {
		return ""
	}
	if offer := applyflow.Offer(root, project, task, proj, reply); offer != "" {
		return "\n\n---\n" + offer
	}
	return ""
}
