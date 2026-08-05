// run.go — executes agents belonging to a group, sequentially (spec
// section 7: "ejecución secuencial ... diseño preparado para ejecución
// paralela futura"). Each agent runs through budget.BuildGatedContext —
// the SAME assemble+gate function `mova run` uses for a normal project
// (see cli/run_cmd.go) — so an agent is never treated as a special case:
// it is just a project addressed as "<group>/<agent>".
package orchestrator

import (
	"fmt"

	"mova.local/budget"
	"mova.local/core"
	"mova.local/logging"
)

// AgentResult is what RunAgent/RunGroup return — one per agent, in the
// same order they were requested, so CLI/HTTP/MCP can report every
// agent's outcome uniformly.
type AgentResult struct {
	Agent   string
	Project string // fully-qualified project name: "<group>/<agent>"
	Text    string // assembled, budget-gated context (empty if Err != nil)
	Tokens  int
	Err     error
}

// RunAgent runs a single agent ("<group>/<agent>") for task (may be "",
// meaning the agent's own default_task). This is intentionally the exact
// same operation as `mova run <group>/<agent> <task>` — nothing here is
// orchestrator-specific except the fully-qualified name it builds.
func RunAgent(adapter core.Adapter, root, group, agent, task string) AgentResult {
	project := group + "/" + agent
	logging.L().Info("orchestrator", "running agent %s", project)
	gated := budget.BuildGatedContext(adapter, root, project, task)
	if gated.Err != nil {
		logging.L().Warning("orchestrator", "agent %s failed: %v", project, gated.Err)
	}
	return AgentResult{Agent: agent, Project: project, Text: gated.Text, Tokens: gated.Tokens, Err: gated.Err}
}

// RunGroup runs every agent in only (or, when only is empty, every agent
// declared/discovered in the group's config.json — see LoadGroupConfig),
// SEQUENTIALLY, one after another, in the order given.
//
// Extensibility (parallel execution, spec section 7's "diseño preparado
// para ejecución paralela futura"): this loop is the single place that
// would change — replacing the for-loop body with a goroutine per agent
// and a sync.WaitGroup — without touching RunAgent, GroupConfig, or any
// caller's contract (AgentResult stays the same either way). Not done
// today because unattended parallel runs share nothing that needs
// coordinating yet (no shared mutable state across agents), so the
// simplest correct implementation is sequential until a concrete need
// for parallel throughput appears.
func RunGroup(adapter core.Adapter, root, group string, only []string, task string) ([]AgentResult, error) {
	cfg, err := LoadGroupConfig(root, group)
	if err != nil {
		return nil, err
	}
	agents := cfg.Agents
	if len(only) > 0 {
		agents = only
	}
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents found for group %q (checked config.json \"agents\" and subdirectories under projects/%s/)", group, group)
	}

	results := make([]AgentResult, 0, len(agents))
	for _, agent := range agents {
		results = append(results, RunAgent(adapter, root, group, agent, task))
	}
	return results, nil
}
