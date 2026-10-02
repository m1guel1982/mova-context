// engine_env_focus.go — focus, grafo, memoria (+ resultados de tareas) e
// instrucción del ensamblado de contexto (ver engine_env.go).
package core

import (
	"fmt"
	"strings"

	focusrender "mova.local/core/focus/render"
)

// buildFocus: el focus de cada tarea en alcance, con SU exclude. Tareas
// que resuelven exactamente al mismo focus+exclude se renderizan una vez.
func (e *buildEnv) buildFocus() {
	done := map[string]bool{}
	var text strings.Builder
	for _, n := range e.tasks {
		task := e.proj.Tasks[n]
		items := resolveTaskFocus(e.proj, &task)
		if len(items) == 0 {
			continue
		}
		exclude := resolveTaskExclude(e.proj, &task)
		sig := strings.Join(items, "\x00") + "\x01" + strings.Join(exclude, "\x00")
		if done[sig] {
			continue
		}
		done[sig] = true
		out, stats := focusrender.RenderFocusContextWithSeen(e.root, e.proj.Repo, items, nil, exclude, e.dedupSeen)
		e.sections.DuplicatesRemoved += stats.DuplicatesRemoved
		e.sections.DuplicatesRemovedChars += stats.DuplicatesRemovedChars
		e.sections.FocusItems = append(e.sections.FocusItems, stats.Items...)
		text.WriteString(out)
		if e.proj.Debug {
			writeFocusDebug(&e.dbg, e.repoPath, stats.Items)
			writeExcludeDebug(&e.dbg, focusrender.ResolveExcludeTargets(e.root, e.proj.Repo, exclude), e.repoPath)
		}
	}
	if strings.TrimSpace(text.String()) != "" {
		e.sections.Focus = "\n\n---\n## FOCUS\n" + text.String()
	}
}

// buildGraph: un grafo por cada tarea EN ALCANCE que declare "graph"
// (ver graphJobs). Sin LLM: análisis AST local de su focus/exclude. En
// chat/MCP/HTTP corre en segundo plano (GraphAsync).
func (e *buildEnv) buildGraph() {
	if GraphHook == nil {
		return
	}
	jobs := graphJobs(e.proj, e.taskName)
	if len(jobs) == 0 {
		return
	}
	e.sections.GraphStatus = GraphHook(GraphRequest{
		Root: e.root, ProjectName: e.projectName, ActiveTask: e.taskName, Lang: e.proj.Lang,
		RepoPath: e.repoPath, Jobs: jobs,
	})
	if e.proj.Debug && e.sections.GraphStatus != "" {
		for _, l := range strings.Split(e.sections.GraphStatus, "\n") {
			e.dbg.WriteString("[debug] " + l + "\n")
		}
	}
}

// buildMemory: memory.md (la memoria viva del proyecto) acotada por
// FormatMemoryForContext. La leen igual todas las tareas y las tres
// puertas (chat, MCP, HTTP) porque todas ensamblan el contexto aquí.
func (e *buildEnv) buildMemory() {
	m, _ := e.adapter.GetMemory(e.projectName)
	if m == "" {
		return
	}
	m = FormatMemoryForContext(m, e.proj.MemoryMaxChars)
	if m = dedupSection(m, e.dedupSeen, e.sections); strings.TrimSpace(m) != "" {
		e.sections.Memory = "\n\n---\n## MEMORY\n" + m
	}
}

func (e *buildEnv) instruction() string {
	var b strings.Builder
	b.WriteString("\n\n---\n## INSTRUCTION\n")
	b.WriteString(fmt.Sprintf("Project: **%s** | Repo: `%s`\n", e.proj.Project, e.proj.Repo))
	es := e.lang == "es"
	if IsAllTasks(e.taskName) {
		if es {
			b.WriteString(fmt.Sprintf("Tareas disponibles: %s. Ejecuta la tarea que el usuario nombre; si no nombra ninguna, ejecuta todas en ese orden, cada una con su propio prompt.\n", strings.Join(e.tasks, ", ")))
		} else {
			b.WriteString(fmt.Sprintf("Available tasks: %s. Run the task the user names; if none is named, run all of them in that order, each with its own prompt.\n", strings.Join(e.tasks, ", ")))
		}
	}
	if es {
		b.WriteString("Aplica los prompts y contexto anterior. Entrega tu informe técnico y finaliza ÚNICAMENTE con el siguiente bloque de síntesis. Es la MEMORIA que leerán las demás tareas: no guardes el chat completo, pero tampoco omitas nada que otra tarea necesite para continuar sin repetir tu análisis:\n\n")
		b.WriteString("```memory\n## YYYY-MM-DD — session\n**Tarea:** <nombre de la tarea ejecutada>\n**Realizado:** <resumen corto de 1 línea>\n**Hallazgos:** <uno por línea: `archivo::función` — dato/clave — estado (existe / se pierde / se sobrescribe) — causa o evidencia>\n**Datos clave:** <identificadores EXACTOS que otra tarea necesitará: funciones, claves, columnas, rutas, valores>\n**Resuelto:** <hallazgos resueltos>\n**Decisiones:** <reglas de negocio y decisiones de diseño acordadas>\n**Pendiente:** <próximos pasos concretos y deuda técnica>\n**Errores del LLM:** <ninguno u observaciones>\n```\n\nReglas del bloque: copia literalmente archivos, funciones, claves y columnas (sin parafrasear); incluye solo el código mínimo imprescindible; si trabajaste varias tareas, entrega un bloque por tarea.\n")
	} else {
		b.WriteString("Apply the prompt and context above. Deliver your technical report and conclude EXCLUSIVELY with the following summary block. It is the MEMORY other tasks will read: do not store the full chat, but do not omit anything another task needs to continue without redoing your analysis:\n\n")
		b.WriteString("```memory\n## YYYY-MM-DD — session\n**Task:** <name of the task executed>\n**Done:** <1-line summary>\n**Findings:** <one per line: `file::function` — field/key — status (exists / lost / overwritten) — cause or evidence>\n**Key data:** <EXACT identifiers another task will need: functions, keys, columns, paths, values>\n**Resolved:** <key findings fixed>\n**Decisions:** <business rules and design choices agreed>\n**Pending:** <concrete next steps and tech debt>\n**LLM Errors:** <none or notes>\n```\n\nBlock rules: copy files, functions, keys and columns verbatim (no paraphrasing); include only the minimum necessary code; if you worked several tasks, give one block per task.\n")
	}
	return b.String()
}
