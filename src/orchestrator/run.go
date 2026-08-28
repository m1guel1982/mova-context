// run.go — executes agents belonging to a group, sequentially (spec
// section 7: "ejecución secuencial ... diseño preparado para ejecución
// paralela futura"). Each agent runs through budget.BuildGatedContext —
// the SAME assemble+gate function `mova run` uses for a normal project
// (see cli/run_cmd.go) — so an agent is never treated as a special case:
// it is just a project addressed as "<group>/<agent>".
package orchestrator

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"

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
// declared/discovered in the group's config.json — see LoadGroupConfig)
// through a bounded worker pool, and returns results in the SAME order
// agents were requested regardless of which goroutine finishes first.
//
// Why a worker pool instead of one goroutine per agent: a group can
// list many agents, and this same process may be serving other
// concurrent callers at once (CLI, Chat, MCP, HTTP API — see
// http/server.go's own concurrency limiter). Bounding how many agents
// of ONE group run at the same time keeps total goroutine/file-handle
// usage predictable on a shared server instead of growing unbounded
// with group size. RunAgent itself is untouched — it is already safe to
// call concurrently, since every piece of state it can touch
// (mova-token-history.json, mova-spend.json, mova-context-cache.json)
// is now guarded by budget.withFileLock (see budget/filelock.go).
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

	results := make([]AgentResult, len(agents))
	sem := make(chan struct{}, groupWorkerLimit())
	var wg sync.WaitGroup
	for i, agent := range agents {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, agent string) {
			defer wg.Done()
			defer func() { <-sem }()
			results[i] = RunAgent(adapter, root, group, agent, task)
		}(i, agent)
	}
	wg.Wait()
	return results, nil
}

// groupWorkerLimit caps how many agents of one RunGroup call execute at
// once. Overridable with MOVA_MAX_CONCURRENCY for tuning on a given
// host (e.g. a small shared Oracle Cloud instance vs. a beefy
// workstation); defaults to runtime.NumCPU(), capped at 8 so a group
// with dozens of agents never opens dozens of simultaneous model
// requests by accident.
func groupWorkerLimit() int {
	if v := os.Getenv("MOVA_MAX_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	n := runtime.NumCPU()
	if n < 1 {
		n = 1
	}
	if n > 8 {
		n = 8
	}
	return n
}
