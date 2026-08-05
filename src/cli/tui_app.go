// tui_app.go — the root Bubble Tea model. Mova's TUI is one program
// (`mova ui`) with a STACK of screens: opening a section pushes a new
// screen; Esc pops back to the previous one; Ctrl+C quits from
// anywhere. Every section (chat, project.json, jobs, agents...) is a
// small tuiScreen implementation in its own tui_*.go file — this file
// only routes Update/View to whichever screen is on top and never
// contains section-specific logic itself, so adding a new section
// never touches this file.
package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

// tuiScreen is any full-screen view the app can push onto the stack.
// Update returns the (possibly replaced) screen plus a tea.Cmd, mirroring
// bubbletea's own Model.Update contract — screens are values, not
// pointers, for the same reason bubbles' own components are.
type tuiScreen interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (tuiScreen, tea.Cmd)
	View() string
}

// tuiApp is the tea.Model passed to tea.NewProgram — see ui_cmd.go.
type tuiApp struct {
	root          string
	stack         []tuiScreen
	width, height int
}

func newTUIApp(root string) *tuiApp {
	app := &tuiApp{root: root}
	app.stack = []tuiScreen{newMainMenu(app)}
	return app
}

func (a *tuiApp) current() tuiScreen { return a.stack[len(a.stack)-1] }

// tuiPushMsg/tuiPopMsg are how a child screen asks the app to navigate —
// returned wrapped in a tea.Cmd (func() tea.Msg { return tuiPushMsg{...} }),
// exactly like any other bubbletea message. Screens never mutate
// a.stack directly; only Update below does, keeping navigation in one
// place.
type tuiPushMsg struct{ screen tuiScreen }
type tuiPopMsg struct{}

// tuiStatusMsg lets any screen show a transient status line (success or
// error) through the same status bar style — see tui_style.go.
type tuiStatusMsg struct {
	text  string
	isErr bool
}

func (a *tuiApp) Init() tea.Cmd {
	return a.current().Init()
}

func (a *tuiApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return a, tea.Quit
		case "esc":
			if len(a.stack) > 1 {
				a.stack = a.stack[:len(a.stack)-1]
				return a, nil
			}
			return a, tea.Quit
		}

	case tuiPushMsg:
		a.stack = append(a.stack, msg.screen)
		return a, msg.screen.Init()

	case tuiPopMsg:
		if len(a.stack) > 1 {
			a.stack = a.stack[:len(a.stack)-1]
		}
		return a, nil
	}

	updated, cmd := a.current().Update(msg)
	a.stack[len(a.stack)-1] = updated
	return a, cmd
}

func (a *tuiApp) View() string {
	return a.current().View()
}

// tuiPush is the tea.Cmd helper every screen uses to navigate forward.
func tuiPush(s tuiScreen) tea.Cmd {
	return func() tea.Msg { return tuiPushMsg{screen: s} }
}

// tuiPop is the tea.Cmd helper to navigate back programmatically
// (Esc already does this globally — this is for "Volver" menu items).
func tuiPop() tea.Cmd {
	return func() tea.Msg { return tuiPopMsg{} }
}
