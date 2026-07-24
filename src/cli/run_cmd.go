// run_cmd.go — `mova run [project] [task]`.
//
// Prints the exact same assembled context `mova chat`/MCP get_full_context
// build (core.BuildContextSections — one assembly, every transport), but
// BEFORE printing anything, the configured "budget": {"max_tokens": N}
// limit (if any) is enforced — see mova.local/budget.EnforceLimit. This
// is the same hard gate `mova chat` and the MCP/HTTP chat_completion tool
// apply (see chat_cmd.go and mova.local/mcp/chat_tool.go): if the context
// exceeds the limit, NOTHING is printed (and, by extension, nothing would
// ever reach a model that consumed this output) — only the error and its
// suggestion.
package main

import (
	"fmt"

	"mova.local/budget"
	"mova.local/core"
	"mova.local/models"
)

// runProject implements `mova run`. Mirrors the status lines `mova chat`
// prints ([Project]/[Context]/[Focus]) so both commands read the same way,
// then applies the Budget gate before ever printing the context.
func runProject(root string, adapter core.Adapter, project, task string) {
	if project == "" {
		die("no project given and none could be auto-detected (see runtime.AutoDetect)")
	}

	consolePrint("[Project] Loading project configuration...\n")
	proj, err := adapter.GetProject(project)
	must(err)

	if proj.LLMProfile != nil && proj.LLMProfile.Config != "" {
		provider := proj.LLMProfile.Provider
		if provider == "" {
			if resolved, rerr := models.ResolveConfigProvider(root, proj.LLMProfile.Config); rerr == nil {
				provider = resolved
			}
		}
		if provider != "" {
			consolePrint(fmt.Sprintf("[Project] Using configured provider: %s (%s)\n", provider, proj.LLMProfile.Config))
		}
	}

	consolePrint("[Context] Building context...\n")
	sections, err := core.BuildContextSections(adapter, root, project, task)
	must(err)
	printContextSummary(sections)

	resolvedTask := task
	if resolvedTask == "" {
		resolvedTask = proj.DefaultTask
	}
	ctxText := sections.Full()

	// ── Budget gate: always runs first, before anything is printed. ────
	if t, ok := proj.Tasks[resolvedTask]; ok {
		if gateErr := budget.EnforceLimit(proj, &t, tokensOf(ctxText, proj)); gateErr != nil {
			consolePrint("\n" + gateErr.Error() + "\n")
			return
		}
	}

	consolePrint(ctxText)
}
