// compiler_run.go — Context Compiler entry points: `mova run` (automatic
// dispatch) and `mova compile` (contexto.txt assembly). Kept separate from
// engine.go so buildContext (the original, always-available path) never
// has to know the Context Compiler exists.
package main

import (
	"fmt"
	"strings"
)

// resolveContext is the single entry point used by `mova run` and MCP's
// get_full_context. It implements the workflow.md decision:
//   workflow.md → project.json → Context Compiler enabled + automatic?
//     yes → compiled context   |   no → normal context
// Manual mode never triggers here — only via `mova compile`.
func resolveContext(adapter Adapter, root, projectName, taskName string) (string, error) {
	proj, err := adapter.GetProject(projectName)
	if err != nil {
		return "", err
	}
	if compilerEnabled(proj.ContextCompiler) && compilerMode(proj.ContextCompiler) == "automatic" {
		return buildCompiledContext(adapter, root, projectName, taskName)
	}
	return buildContext(adapter, projectName, taskName)
}

// buildCompiledContext is the Context Compiler (mova compile). It produces
// contexto.txt: the same knowledge as buildContext, but distilled per
// Fase 1 (Telegrama Semántico) and pruned per Fase 2 (focus). Output is
// compact — no headers for humans, no double blank lines.
func buildCompiledContext(adapter Adapter, root, projectName, taskName string) (string, error) {
	proj, err := adapter.GetProject(projectName)
	if err != nil {
		return "", err
	}
	if taskName == "" {
		taskName = proj.DefaultTask
	}
	task, ok := proj.Tasks[taskName]
	if !ok {
		return "", fmt.Errorf("task %q not found — available: %s", taskName, availableTasks(proj))
	}

	vars := mergeVars(proj.Variables, task.Variables)
	vars["project"] = proj.Project
	vars["repo"] = proj.Repo
	vars["task"] = taskName
	vars["lang"] = proj.Lang

	domain := proj.Agents.Domain
	lang := proj.Lang
	profile := resolveProfile(proj)
	strategy := compilerStrategy(proj.ContextCompiler)
	coreLoaded := map[string]bool{}

	distill := func(c string) string {
		c = inject(adaptContent(c, profile), vars)
		if strategy == "semantic" {
			c = CompileInstruction(c)
		}
		return c
	}

	var out strings.Builder
	out.WriteString(fmt.Sprintf("PROJECT:%s TASK:%s LANG:%s STRATEGY:%s\n",
		proj.Project, taskName, orDefault(lang, "legacy"), strategy))

	if core := loadCore(adapter, "agent", domain, lang, coreFiles["agent"], coreLoaded); core != "" {
		out.WriteString("AGENT:" + coreFiles["agent"] + "\n" + distill(core) + "\n")
	}
	allAgents := dedupe(append(append(append([]string{}, proj.Agents.Use...), task.Agents...), proj.Agents.Custom...))
	for _, name := range allAgents {
		if name == coreFiles["agent"] {
			continue
		}
		if c, err := adapter.GetKnowledge("agent", domain, lang, name); err == nil && c != "" {
			out.WriteString("AGENT:" + name + "\n" + distill(c) + "\n")
		}
	}

	if core := loadCore(adapter, "skill", proj.Skills.Domain, lang, coreFiles["skill"], coreLoaded); core != "" {
		out.WriteString("SKILL:" + coreFiles["skill"] + "\n" + distill(core) + "\n")
	}
	allSkills := dedupe(append(append(append([]string{}, proj.Skills.Use...), task.Skills...), proj.Skills.Custom...))
	for _, name := range allSkills {
		if name == coreFiles["skill"] {
			continue
		}
		if c, err := adapter.GetKnowledge("skill", proj.Skills.Domain, lang, name); err == nil && c != "" {
			out.WriteString("SKILL:" + name + "\n" + distill(c) + "\n")
		}
	}

	if core := loadCore(adapter, "prompt", domain, lang, coreFiles["prompt"], coreLoaded); core != "" {
		out.WriteString("PROMPT:" + coreFiles["prompt"] + "\n" + distill(core) + "\n")
	}
	if task.Prompt != "" {
		if c, err := adapter.GetKnowledge("prompt", domain, lang, task.Prompt); err == nil && c != "" {
			out.WriteString("PROMPT:" + task.Prompt + "\n" + distill(c) + "\n")
		}
	}

	if mem, _ := adapter.GetMemory(projectName); mem != "" {
		out.WriteString("MEMORY:\n" + distill(mem) + "\n")
	}

	// Fase 2 — never applies Fase 1 text-distillation to focus content:
	// it may be source code, which must never be rewritten.
	if focus := resolveFocus(proj, &task); len(focus) > 0 {
		out.WriteString(buildFocusContext(root, proj, focus))
	}

	out.WriteString("CONTRACT:reply with ```memory block (Done/Resolved/Pending/Decisions/LLM Errors)\n")

	return collapseBlankLines(out.String()) + "\n", nil
}
