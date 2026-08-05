// actions_save.go — the job engine's "save" action: writes jc.TaskOutput
// (built by runTasks, actions_tasks.go) through documents.Save — the
// exact same SaveService `/save` (chat), the "save" MCP tool, and
// `POST /save` already use (see documents/save_service.go). The output
// FORMAT is picked automatically from the extension in spec.Save
// (.md/.pdf/.docx/.xlsx/.svg/...), same as every other Save caller — the
// job engine never picks a generator itself.
package jobs

import (
	"strings"
	"time"

	"mova.local/documents"
)

// ExpandDate replaces every "{date}" placeholder in path with today's
// date (YYYY-MM-DD) — the one dynamic token spec.Save supports, e.g.
// "reports/auditoria_{date}.pdf". Exported so CLI/tests can preview the
// resolved path without running the job.
func ExpandDate(path string, now time.Time) string {
	return strings.ReplaceAll(path, "{date}", now.Format("2006-01-02"))
}

// runSave writes jc.TaskOutput to spec.Save (with {date} expanded). If no
// task ran (TaskOutput is empty) it still writes an empty-but-valid
// report rather than silently skipping — a job author who configured
// "save" expects a file to exist after every scheduled run.
func runSave(jc *jobContext, res *Result) {
	if jc.Spec.Save == "" {
		return
	}
	repo := "."
	if jc.Proj != nil && jc.Proj.Repo != "" {
		repo = jc.Proj.Repo
	}
	path := ExpandDate(jc.Spec.Save, jc.Now)
	content := jc.TaskOutput
	if content == "" {
		content = "_(no tasks configured or produced output for this run — " + jc.Now.Format("2006-01-02 15:04") + ")_"
	}

	result, err := documents.Save(jc.Root, documents.SaveRequest{
		Path: path, Content: content, Overwrite: true, Repo: repo,
	})
	if err != nil {
		res.fail("save %q: %v", path, err)
		return
	}
	res.log("✓ %s", result.Message)
}
