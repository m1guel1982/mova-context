// context_tool.go — MCP/HTTP tool "get_full_context" (= `mova run`).
//
// Before returning anything, the configured "budget": {"max_tokens": N}
// limit (if any) is enforced — see mova.local/budget.EnforceLimit. This
// is the same hard gate cli/run_cmd.go, cli/chat_cmd.go, and
// chat_tool.go's chat_completion apply: every door that can generate a
// project's context checks Budget FIRST, before a single byte of that
// context is handed to whoever is asking for it (an MCP client sending
// this straight to a model, a script, curl, anything).
package mcp

import (
	"fmt"

	"mova.local/budget"
	"mova.local/core"
)

// fullContextTool builds the same context `mova run` and chat_completion
// build (core.BuildContextSections — one assembly, every transport), then
// applies the Budget gate before returning it.
func fullContextTool(adapter core.Adapter, root, project, task string) (string, error) {
	proj, err := adapter.GetProject(project)
	if err != nil {
		return "", err
	}
	sections, err := core.BuildContextSections(adapter, root, project, task)
	if err != nil {
		return "", err
	}

	resolvedTask := task
	if resolvedTask == "" {
		resolvedTask = proj.DefaultTask
	}
	ctxText := sections.Full()

	if t, ok := proj.Tasks[resolvedTask]; ok {
		modelHint := ""
		if proj.LLMProfile != nil {
			modelHint = proj.LLMProfile.Config
		}
		tokens, _, cerr := budget.CountTokens(ctxText, modelHint)
		if cerr != nil {
			return "", fmt.Errorf("could not count tokens for Budget check: %w", cerr)
		}
		if gateErr := budget.EnforceLimit(proj, &t, tokens); gateErr != nil {
			return "", gateErr
		}
	}

	return ctxText, nil
}
