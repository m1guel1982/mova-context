// engine_helpers.go — funciones de apoyo de BuildContextSections: carga de
// core files, inyección de variables, resolución de focus y de perfil LLM,
// y adaptación de contenido para modelos locales. Separado de engine.go
// únicamente para respetar el límite de 300 líneas por archivo — misma
// unidad lógica, mismo paquete.
package core

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

func loadCore(adapter Adapter, kind, domain, lang, name string, loaded map[string]bool) string {
	if loaded[name] {
		return ""
	}
	if domain == "" {
		domain = "base"
	}
	if lang == "" {
		lang = "es"
	}
	c, err := adapter.GetKnowledge(kind, domain, lang, name)
	if err != nil || c == "" {
		return ""
	}
	loaded[name] = true
	return c
}

// ExtractMemoryBlock pulls ONLY the ```memory ... ``` block content from an LLM response.
func ExtractMemoryBlock(response string) (string, error) {
	// Usamos comillas dobles para construir la regex sin conflictos de backticks
	re := regexp.MustCompile("(?s)```memory\\s*(.*?)\\s*```")
	matches := re.FindStringSubmatch(response)

	if len(matches) < 2 {
		return "", fmt.Errorf("no ```memory block found")
	}

	return strings.TrimSpace(matches[1]), nil
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

// resolveTaskFocus decides which `focus` list applies to this run: the
// task's own `focus` (if set) always wins and REPLACES the project's
// global focus — it never merges the two lists. If the task has no
// `focus`, the project-level `focus` (if any) is used instead.
func resolveTaskFocus(proj *Project, task *Task) []string {
	if len(task.Focus) > 0 {
		return task.Focus
	}
	return proj.Focus
}

// ResolveFocus expone resolveTaskFocus fuera del paquete — usado por
// mova.local/budget para saber, sin duplicar la regla, qué lista de focus
// aplica a un proyecto+task antes de comparar "con focus" vs "sin focus".
func ResolveFocus(proj *Project, task *Task) []string {
	return resolveTaskFocus(proj, task)
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
	if !profile.IsLocal() {
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
	if profile.Config != "" {
		label += ":" + profile.Config
	}
	return label
}