// limit.go implements `max_tokens` as a real, enforced rule — not just
// informational. If project.json (or a task) declares a
// "budget": {"max_tokens": N}, this file is what actually stops execution
// BEFORE any context is sent to a model, from every entry point (CLI
// chat, MCP chat_completion, HTTP) — see EnforceLimit's call sites in
// cli/chat_cmd.go and mcp/chat_tool.go. `mova budget` itself only reports
// OverBudget (see estimate.go) since it never sends anything anywhere.
package budget

import (
	"fmt"

	"mova.local/core"
)

// ResolveTask looks up taskName in proj.Tasks and returns a pointer to
// it — or a pointer to a zero-value Task (never nil) when taskName is
// empty or doesn't match any task. This exists so every caller of
// EnforceLimit/CheckLimit ALWAYS has a *Task to pass, even for "read the
// whole project, no specific task" (e.g. "mova run my-project" or "lee
// my-project" with no task named): a zero-value Task has no Task.Budget
// of its own, so core.ResolveBudget correctly falls back to the
// project-level "budget" — the Budget gate must run in that case too,
// not be skipped because no task happened to match.
func ResolveTask(proj *core.Project, taskName string) *core.Task {
	if proj != nil && taskName != "" {
		if t, ok := proj.Tasks[taskName]; ok {
			return &t
		}
	}
	return &core.Task{}
}

// CheckLimit resolves the effective BudgetConfig (task wins over project,
// see core.ResolveBudget) and returns (maxTokens, overBudget). maxTokens=0
// means "no limit configured" — overBudget is always false in that case,
// never a false positive from missing configuration.
func CheckLimit(proj *core.Project, task *core.Task, totalTokens int) (maxTokens int, overBudget bool) {
	cfg := core.ResolveBudget(proj, task)
	if cfg == nil || cfg.MaxTokens <= 0 {
		return 0, false
	}
	return cfg.MaxTokens, totalTokens > cfg.MaxTokens
}

// EnforceLimit is the hard gate: if a budget is configured and tokens
// exceeds it, execution must stop before sending anything to a model.
// Returns nil when there's no limit configured or the context fits.
func EnforceLimit(proj *core.Project, task *core.Task, tokens int) error {
	maxTokens, over := CheckLimit(proj, task, tokens)
	if !over {
		return nil
	}
	return fmt.Errorf(
		"ERROR\n\nCurrent context (%s tokens) exceeds the configured limit (%s).\n\nSuggestion:\nUse --focus to reduce the included files.",
		formatThousands(tokens), formatThousands(maxTokens),
	)
}

// formatThousands renders an int with thousands separators (14250 →
// "14,250") — no new dependency, just a small loop.
func formatThousands(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	return string(out)
}
