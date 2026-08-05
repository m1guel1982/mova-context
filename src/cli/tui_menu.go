// tui_menu.go — ONE generic, list-based menu screen, reused for the
// main menu, the project picker, the project dashboard, the models
// list, the multiagent group/agent lists, and the reports list. Adding
// a new section to the TUI almost always means calling newMenuScreen
// with a different title/items, not writing a new screen type — this
// is the single biggest lever against "implementaciones paralelas".
package main

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// menuItem is one selectable row. onSelect returns the tea.Cmd to run
// when Enter is pressed on it — usually tuiPush(newSomeScreen(...)).
type menuItem struct {
	title, desc string
	onSelect    func() tea.Cmd
}

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.desc }
func (i menuItem) FilterValue() string { return i.title }

type menuScreen struct {
	title string
	list  list.Model
	help  string
}

// newMenuScreen builds a menu titled "section" with the given items.
// help is a short footer hint line (e.g. "enter: abrir · esc: volver").
func newMenuScreen(section string, items []menuItem, help string) *menuScreen {
	litems := make([]list.Item, len(items))
	for i, it := range items {
		litems[i] = it
	}
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(tuiAccent).BorderLeftForeground(tuiAccent)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(tuiMuted)

	l := list.New(litems, delegate, 80, 20)
	l.Title = section
	l.Styles.Title = tuiTitleStyle
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)

	if help == "" {
		help = "↑/↓: move · enter: open · /: search · esc: back · ctrl+c: exit"
	}
	return &menuScreen{title: section, list: l, help: help}
}

func (m *menuScreen) Init() tea.Cmd { return nil }

func (m *menuScreen) Update(msg tea.Msg) (tuiScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width-4, msg.Height-6)
	case tea.KeyMsg:
		if msg.String() == "enter" {
			if it, ok := m.list.SelectedItem().(menuItem); ok && it.onSelect != nil {
				return m, it.onSelect()
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *menuScreen) View() string {
	return tuiDocStyle.Render(m.list.View() + tuiFooter(m.help))
}
