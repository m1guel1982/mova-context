// engine_helpers.go — funciones de apoyo de BuildContextSections: carga de
// core files, inyección de variables, resolución de focus y de perfil LLM,
// y adaptación de contenido para modelos locales. Separado de engine.go
// únicamente para respetar el límite de 300 líneas por archivo — misma
// unidad lógica, mismo paquete.
package core

import (
	"fmt"
	"path/filepath"
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

// resolveKnowledgeOrLiteral extends "use"/"custom" (agents, skills) and
// a task's "prompt" so each entry can be EITHER a file name (existing,
// unchanged behavior — tried first, so nothing that already works
// today changes) OR inline literal text typed straight into
// project.json (e.g. "use": ["revisa el estilo de commits del repo"]).
// A value only counts as "found as a file" when GetKnowledge returns
// non-empty content; any other case (not found, adapter error, empty
// file) falls back to treating value itself as the content, so a
// person is never required to create a one-line .md file just to add
// a short instruction. See docs/i18n/{es,en}/PROJECT.md § texto libre.
func resolveKnowledgeOrLiteral(adapter Adapter, kind, domain, lang, value string) (content string, fromFile bool) {
	if strings.TrimSpace(value) == "" {
		return "", false
	}
	if c, err := adapter.GetKnowledge(kind, domain, lang, value); err == nil && c != "" {
		return c, true
	}
	return value, false
}

// resolveDebugPath turns a repo-relative (or already-absolute) path
// into the absolute path debug output should show, without asserting
// the path actually exists — debug is a trace of what was RESOLVED,
// not a validity check.
// writePolicyDebugLines appends one "[debug] policy include/exclude: ..."
// line per resolved config/policy/*.json file — same plain, non-i18n
// style as every other [debug] line in this file (see the package
// comment's rationale), so `mova chat`'s debug output shows the exact
// same policy paths `mova context-trace --debug` already does (see
// trace/console.go's renderPolicyDebug, the OTHER renderer of this
// same core.ResolvedPolicyDebug data). CLI flags aren't available
// here (BuildContext has no --policies_include/--policies_exclude of
// its own), so only project.json and config/policy.json are resolved
// — exactly PolicyRequest's precedence with nil CLI overrides.
func writePolicyDebugLines(dbg *strings.Builder, root, projectName string, proj *Project) {
	req := ResolvePolicyRequest(proj, OrchestratorPolicySelector(root), nil, nil)
	if !req.Declared {
		return
	}
	for _, e := range ResolvedPolicyDebug(root, req) {
		path := e.Path
		if path == "" {
			path = "(not found)"
		}
		if e.Included {
			fmt.Fprintf(dbg, "[debug] policy include: %s -> %s\n", e.Name, path)
		} else {
			reason := "excluded by config"
			if e.Reason == "excluded_not_found" {
				reason = "excluded by config, not found"
			} else if e.Reason == "not_found" {
				reason = "not found"
			}
			fmt.Fprintf(dbg, "[debug] policy exclude: %s -> %s (%s)\n", e.Name, path, reason)
		}
	}
	_ = projectName // kept in the signature for symmetry with other debug writers and possible future per-project logging
}

func resolveDebugPath(root, p string) string {
	if p == "" {
		return root
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(root, p)
}

// debugKnowledgeLoc renders where an agent/skill/prompt entry actually
// came from for --debug output: its resolved file path when found on
// disk, or "inline (free text)" when resolveKnowledgeOrLiteral fell
// back to literal content (see engine.go's agents/skills/prompt
// loops). Shows the PRIMARY candidate path GetKnowledge tries first
// (domain/i18n/lang/name.md) - an approximation when one of its later
// fallback candidates (see file_adapter.go's GetKnowledge doc comment,
// steps 2-10) is what actually matched, since GetKnowledge itself
// doesn't report which candidate won.
func debugKnowledgeLoc(fromFile bool, kind, domain, lang, name string) string {
	if !fromFile {
		return "inline (free text)"
	}
	if domain == "" {
		domain = "base"
	}
	if lang == "" {
		lang = "es"
	}
	return fmt.Sprintf("%ss/%s/i18n/%s/%s.md", kind, domain, lang, name)
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

// resolveTaskExclude espeja resolveTaskFocus exactamente — Task.Exclude
// (si trae al menos un elemento) sobreescribe Project.Exclude, nunca se
// combinan.
func resolveTaskExclude(proj *Project, task *Task) []string {
	if len(task.Exclude) > 0 {
		return task.Exclude
	}
	return proj.Exclude
}

// ResolveExclude expone resolveTaskExclude fuera del paquete — mismo
// motivo que ResolveFocus.
func ResolveExclude(proj *Project, task *Task) []string {
	return resolveTaskExclude(proj, task)
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
