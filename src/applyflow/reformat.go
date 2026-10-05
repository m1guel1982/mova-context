package applyflow

import (
	"regexp"
	"strings"

	"mova.local/core"
	"mova.local/documents"
	"mova.local/i18n"
)

// askRe reconoce la pregunta típica de un prompt que le pide al MODELO
// aplicar los cambios: «(Sí/No)», «aplique estas modificaciones…».
var askRe = regexp.MustCompile(`(?i)\(\s*s[ií]\s*/\s*no\s*\)|\(\s*yes\s*/\s*no\s*\)|aplique\s+estas\s+modificaciones|apply\s+these\s+(?:changes|modifications)`)

// ReformatRequest resuelve el caso que hacía que la confirmación «nunca se
// reflejara»: el prompt le pide al modelo «¿Deseas que aplique estas
// modificaciones…? (Sí/No)», la persona contesta «Sí», pero el modelo no puede
// escribir archivos y su propuesta no traía bloques con destino, así que no
// había nada que Mova pudiera aplicar (y el modelo respondía «listo» sin haber
// hecho nada). Si la tarea tiene "apply" activo, el último mensaje del modelo
// hace esa pregunta, trae código pero NINGÚN bloque etiquetado, y la persona
// responde con una confirmación escueta, devuelve el mensaje que se envía en
// su lugar: pide al modelo convertir la propuesta aprobada en bloques con
// destino. Esa respuesta pasa luego por Offer/Render, que muestran la lista
// real de archivos con la pregunta final de Mova.
func ReformatRequest(proj *core.Project, task, line, lastAssistant string) (string, bool) {
	if !core.ApplyEnabled(proj, task) {
		return "", false
	}
	if ok, _ := documents.DetectApplyConfirmation(line); !ok {
		return "", false
	}
	if len(documents.ExtractLabeledCodeBlocks(lastAssistant)) > 0 {
		return "", false // ya hay bloques con destino: se aplican tal cual
	}
	if !askRe.MatchString(lastAssistant) || !strings.Contains(lastAssistant, "```") {
		return "", false
	}
	return i18n.TIn(proj.Lang, "apply.reformat_request"), true
}
