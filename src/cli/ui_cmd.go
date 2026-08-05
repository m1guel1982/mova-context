// ui_cmd.go — `mova ui`, the single command that opens the whole
// terminal interface (see tui_app.go for the screen-stack model and
// every tui_*.go file for one section each). Everything the TUI does
// calls straight into the same packages the plain CLI commands already
// use (core, budget, jobs, orchestrator, documents, models, logging) —
// this file only wires a tea.Program around tuiApp, nothing else.
package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

// runUI starts the terminal interface. project, if non-empty, jumps
// straight into that project's dashboard (`mova ui <project>`) instead
// of showing the main menu first.
func runUI(root, project string) {
	app := newTUIApp(root)
	if project != "" {
		app.stack = append(app.stack, newProjectDashboard(app, project))
	}

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		die("failed to start interface: " + err.Error())
	}
}
