// engine.go — assembles context from knowledge pieces.
// Reads project.json, loads agents/skills/prompt/memory, injects variables.
// Does not know where data comes from. That's the adapter's job.
package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// coreFiles maps each knowledge kind to its core filename.
// Core files are always loaded once, before their section, never duplicated.
var coreFiles = map[string]string{
	"agent":  "yagni-core",
	"skill":  "kiss-dry-core",
	"prompt": "ockham-core",
}

// buildContext is the core operation of Mova Context.
// Equivalent to the original cmdRun, decoupled from I/O.
func buildContext(adapter Adapter, projectName, taskName string) (string, error) {
	proj, err := adapter.GetProject(projectName)
	if err != nil {
		return "", err
	}

	if taskName == "" {
		taskName = proj.DefaultTask
	}
	task, ok := proj.Tasks[taskName]
	if !ok {
		return "", fmt.Errorf("task %q not found — available: %s",
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

	var out strings.Builder
	out.WriteString(fmt.Sprintf("# Mova Context — %s / %s\n", proj.Project, taskName))
	out.WriteString(fmt.Sprintf("Generated: %s | Repo: %s | Lang: %s | LLM: %s | Profile: %s\n",
		time.Now().Format("2006-01-02 15:04"), proj.Repo, orDefault(lang, "legacy"), orDefault(proj.LLM, "not set"), profileLabel(profile)))

	// ── AGENTS ──────────────────────────────────────────────────────────────
	out.WriteString("\n\n---\n## AGENTS\n")

	// Load yagni-core once before any agent
	if core := loadCore(adapter, "agent", domain, lang, coreFiles["agent"], coreLoaded); core != "" {
		out.WriteString(fmt.Sprintf("\n<!-- core: %s -->\n%s\n", coreFiles["agent"], inject(adaptContent(core, profile), vars)))
	}

	allAgents := append(append([]string{}, proj.Agents.Use...), task.Agents...)
	allAgents = append(allAgents, proj.Agents.Custom...)
	for _, name := range dedupe(allAgents) {
		if name == coreFiles["agent"] {
			continue // already loaded above
		}
		c, err := adapter.GetKnowledge("agent", domain, lang, name)
		if err != nil || c == "" {
			continue
		}
		out.WriteString(fmt.Sprintf("\n<!-- agent: %s -->\n%s\n", name, inject(adaptContent(c, profile), vars)))
	}

	// ── SKILLS ──────────────────────────────────────────────────────────────
	out.WriteString("\n\n---\n## SKILLS\n")

	// Load kiss-dry-core once before any skill
	if core := loadCore(adapter, "skill", proj.Skills.Domain, lang, coreFiles["skill"], coreLoaded); core != "" {
		out.WriteString(fmt.Sprintf("\n<!-- core: %s -->\n%s\n", coreFiles["skill"], inject(adaptContent(core, profile), vars)))
	}

	allSkills := append(append([]string{}, proj.Skills.Use...), task.Skills...)
	allSkills = append(allSkills, proj.Skills.Custom...)
	for _, name := range dedupe(allSkills) {
		if name == coreFiles["skill"] {
			continue // already loaded above
		}
		c, err := adapter.GetKnowledge("skill", proj.Skills.Domain, lang, name)
		if err != nil || c == "" {
			continue
		}
		out.WriteString(fmt.Sprintf("\n<!-- skill: %s -->\n%s\n", name, inject(adaptContent(c, profile), vars)))
	}

	// ── PROMPT ──────────────────────────────────────────────────────────────
	out.WriteString("\n\n---\n## PROMPT\n")

	// Load ockham-core once before the prompt
	if core := loadCore(adapter, "prompt", domain, lang, coreFiles["prompt"], coreLoaded); core != "" {
		out.WriteString(fmt.Sprintf("\n<!-- core: %s -->\n%s\n", coreFiles["prompt"], inject(adaptContent(core, profile), vars)))
	}

	if task.Prompt != "" {
		c, err := adapter.GetKnowledge("prompt", domain, lang, task.Prompt)
		if err == nil && c != "" {
			out.WriteString(fmt.Sprintf("\n<!-- prompt: %s -->\n%s\n", task.Prompt, inject(adaptContent(c, profile), vars)))
		}
	}

	// ── MEMORY ──────────────────────────────────────────────────────────────
	if mem, _ := adapter.GetMemory(projectName); mem != "" {
		out.WriteString("\n\n---\n## MEMORY\n")
		out.WriteString(mem)
	}

	// ── INSTRUCTION ─────────────────────────────────────────────────────────
	out.WriteString("\n\n---\n## INSTRUCTION\n")
	out.WriteString(fmt.Sprintf("Project: **%s** | Repo: `%s`\n", proj.Project, proj.Repo))
	out.WriteString("Apply agents, skills and prompt above. When done, reply with:\n\n")
	out.WriteString("```memory\n## YYYY-MM-DD — session\n**Done:**\n**Resolved:**\n**Pending:**\n**Decisions:**\n**LLM Errors:**\n```\n")

	return out.String(), nil
}

func loadCore(adapter Adapter, kind, domain, lang, name string, loaded map[string]bool) string {
	if loaded[name] {
		return ""
	}
	c, err := adapter.GetKnowledge(kind, domain, lang, name)
	if err != nil || c == "" {
		return ""
	}
	loaded[name] = true
	return c
}

// extractMemoryBlock pulls the ```memory block from an LLM response.
func extractMemoryBlock(response string) (string, error) {
	start := strings.Index(response, "```memory")
	end := strings.LastIndex(response, "```")
	if start == -1 || end <= start {
		return "", fmt.Errorf("no ```memory block found")
	}
	return strings.TrimSpace(response[start+len("```memory") : end]), nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func mergeVars(global, task map[string]string) map[string]string {
	out := make(map[string]string, len(global)+len(task))
	for k, v := range global {
		out[k] = v
	}
	for k, v := range task {
		out[k] = v
	}
	return out
}

func inject(text string, vars map[string]string) string {
	for k, v := range vars {
		text = strings.ReplaceAll(text, "{{"+strings.ToUpper(k)+"}}", v)
	}
	return text
}

func availableTasks(p *Project) string {
	names := make([]string, 0, len(p.Tasks))
	for k := range p.Tasks {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func dedupe(items []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range items {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// resolveProfile returns the effective LLM profile for a project.
// Priority: llm_profile block > llm string field > default (powerful).
func resolveProfile(proj *Project) *LLMProfile {
	if proj.LLMProfile != nil {
		return proj.LLMProfile
	}
	switch proj.LLM {
	case "ollama", "llama", "mistral", "deepseek", "qwen", "gemma", "phi":
		return &LLMProfile{Type: "local", Provider: proj.LLM}
	default:
		return &LLMProfile{Type: "powerful", Provider: proj.LLM}
	}
}

// adaptContent applies light formatting normalization for local models.
// For powerful models it returns the content unchanged.
// The original files are NEVER modified.
func adaptContent(content string, profile *LLMProfile) string {
	if !profile.isLocal() {
		return content
	}
	lines := strings.Split(content, "\n")
	var out []string
	stepNum := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- ") {
			stepNum++
			line = fmt.Sprintf("%d. %s", stepNum, trimmed[2:])
		} else if trimmed == "" {
			stepNum = 0
		}
		out = append(out, line)
	}
	adapted := strings.Join(out, "\n")
	if !strings.HasPrefix(strings.TrimSpace(adapted), "INSTRUCTIONS:") &&
		!strings.HasPrefix(strings.TrimSpace(adapted), "#") {
		adapted = "INSTRUCTIONS:\n" + adapted
	}
	return adapted
}

// profileLabel returns a short label for context header display.
func profileLabel(profile *LLMProfile) string {
	if profile == nil {
		return "powerful"
	}
	label := profile.Type
	if profile.Provider != "" {
		label += "/" + profile.Provider
	}
	if profile.Model != "" {
		label += ":" + profile.Model
	}
	return label
}
