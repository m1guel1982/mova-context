// tui_textscreen.go — a read-only screen for TEXT THAT ISN'T A FILE:
// the output of running a job (tui_jobs.go) or a multiagent group
// (tui_agents.go). Distinct from fileScreen (tui_fileview.go), which
// always reads/writes an actual path — this one just displays a string
// already in memory, via the same viewport component bubbles ships.
package main

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type textScreen struct {
	title string
	vp    viewport.Model
}

func newTextScreen(title, content string) *textScreen {
	vp := viewport.New(90, 24)
	vp.SetContent(content)
	return &textScreen{title: title, vp: vp}
}

func (t *textScreen) Init() tea.Cmd { return nil }

func (t *textScreen) Update(msg tea.Msg) (tuiScreen, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		t.vp.Width, t.vp.Height = wsz.Width-6, wsz.Height-8
	}
	var cmd tea.Cmd
	t.vp, cmd = t.vp.Update(msg)
	return t, cmd
}

func (t *textScreen) View() string {
	return tuiDocStyle.Render(tuiHeader(t.title) + "\n" + t.vp.View() + tuiFooter("↑/↓: move · esc: back"))
}
