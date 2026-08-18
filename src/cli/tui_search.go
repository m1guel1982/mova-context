// tui_search.go — "Search" from the main menu: a query box (reusing
// the same core.FileAdapter.Search agents/skills/prompts already index
// — see core/file_adapter.go, unchanged logic, this is only a new
// PRESENTATION of it) followed by a real, navigable results list. Each
// result opens the exact file at the exact matching line via
// newFileScreenAtQuery (tui_fileview.go) — this is the fix for the
// "search only shows a count, never takes you anywhere" gap: previously
// there was no TUI presentation of Search at all (CLI-only, via `mova
// search`); this screen is what makes results clickable/navigable
// instead of just counted.
package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"

	"mova.local/core"
)

type searchScreen struct {
	app    *tuiApp
	query  textinput.Model
	status string
}

func newSearchScreen(app *tuiApp) *searchScreen {
	ti := textinput.New()
	ti.Placeholder = "search agents, skills, prompts…"
	ti.Focus()
	return &searchScreen{app: app, query: ti}
}

func (s *searchScreen) Init() tea.Cmd { return textinput.Blink }

func (s *searchScreen) Update(msg tea.Msg) (tuiScreen, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		s.query, cmd = s.query.Update(msg)
		return s, cmd
	}
	switch keyMsg.String() {
	case "enter":
		return s.runSearch()
	default:
		var cmd tea.Cmd
		s.query, cmd = s.query.Update(msg)
		return s, cmd
	}
}

// runSearch pushes a results screen (a plain newMenuScreen — see
// tui_projectpicker.go for the same reuse pattern) built from real
// core.FileAdapter.Search results. An empty/error/no-match outcome is
// shown as a single explanatory item rather than silently doing
// nothing, so the person always knows what happened.
func (s *searchScreen) runSearch() (tuiScreen, tea.Cmd) {
	query := s.query.Value()
	if query == "" {
		return s, nil
	}
	fa := core.NewFileAdapter(s.app.root)
	results, err := fa.Search(query, "")
	if err != nil {
		s.status = "Search error: " + err.Error()
		return s, nil
	}
	return s, tuiPush(newSearchResultsScreen(s.app, query, results))
}

func (s *searchScreen) View() string {
	body := tuiHeader("Search") + "\n\n"
	body += tuiHelpStyle.Render("Query: ") + s.query.View() + "\n"
	if s.status != "" {
		body += "\n" + tuiWarnStyle.Render(s.status)
	}
	body += tuiFooter("enter: search · esc: back")
	return tuiDocStyle.Render(body)
}

// newSearchResultsScreen turns each core.SearchResult into a
// navigable menuItem: selecting one opens ITS OWN file, in find mode,
// already jumped to the matching line — see newFileScreenAtQuery.
// This is the actual fix: previously nothing did this, a match's
// existence was all a person could learn.
func newSearchResultsScreen(app *tuiApp, query string, results []core.SearchResult) tuiScreen {
	title := fmt.Sprintf("Search results for %q", query)
	if len(results) == 0 {
		items := []menuItem{{title: "No matches", desc: "Try a different query — searches agent/skill/prompt names and full text"}}
		return newMenuScreen(title, items, "esc: back")
	}
	items := make([]menuItem, 0, len(results))
	for _, r := range results {
		r := r // capture for the closure below
		loc := r.Kind + "/" + r.Domain + "/" + r.Lang
		if r.Line > 0 {
			loc += fmt.Sprintf(" · line %d", r.Line)
		}
		items = append(items, menuItem{
			title: r.Name,
			desc:  loc + " — " + r.Excerpt,
			onSelect: func() tea.Cmd {
				return tuiPush(newFileScreenAtQuery(app, r.Name, r.Path, false, query))
			},
		})
	}
	return newMenuScreen(title, items, "↑/↓: navigate · enter: open at match · esc: back")
}
