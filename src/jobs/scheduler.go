// scheduler.go — the daemon side of the Job Engine (`mova jobs start`,
// section 2 of the spec: "Ejecución ... desde CLI/Chat/HTTP/MCP"). Ticks
// once a minute, and for every project/job whose "schedule" matches the
// current minute, calls the exact same RunJob every on-demand door uses
// (engine.go) — the daemon is a scheduler wrapped around RunJob, not a
// second execution path.
//
// State (which minute a job last fired) is kept in memory only: a
// process restart simply resumes matching from "now", which is the
// correct behavior for a cron-style schedule (never "catch up" on
// missed runs — same semantics as system cron/systemd timers).
package jobs

import (
	"fmt"
	"time"

	"mova.local/core"
)

// ProjectLister is the minimal slice of core.Adapter the scheduler needs
// to discover which projects to scan — kept separate from core.Adapter
// so tests can fake it trivially.
type ProjectLister interface {
	ListProjects() ([]core.ProjectSummary, error)
	GetProject(name string) (*core.Project, error)
}

// RunDueJobs scans every project once, running (via RunJob) any job
// whose CronSpec matches `at`. Invalid "schedule" strings are reported
// as errors on that job's Result rather than aborting the scan — one
// malformed job.json entry never blocks every other project's jobs.
func RunDueJobs(adapter core.Adapter, lister ProjectLister, root string, at time.Time) []*Result {
	summaries, err := lister.ListProjects()
	if err != nil {
		return []*Result{{Errors: []string{"jobs scan: " + err.Error()}}}
	}

	var results []*Result
	for _, summary := range summaries {
		proj, err := lister.GetProject(summary.Name)
		if err != nil || len(proj.Jobs) == 0 {
			continue
		}
		for _, spec := range proj.Jobs {
			spec := spec
			if spec.Schedule == "" {
				continue
			}
			cron, err := ParseSchedule(spec.Schedule)
			if err != nil {
				results = append(results, &Result{
					Project: summary.Name,
					Errors:  []string{fmt.Sprintf("schedule: %v", err)},
				})
				continue
			}
			if cron.Matches(at) {
				results = append(results, RunJob(adapter, root, summary.Name, proj, spec))
			}
		}
	}
	return results
}

// RunScheduler blocks forever, checking every project's jobs once per
// minute and calling onResult for each job that fired. Callers (CLI's
// `mova jobs start`) pass a print/log function; stop by cancelling ctx-
// style via the returned stop channel close, or simply killing the
// process (same as any long-running daemon).
func RunScheduler(adapter core.Adapter, lister ProjectLister, root string, onResult func(*Result)) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	// Fire immediately once at startup for the current minute, then on
	// every subsequent tick — matches typical cron-daemon behavior of
	// not waiting up to 60s for the first check.
	check := func() {
		for _, res := range RunDueJobs(adapter, lister, root, time.Now()) {
			onResult(res)
		}
	}
	check()
	for range ticker.C {
		check()
	}
}
