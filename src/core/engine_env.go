// engine_env.go — ensamblado por piezas de ContextSections. Antes todo
// vivía en una sola función de ~230 líneas dentro de engine.go; ahora cada
// sección (agents, skills, prompts, focus, grafo, memoria) es un método de
// buildEnv, y el MISMO código sirve para una tarea concreta y para TaskAll
// (todas las tareas: unión de agents/skills, un prompt por tarea con SUS
// variables, focus/grafo de cada una). Con una sola tarea la salida es
// byte a byte la de siempre.
package core

import (
	"fmt"
	"strings"

	focusrender "mova.local/core/focus/render"
)

type buildEnv struct {
	adapter                     Adapter
	root, projectName, taskName string
	proj                        *Project
	tasks                       []string // tareas en alcance (una, o todas)
	profile                     *LLMProfile
	lang                        string
	builtin                     map[string]string
	agentVars, skillVars        map[string]string
	coreLoaded, dedupSeen       map[string]bool
	sections                    *ContextSections
	dbg                         strings.Builder
	repoPath                    string
}

func newBuildEnv(adapter Adapter, root, projectName, taskName string, proj *Project) *buildEnv {
	e := &buildEnv{
		adapter: adapter, root: root, projectName: projectName, taskName: taskName, proj: proj,
		tasks: tasksInScope(proj, taskName), profile: resolveProfile(proj), lang: proj.Lang,
		coreLoaded: map[string]bool{}, dedupSeen: map[string]bool{}, sections: &ContextSections{},
		repoPath: focusrender.ResolveRepoPath(root, proj.Repo),
	}
	taskLabel := taskName
	if IsAllTasks(taskName) {
		taskLabel = "all"
	}
	// Variables — ONE dynamic engine for agents, skills and prompt. Layers,
	// lowest to highest: built-ins (PROJECT, REPO, TASK, LANG) < project
	// "variables" < this block's own "variables" (agents / skills) < task
	// "variables". Any KEY is injected as ${KEY} or {{KEY}}, whatever its name.
	e.builtin = map[string]string{"PROJECT": proj.Project, "REPO": proj.Repo, "TASK": taskLabel, "LANG": proj.Lang}
	tv := e.sharedTaskVars()
	e.agentVars = mergeVars(e.builtin, proj.Variables, proj.Agents.Variables, tv)
	e.skillVars = mergeVars(e.builtin, proj.Variables, proj.Skills.Variables, tv)
	if proj.Debug {
		fmt.Fprintf(&e.dbg, "[debug] repo: %s\n", e.repoPath)
		writePolicyDebugLines(&e.dbg, root, projectName, proj)
	}
	return e
}

// sharedTaskVars: variables de tarea para agents/skills. Una tarea → las
// suyas. TaskAll → unión; si dos tareas definen la misma clave gana la
// primera por nombre (los prompts siempre usan las de SU tarea).
func (e *buildEnv) sharedTaskVars() map[string]string {
	out := map[string]string{}
	for _, n := range e.tasks {
		for k, v := range e.proj.Tasks[n].Variables {
			if _, dup := out[k]; !dup {
				out[k] = v
			}
		}
	}
	return out
}

func (e *buildEnv) header() string {
	scope := e.taskName
	if IsAllTasks(e.taskName) {
		scope = "todas las tareas (" + strings.Join(e.tasks, ", ") + ")"
	}
	// No timestamp here: the released bytes must be a pure function of
	// (repo, project.json, policies, memory) so their sha256 in the run
	// evidence is comparable across runs. The time lives in manifest.json.
	return fmt.Sprintf("# Mova Context — %s / %s\nRepo: %s | Lang: %s | LLM: %s | Profile: %s\n",
		e.proj.Project, scope, e.proj.Repo,
		orDefault(e.lang, "legacy"), orDefault(e.proj.LLM, "not set"), profileLabel(e.profile))
}

func (e *buildEnv) agentNames() []string {
	names := append([]string{}, e.proj.Agents.Use...)
	for _, n := range e.tasks {
		names = append(names, e.proj.Tasks[n].Agents...)
	}
	return dedupe(append(names, e.proj.Agents.Custom...))
}

func (e *buildEnv) skillNames() []string {
	names := append([]string{}, e.proj.Skills.Use...)
	for _, n := range e.tasks {
		names = append(names, e.proj.Tasks[n].Skills...)
	}
	return dedupe(append(names, e.proj.Skills.Custom...))
}

// buildKnowledge arma AGENTS o SKILLS (kind "agent"/"skill").
func (e *buildEnv) buildKnowledge(kind, title, domain string, names []string, vars map[string]string) {
	if len(names) == 0 {
		return
	}
	paths := e.proj.Paths.forKind(kind)
	var sb strings.Builder
	sb.WriteString("\n\n---\n## " + title + "\n")
	if c := loadCore(e.adapter, kind, domain, e.lang, coreFiles[kind], paths, e.coreLoaded); c != "" {
		text := dedupSection(inject(adaptContent(c, e.profile), vars), e.dedupSeen, e.sections)
		sb.WriteString(fmt.Sprintf("\n<!-- core: %s -->\n%s\n", coreFiles[kind], text))
	}
	for _, name := range names {
		if name == coreFiles[kind] {
			continue
		}
		c, fromFile, loc := resolveKnowledgeOrLiteral(e.adapter, kind, domain, e.lang, name, paths)
		if c == "" {
			continue
		}
		text := dedupSection(inject(adaptContent(c, e.profile), vars), e.dedupSeen, e.sections)
		label := name
		if !fromFile {
			label = "inline"
		}
		sb.WriteString(fmt.Sprintf("\n<!-- %s: %s -->\n%s\n", kind, label, text))
		if e.proj.Debug {
			fmt.Fprintf(&e.dbg, "[debug] %s: %s -> %s\n", kind, name, debugLoc(fromFile, loc))
		}
	}
	if kind == "agent" {
		e.sections.Agents = sb.String()
	} else {
		e.sections.Skills = sb.String()
	}
}

// buildPrompts: el prompt de CADA tarea en alcance (con sus variables). En
// TaskAll cada prompt lleva su rótulo de tarea para que el modelo sepa
// cuál ejecutar.
func (e *buildEnv) buildPrompts() {
	paths := e.proj.Paths.forKind("prompt")
	var sb strings.Builder
	hasPrompt := false
	for _, n := range e.tasks {
		task := e.proj.Tasks[n]
		if task.Prompt == "" {
			continue
		}
		vars := mergeVars(e.builtin, map[string]string{"TASK": n}, e.proj.Variables, task.Variables)
		if !hasPrompt {
			hasPrompt = true
			sb.WriteString("\n\n---\n## PROMPT\n")
			if c := loadCore(e.adapter, "prompt", e.proj.Agents.Domain, e.lang, coreFiles["prompt"], paths, e.coreLoaded); c != "" {
				text := dedupSection(inject(adaptContent(c, e.profile), vars), e.dedupSeen, e.sections)
				sb.WriteString(fmt.Sprintf("\n<!-- core: %s -->\n%s\n", coreFiles["prompt"], text))
			}
		}
		c, fromFile, loc := resolveKnowledgeOrLiteral(e.adapter, "prompt", e.proj.Agents.Domain, e.lang, task.Prompt, paths)
		if c == "" {
			continue
		}
		text := dedupSection(inject(adaptContent(c, e.profile), vars), e.dedupSeen, e.sections)
		label := task.Prompt
		if !fromFile {
			label = "inline"
		}
		sb.WriteString(fmt.Sprintf("\n<!-- prompt: %s -->\n", label))
		if IsAllTasks(e.taskName) {
			sb.WriteString(fmt.Sprintf("### Tarea: %s\n", n))
		}
		sb.WriteString(text + "\n")
		if e.proj.Debug {
			fmt.Fprintf(&e.dbg, "[debug] prompt: %s -> %s\n", task.Prompt, debugLoc(fromFile, loc))
		}
	}
	e.sections.Prompt = sb.String()
}
