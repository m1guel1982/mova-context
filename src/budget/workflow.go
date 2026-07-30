// workflow.go — the SINGLE pipeline behind "lee workflow.md" / "ejecuta
// workflow.md" / "workflow.md <project> <task>", reachable identically
// from Chat (see cli/workflow_cmd.go), MCP (mcp/server.go's
// "get_workflow" tool), and HTTP (http/server.go's `/workflow` route —
// which is itself a thin wrapper over the MCP tool, same as every other
// HTTP endpoint in Mova Context).
//
// workflow.md is never opened directly. Every door goes through
// LoadWorkflow, which enforces the mandatory order the spec describes:
//
//  1. resolve the project (project.json)
//  2. resolve the task, if one was named
//  3. resolve which workflow.md file applies (core.ResolveWorkflowPath —
//     project.json's "workflow_path", or a repo's own "workflow" entry,
//     see core/types.go)
//  4. build the project's context (core.BuildContextSections — agents +
//     skills + prompt + focus + memory)
//  5./6. Dedup and Focus are already applied INSIDE BuildContextSections
//     (core/engine.go) — there is no separate step to run, only to
//     report what it did
//  7. estimate the token cost of that context PLUS workflow.md itself
//  8. validate it against the project/task's configured "budget"
//     (EnforceLimit — the exact same hard gate `mova chat`/
//     chat_completion already apply before ever calling a model)
//
// Only if step 8 passes does LoadWorkflow read workflow.md's content.
// This is deliberately the SAME BuildReport/CheckLimit/EnforceLimit code
// `mova budget` and chat_completion already use — no parallel budget
// implementation for workflow.md, no matter which door asked for it.
package budget

import (
	"fmt"
	"os"

	"mova.local/core"
)

// WorkflowResult is what every door (CLI, MCP, HTTP) receives back from
// LoadWorkflow. Log holds the exact step-by-step progress lines the spec
// requires ("[Project] Loading project configuration...", "[Context]
// Building context...", ...) so Chat/MCP/HTTP render IDENTICAL progress
// text; only how each door displays Log differs (console print vs. an
// MCP/HTTP text result) — never the wording itself.
type WorkflowResult struct {
	Log       []string // step-by-step progress lines, always present
	Path      string    // resolved workflow.md path
	Content   string    // workflow.md content — only set when the Budget check passed
	Tokens    int       // estimated tokens for context+workflow.md together
	MaxTokens int       // 0 = no limit configured
	Project   string
	Task      string
}

// LoadWorkflow runs the full pipeline described in the package doc
// comment above. explicitWorkflow overrides project.json's configured
// workflow_path when the caller named a specific file explicitly;
// leave "" to use project.json's own resolution (the normal case).
// modelHint picks the tokenizer encoding the same way estimate_budget
// already does (empty = universal cl100k_base approximation).
//
// On success, res.Content holds workflow.md's text and the caller may
// proceed to use it. On a Budget failure, res.Content is empty, res.Log
// holds every step that DID complete, and the returned error is exactly
// EnforceLimit's "ERROR\n\nCurrent context (...) exceeds the configured
// limit (...).\n\nSuggestion:\nUse --focus..." text — the same block
// `mova chat`/chat_completion already show, so a person sees one
// consistent message regardless of which door asked.
func LoadWorkflow(adapter core.Adapter, root, projectName, taskName, explicitWorkflow, modelHint string) (*WorkflowResult, error) {
	if projectName == "" {
		return nil, fmt.Errorf(
			"workflow.md requires a project — use \"workflow.md <project>\" or \"workflow.md <project> <task>\" so its Budget can be validated first")
	}

	res := &WorkflowResult{Project: projectName, Task: taskName}

	res.Log = append(res.Log, "[Project] Loading project configuration...")
	proj, err := adapter.GetProject(projectName)
	if err != nil {
		return nil, err
	}
	res.Log = append(res.Log, "[Project] Using configured provider...")

	resolvedTaskName := core.ResolveTaskName(proj, taskName)
	task := ResolveTask(proj, resolvedTaskName)

	res.Path = core.ResolveWorkflowPath(root, proj, explicitWorkflow)

	res.Log = append(res.Log, "[Context] Building context...")
	sections, err := core.BuildContextSections(adapter, root, projectName, taskName)
	if err != nil {
		return nil, err
	}

	if sections.DuplicatesRemoved > 0 {
		res.Log = append(res.Log, fmt.Sprintf("[Dedup] Removed %d duplicated paragraph(s) (%d chars).",
			sections.DuplicatesRemoved, sections.DuplicatesRemovedChars))
	} else {
		res.Log = append(res.Log, "[Dedup] No duplicates found.")
	}

	if sections.Focus != "" {
		res.Log = append(res.Log, "[Focus] Selected the configured focus for this project/task.")
	} else {
		res.Log = append(res.Log, "[Focus] No focus configured — using the full agents/skills/prompt context.")
	}

	workflowBytes, err := os.ReadFile(res.Path)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w — configure \"workflow_path\" in project.json or place a workflow.md there", res.Path, err)
	}
	workflowContent := string(workflowBytes)

	tokens, _, err := CountTokens(sections.Full()+"\n"+workflowContent, modelHint)
	if err != nil {
		return nil, err
	}
	res.Tokens = tokens
	res.MaxTokens, _ = CheckLimit(proj, task, tokens)

	if err := EnforceLimit(proj, task, tokens); err != nil {
		return res, err
	}

	res.Log = append(res.Log, fmt.Sprintf("[Workflow] Loaded %s (%s tokens).", res.Path, formatThousands(tokens)))
	res.Content = workflowContent
	return res, nil
}

// RenderLog joins a WorkflowResult's progress lines the same way every
// door needs to display them — one line per step, in order.
func (r *WorkflowResult) RenderLog() string {
	out := ""
	for _, line := range r.Log {
		out += line + "\n"
	}
	return out
}
