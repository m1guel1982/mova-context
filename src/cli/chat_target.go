// chat_target.go — resolución de proyecto/grupo para `mova chat`.
// Movido tal cual desde chat_cmd.go (≤300 líneas por archivo).
package main

import (
	"strings"

	"mova.local/core"
	"mova.local/orchestrator"
)

// resolveChatTarget turns the raw (project, task) positionals `mova
// chat <project> [task]` was given into what core.FileAdapter.GetProject
// actually needs, handling the ONE case it can't resolve on its own: a
// multiagent GROUP name (projects/<group>/config.json — see
// mova.local/orchestrator) has no project.json of its own, so
// `mova chat <group>` and `mova chat <group> <agent>` both fail
// GetProject verbatim. This never changes behavior for an ordinary
// project or for the already-supported `mova chat <group>/<agent>`
// single-argument form (core.FileAdapter.GetProject already resolves
// that slash path) — it only fires when the literal name given isn't a
// project by itself.
//
// Returns ok=false when it already printed enough guidance that the
// caller should just return (ambiguous group with no agent chosen, or
// an agent name that doesn't belong to this group).
func resolveChatTarget(root, project, task string) (resolvedProject, resolvedTask string, ok bool) {
	fa := core.NewFileAdapter(root)
	if _, err := fa.GetProject(project); err == nil {
		return project, task, true // ordinary project — unchanged path
	}
	if !orchestrator.IsGroup(root, project) {
		return project, task, true // not a group either — let GetProject's own error surface as before
	}

	cfg, err := orchestrator.LoadGroupConfig(root, project)
	if err != nil || len(cfg.Agents) == 0 {
		consolePrint("[Project] \"" + project + "\" looks like a multiagent group, but its config.json couldn't be read or lists no agents.\n")
		return "", "", false
	}
	if task != "" {
		for _, agent := range cfg.Agents {
			if agent == task {
				return project + "/" + agent, "", true // "<group> <agent>" → "<group>/<agent>", no task filter applied
			}
		}
		consolePrint("[Project] \"" + task + "\" is not an agent of group \"" + project + "\". Agents: " + strings.Join(cfg.Agents, ", ") + "\n")
		return "", "", false
	}
	consolePrint("[Project] \"" + project + "\" is a multiagent group — pick one agent to chat with:\n")
	for _, agent := range cfg.Agents {
		consolePrint("  mova chat " + project + " " + agent + "\n")
	}
	return "", "", false
}
