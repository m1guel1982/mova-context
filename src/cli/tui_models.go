// tui_models.go — lists every config/models/**/*.json file (provider
// configs plus active.json) and opens the chosen one in the generic
// fileScreen (tui_fileview.go). Mova resolves providers/models purely
// from these files already (see models/config.go) — the TUI only adds
// a browsable menu in front of the same directory, no new config format.
package main

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func newModelsMenu(app *tuiApp) tuiScreen {
	root := filepath.Join(app.root, "config", "models")
	files := walkJSONFiles(root)
	sort.Strings(files)

	var items []menuItem
	for _, full := range files {
		full := full
		rel := strings.TrimPrefix(strings.TrimPrefix(full, root), string(filepath.Separator))
		items = append(items, menuItem{
			title: rel, desc: full,
			onSelect: func() tea.Cmd { return tuiPush(newFileScreen(app, rel, full, true)) },
		})
	}
	if len(items) == 0 {
		items = append(items, menuItem{title: "(no configurations)", desc: "No .json files found in config/models/"})
	}
	return newMenuScreen("Modelos", items, "")
}

// walkJSONFiles lists every *.json file under dir, recursively.
func walkJSONFiles(dir string) []string {
	var out []string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".json") {
			out = append(out, path)
		}
		return nil
	})
	return out
}
