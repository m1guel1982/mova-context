// agents_cmd.go — `mova agents list|run` (spec section 7-9: the parent
// config.json orchestrator). Both call straight into
// mova.local/orchestrator — LoadGroupConfig/RunAgent/RunGroup — the same
// functions HTTP's /agents/run route and MCP's "run_agent"/"list_agents"
// tools call (see mcp/agent_group_tool.go, http/server.go). No separate
// orchestration logic here: this file only parses os.Args and prints
// orchestrator.AgentResult. Counting a group's tokens instead of running
// it lives at `mova run --count <group>` now (see run_cmd.go /
// orchestrator/count.go) — same group-vs-project detection, so both
// commands treat a group name the same way.
package main

import (
	"fmt"

	"mova.local/core"
	"mova.local/orchestrator"
)

// runAgentsList implements `mova agents list <group>`.
func runAgentsList(root, group string) {
	cfg, err := orchestrator.LoadGroupConfig(root, group)
	must(err)
	consolePrint(fmt.Sprintf("Group: %s\n", cfg.Group))
	if cfg.Description != "" {
		consolePrint(cfg.Description + "\n")
	}
	consolePrint("Agents:\n")
	for _, a := range cfg.Agents {
		consolePrint("  - " + group + "/" + a + "\n")
	}
}

// runAgentsRun implements `mova agents run <group> [agent|--all]`.
func runAgentsRun(root, group, agentArg string) {
	fa := core.NewFileAdapter(root)
	adapter := core.Adapter(fa)

	var only []string
	if agentArg != "" && agentArg != "--all" {
		only = []string{agentArg}
	}

	results, err := orchestrator.RunGroup(adapter, root, group, only, "")
	must(err)
	printAgentResults(results)
}

func printAgentResults(results []orchestrator.AgentResult) {
	for _, r := range results {
		consolePrint(fmt.Sprintf("\n=== %s ===\n", r.Project))
		if r.Err != nil {
			consolePrint(r.Err.Error() + "\n")
			continue
		}
		consolePrint(r.Text)
	}
}
