// tui_mainmenu.go — builds the main menu (`mova ui`'s first screen):
// one entry per major Mova Context capability. Every entry just pushes
// another screen built from an existing tui_*.go constructor — no
// business logic lives here.
package main

import tea "github.com/charmbracelet/bubbletea"

func newMainMenu(app *tuiApp) tuiScreen {
	items := []menuItem{
		{
			title: "Chat", desc: "Chat with active model (same as `mova chat`)",
			onSelect: func() tea.Cmd {
				return tuiPush(newProjectPicker(app, "Chat — choose a project (or Esc to chat without project)", true, func(project string) tuiScreen {
					return newChatScreen(app, project)
				}))
			},
		},
		{
			title: "Projects", desc: "project.json, workflow.md, memory, jobs, and reports per project",
			onSelect: func() tea.Cmd {
				return tuiPush(newProjectPicker(app, "Projects", false, func(project string) tuiScreen {
					return newProjectDashboard(app, project)
				}))
			},
		},
		{
			title: "Workflow.md", desc: "Main execution document (project root)",
			onSelect: func() tea.Cmd { return tuiPush(newFileScreen(app, "workflow.md", workflowMDPath(app.root), true)) },
		},
		{
			title: "Multi-agents", desc: "Agent groups — projects/<group>/config.json",
			onSelect: func() tea.Cmd { return tuiPush(newGroupPicker(app)) },
		},
		{
			title: "Models", desc: "config/models/*.json — view and edit provider settings",
			onSelect: func() tea.Cmd { return tuiPush(newModelsMenu(app)) },
		},
		{
			title: "Logging", desc: "config/log/logging.json — enable/disable and adjust logging",
			onSelect: func() tea.Cmd { return tuiPush(newFileScreen(app, "Logging", loggingConfigPath(app.root), true)) },
		},
		{
			title: "Logs", desc: "View active log file (read-only, auto-updates)",
			onSelect: func() tea.Cmd { return tuiPush(newLogsScreen(app)) },
		},
		{
			title: "Exit", desc: "Close interface",
			onSelect: func() tea.Cmd { return tea.Quit },
		},
	}
	return newMenuScreen("Mova Context", items, "↑/↓: navigate · enter: select · ctrl+c: exit")
}