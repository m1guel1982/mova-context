// agent_group_tool.go — exposes mova.local/orchestrator as MCP tools
// "list_agents" and "run_agent", reachable identically from stdio and
// HTTP (same Process(), see server.go) — the exact same
// orchestrator.LoadGroupConfig/RunGroup the CLI's `mova agents`
// (cli/agents_cmd.go) calls. One implementation, every door.
package mcp

import (
	"fmt"

	"mova.local/core"
	"mova.local/orchestrator"
)

func listAgentsTool(root string, args map[string]any) (string, error) {
	group := str(args, "group")
	cfg, err := orchestrator.LoadGroupConfig(root, group)
	if err != nil {
		return "", err
	}
	out := fmt.Sprintf("Group: %s\n", cfg.Group)
	if cfg.Description != "" {
		out += cfg.Description + "\n"
	}
	for _, a := range cfg.Agents {
		out += "  - " + group + "/" + a + "\n"
	}
	return out, nil
}

func runAgentTool(adapter core.Adapter, root string, args map[string]any) (string, error) {
	group := str(args, "group")
	agent := str(args, "agent")
	task := str(args, "task")

	var only []string
	if agent != "" {
		only = []string{agent}
	}
	results, err := orchestrator.RunGroup(adapter, root, group, only, task)
	if err != nil {
		return "", err
	}

	out := ""
	for _, r := range results {
		out += fmt.Sprintf("=== %s ===\n", r.Project)
		if r.Err != nil {
			out += r.Err.Error() + "\n\n"
			continue
		}
		out += r.Text + "\n\n"
	}
	return out, nil
}
