// workflow_cmd.go — natural-language "lee workflow.md" / "ejecuta
// workflow.md" / "workflow.md <project> [task]" support for `mova chat`
// (see chat_cmd.go). Every phrasing below runs through the exact same
// Budget-gated pipeline as MCP's "get_workflow" tool and HTTP's
// /workflow endpoint — see mova.local/budget.LoadWorkflow. workflow.md
// is never opened directly: the project is always resolved and its
// Budget validated first, and only on success does its content become
// part of the active chat context (same mechanism `mova chat <project>`
// already uses to inject a project's context via sess.SetSystem).
package main

import (
	"regexp"
	"strings"

	"mova.local/budget"
	"mova.local/core"
	"mova.local/models"
)

// workflowVerbPattern matches the plain-verb phrasings that apply to the
// project/task the chat was already started with: "lee workflow.md",
// "leer workflow.md", "ejecuta workflow.md", "ejecutar workflow.md",
// "run workflow.md", "execute workflow.md".
var workflowVerbPattern = regexp.MustCompile(`(?i)^(lee|leer|ejecuta|ejecutar|run|execute)\s+workflow\.md\s*$`)

// workflowArgsPattern matches "workflow.md <project>" and
// "workflow.md <project> <task>".
var workflowArgsPattern = regexp.MustCompile(`(?i)^workflow\.md(?:\s+(\S+))?(?:\s+(\S+))?\s*$`)

// handleWorkflowCommand recognizes every workflow.md phrasing the spec
// lists. Returns true (and has already printed a result) when the line
// matched, so chat_cmd.go's default branch knows not to also treat the
// line as an ordinary chat turn or a natural-language save/edit.
func handleWorkflowCommand(sessionProject, sessionTask string, sess *models.Session, root, line string) bool {
	line = strings.TrimSpace(line)

	project, task := sessionProject, sessionTask
	switch {
	case workflowVerbPattern.MatchString(line):
		// use the project/task this chat already started with
	case workflowArgsPattern.MatchString(line):
		m := workflowArgsPattern.FindStringSubmatch(line)
		if m[1] != "" {
			project = m[1]
		}
		if m[2] != "" {
			task = m[2]
		}
	default:
		return false
	}

	if project == "" {
		consolePrint("workflow.md requires a project — try \"workflow.md <project>\" or start the chat with one (mova chat <project>).\n")
		return true
	}

	fa := core.NewFileAdapter(root)
	proj, err := fa.GetProject(project)
	if err != nil {
		consolePrint("Error: " + err.Error() + "\n")
		return true
	}
	adapter := newAdapter(root, proj)

	result, err := budget.LoadWorkflow(adapter, root, project, task, "", "")
	if result != nil {
		for _, l := range result.Log {
			consolePrint(l + "\n")
		}
	}
	if err != nil {
		consolePrint("\n" + err.Error() + "\n")
		return true
	}

	consolePrint("\n" + renderMarkdown(result.Content) + "\n")
	sess.SetSystem(strings.TrimSpace(sess.System + "\n\n" + result.Content))
	return true
}
