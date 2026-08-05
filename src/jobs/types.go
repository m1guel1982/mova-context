// Package jobs implements the Job Engine: a single, transport-agnostic
// executor for project.json's "jobs" array (core.JobSpec — see
// core/types.go). CLI, chat, HTTP, and MCP all call RunJob/RunProjectJobs
// below — never their own copy of "read tasks, save, memory, delete,
// budget" logic. See docs/i18n/en/PROJECT_JSON.md § Jobs and workflow.md
// § Job Engine for the user-facing spec this package implements.
//
// Extensibility: adding a new job action (e.g. a future "notify") never
// touches RunJob's core loop — add a field to core.JobSpec, then one
// "if spec.NewThing != nil { ... }" block calling a new actions_*.go
// file, following the exact shape of runSave/runMemory/runDelete/
// runBudget below. See docs/SOURCE.md § Job Engine extensibility.
package jobs

import (
	"fmt"
	"time"

	"mova.local/core"
)

// Result summarizes one job run — returned to every door (CLI prints it,
// MCP/HTTP wrap it as the tool's text result) so all three report the
// exact same information.
type Result struct {
	Project string
	Steps   []string // human-readable log of what happened, in execution order
	Errors  []string // non-fatal errors collected along the way (one job step failing doesn't abort the others)
}

func (r *Result) log(format string, args ...any) {
	r.Steps = append(r.Steps, fmt.Sprintf(format, args...))
}

func (r *Result) fail(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	r.Errors = append(r.Errors, msg)
	r.log("✗ %s", msg)
}

// OK reports whether the run had zero errors.
func (r *Result) OK() bool { return len(r.Errors) == 0 }

// jobContext bundles what every action needs — avoids a long parameter
// list across runTasks/runSave/runMemory/runMemoryArchive/runDelete/runBudget.
type jobContext struct {
	Adapter core.Adapter
	Root    string
	Project string
	Proj    *core.Project
	Spec    core.JobSpec
	Now     time.Time
	// TaskOutput accumulates the assembled context of every task this
	// job ran (see actions_tasks.go) — what "save" (actions_save.go)
	// writes out, if the job declares one.
	TaskOutput string
}
