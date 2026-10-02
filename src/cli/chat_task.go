// chat_task.go — cambiar de tarea DENTRO de un chat abierto:
//
//	/tasks            lista las tareas del proyecto
//	/task <nombre>    cambia a esa tarea (solo su prompt/focus/grafo)
//	/task all         pasa al modo "todas las tareas"
//	/run [nombre]     cambia a la tarea y le envía su variable QUERY
//
// Cambiar de tarea reconstruye el contexto (sin salir del chat, sin
// perder el historial de la conversación) y relee memory.md, donde las
// demás tareas dejaron su síntesis (ver core.RecordMemory).
package main

import (
	"fmt"
	"strings"

	"mova.local/core"
	"mova.local/models"
)

// taskListing arma el texto de /tasks.
func taskListing(root, project string, proj *core.Project, active string) string {
	if proj == nil || len(proj.Tasks) == 0 {
		return "[Task] Este proyecto no declara tareas.\n"
	}
	var b strings.Builder
	cur := active
	if core.IsAllTasks(active) {
		cur = "all"
	}
	b.WriteString("[Task] Tarea activa: " + cur + "\n")
	for _, n := range core.SortedTaskNames(proj) {
		mark := " "
		if n == active {
			mark = "*"
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", mark, n))
	}
	b.WriteString("  /task <nombre> | /task all | /run <nombre>\n")
	return b.String()
}

// switchChatTask cambia la tarea activa y reconstruye el contexto ya mismo.
// Devuelve la nueva tarea (o la anterior si falló) y la firma a usar.
func switchChatTask(root, project, newTask string, sess *models.Session, proj *core.Project,
	adapter core.Adapter, active string) (string, *core.Project, core.Adapter, string, bool) {
	target := core.NormalizeTaskArg(proj, newTask)
	if !core.IsAllTasks(target) {
		if _, ok := proj.Tasks[target]; !ok {
			consolePrint(fmt.Sprintf("[Task] \"%s\" no existe. Disponibles: %s\n", newTask, strings.Join(core.SortedTaskNames(proj), ", ")))
			return active, proj, adapter, "", false
		}
	}
	// firma vacía ⇒ refreshProjectContext reconstruye sí o sí.
	freshProj, freshAdapter, sig := refreshProjectContext(root, project, target, sess, proj, adapter, "", nil)
	if sig == "" { // el nuevo contexto no pasó el gate: se queda la tarea anterior
		return active, proj, adapter, "", false
	}
	label := target
	if core.IsAllTasks(target) {
		label = "todas"
	}
	consolePrint("[Task] Tarea activa: " + label + " — contexto recargado; el historial del chat se conserva.\n")
	return target, freshProj, freshAdapter, sig, true
}

// taskQuery: el mensaje que /run envía al modelo para una tarea.
func taskQuery(proj *core.Project, task string) string {
	if t, ok := proj.Tasks[task]; ok {
		if q := strings.TrimSpace(t.Variables["QUERY"]); q != "" {
			return q
		}
	}
	return "Ejecuta la tarea \"" + task + "\" según su prompt y el contexto, usando los resultados guardados de otras tareas si existen."
}
