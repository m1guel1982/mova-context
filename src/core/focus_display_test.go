// focus_display_test.go — cubre el mensaje "[Focus] Selected ..." que
// `mova chat`/chat_completion (MCP/HTTP) imprimen: antes contaba
// marcadores "FOCUS:" en el texto ya renderizado (confundiendo targets
// con archivos, y sin distinguir file/dir ni listar nombres — ver
// evidencia.jpg). FormatFocusSelection/FocusDisplayLimit reemplazan esa
// cuenta por datos reales (focus.FocusItem).
package core

import (
	"strings"
	"testing"

	"mova.local/core/focus"
)

func TestFormatFocusSelection_Empty(t *testing.T) {
	if got := FormatFocusSelection(nil, 2); got != "" {
		t.Fatalf("expected empty string for no items, got %q", got)
	}
}

func TestFormatFocusSelection_FilesOnly_WithinLimit(t *testing.T) {
	items := []focus.FocusItem{
		{Name: "server.js", Kind: "file", Files: 1},
		{Name: "backend-test.py", Kind: "file", Files: 1},
	}
	got := FormatFocusSelection(items, 2)

	if !strings.Contains(got, "Selected 2 files") {
		t.Fatalf("expected the line to say \"Selected 2 files\", got %q", got)
	}
	if !strings.Contains(got, "server.js") || !strings.Contains(got, "backend-test.py") {
		t.Fatalf("expected both file names listed within the limit, got %q", got)
	}
	if strings.Contains(got, "+") {
		t.Fatalf("expected no \"+N\" badge when everything fits within the limit, got %q", got)
	}
}

func TestFormatFocusSelection_CollapsesBeyondLimit(t *testing.T) {
	items := []focus.FocusItem{
		{Name: "a.go", Kind: "file", Files: 1},
		{Name: "b.go", Kind: "file", Files: 1},
		{Name: "c.go", Kind: "file", Files: 1},
		{Name: "d.go", Kind: "file", Files: 1},
	}
	got := FormatFocusSelection(items, 2)

	if !strings.Contains(got, "a.go") || !strings.Contains(got, "b.go") {
		t.Fatalf("expected the first 2 names to be listed, got %q", got)
	}
	if strings.Contains(got, "c.go") || strings.Contains(got, "d.go") {
		t.Fatalf("expected names beyond the limit to be collapsed, not listed, got %q", got)
	}
	if !strings.Contains(got, "+2") {
		t.Fatalf("expected a \"+2\" badge for the 2 collapsed names, got %q", got)
	}
}

func TestFormatFocusSelection_RespectsConfiguredLimit(t *testing.T) {
	items := []focus.FocusItem{
		{Name: "a.go", Kind: "file", Files: 1},
		{Name: "b.go", Kind: "file", Files: 1},
		{Name: "c.go", Kind: "file", Files: 1},
		{Name: "d.go", Kind: "file", Files: 1},
		{Name: "e.go", Kind: "file", Files: 1},
	}
	// project.json configuró 4 — con 5 items, el 5to se colapsa igual.
	got := FormatFocusSelection(items, 4)

	for _, name := range []string{"a.go", "b.go", "c.go", "d.go"} {
		if !strings.Contains(got, name) {
			t.Fatalf("expected %q within the configured limit of 4, got %q", name, got)
		}
	}
	if strings.Contains(got, "e.go") {
		t.Fatalf("expected e.go to be collapsed beyond the configured limit of 4, got %q", got)
	}
	if !strings.Contains(got, "+1") {
		t.Fatalf("expected a \"+1\" badge, got %q", got)
	}
}

func TestFormatFocusSelection_MixedFilesAndDirs(t *testing.T) {
	items := []focus.FocusItem{
		{Name: "src", Kind: "dir", Files: 12},
		{Name: "server.js", Kind: "file", Files: 1},
	}
	got := FormatFocusSelection(items, 2)

	if !strings.Contains(got, "item(s)") {
		t.Fatalf("expected a mixed file+dir selection to use the generic \"item(s)\" label, got %q", got)
	}
	if !strings.Contains(got, "13 file(s) total") {
		t.Fatalf("expected the total file count (12+1=13) to be reported, got %q", got)
	}
}

func TestFocusDisplayLimit_DefaultsToTwo(t *testing.T) {
	if got := FocusDisplayLimit(nil); got != DefaultFocusDisplayLimit {
		t.Fatalf("expected default limit %d for nil project, got %d", DefaultFocusDisplayLimit, got)
	}
	if got := FocusDisplayLimit(&Project{}); got != DefaultFocusDisplayLimit {
		t.Fatalf("expected default limit %d when project.json omits focus_display_limit, got %d", DefaultFocusDisplayLimit, got)
	}
}

func TestFocusDisplayLimit_HonorsProjectConfig(t *testing.T) {
	proj := &Project{FocusDisplayLimit: 4}
	if got := FocusDisplayLimit(proj); got != 4 {
		t.Fatalf("expected configured focus_display_limit=4 to win, got %d", got)
	}
}
