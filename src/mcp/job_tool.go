// job_tool.go — exposes the Job Engine (mova.local/jobs) as MCP tools
// "run_job" and "list_jobs", reachable identically from stdio and HTTP
// (same Process(), see server.go) — the exact same jobs.RunJob/
// RunProjectJobs/RunJobByIndex the CLI's `mova jobs run` (cli/jobs_cmd.go)
// calls. One implementation, every door.
package mcp

import (
	"fmt"
	"strconv"

	"mova.local/core"
	"mova.local/jobs"
)

func listJobsTool(root string, args map[string]any) (string, error) {
	project := str(args, "project")
	fa := core.NewFileAdapter(root)
	proj, err := fa.GetProject(project)
	if err != nil {
		return "", err
	}
	if len(proj.Jobs) == 0 {
		return "no jobs configured for project: " + project, nil
	}
	out := ""
	for i, spec := range proj.Jobs {
		label := spec.Comment
		if label == "" {
			label = fmt.Sprintf("tasks=%v save=%q", spec.Tasks, spec.Save)
		}
		out += fmt.Sprintf("[%d] schedule=%q  %s\n", i, spec.Schedule, label)
	}
	return out, nil
}

func runJobTool(adapter core.Adapter, root string, args map[string]any) (string, error) {
	project := str(args, "project")
	proj, err := adapter.GetProject(project)
	if err != nil {
		return "", err
	}

	var results []*jobs.Result
	if idxStr := str(args, "index"); idxStr != "" {
		idx, convErr := strconv.Atoi(idxStr)
		if convErr != nil {
			return "", fmt.Errorf("run_job: \"index\" must be a number, got %q", idxStr)
		}
		res, runErr := jobs.RunJobByIndex(adapter, root, project, proj, idx)
		if runErr != nil {
			return "", runErr
		}
		results = []*jobs.Result{res}
	} else {
		results = jobs.RunProjectJobs(adapter, root, project, proj)
	}
	return renderJobResults(results), nil
}

func renderJobResults(results []*jobs.Result) string {
	out := ""
	for _, res := range results {
		out += fmt.Sprintf("[%s]\n", res.Project)
		for _, step := range res.Steps {
			out += "  " + step + "\n"
		}
		if !res.OK() {
			out += fmt.Sprintf("  (%d error(s))\n", len(res.Errors))
		}
	}
	return out
}
