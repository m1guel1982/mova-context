// closure.go — dependency-closure validation of a task's context spec.
//
// Question answered: "do the symbols selected by `focus` depend on code
// that the same spec explicitly EXCLUDES?" If so, the context Mova is
// about to release is incomplete by construction (the model is told the
// excluded code "does not exist" while the selected code calls it).
//
// It reuses the exact parser and edge resolution of the dependency graph
// (build.go): same nodes, same call/ref edges, same module resolution —
// no second analysis. Files imported with a RELATIVE module path by a
// focus file are loaded too, so callees outside focus/exclude are
// reported as "outside" (in the repo, not in the context).
//
// Limits (stated, not hidden): static calls/refs only (no dynamic
// dispatch, reflection or string-built calls); only relative imports are
// followed; a callee name that cannot be resolved is not reported.
package graph

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"mova.local/core"
	"mova.local/core/focus/astfilter"
)

// Dependency / ClosureReport are defined in core (see core/closure.go).
type (
	Dependency    = core.Dependency
	ClosureReport = core.ClosureReport
)

func init() { core.ClosureHook = ProjectClosure }

var relExts = []string{"", ".js", ".mjs", ".cjs", ".ts", ".tsx", ".jsx", ".py", "/index.js", "/index.ts"}

// relativeImports resolves relative module imports of the located files
// to files on disk, returned as extra located entries without selectors.
func relativeImports(files []located) []located {
	have := map[string]bool{}
	for _, f := range files {
		have[filepath.Clean(f.path)] = true
	}
	var extra []located
	for _, f := range files {
		data, err := os.ReadFile(f.path)
		if err != nil {
			continue
		}
		facts, ok := astfilter.Analyze(data, f.path)
		if !ok {
			continue
		}
		for _, im := range facts.Imports {
			m := slash(strings.TrimSpace(im.Module))
			if !strings.HasPrefix(m, "./") && !strings.HasPrefix(m, "../") {
				continue
			}
			base := filepath.Join(filepath.Dir(f.path), filepath.FromSlash(m))
			for _, ext := range relExts {
				cand := filepath.Clean(base + ext)
				if st, err := os.Stat(cand); err == nil && !st.IsDir() {
					if !have[cand] {
						have[cand] = true
						extra = append(extra, located{path: cand})
					}
					break
				}
			}
		}
	}
	return extra
}

func symbolID(nodeID string) string { return strings.Replace(nodeID, "#", "::", 1) }

func accepted(accept []core.AcceptedDependency, to string) (bool, string) {
	name := to
	if i := strings.LastIndex(to, "::"); i >= 0 {
		name = to[i+2:]
	}
	for _, a := range accept {
		s := strings.TrimSpace(a.Symbol)
		if s == to || s == name || strings.HasSuffix(to, "/"+s) {
			return true, a.Reason
		}
	}
	return false, ""
}

// Closure validates focus/exclude of one task against the repo at
// repoPath. policy "off" or an empty focus returns Checked=false.
func Closure(repoPath, task, policy string, focus, exclude []string, accept []core.AcceptedDependency) ClosureReport {
	rep := ClosureReport{Task: task, Policy: policy, Conflicts: []Dependency{}, Outside: []Dependency{}}
	if policy == "off" {
		rep.Note = "dependency_policy: off"
		return rep
	}
	if len(focus) == 0 {
		rep.Note = "sin focus: nada que validar"
		return rep
	}
	sp := spec{RepoPath: repoPath, TaskName: task, Focus: focus, Exclude: exclude}
	files := locate(sp)
	files = append(files, relativeImports(files)...)
	g, _ := build(sp, files)
	style := map[string]string{}
	for _, n := range g.Nodes {
		style[n.ID] = n.Style
		if n.Style == "focus" {
			rep.Focused++
		}
	}
	rep.Checked = rep.Focused > 0
	if !rep.Checked {
		rep.Note = "ningún símbolo de focus pudo analizarse por AST (archivos de datos o lenguaje no soportado)"
	}
	for _, e := range g.Edges {
		if (e.Kind != "call" && e.Kind != "ref") || style[e.From] != "focus" {
			continue
		}
		var status string
		switch style[e.To] {
		case "excluded":
			status = "excluded"
		case "outside":
			status = "outside"
		default:
			continue
		}
		d := Dependency{From: symbolID(e.From), To: symbolID(e.To), Kind: e.Kind, Status: status, Inferred: e.Inferred}
		if status == "outside" {
			rep.Outside = append(rep.Outside, d)
			continue
		}
		if ok, reason := accepted(accept, d.To); ok {
			d.Accepted, d.Reason = true, reason
			rep.Accepted = append(rep.Accepted, d)
			continue
		}
		rep.Conflicts = append(rep.Conflicts, d)
	}
	for _, list := range [][]Dependency{rep.Conflicts, rep.Accepted, rep.Outside} {
		sort.Slice(list, func(i, j int) bool { return list[i].From+list[i].To < list[j].From+list[j].To })
	}
	return rep
}

// ProjectClosure runs Closure for every task in scope of taskName and
// returns one report per task (TaskAll → all tasks).
func ProjectClosure(root string, proj *core.Project, taskName string) []ClosureReport {
	repo := core.RepoDir(root, proj)
	names := []string{taskName}
	if core.IsAllTasks(taskName) {
		names = core.SortedTaskNames(proj)
	}
	var out []ClosureReport
	for _, n := range names {
		t := proj.Tasks[n]
		out = append(out, Closure(repo, n, core.DependencyPolicyFor(proj, &t),
			core.ResolveFocus(proj, &t), core.ResolveExclude(proj, &t), t.AcceptMissing))
	}
	return out
}
