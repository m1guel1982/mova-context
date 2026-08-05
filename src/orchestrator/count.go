// count.go — the ONE implementation behind `mova run --count`, the
// chat "/budget" command, and the MCP "estimate_budget" tool (used
// as-is by both stdio and HTTP — see mcp/budget_tool.go, http/server.go
// mounting /mcp). All four doors call orchestrator.Count and format the
// same CountResult; none of them re-implements group iteration.
//
// Count is the group-aware generalization of budget.BuildReport: budget
// itself cannot know about multiagent groups (mova.local/orchestrator
// already imports mova.local/budget, so the reverse import would be a
// cycle), so this is exactly where that generalization belongs — same
// reasoning as RunAgent/RunGroup in run.go, which already do the same
// thing for actually executing an agent instead of just counting its
// tokens. A group name has no project.json of its own (see
// LoadGroupConfig/ConfigPath); IsGroup is how every door tells
// "ordinary project" and "multiagent group" apart before deciding how
// to count it.
package orchestrator

import (
	"fmt"
	"os"

	"mova.local/budget"
	"mova.local/core"
)

// IsGroup reports whether name is a multiagent group — a directory
// under projects/ with its own config.json (see ConfigPath) — rather
// than an ordinary project addressed by a project.json. Every door that
// takes a project name and needs to treat "it might be a group"
// uniformly should check this first (jobs list/run and Count both do).
func IsGroup(root, name string) bool {
	_, err := os.Stat(ConfigPath(root, name))
	return err == nil
}

// AgentCount is one agent's token estimate within a group Count — the
// counting equivalent of AgentResult (run.go).
type AgentCount struct {
	Agent   string
	Project string         // fully-qualified: "<group>/<agent>"
	Report  *budget.Report // nil if Err != nil
	Err     error
}

// CountResult is what Count returns, whether name was a single project
// or a multiagent group — every door (CLI/chat/MCP/HTTP) formats this
// one shape.
type CountResult struct {
	Name    string
	IsGroup bool

	// Set when !IsGroup: the single project's own report, exactly what
	// budget.BuildReport/`mova budget` already return.
	Report *budget.Report

	// Set when IsGroup: one entry per agent, in config.json order.
	Agents []AgentCount

	// Always set: TotalTokens/TotalCosts equal Report's own totals when
	// !IsGroup, or the sum across every agent that could be estimated
	// when IsGroup — so callers never need an if/else to read a total.
	TotalTokens int
	TotalCosts  []budget.ModelCost
}

// Count estimates, 100% locally (tiktoken-go, no model ever called),
// how many tokens ONE run of name would send — name may be an ordinary
// project (delegates straight to budget.BuildReport, unchanged) or a
// multiagent group (sums one budget.BuildReport per agent, each using
// its own default_task when task is ""). This is the shared
// implementation behind:
//   - CLI:  mova run --count <project>   (see cli/run_cmd.go)
//   - chat: /budget                      (see cli/chat_save.go)
//   - MCP:  the "estimate_budget" tool   (see mcp/budget_tool.go),
//     reachable identically over stdio and HTTP's /mcp route.
func Count(adapter core.Adapter, root, name, task string, withFocus bool) (CountResult, error) {
	if !IsGroup(root, name) {
		report, err := budget.BuildReport(adapter, root, name, task, withFocus)
		if err != nil {
			return CountResult{}, err
		}
		return CountResult{
			Name:        name,
			Report:      report,
			TotalTokens: report.TotalTokens,
			TotalCosts:  report.TotalCosts,
		}, nil
	}

	cfg, err := LoadGroupConfig(root, name)
	if err != nil {
		return CountResult{}, err
	}
	if len(cfg.Agents) == 0 {
		return CountResult{}, fmt.Errorf("no agents found for group %q (checked config.json \"agents\" and subdirectories under projects/%s/)", name, name)
	}

	result := CountResult{Name: name, IsGroup: true}
	var costList []*budget.ModelCost
	costIndex := map[string]*budget.ModelCost{}

	for _, agent := range cfg.Agents {
		project := name + "/" + agent
		report, err := budget.BuildReport(adapter, root, project, task, withFocus)
		if err != nil {
			result.Agents = append(result.Agents, AgentCount{Agent: agent, Project: project, Err: err})
			continue
		}
		result.Agents = append(result.Agents, AgentCount{Agent: agent, Project: project, Report: report})
		result.TotalTokens += report.TotalTokens
		for _, c := range report.TotalCosts {
			key := c.Provider + "/" + c.Model
			ct, ok := costIndex[key]
			if !ok {
				ct = &budget.ModelCost{Provider: c.Provider, Model: c.Model}
				costIndex[key] = ct
				costList = append(costList, ct)
			}
			ct.USD += c.USD
			ct.CLP += c.CLP
		}
	}

	for _, ct := range costList {
		result.TotalCosts = append(result.TotalCosts, *ct)
	}
	return result, nil
}
