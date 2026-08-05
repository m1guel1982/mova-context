// tui_agents.go — the multiagent screens: pick a group (a
// projects/<name>/config.json — see orchestrator/config.go), then list
// its agents and run one or all of them, calling the exact same
// orchestrator.RunGroup the CLI's `mova agents run`, the MCP
// "run_agent" tool, and `POST /agents/run` all call.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"mova.local/core"
	"mova.local/orchestrator"
)

func newGroupPicker(app *tuiApp) tuiScreen {
	dir := filepath.Join(app.root, "projects")
	entries, err := os.ReadDir(dir)
	var items []menuItem
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			group := e.Name()
			if _, statErr := os.Stat(groupConfigPath(app.root, group)); statErr != nil {
				continue
			}
			items = append(items, menuItem{
				title: group, desc: "projects/" + group + "/config.json",
				onSelect: func() tea.Cmd { return tuiPush(newAgentsScreen(app, group)) },
			})
		}
	}
	if len(items) == 0 {
		items = append(items, menuItem{title: "(no groups)", desc: "Create projects/<group>/config.json — see PROJECT_JSON.md § Multiagent"})
	}
	return newMenuScreen("Multiagent", items, "")
}

func newAgentsScreen(app *tuiApp, group string) tuiScreen {
	cfg, err := orchestrator.LoadGroupConfig(app.root, group)
	if err != nil {
		return newMenuScreen(group, []menuItem{{title: "Error", desc: err.Error()}}, "")
	}

	fa := core.NewFileAdapter(app.root)
	items := []menuItem{
		{
			title: "▶ Run all", desc: fmt.Sprintf("Executes all %d agents in sequence", len(cfg.Agents)),
			onSelect: func() tea.Cmd {
				return func() tea.Msg { return tuiPushMsg{screen: runAgentsAndShow(app, fa, group, nil)} }
			},
		},
	}
	for _, agent := range cfg.Agents {
		agent := agent
		items = append(items, menuItem{
			title: agent, desc: group + "/" + agent,
			onSelect: func() tea.Cmd {
				return func() tea.Msg { return tuiPushMsg{screen: runAgentsAndShow(app, fa, group, []string{agent})} }
			},
		})
	}
	return newMenuScreen(group+" — agents", items, "enter: run · esc: back")
}

func runAgentsAndShow(app *tuiApp, adapter core.Adapter, group string, only []string) tuiScreen {
	results, err := orchestrator.RunGroup(adapter, app.root, group, only, "")
	if err != nil {
		return newTextScreen(group, "Error: "+err.Error())
	}
	var b strings.Builder
	for _, r := range results {
		fmt.Fprintf(&b, "=== %s ===\n", r.Project)
		if r.Err != nil {
			b.WriteString(r.Err.Error() + "\n\n")
			continue
		}
		b.WriteString(r.Text + "\n\n")
	}
	return newTextScreen(group, b.String())
}