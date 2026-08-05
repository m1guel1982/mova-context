// run_cmd.go — `mova run [project] [task]` and `mova run --count
// [project] [task] [--focus]`.
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
//
// --count switches to a read-only estimate instead — how many tokens
// this run WOULD send, with no context assembled and no model ever
// called — via orchestrator.Count, the same group-aware counting used
// by chat's "/budget" command and the MCP "estimate_budget" tool (see
// mcp/budget_tool.go, reachable identically over stdio and HTTP's /mcp
// route): one implementation, every door. [project] may be an ordinary
// project OR a multiagent group — orchestrator.Count sums one estimate
// per agent when it's a group, exactly like `mova agents run` would
// execute one agent at a time.
package main

import (
	"fmt"

	"mova.local/budget"
	"mova.local/core"
	"mova.local/models"
	"mova.local/orchestrator"
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
	// Budget gate: always runs first, before anything is printed — even
	// for "mova run <project>" with no task at all, which falls back to
	// the project-level "budget" (see budget.BuildGatedContext, shared
	// with the multiagent orchestrator's per-agent run).
	gated := budget.BuildGatedContext(adapter, root, project, task)
	if gated.Sections != nil {
		printContextSummary(gated.Sections)
	}
	if gated.Err != nil {
		consolePrint("\n" + gated.Err.Error() + "\n")
		return
	}

	consolePrint(gated.Text)
}

// runProjectCount implements `mova run --count <project> [task]
// [--focus]` — see the package doc comment above for why this delegates
// to orchestrator.Count instead of duplicating group-vs-project logic
// here. Never calls a model; never writes a report file (unlike `mova
// budget`, which is the same estimate PLUS a saved mova-budget-report.md
// — use that instead when you want the file).
func runProjectCount(root string, adapter core.Adapter, name, task string, withFocus bool) {
	if name == "" {
		die("no project given and none could be auto-detected (see runtime.AutoDetect)")
	}

	result, err := orchestrator.Count(adapter, root, name, task, withFocus)
	must(err)

	if !result.IsGroup {
		printCountReport(name, result.Report)
		return
	}

	consolePrint(fmt.Sprintf("Group: %s (%d agents)\n\n", name, len(result.Agents)))
	for _, ac := range result.Agents {
		if ac.Err != nil {
			consolePrint(fmt.Sprintf("  %-40s error: %s\n", ac.Project, ac.Err.Error()))
			continue
		}
		consolePrint(fmt.Sprintf("  %-40s %8d tokens  (task: %s)\n", ac.Project, ac.Report.TotalTokens, ac.Report.TaskName))
	}
	consolePrint(fmt.Sprintf("\nTOTAL — one run of every agent: %d tokens (approx.)\n", result.TotalTokens))
	for _, c := range result.TotalCosts {
		consolePrint(fmt.Sprintf("  %s/%s: $%.4f\n", c.Provider, c.Model, c.USD))
	}
	consolePrint("\nLocal estimate (tiktoken-go) — no model was called. Actual usage depends on which tasks run and how often (e.g. each agent's own scheduled jobs firing repeatedly adds up separately — see `mova jobs list " + name + "`).\n")
}

func printCountReport(project string, report *budget.Report) {
	consolePrint(fmt.Sprintf("%s: %d tokens (%s), task: %s\n", project, report.TotalTokens, report.Encoding, report.TaskName))
	for _, c := range report.TotalCosts {
		consolePrint(fmt.Sprintf("  %s/%s: $%.4f %s\n", c.Provider, c.Model, c.USD, report.Currency))
	}
	if report.Focus != nil {
		consolePrint(fmt.Sprintf("Focus savings: %.1f%% fewer tokens (%d → %d)\n",
			report.Focus.SavingsPercent, report.Focus.TokensWithoutFocus, report.Focus.TokensWithFocus))
	}
	consolePrint("\nLocal estimate (tiktoken-go) — no model was called. Run `mova budget " + project + "` for the same estimate saved to a report file.\n")
}
