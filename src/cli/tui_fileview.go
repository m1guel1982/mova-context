// tui_fileview.go — ONE screen type to view and edit any text file:
// project.json, workflow.md, memory.md, config/models/*.json,
// config/log/logging.json. Loads the file with plain os.ReadFile,
// saves with os.WriteFile after validating with the exact same
// documents.ValidateTextFormat used by the "save"/"write_file"
// MCP tools (see documents/textfile.go) — so a malformed JSON edit is
// rejected the same way it would be from chat or MCP, never a second
// validation rule. Read-only mode (reports, logs, execution history)
// simply disables Ctrl+S instead of blocking keystrokes, so the same
// component still scrolls/searches normally.
//
// ctrl+f opens a find bar (search/searchInput below) to look for text
// inside the document currently being viewed/edited — case-insensitive,
// cycles through every match with enter (next) / up (previous). This is
// intentionally NOT bound to "esc" or "/": esc is reserved globally by
// the app (tui_app.go) to pop back to the previous screen, and "/" is a
// perfectly ordinary character to type into a document (paths, Markdown,
// JSON strings...), so it must keep going straight into the textarea.
// ctrl+f both opens AND closes the find bar, so no key that would
// otherwise navigate away or edit text is ever repurposed.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"mova.local/documents"
)

type fileScreen struct {
	app       *tuiApp
	title     string
	path      string
	editable  bool
	area      textarea.Model
	status    string
	statusErr bool
	loadErr   error

	// Find/search — see the package doc comment above.
	searchMode bool
	search     textinput.Model
	matches    []fileMatch
	matchIdx   int // index into matches of the current match, -1 = none jumped to yet
	searchNote string

	// pendingQuery: set by newFileScreenAtQuery (tui_search.go) to open
	// this screen already positioned at a match, instead of an empty
	// find bar — consumed once, in Init(), right after content loads.
	pendingQuery string
}

// fileMatch is one occurrence of the search query, as a (row, col)
// textarea position — the same coordinate space textarea.Model.Line()/
// SetCursor(col) use, since textarea has no byte-offset cursor API.
type fileMatch struct {
	row, col int
}

func newFileScreen(app *tuiApp, title, path string, editable bool) *fileScreen {
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.SetWidth(90)
	ta.SetHeight(24)

	si := textinput.New()
	si.Placeholder = "search this document…"

	return &fileScreen{app: app, title: title, path: path, editable: editable, area: ta, search: si, matchIdx: -1}
}

// newFileScreenAtQuery is what tui_search.go's global search results
// open into: the SAME file viewer newFileScreen always uses, just
// already in find mode with query typed in and the cursor jumped to
// its first match — this is what turns a search result into "taken to
// the exact place", reusing 100% of the existing find/jump machinery
// (recomputeMatches/jumpMatch/moveCursorTo) instead of a second
// implementation of it.
func newFileScreenAtQuery(app *tuiApp, title, path string, editable bool, query string) *fileScreen {
	f := newFileScreen(app, title, path, editable)
	f.pendingQuery = query
	return f
}

func (f *fileScreen) Init() tea.Cmd {
	data, err := os.ReadFile(f.path)
	if err != nil {
		f.loadErr = err
		f.area.SetValue("")
	} else {
		f.area.SetValue(string(data))
	}
	f.area.Focus()
	if f.pendingQuery != "" {
		f.searchMode = true
		f.search.SetValue(f.pendingQuery)
		f.search.Focus()
		f.recomputeMatches()
		f.jumpMatch(1)
		f.pendingQuery = ""
	}
	return textarea.Blink
}

func (f *fileScreen) Update(msg tea.Msg) (tuiScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.area.SetWidth(msg.Width - 6)
		f.area.SetHeight(msg.Height - 8)
		if f.searchMode {
			f.search.Width = msg.Width - 20
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			if f.editable {
				f.save()
			}
			return f, nil

		case "ctrl+f":
			f.toggleSearch()
			return f, nil
		}

		if f.searchMode {
			return f.updateSearch(msg)
		}
	}
	var cmd tea.Cmd
	f.area, cmd = f.area.Update(msg)
	return f, cmd
}

// toggleSearch opens the find bar (focusing it, blurring the document)
// or closes it (the reverse), clearing any in-progress query/matches
// each time it closes so reopening it always starts fresh.
func (f *fileScreen) toggleSearch() {
	f.searchMode = !f.searchMode
	if f.searchMode {
		f.area.Blur()
		f.search.Focus()
	} else {
		f.search.Blur()
		f.search.SetValue("")
		f.matches = nil
		f.matchIdx = -1
		f.searchNote = ""
		f.area.Focus()
	}
}

// updateSearch handles keystrokes while the find bar has focus: enter
// jumps to the next match, up jumps to the previous one, anything else
// is ordinary text input that live-recomputes the match list.
func (f *fileScreen) updateSearch(msg tea.KeyMsg) (tuiScreen, tea.Cmd) {
	switch msg.String() {
	case "enter", "down":
		f.jumpMatch(1)
		return f, nil
	case "up", "shift+tab":
		f.jumpMatch(-1)
		return f, nil
	}

	var cmd tea.Cmd
	before := f.search.Value()
	f.search, cmd = f.search.Update(msg)
	if f.search.Value() != before {
		f.recomputeMatches()
	}
	return f, cmd
}

// recomputeMatches rebuilds the match list for the current query
// (case-insensitive) and resets navigation — called on every edit to
// the search box so the "N matches" count always reflects what's typed.
func (f *fileScreen) recomputeMatches() {
	query := f.search.Value()
	f.matchIdx = -1
	if query == "" {
		f.matches = nil
		f.searchNote = ""
		return
	}

	lines := strings.Split(f.area.Value(), "\n")
	lowerQuery := strings.ToLower(query)
	var matches []fileMatch
	for row, line := range lines {
		lowerLine := strings.ToLower(line)
		start := 0
		for start <= len(lowerLine)-len(lowerQuery) {
			idx := strings.Index(lowerLine[start:], lowerQuery)
			if idx == -1 {
				break
			}
			col := start + idx
			matches = append(matches, fileMatch{row: row, col: col})
			start = col + 1
		}
	}
	f.matches = matches

	if len(matches) == 0 {
		f.searchNote = "no matches for " + queryLabel(query)
	} else {
		f.searchNote = ""
	}
}

// jumpMatch moves to the next (dir=1) or previous (dir=-1) match,
// wrapping around either end of the list.
func (f *fileScreen) jumpMatch(dir int) {
	if len(f.matches) == 0 {
		if f.search.Value() != "" {
			f.searchNote = "no matches for " + queryLabel(f.search.Value())
		}
		return
	}
	if f.matchIdx == -1 {
		if dir >= 0 {
			f.matchIdx = 0
		} else {
			f.matchIdx = len(f.matches) - 1
		}
	} else {
		f.matchIdx = (f.matchIdx + dir + len(f.matches)) % len(f.matches)
	}
	f.moveCursorTo(f.matches[f.matchIdx])
	// Bubbles' textarea has no inline text-highlighting API (it's a
	// plain-text edit widget, not a syntax-highlighted viewer) — this
	// status line is the honest substitute: exact match number and
	// line, always visible, so a jump is never silently ambiguous even
	// without colored highlighting of the matched text itself.
	f.searchNote = fmt.Sprintf("match %d/%d — line %d", f.matchIdx+1, len(f.matches), f.matches[f.matchIdx].row+1)
}

// moveCursorTo positions the textarea's cursor at (row, col) using only
// its public API (CursorUp/CursorDown move by row, SetCursor sets the
// column within the current row — there is no direct "jump to row N"
// method), then feeds it a message so its internal repositionView runs
// and the viewport scrolls to follow the cursor, exactly as it would
// after any real keystroke.
func (f *fileScreen) moveCursorTo(m fileMatch) {
	for f.area.Line() > m.row {
		f.area.CursorUp()
	}
	for f.area.Line() < m.row {
		f.area.CursorDown()
	}
	f.area.SetCursor(m.col)
	f.area, _ = f.area.Update(fileviewRepositionMsg{})
}

// fileviewRepositionMsg is an otherwise-meaningless message whose only
// purpose is to make textarea.Model.Update run its trailing
// repositionView() step — see moveCursorTo.
type fileviewRepositionMsg struct{}

func queryLabel(q string) string {
	return "\"" + q + "\""
}

func (f *fileScreen) save() {
	content := f.area.Value()
	if err := documents.ValidateTextFormat(f.path, content); err != nil {
		f.status, f.statusErr = "Not saved: "+err.Error(), true
		return
	}
	if err := os.WriteFile(f.path, []byte(content), 0644); err != nil {
		f.status, f.statusErr = "Not saved: "+err.Error(), true
		return
	}
	f.status, f.statusErr = "Saved: "+f.path, false
}

func (f *fileScreen) View() string {
	body := tuiHeader(f.title) + "\n"
	if f.loadErr != nil {
		body += tuiWarnStyle.Render("Could not read "+f.path+": "+f.loadErr.Error()) + "\n\n"
	}
	body += f.area.View()

	if f.searchMode {
		body += "\n\n" + tuiHelpStyle.Render("Find: ") + f.search.View()
		if len(f.matches) > 0 {
			body += "  " + tuiOKStyle.Render(matchCountLabel(f.matchIdx, len(f.matches)))
		} else if f.searchNote != "" {
			body += "  " + tuiWarnStyle.Render(f.searchNote)
		}
	}

	help := "↑/↓/←/→: move · ctrl+f: find · esc: back"
	if f.searchMode {
		help = "enter/↓: next match · ↑: previous match · ctrl+f: close find · esc: back"
	} else if f.editable {
		help = "ctrl+s: save · " + help
	} else {
		help += " · (read-only)"
	}
	body += tuiFooter(help)

	if f.status != "" {
		style := tuiOKStyle
		if f.statusErr {
			style = tuiErrStyle
		}
		body += "\n" + style.Render(f.status)
	}
	return tuiDocStyle.Render(body)
}

// matchCountLabel renders "N/M matches" (1-indexed) or "M matches —
// press enter" when nothing has been jumped to yet.
func matchCountLabel(idx, total int) string {
	if idx < 0 {
		if total == 1 {
			return "1 match — press enter"
		}
		return strconv.Itoa(total) + " matches — press enter"
	}
	return strconv.Itoa(idx+1) + "/" + strconv.Itoa(total) + " matches"
}
