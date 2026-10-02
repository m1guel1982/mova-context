// task_scope.go — qué tarea(s) abarca un contexto. Una tarea concreta
// ("analizar") usa SOLO su prompt/focus/grafo; TaskAll ("*") abarca todas
// las tareas del project.json a la vez (todos sus prompts, focus y grafos).
// Lo usan chat, MCP chat_completion y HTTP (que es MCP) — ver ChatTaskName.
package core

import (
	"sort"
	"strings"
)

// TaskAll es el "nombre de tarea" que significa "todas las tareas".
const TaskAll = "*"

// IsAllTasks: ¿el nombre pedido abarca todas las tareas?
func IsAllTasks(name string) bool { return name == TaskAll }

// NormalizeTaskArg convierte los alias humanos de "todas" ("all",
// "todas", "*") en TaskAll, salvo que exista una tarea con ese nombre.
func NormalizeTaskArg(proj *Project, arg string) string {
	arg = strings.TrimSpace(arg)
	if proj != nil {
		if _, ok := proj.Tasks[arg]; ok {
			return arg
		}
	}
	switch strings.ToLower(arg) {
	case "*", "all", "todas", "todo", "todos":
		return TaskAll
	}
	return arg
}

// ChatTaskName decide la tarea de una sesión de chat/MCP/HTTP: la pedida
// explícitamente; si no se pidió ninguna y el proyecto tiene VARIAS
// tareas, TaskAll (se ejecutan todas); con una sola tarea, esa. (`mova
// run` sigue usando ResolveTaskName/default_task: no cambia.)
func ChatTaskName(proj *Project, requested string) string {
	if requested = NormalizeTaskArg(proj, requested); requested != "" {
		return requested
	}
	if proj != nil && len(proj.Tasks) > 1 {
		return TaskAll
	}
	return ResolveTaskName(proj, "")
}

// SortedTaskNames: nombres de tarea en orden estable.
func SortedTaskNames(proj *Project) []string {
	names := make([]string, 0, len(proj.Tasks))
	for n := range proj.Tasks {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// tasksInScope: las tareas que cubre taskName (todas, o solo esa).
func tasksInScope(proj *Project, taskName string) []string {
	if IsAllTasks(taskName) {
		return SortedTaskNames(proj)
	}
	return []string{taskName}
}
