// actions_memory.go — the job engine's "memory" and "memory_archive"
// actions. Both delegate to the exact same core.Adapter methods `mova
// memory`/`mova memory-archive` and the "save_memory" MCP tool already
// use (AppendMemory/ArchiveMemory, see core/adapter.go) — no separate
// memory-writing path for jobs.
package jobs

import (
	"strings"
	"time"

	"mova.local/core"
)

// runMemory appends spec.Memory to the project's memory.md, with
// {date}/{time} placeholders expanded so a single job.json entry can be
// reused run after run (e.g. "Auditoría {date} completada").
func runMemory(jc *jobContext, res *Result) {
	if jc.Spec.Memory == "" {
		return
	}
	entry := ExpandDate(jc.Spec.Memory, jc.Now)
	entry = replaceTime(entry, jc.Now)
	if err := jc.Adapter.AppendMemory(jc.Project, entry); err != nil {
		res.fail("memory: %v", err)
		return
	}
	res.log("✓ memory updated: projects/%s/memory.md", jc.Project)
}

// runMemoryArchive archives entries older than spec.MemoryArchive.Days
// (or the project's own ArchiveConfig default when Days is 0/unset —
// see core.RetentionDays), via the same ArchiveMemory used by `mova
// memory-archive`.
func runMemoryArchive(jc *jobContext, res *Result) {
	spec := jc.Spec.MemoryArchive
	if spec == nil {
		return
	}
	days := spec.Days
	if days <= 0 {
		days = core.RetentionDays(jc.Proj.Archive)
	}
	if err := jc.Adapter.ArchiveMemory(jc.Project, days); err != nil {
		res.fail("memory_archive: %v", err)
		return
	}
	res.log("✓ memory archived: %s (entries older than %d days)", jc.Project, days)
}

func replaceTime(s string, now time.Time) string {
	return strings.ReplaceAll(s, "{time}", now.Format("15:04"))
}
