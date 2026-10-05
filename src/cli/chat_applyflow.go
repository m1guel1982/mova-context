// chat_applyflow.go — puerta CHAT del flujo «el modelo propone → Mova
// pregunta → se modifican los archivos» (lógica compartida con MCP/HTTP en
// mova.local/applyflow). Aquí solo se decide CÓMO preguntar: por la
// terminal, leyendo la respuesta del mismo scanner del REPL.
package main

import (
	"bufio"
	"strings"

	"mova.local/applyflow"
	"mova.local/core"
	"mova.local/i18n"
	"mova.local/models"
)

// offerChatApply muestra la propuesta del modelo (archivos, +/− líneas,
// avisos de exclude/no encontrado, estado del respaldo) y PREGUNTA si se
// quiere modificar todo, solo algunos números o nada. Devuelve true si
// escribió algún archivo. No hace nada si la tarea no tiene "apply" activo
// o la respuesta no trae bloques con destino.
func offerChatApply(root, project, task string, proj *core.Project, reply string, scanner *bufio.Scanner) bool {
	p := applyflow.Build(root, project, task, proj, reply)
	if p == nil || scanner == nil {
		return false
	}
	consolePrint("\n" + p.Render(false))
	if !scanner.Scan() {
		return false
	}
	text, wrote := p.Answer(root, strings.TrimSpace(scanner.Text()))
	consolePrint(text + "\n")
	return wrote
}

// reformatIfNeeded cubre el prompt clásico que le pide al MODELO «aplicar» y
// preguntar (Sí/No): si la persona responde «sí» pero la propuesta no trae
// bloques con destino, el modelo no puede escribir nada (esa era la
// confirmación que «nunca se reflejaba»). Se reemplaza el «sí» por la
// petición de convertir la propuesta en bloques; la respuesta pasa luego por
// offerChatApply, que muestra la lista real y pregunta.
func reformatIfNeeded(sess *models.Session, proj *core.Project, task, line string) string {
	_, assistant, has := sess.LastExchange()
	if !has {
		return line
	}
	if rq, ok := applyflow.ReformatRequest(proj, task, line, assistant); ok {
		consolePrint(i18n.T("apply.reformatting") + "\n")
		return rq
	}
	return line
}
