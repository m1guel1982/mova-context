// engine.go — assembles context from knowledge pieces.
// Reads project.json, loads agents/skills/prompt/memory, injects variables.
// Does not know where data comes from. That's the adapter's job.
package core

import (
	"fmt"
	"strings"

	corefocus "mova.local/core/focus"
)

// coreFiles maps each knowledge kind to its core filename.
// Core files are always loaded once, before their section, never duplicated.
var coreFiles = map[string]string{
	"agent":  "yagni-core",
	"skill":  "kiss-dry-core",
	"prompt": "ockham-core",
}

// ContextSections holds the assembled context split into its individual
// pieces — Header/Instruction are fixed engine boilerplate; Agents,
// Skills, Prompt, Focus, and Memory each correspond to a concrete part of
// project.json (or of the running session, for Memory). BuildContext
// concatenates these exactly as it always has (see Full()); mova.local/budget
// keeps them separate to report token cost per component instead of only
// a single total — same assembly, two consumers, zero duplicated logic.
//
// DuplicatesRemoved counts exact-paragraph duplicates removed across the
// WHOLE assembly (agents+skills+prompt+focus+memory share one dedup.Paragraphs
// "seen" map, see BuildContextSections) — a paragraph pasted into two
// agents, or repeated between a skill and the task prompt, only survives
// once in the final context. Never a reformulation, never a "similar"
// match: exact text only (see mova.local/dedup).
type ContextSections struct {
	Header                 string
	Agents                 string
	Skills                 string
	Prompt                 string
	Focus                  string // "" when the project/task has no focus configured
	Memory                 string // "" when memory.md is empty
	Instruction            string
	DuplicatesRemoved      int
	DuplicatesRemovedChars int

	// FocusItems: un ítem por target de `focus`/`memory` YA resuelto
	// (ver mova.local/core/focus.FocusItem) — nunca uno por archivo
	// individual dentro de un directorio. Vacío cuando el proyecto/task
	// no tiene `focus` configurado. Alimenta el resumen "[Focus]
	// Selected ..." de `mova chat`/chat_completion (ver
	// FormatFocusSelection) sin volver a resolver nada.
	FocusItems []corefocus.FocusItem

	// DebugLog: human-readable trace of what THIS run resolved
	// (repo, each agent/skill/prompt's name and file path or
	// "inline", each focus/exclude entry and its resolved absolute
	// path) — only populated when project.json's "debug" is true
	// (see core.Project.Debug). Every door (chat, mova ui chat, CLI,
	// HTTP API, MCP) prints this exactly as-is when non-empty; none
	// of them recomputes their own version.
	DebugLog string

	// GraphStatus: línea traducida del generador de grafos ("" si la
	// tarea no define "graph"). Ver graph_hook.go.
	GraphStatus string
}

// Full concatenates every section in the exact order and format
// BuildContext has always produced. `mova run`, the MCP get_full_context
// tool, and the HTTP transport all go through BuildContext → Full(), so
// none of them sees any change in output from this refactor.
func (s *ContextSections) Full() string {
	var out strings.Builder
	out.WriteString(s.Header)
	out.WriteString(s.Agents)
	out.WriteString(s.Skills)
	out.WriteString(s.Prompt)
	out.WriteString(s.Focus)
	out.WriteString(s.Memory)
	out.WriteString(s.Instruction)
	return out.String()
}

// BuildContext is the core operation of Mova Context.
// Equivalent to the original cmdRun, decoupled from I/O.
func BuildContext(adapter Adapter, root, projectName, taskName string) (string, error) {
	sections, err := BuildContextSections(adapter, root, projectName, taskName)
	if err != nil {
		return "", err
	}
	return sections.Full(), nil
}

// ResolveTaskName decides which task applies when the caller may not
// have named one: the explicit taskName, or proj.DefaultTask, or — if
// the project only declares a SINGLE task — that one task, since there's
// nothing ambiguous to resolve. Returns "" when none of those apply
// (multiple tasks exist and none was specified or set as default).
//
// This is the ONE place that decision is made — BuildContextSections
// uses it to pick which task's prompt/agents/skills to assemble, and
// every Budget-gate call site (cli/run_cmd.go, cli/chat_cmd.go,
// mcp/context_tool.go, mcp/chat_tool.go) uses it too, via
// budget.ResolveTask(proj, core.ResolveTaskName(proj, taskName)), so the
// task the Budget check validates against is ALWAYS the same one the
// context was actually built from — never a mismatch between the two.
func ResolveTaskName(proj *Project, taskName string) string {
	if taskName != "" {
		return taskName
	}
	if proj.DefaultTask != "" {
		return proj.DefaultTask
	}
	if len(proj.Tasks) == 1 {
		for name := range proj.Tasks {
			return name
		}
	}
	return ""
}

// BuildContextSections does the exact same assembly as BuildContext,
// split into its individual pieces — used by mova.local/budget to report
// token cost per component (agents, skills, prompt, focus, memory)
// instead of only a single opaque total. Whatever project.json/task
// declare here is exactly what `mova run`, `mova budget`, the MCP
// get_full_context/estimate_budget tools, and chat_completion all see —
// one assembly, every transport.
//
// taskName puede ser una tarea concreta (solo su prompt/focus/grafo) o
// TaskAll (todas las tareas a la vez, ver task_scope.go). El ensamblado
// por piezas vive en engine_env.go.
func BuildContextSections(adapter Adapter, root, projectName, taskName string) (*ContextSections, error) {
	proj, err := adapter.GetProject(projectName)
	if err != nil {
		return nil, err
	}
	if IsAllTasks(taskName) {
		if len(proj.Tasks) == 0 {
			return nil, fmt.Errorf("task %q not found — available: %s", taskName, availableTasks(proj))
		}
	} else {
		taskName = ResolveTaskName(proj, taskName)
		if _, ok := proj.Tasks[taskName]; !ok {
			return nil, fmt.Errorf("task %q not found — available: %s", taskName, availableTasks(proj))
		}
	}

	e := newBuildEnv(adapter, root, projectName, taskName, proj)
	e.sections.Header = e.header()
	e.buildKnowledge("agent", "AGENTS", proj.Agents.Domain, e.agentNames(), e.agentVars)
	e.buildKnowledge("skill", "SKILLS", proj.Skills.Domain, e.skillNames(), e.skillVars)
	e.buildPrompts()
	e.buildFocus()
	e.buildGraph()
	e.buildMemory()
	e.sections.Instruction = e.instruction()
	e.sections.DebugLog = e.dbg.String()
	return e.sections, nil
}
