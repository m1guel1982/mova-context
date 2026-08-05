// jobs_cmd.go — `mova jobs list|run|start` (section 2 of the spec:
// running Jobs from CLI/Chat/HTTP/MCP through one shared flow). Every
// case here calls straight into mova.local/jobs — RunJob/RunProjectJobs/
// RunJobByIndex/RunScheduler — the exact same functions the MCP
// "run_job"/"list_jobs" tools and HTTP's /jobs/run route call (see
// mcp/job_tool.go, http/server.go). No separate execution logic lives
// in this file: it only parses os.Args and prints jobs.Result.
package main

import (
	"fmt"
	"os"
	"time"

	"mova.local/core"
	"mova.local/jobs"
	"mova.local/orchestrator"
)

// tryLoadGroup checks whether `name` is a multiagent group — a directory
// under projects/ with its own config.json (see orchestrator.ConfigPath)
// rather than an ordinary project.json — and loads it if so. Every job
// command below tries this FIRST, because a group name has no
// project.json of its own: its jobs live one level down, in each agent's
// own project.json (projects/<group>/<agent>/project.json). Without this
// check, `mova jobs list/run <group>` failed with "project not found"
// even though the group's agents had jobs configured.
func tryLoadGroup(root, name string) (*orchestrator.GroupConfig, bool) {
	if _, err := os.Stat(orchestrator.ConfigPath(root, name)); err != nil {
		return nil, false
	}
	cfg, err := orchestrator.LoadGroupConfig(root, name)
	if err != nil {
		return nil, false
	}
	return cfg, true
}

// runJobsList implements `mova jobs list <project>`. When `project` is
// actually a multiagent group, it lists every agent's jobs in turn
// instead of failing (see tryLoadGroup).
func runJobsList(root, project string) {
	fa := core.NewFileAdapter(root)

	if cfg, ok := tryLoadGroup(root, project); ok {
		any := false
		for _, agentName := range cfg.Agents {
			full := project + "/" + agentName
			proj, err := fa.GetProject(full)
			if err != nil || len(proj.Jobs) == 0 {
				continue
			}
			any = true
			consolePrint(full + ":\n")
			printJobsList(proj)
		}
		if !any {
			consolePrint("no jobs configured for any agent in group: " + project + "\n")
		}
		return
	}

	proj, err := fa.GetProject(project)
	must(err)
	if len(proj.Jobs) == 0 {
		consolePrint("no jobs configured for project: " + project + "\n")
		return
	}
	printJobsList(proj)
}

func printJobsList(proj *core.Project) {
	for i, spec := range proj.Jobs {
		label := spec.Comment
		if label == "" {
			label = fmt.Sprintf("tasks=%v save=%q", spec.Tasks, spec.Save)
		}
		consolePrint(fmt.Sprintf("  [%d] schedule=%q  %s\n", i, spec.Schedule, label))
	}
}

// runJobsRun implements `mova jobs run <project> [index]` (all jobs when
// index is omitted) and `mova jobs run <project> --all` (explicit, same
// behavior — accepted for readability in scripts).
//
// When `project` is a multiagent group, `index` is instead read as an
// optional agent name: omitted/--all runs every job for every agent in
// the group; a name runs every job for just that agent (job indices are
// per-agent, so a bare number is ambiguous across agents — to run a
// single job by index inside one agent, address it directly, e.g.
// `mova jobs run <group>/<agent> <index>`, which already works today
// since that full path is itself an ordinary project name).
func runJobsRun(root, project, indexArg string) {
	fa := core.NewFileAdapter(root)

	if cfg, ok := tryLoadGroup(root, project); ok {
		runJobsRunGroup(root, fa, cfg, project, indexArg)
		return
	}

	proj, err := fa.GetProject(project)
	must(err)
	adapter := newAdapter(root, proj)

	var results []*jobs.Result
	if indexArg == "" || indexArg == "--all" {
		results = jobs.RunProjectJobs(adapter, root, project, proj)
	} else {
		idx, convErr := parseJobIndex(indexArg)
		must(convErr)
		res, runErr := jobs.RunJobByIndex(adapter, root, project, proj, idx)
		must(runErr)
		results = []*jobs.Result{res}
	}
	printJobResults(results)
}

func runJobsRunGroup(root string, fa core.Adapter, cfg *orchestrator.GroupConfig, group, agentArg string) {
	var agentNames []string
	if agentArg == "" || agentArg == "--all" {
		agentNames = cfg.Agents
	} else {
		agentNames = []string{agentArg}
	}

	var results []*jobs.Result
	for _, agentName := range agentNames {
		full := group + "/" + agentName
		proj, err := fa.GetProject(full)
		if err != nil {
			consolePrint("skipping " + full + ": " + err.Error() + "\n")
			continue
		}
		adapter := newAdapter(root, proj)
		results = append(results, jobs.RunProjectJobs(adapter, root, full, proj)...)
	}
	if len(results) == 0 {
		consolePrint("no jobs ran for group: " + group + "\n")
		return
	}
	printJobResults(results)
}

// runJobsStart implements `mova jobs start` — the daemon (section 2):
// scans every project once a minute and fires any job whose "schedule"
// matches, via the exact same jobs.RunJob every on-demand door uses (see
// mova.local/jobs.RunScheduler). Blocks until killed, like any long-
// running scheduler process (cron/systemd timer).
func runJobsStart(root string) {
	fa := core.NewFileAdapter(root)
	consolePrint("[Jobs] scheduler started — checking every project once a minute (Ctrl+C to stop)\n")
	jobs.RunScheduler(fa, fa, root, func(res *jobs.Result) {
		printJobResults([]*jobs.Result{res})
	})
}

func printJobResults(results []*jobs.Result) {
	for _, res := range results {
		consolePrint(fmt.Sprintf("[%s] %s\n", res.Project, time.Now().Format("2006-01-02 15:04:05")))
		for _, step := range res.Steps {
			consolePrint("  " + step + "\n")
		}
		if !res.OK() {
			consolePrint(fmt.Sprintf("  (%d error(s))\n", len(res.Errors)))
		}
	}
}

func parseJobIndex(s string) (int, error) {
	var idx int
	if _, err := fmt.Sscanf(s, "%d", &idx); err != nil {
		return 0, fmt.Errorf("invalid job index %q: expected a number or --all", s)
	}
	return idx, nil
}
