// access_policy.go — the ONE read policy for every content-returning
// door that is not the context assembly itself: MCP/HTTP read_file and
// read_document_layer, the tool loop of `mova chat` / chat_completion,
// and the check_read hook tool. Before this file each of those read the
// disk directly, ignoring "exclude", "focus" and the repo boundary.
//
// Rules, in order (first failing rule denies):
//  1. a project is required (no project → no policy → no read);
//  2. the path must be inside the project's repo (documents.WithinRepo,
//     symlinks resolved);
//  3. a whole-file/dir "exclude" entry denies it (same matcher as focus);
//  4. with read scope "focus", the file must be targeted by the task's
//     focus; symbol-level focus ("file::func=a,b") only exposes those
//     symbols.
//
// Symbol-level "exclude" entries never deny a file: they are stripped
// from whatever is returned (same as the context assembly).
package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	corefocus "mova.local/core/focus"
	focusrender "mova.local/core/focus/render"
	"mova.local/core/focus/resolvers"
	"mova.local/documents"
)

// Read modes returned by CheckRead.
const (
	ReadModeFull    = "full"    // whole file (minus symbol-level excludes)
	ReadModeSymbols = "symbols" // only the focused symbols of the file
)

// ReadDecision is the outcome of CheckRead: whether the read is allowed,
// the rule that decided it, and what view of the file may be returned.
type ReadDecision struct {
	Allowed bool     `json:"allowed"`
	Rule    string   `json:"rule"`              // e.g. "repo_boundary", "exclude", "focus_scope", "allowed"
	Reason  string   `json:"reason"`            // human-readable
	Path    string   `json:"path"`              // absolute path evaluated
	RelPath string   `json:"rel_path"`          // repo-relative, slash-separated
	Scope   string   `json:"scope"`             // "focus" | "repo"
	Mode    string   `json:"mode,omitempty"`    // ReadModeFull | ReadModeSymbols
	Entries []string `json:"entries,omitempty"` // focus entries that grant the read (symbols mode)
	Exclude []string `json:"-"`
}

// RepoDir returns the absolute repo directory of proj.
func RepoDir(root string, proj *Project) string {
	if proj == nil {
		return ""
	}
	return focusrender.ResolveRepoPath(root, proj.Repo)
}

// ReadScopeFor resolves proj.ReadScope for a task scope.
func ReadScopeFor(proj *Project, taskName string) string {
	switch strings.ToLower(strings.TrimSpace(proj.ReadScope)) {
	case "repo":
		return "repo"
	case "focus":
		return "focus"
	}
	if len(FocusInScope(proj, taskName)) > 0 {
		return "focus"
	}
	return "repo"
}

// FocusInScope is the union of the focus lists of the tasks in scope.
func FocusInScope(proj *Project, taskName string) []string {
	var out []string
	names := tasksInScope(proj, taskName)
	if len(names) == 0 {
		return dedupe(proj.Focus)
	}
	for _, n := range names {
		t := proj.Tasks[n]
		out = append(out, resolveTaskFocus(proj, &t)...)
	}
	return dedupe(out)
}

// ExcludeInScope is the union of the exclude lists of the tasks in scope.
func ExcludeInScope(proj *Project, taskName string) []string {
	if len(tasksInScope(proj, taskName)) == 0 {
		return dedupe(proj.Exclude)
	}
	return ApplyExcludes(proj, taskName)
}

// CheckRead decides whether requestedAbs (already resolved by
// documents.ResolveFilePath, or an absolute path from a hook) may be read
// for project/task. It never touches file contents.
func CheckRead(root string, proj *Project, taskName, requestedAbs string) ReadDecision {
	d := ReadDecision{Path: requestedAbs}
	if proj == nil {
		d.Rule, d.Reason = "project_required", "se requiere \"project\": sin proyecto no hay política que aplicar"
		return d
	}
	repoDir := RepoDir(root, proj)
	abs := filepath.Clean(requestedAbs)
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(repoDir, abs)
	}
	d.Path = abs
	if !documents.WithinRepo(repoDir, abs) {
		d.Rule, d.Reason = "repo_boundary", "fuera del repo del proyecto"
		return d
	}
	if rel, err := filepath.Rel(repoDir, abs); err == nil {
		d.RelPath = filepath.ToSlash(rel)
	}
	d.Exclude = ExcludeInScope(proj, taskName)
	if resolvers.PathExcluded(d.Exclude, abs) {
		d.Rule, d.Reason = "exclude", "excluido por \"exclude\" de project.json"
		return d
	}
	d.Scope = ReadScopeFor(proj, taskName)
	if d.Scope == "repo" {
		d.Allowed, d.Rule, d.Mode, d.Reason = true, "allowed", ReadModeFull, "dentro del repo y no excluido (read_scope: repo)"
		return d
	}

	ctx := corefocus.Context{RepoPath: repoDir, Index: &corefocus.FileIndex{}}
	var symbolEntries []string
	for _, entry := range FocusInScope(proj, taskName) {
		file, _, hasSym := strings.Cut(entry, "::")
		if !focusTargets(ctx, repoDir, file, abs) {
			continue
		}
		if !hasSym {
			d.Allowed, d.Rule, d.Mode, d.Reason = true, "allowed", ReadModeFull, "en focus: "+entry
			return d
		}
		symbolEntries = append(symbolEntries, entry)
	}
	if len(symbolEntries) > 0 {
		sort.Strings(symbolEntries)
		d.Allowed, d.Rule, d.Mode, d.Entries = true, "allowed", ReadModeSymbols, symbolEntries
		d.Reason = "solo los símbolos en focus: " + strings.Join(symbolEntries, "; ")
		return d
	}
	d.Rule, d.Reason = "focus_scope", "no está en el focus de la tarea (read_scope: focus)"
	return d
}

// focusTargets reports whether a focus entry's file part covers abs:
// exact file, a directory containing it, or any file the focus engine's
// own locator resolves the entry to.
func focusTargets(ctx corefocus.Context, repoDir, file, abs string) bool {
	file = strings.TrimSpace(file)
	if file == "" {
		return false
	}
	candidate := focusrender.AbsUnder(repoDir, file)
	if info, err := os.Stat(candidate); err == nil {
		if info.IsDir() {
			return documents.WithinRepo(candidate, abs)
		}
		return filepath.Clean(candidate) == abs
	}
	for _, p := range resolvers.LocateTarget(ctx, file) {
		if filepath.Clean(focusrender.AbsUnder(repoDir, p)) == abs {
			return true
		}
	}
	return false
}

// GovernedRead applies CheckRead and returns the view of the file the
// policy allows: the focused symbols only (symbols mode) or the whole
// file minus symbol-level excludes (full mode). Office/PDF files go
// through documents.ReadDocumentLayer. The result is NOT yet sanitized —
// callers pass it through budget.GovernText (PII/secrets), which core
// cannot import.
func GovernedRead(root string, proj *Project, taskName, requestedAbs string) (string, ReadDecision, error) {
	d := CheckRead(root, proj, taskName, requestedAbs)
	if !d.Allowed {
		return "", d, fmt.Errorf("lectura denegada por Mova (%s): %s — %s", d.Rule, d.RelPathOr(), d.Reason)
	}
	if d.Mode == ReadModeSymbols {
		text, _ := focusrender.RenderFocusContext(root, proj.Repo, d.Entries, nil, d.Exclude)
		return text, d, nil
	}
	var content string
	var err error
	switch strings.ToLower(filepath.Ext(d.Path)) {
	case ".docx", ".xlsx", ".pdf":
		content, err = documents.ReadDocumentLayer(d.Path)
	default:
		content, err = documents.ReadFile(d.Path)
	}
	if err != nil {
		return "", d, err
	}
	if resolvers.HasSymbolExcludes(d.Exclude, d.RelPath) {
		content = resolvers.ApplyAstSymbolExcludes(d.Exclude, d.RelPath, content)
	}
	return content, d, nil
}

// RelPathOr returns RelPath, or Path when the file is outside the repo.
func (d ReadDecision) RelPathOr() string {
	if d.RelPath != "" {
		return d.RelPath
	}
	return d.Path
}
