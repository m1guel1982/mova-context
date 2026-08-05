// tui_projectpicker.go — lists every project via the same
// core.Adapter.ListProjects() `mova list` uses (nested multiagent
// agents included automatically, since they're ordinary projects with
// a "/" in their name — see orchestrator/config.go), then hands the
// chosen name to onPick. Reused by "Chat" (project context) and
// "Projects" (project dashboard) from the main menu.
package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"mova.local/core"
)

func newProjectPicker(app *tuiApp, title string, allowNone bool, onPick func(project string) tuiScreen) tuiScreen {
	var items []menuItem
	if allowNone {
		items = append(items, menuItem{
			title: "— No project —", desc: "Continue without loading context from any project",
			onSelect: func() tea.Cmd { return tuiPush(onPick("")) },
		})
	}

	fa := core.NewFileAdapter(app.root)
	projects, err := fa.ListProjects()
	if err != nil {
		items = append(items, menuItem{title: "Error", desc: err.Error()})
		return newMenuScreen(title, items, "")
	}
	for _, p := range projects {
		name := p.Name
		desc := p.Description
		if desc == "" {
			desc = fmt.Sprintf("tasks: %v", p.Tasks)
		}
		items = append(items, menuItem{
			title: name, desc: desc,
			onSelect: func() tea.Cmd { return tuiPush(onPick(name)) },
		})
	}
	if len(items) == 0 {
		items = append(items, menuItem{title: "(no projects)", desc: "Create one with `mova init <name>`"})
	}
	return newMenuScreen(title, items, "")
}