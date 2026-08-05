// tui_logs.go — shows the active log file (config/log/logging.json's
// "file"."path", resolved the exact same way mova.local/logging does —
// see logging.LoadConfig), read-only, refreshing every second so newly
// appended lines (e.g. from `mova jobs start` running elsewhere) show
// up without leaving this screen.
package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"mova.local/logging"
)

type logsScreen struct {
	path string
	vp   viewport.Model
}

type logsTickMsg time.Time

func newLogsScreen(app *tuiApp) *logsScreen {
	cfg := logging.LoadConfig(app.root)
	path := cfg.File.Path
	if path == "" {
		path = "logs/mova.log"
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(app.root, path)
	}
	vp := viewport.New(90, 24)
	return &logsScreen{path: path, vp: vp}
}

func (l *logsScreen) Init() tea.Cmd {
	l.reload()
	return logsTick()
}

func logsTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return logsTickMsg(t) })
}

func (l *logsScreen) reload() {
	data, err := os.ReadFile(l.path)
	if err != nil {
		l.vp.SetContent("(logging is disabled or log file does not exist yet)\n\n" +
			"Expected path: " + l.path +
			"\n\nEnable it in config/log/logging.json → \"enabled\": true")
		return
	}
	l.vp.SetContent(string(data))
	l.vp.GotoBottom()
}

func (l *logsScreen) Update(msg tea.Msg) (tuiScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		l.vp.Width, l.vp.Height = msg.Width-6, msg.Height-8
	case logsTickMsg:
		l.reload()
		return l, logsTick()
	}
	var cmd tea.Cmd
	l.vp, cmd = l.vp.Update(msg)
	return l, cmd
}

func (l *logsScreen) View() string {
	return tuiDocStyle.Render(tuiHeader("Logs") + "\n" + l.vp.View() +
		tuiFooter("↑/↓: move · auto-refreshes every second · esc: back"))
}