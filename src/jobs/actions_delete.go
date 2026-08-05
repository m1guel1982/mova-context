// actions_delete.go — the job engine's "delete" action. spec.Delete
// entries may be glob patterns ("reports/temp_*.csv"), which
// documents.Delete itself doesn't expand (its Paths are literal,
// already-resolved-by-convention paths — see documents/delete_service.go)
// — so this file expands globs against the repo root and then hands the
// literal, matched paths to the exact same documents.Delete used by
// chat's `/delete`, the "delete_path" MCP tool, and `POST /delete`.
// Jobs run unattended, so Confirm is always true here — there's no
// person to ask y/n (same rule chat_completion's apply_edits and
// MCP/HTTP delete already follow: automated callers must opt in
// explicitly, which a job author does simply by listing the path).
package jobs

import (
	"path/filepath"

	"mova.local/documents"
)

func runDelete(jc *jobContext, res *Result) {
	if len(jc.Spec.Delete) == 0 {
		return
	}
	repo := "."
	if jc.Proj != nil && jc.Proj.Repo != "" {
		repo = jc.Proj.Repo
	}
	repoRoot := jc.Root
	if !filepath.IsAbs(repo) {
		repoRoot = filepath.Join(jc.Root, repo)
	} else {
		repoRoot = repo
	}

	var matched []string
	for _, pattern := range jc.Spec.Delete {
		full := ExpandDate(pattern, jc.Now)
		abs := full
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(repoRoot, full)
		}
		hits, err := filepath.Glob(abs)
		if err != nil {
			res.fail("delete %q: %v", pattern, err)
			continue
		}
		if len(hits) == 0 {
			res.log("· delete %q: no matching files", pattern)
			continue
		}
		matched = append(matched, hits...)
	}
	if len(matched) == 0 {
		return
	}

	result, err := documents.Delete(jc.Root, documents.DeleteRequest{
		Paths: matched, Repo: repo, Confirm: true,
	})
	if err != nil {
		res.fail("delete: %v", err)
		return
	}
	for _, d := range result.Deleted {
		res.log("✓ deleted: %s", d)
	}
}
