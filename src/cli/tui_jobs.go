// tui_jobs.go — lists a project's project.json "jobs" array and runs
// any of them on demand, calling the exact same jobs.RunJob the CLI's
// `mova jobs run`, the MCP "run_job" tool, and `POST /jobs/run` all
// call (see jobs/engine.go) — the TUI adds no execution path of its
// own, only a menu in front of the existing one.
package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"mova.local/core"
	"mova.local/jobs"
)

func newJobsScreen(app *tuiApp, project string) tuiScreen {
	fa := core.NewFileAdapter(app.root)
	proj, err := fa.GetProject(project)
	if err != nil {
		return newMenuScreen(project+" — jobs", []menuItem{{title: "Error", desc: err.Error()}}, "")
	}
	if len(proj.Jobs) == 0 {
		return newMenuScreen(project+" — jobs", []menuItem{
			{title: "(no configured jobs)", desc: "Add a \"jobs\" array to project.json — see PROJECT_JSON.md § Jobs"},
		}, "")
	}

	adapter := newAdapter(app.root, proj)
	var items []menuItem
	for i, spec := range proj.Jobs {
		index := i
		label := spec.Comment
		if label == "" {
			label = fmt.Sprintf("tasks=%v save=%q", spec.Tasks, spec.Save)
		}
		items = append(items, menuItem{
			title: fmt.Sprintf("[%d] %s", index, spec.Schedule), desc: label,
			onSelect: func() tea.Cmd {
				return func() tea.Msg {
					res, runErr := jobs.RunJobByIndex(adapter, app.root, project, proj, index)
					if runErr != nil {
						return tuiPushMsg{screen: newTextScreen(project+" — job "+fmt.Sprint(index), "Error: "+runErr.Error())}
					}
					return tuiPushMsg{screen: newTextScreen(project+" — job "+fmt.Sprint(index), renderJobResultTUI(res))}
				}
			},
		})
	}
	return newMenuScreen(project+" — jobs", items, "enter: run now · esc: back")
}

func renderJobResultTUI(res *jobs.Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Project: %s\n\n", res.Project)
	for _, step := range res.Steps {
		b.WriteString(step + "\n")
	}
	if !res.OK() {
		fmt.Fprintf(&b, "\n%d error(s):\n", len(res.Errors))
		for _, e := range res.Errors {
			b.WriteString("  - " + e + "\n")
		}
	}
	return b.String()
}