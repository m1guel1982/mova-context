package documents

import "testing"

func TestDetectEditIntent(t *testing.T) {
	cases := []struct {
		text  string
		verb  bool
		files []string
	}{
		{"Modifica src/server.js: valida el email en login", true, []string{"src/server.js"}},
		{"Cambia el texto de report.md", true, []string{"report.md"}},
		{"Fix the login bug in auth.go", true, []string{"auth.go"}},
		{"Update server.js to add logging", true, []string{"server.js"}},
		{"Modifica la función login", true, nil},
		{"hola como estas", false, nil},
		{"Genera reporte.pdf", false, nil}, // pure creation verb, no edit verb
	}
	for _, c := range cases {
		got := DetectEditIntent(c.text)
		if got.VerbDetected != c.verb {
			t.Errorf("%q: VerbDetected = %v, want %v", c.text, got.VerbDetected, c.verb)
		}
		if !equalStrSlices(got.Files, c.files) {
			t.Errorf("%q: Files = %v, want %v", c.text, got.Files, c.files)
		}
	}
}

func TestDiffLines(t *testing.T) {
	old := "line1\nline2\nline3"
	new := "line1\nlineTWO\nline3\nline4"
	d := DiffLines(old, new)
	if !d.Changed {
		t.Fatal("expected Changed = true")
	}
	if d.LinesRemoved != 1 || d.LinesAdded != 2 {
		t.Errorf("got +%d/-%d, want +2/-1", d.LinesAdded, d.LinesRemoved)
	}

	same := DiffLines("identical", "identical")
	if same.Changed {
		t.Error("expected Changed = false for identical text")
	}
}

func TestExtractEditedContent(t *testing.T) {
	in := "```go\npackage main\n```"
	got := ExtractEditedContent(in)
	if got != "package main" {
		t.Errorf("got %q, want %q", got, "package main")
	}
	plain := "package main"
	if ExtractEditedContent(plain) != plain {
		t.Errorf("plain content should pass through unchanged")
	}
}
