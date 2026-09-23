// engine.go — assembles context from knowledge pieces.
// Reads project.json, loads agents/skills/prompt/memory, injects variables.
// Does not know where data comes from. That's the adapter's job.
package core

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	corefocus "mova.local/core/focus"
	focusrender "mova.local/core/focus/render"
	"mova.local/dedup"
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
func BuildContextSections(adapter Adapter, root, projectName, taskName string) (*ContextSections, error) {
	proj, err := adapter.GetProject(projectName)
	if err != nil {
		return nil, err
	}

	taskName = ResolveTaskName(proj, taskName)
	task, ok := proj.Tasks[taskName]
	if !ok {
		return nil, fmt.Errorf("task %q not found — available: %s",
			taskName, availableTasks(proj))
	}

	// Variables: project-level merged with task-level (task wins on conflict)
	vars := mergeVars(proj.Variables, task.Variables)
	vars["project"] = proj.Project
	vars["repo"] = proj.Repo
	vars["task"] = taskName
	vars["lang"] = proj.Lang

	domain := proj.Agents.Domain
	lang := proj.Lang
	profile := resolveProfile(proj)

	// Track which core files have been loaded (avoid duplicates)
	coreLoaded := map[string]bool{}
	// Shared across AGENTS/SKILLS/PROMPT/FOCUS/MEMORY — see doc comment above.
	dedupSeen := map[string]bool{}

	sections := &ContextSections{}
	var dbg strings.Builder
	if proj.Debug {
		fmt.Fprintf(&dbg, "[debug] repo: %s\n", resolveDebugPath(root, proj.Repo))
		writePolicyDebugLines(&dbg, root, projectName, proj)
	}

	var header strings.Builder
	header.WriteString(fmt.Sprintf("# Mova Context — %s / %s\n", proj.Project, taskName))
	header.WriteString(fmt.Sprintf("Generated: %s | Repo: %s | Lang: %s | LLM: %s | Profile: %s\n",
		time.Now().Format("2006-01-02 15:04"), proj.Repo, orDefault(lang, "legacy"), orDefault(proj.LLM, "not set"), profileLabel(profile)))
	sections.Header = header.String()

	// ── AGENTS ──────────────────────────────────────────────────────────────
	allAgents := dedupe(append(append(append([]string{}, proj.Agents.Use...), task.Agents...), proj.Agents.Custom...))
	if len(allAgents) > 0 {
		var agents strings.Builder
		agents.WriteString("\n\n---\n## AGENTS\n")

		if core := loadCore(adapter, "agent", domain, lang, coreFiles["agent"], coreLoaded); core != "" {
			text := inject(adaptContent(core, profile), vars)
			text = dedupSection(text, dedupSeen, sections)
			agents.WriteString(fmt.Sprintf("\n<!-- core: %s -->\n%s\n", coreFiles["agent"], text))
		}

		for _, name := range allAgents {
			if name == coreFiles["agent"] {
				continue
			}
			c, fromFile := resolveKnowledgeOrLiteral(adapter, "agent", domain, lang, name)
			if c == "" {
				continue
			}
			text := inject(adaptContent(c, profile), vars)
			text = dedupSection(text, dedupSeen, sections)
			label := name
			if !fromFile {
				label = "inline"
			}
			agents.WriteString(fmt.Sprintf("\n<!-- agent: %s -->\n%s\n", label, text))
			if proj.Debug {
				fmt.Fprintf(&dbg, "[debug] agent: %s -> %s\n", name, debugKnowledgeLoc(fromFile, "agent", domain, lang, name))
			}
		}
		sections.Agents = agents.String()
	}

	// ── SKILLS ──────────────────────────────────────────────────────────────
	allSkills := dedupe(append(append(append([]string{}, proj.Skills.Use...), task.Skills...), proj.Skills.Custom...))
	if len(allSkills) > 0 {
		var skills strings.Builder
		skills.WriteString("\n\n---\n## SKILLS\n")

		if core := loadCore(adapter, "skill", proj.Skills.Domain, lang, coreFiles["skill"], coreLoaded); core != "" {
			text := inject(adaptContent(core, profile), vars)
			text = dedupSection(text, dedupSeen, sections)
			skills.WriteString(fmt.Sprintf("\n<!-- core: %s -->\n%s\n", coreFiles["skill"], text))
		}

		for _, name := range allSkills {
			if name == coreFiles["skill"] {
				continue
			}
			c, fromFile := resolveKnowledgeOrLiteral(adapter, "skill", proj.Skills.Domain, lang, name)
			if c == "" {
				continue
			}
			text := inject(adaptContent(c, profile), vars)
			text = dedupSection(text, dedupSeen, sections)
			label := name
			if !fromFile {
				label = "inline"
			}
			skills.WriteString(fmt.Sprintf("\n<!-- skill: %s -->\n%s\n", label, text))
			if proj.Debug {
				fmt.Fprintf(&dbg, "[debug] skill: %s -> %s\n", name, debugKnowledgeLoc(fromFile, "skill", proj.Skills.Domain, lang, name))
			}
		}
		sections.Skills = skills.String()
	}

	// ── PROMPT ──────────────────────────────────────────────────────────────
	if task.Prompt != "" {
		var prompt strings.Builder
		prompt.WriteString("\n\n---\n## PROMPT\n")

		if core := loadCore(adapter, "prompt", domain, lang, coreFiles["prompt"], coreLoaded); core != "" {
			text := inject(adaptContent(core, profile), vars)
			text = dedupSection(text, dedupSeen, sections)
			prompt.WriteString(fmt.Sprintf("\n<!-- core: %s -->\n%s\n", coreFiles["prompt"], text))
		}

		c, fromFile := resolveKnowledgeOrLiteral(adapter, "prompt", domain, lang, task.Prompt)
		if c != "" {
			text := inject(adaptContent(c, profile), vars)
			text = dedupSection(text, dedupSeen, sections)
			label := task.Prompt
			if !fromFile {
				label = "inline"
			}
			prompt.WriteString(fmt.Sprintf("\n<!-- prompt: %s -->\n%s\n", label, text))
			if proj.Debug {
				fmt.Fprintf(&dbg, "[debug] prompt: %s -> %s\n", task.Prompt, debugKnowledgeLoc(fromFile, "prompt", domain, lang, task.Prompt))
			}
		}
		sections.Prompt = prompt.String()
	}

	// ── FOCUS ───────────────────────────────────────────────────────────────

	if items := resolveTaskFocus(proj, &task); len(items) > 0 {
		exclude := resolveTaskExclude(proj, &task)
		text, stats := focusrender.RenderFocusContextWithSeen(root, proj.Repo, items, nil, exclude, dedupSeen)
		sections.DuplicatesRemoved += stats.DuplicatesRemoved
		sections.DuplicatesRemovedChars += stats.DuplicatesRemovedChars
		sections.FocusItems = stats.Items
		if strings.TrimSpace(text) != "" {
			sections.Focus = "\n\n---\n## FOCUS\n" + text
		}
		if proj.Debug {
			for _, it := range stats.Items {
				fmt.Fprintf(&dbg, "[debug] focus: %s -> %s\n", it.Name, resolveDebugPath(root, filepath.Join(proj.Repo, it.Name)))
			}
			for _, ex := range exclude {
				fmt.Fprintf(&dbg, "[debug] exclude: %s -> %s\n", ex, resolveDebugPath(root, filepath.Join(proj.Repo, ex)))
			}
		}
	}

	// ── MEMORY ──────────────────────────────────────────────────────────────
	if mem, _ := adapter.GetMemory(projectName); mem != "" {
		mem = dedupSection(mem, dedupSeen, sections)
		if strings.TrimSpace(mem) != "" {
			sections.Memory = "\n\n---\n## MEMORY\n" + mem
		}
	}

	// ── INSTRUCTION ─────────────────────────────────────────────────────────
	var instruction strings.Builder
	instruction.WriteString("\n\n---\n## INSTRUCTION\n")
	instruction.WriteString(fmt.Sprintf("Project: **%s** | Repo: `%s`\n", proj.Project, proj.Repo))

	if lang == "es" {
		instruction.WriteString("Aplica los prompts y contexto anterior. Entrega tu informe técnico y finaliza ÚNICAMENTE con el siguiente bloque de síntesis (no guardes el chat completo en memoria):\n\n")
		instruction.WriteString("```memory\n## YYYY-MM-DD — session\n**Realizado:** <resumen corto de 1 línea>\n**Resuelto:** <hallazgos resueltos>\n**Pendiente:** <deuda técnica o tareas futuras>\n**Decisiones:** <decisiones de diseño/stack>\n**Errores del LLM:** <ninguno u observaciones>\n```\n")
	} else {
		instruction.WriteString("Apply the prompt and context above. Deliver your technical report and conclude EXCLUSIVELY with the following summary block (do not store the full chat in memory):\n\n")
		instruction.WriteString("```memory\n## YYYY-MM-DD — session\n**Done:** <1-line summary>\n**Resolved:** <key findings fixed>\n**Pending:** <tech debt or future tasks>\n**Decisions:** <architecture/stack choices>\n**LLM Errors:** <none or notes>\n```\n")
	}
	sections.Instruction = instruction.String()
	sections.DebugLog = dbg.String()

	return sections, nil
}

// FormatFocusSelection arma la línea de estado "[Focus] Selected ..."
// a partir de los targets ya resueltos — usada por AMBOS `mova chat`
// (cli/chat_helpers.go) y el tool MCP/HTTP chat_completion
// (mcp/chat_tool.go), así CLI, HTTP y MCP muestran exactamente la misma
// línea para el mismo project.json. limit es cuántos nombres listar
// antes de colapsar el resto en un badge "+N" (ver FocusDisplayLimit).
// Devuelve "" cuando items está vacío (sin `focus`/`memory` configurado
// — no imprime nada, igual que antes).
func FormatFocusSelection(items []corefocus.FocusItem, limit int) string {
	if len(items) == 0 {
		return ""
	}
	if limit <= 0 {
		limit = DefaultFocusDisplayLimit
	}

	files, dirs, totalFiles := 0, 0, 0
	names := make([]string, 0, len(items))
	for _, it := range items {
		if it.Kind == "dir" {
			dirs++
		} else {
			files++
		}
		totalFiles += it.Files
		names = append(names, it.Name)
	}

	shownCount := limit
	if shownCount > len(names) {
		shownCount = len(names)
	}
	// Bug real reportado en QA: un target de `focus` literal "." (la
	// raíz del proyecto, ver corefocus.FocusItem.Name) hacía que la
	// lista terminara en "." — al concatenar el "." final fijo del
	// formato de abajo, el mensaje quedaba
	// "[Focus] Selected 1 directory (N file(s) total): ..\n" (dos
	// puntos seguidos), que parece un path truncado o corrupto en vez
	// de la raíz del proyecto. Se muestra "." como "(root)", legible,
	// en vez del carácter crudo de project.json.
	displayNames := make([]string, shownCount)
	for i, n := range names[:shownCount] {
		if n == "." {
			n = "(root)"
		}
		displayNames[i] = n
	}
	list := strings.Join(displayNames, ", ")
	if extra := len(names) - limit; extra > 0 {
		list += fmt.Sprintf(" 📎+%d", extra)
	}

	return fmt.Sprintf("[Focus] Selected %d %s (%d file(s) total): %s.\n",
		len(items), focusKindLabel(files, dirs), totalFiles, list)
}

// focusKindLabel describe la MEZCLA de targets resueltos ("2 files",
// "1 directory", "3 items" cuando hay de ambos tipos) — nunca inventa
// una categoría que no corresponda a lo realmente resuelto.
func focusKindLabel(files, dirs int) string {
	switch {
	case dirs == 0 && files == 1:
		return "file"
	case dirs == 0:
		return "files"
	case files == 0 && dirs == 1:
		return "directory"
	case files == 0:
		return "directories"
	default:
		return "item(s)"
	}
}

// DefaultFocusDisplayLimit: cuántos nombres de target muestra "[Focus]
// Selected ..." antes de colapsar el resto en "+N" cuando project.json
// no configura "focus_display_limit" — ver FocusDisplayLimit.
const DefaultFocusDisplayLimit = 2

// FocusDisplayLimit resuelve cuántos nombres mostrar: el
// "focus_display_limit" de project.json siempre gana; 0/ausente usa
// DefaultFocusDisplayLimit. Cualquier valor configurado (2, 4, 10, ...)
// se respeta tal cual — al superarlo SIEMPRE se colapsa el resto en el
// badge "+N", sin importar cuál sea el número.
func FocusDisplayLimit(proj *Project) int {
	if proj != nil && proj.FocusDisplayLimit > 0 {
		return proj.FocusDisplayLimit
	}
	return DefaultFocusDisplayLimit
}

// dedupSection applies dedup.Paragraphs to one prose chunk and accumulates
// its removed-count/removed-chars into sections — a small helper so every
// call site above stays a single readable line instead of repeating the
// same statements six times.
func dedupSection(text string, seen map[string]bool, sections *ContextSections) string {
	deduped, removed, chars := dedup.Paragraphs(text, seen)
	sections.DuplicatesRemoved += removed
	sections.DuplicatesRemovedChars += chars
	return deduped
}
