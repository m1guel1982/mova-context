package jobs

import (
	"fmt"
	"time"

	"mova.local/core"
	"mova.local/logging"
)

// RunJob executes ONE job spec for one project, right now — the single
// flow CLI (`mova jobs run`), chat, HTTP (`POST /jobs/run`), and MCP
// ("run_job") all call. Order is fixed and deliberate: tasks first (so
// "save" below has something to write), then save, memory, memory_archive,
// delete, and finally budget — a job author lists all six independently,
// but the engine itself always applies them in this order regardless of
// how they're written in project.json.
func RunJob(adapter core.Adapter, root, project string, proj *core.Project, spec core.JobSpec) *Result {
	logging.L().Info("jobs", "starting job for project=%s schedule=%q", project, spec.Schedule)

	res := &Result{Project: project}
	jc := &jobContext{
		Adapter: adapter, Root: root, Project: project, Proj: proj,
		Spec: spec, Now: time.Now(),
	}

	runTasks(jc, res)
	runSave(jc, res)
	runMemory(jc, res)
	runMemoryArchive(jc, res)
	runDelete(jc, res)
	runBudget(jc, res)

	if len(res.Steps) == 0 && len(res.Errors) == 0 {
		res.log("· job has no actions configured (tasks/save/memory/memory_archive/delete/budget) — nothing to do")
	}

	if res.OK() {
		logging.L().Info("jobs", "finished job for project=%s (%d step(s))", project, len(res.Steps))
	} else {
		logging.L().Warning("jobs", "finished job for project=%s with %d error(s)", project, len(res.Errors))
	}
	return res
}

// RunProjectJobs runs every job declared in proj.Jobs for project,
// unconditionally (ignores "schedule" — used by `mova jobs run <project>
// --all` and by tests). See RunDueJobs for the schedule-aware entry
// point the daemon (`mova jobs start`) uses.
func RunProjectJobs(adapter core.Adapter, root, project string, proj *core.Project) []*Result {
	results := make([]*Result, 0, len(proj.Jobs))
	for _, spec := range proj.Jobs {
		results = append(results, RunJob(adapter, root, project, proj, spec))
	}
	return results
}

// RunJobByIndex runs a single job from proj.Jobs by its position (0-based,
// the order it appears in project.json) — used by `mova jobs run
// <project> <index>` to run one job on demand without waiting for its
// schedule.
func RunJobByIndex(adapter core.Adapter, root, project string, proj *core.Project, index int) (*Result, error) {
	if index < 0 || index >= len(proj.Jobs) {
		return nil, fmt.Errorf("job index %d out of range: project %q has %d job(s)", index, project, len(proj.Jobs))
	}
	return RunJob(adapter, root, project, proj, proj.Jobs[index]), nil
}
