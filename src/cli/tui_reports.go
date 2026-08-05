// tui_reports.go — a small directory-listing screen, reused for a
// project's reports/ (job "save" output — see jobs/actions_save.go)
// and memory-archive/ (job "memory_archive" output — see
// jobs/actions_memory.go). Selecting a file opens it read-only in the
// generic fileScreen (tui_fileview.go) — reports are generated
// artifacts, not something the TUI edits.
package main

import (
	"os"
	"path/filepath"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
)

func newDirListScreen(app *tuiApp, title, dir string) tuiScreen {
	entries, err := os.ReadDir(dir)
	var items []menuItem
	if err != nil {
		items = append(items, menuItem{title: "(empty)", desc: "No files in " + dir + " yet"})
		return newMenuScreen(title, items, "")
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names))) // más recientes primero (nombres con fecha)

	for _, name := range names {
		full := filepath.Join(dir, name)
		items = append(items, menuItem{
			title: name, desc: full,
			onSelect: func() tea.Cmd { return tuiPush(newFileScreen(app, name, full, false)) },
		})
	}
	if len(items) == 0 {
		items = append(items, menuItem{title: "(empty)", desc: "No files in " + dir + " yet"})
	}
	return newMenuScreen(title, items, "")
}
