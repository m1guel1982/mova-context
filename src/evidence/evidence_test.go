package evidence

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStart_WriteOnceAndIsolated(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "runs")
	a, err := Start(dir, Manifest{Door: "cli:run", Project: "p", Task: "t", Decision: Decision{Outcome: "released"}}, []byte("ctx"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := Start(dir, Manifest{Door: "cli:run", Project: "p", Task: "u"}, []byte("ctx"))
	if err != nil || a.Dir == b.Dir {
		t.Fatalf("second run must get its own directory: %v %v", a.Dir, b.Dir)
	}
	if _, err := Start(dir, Manifest{RunID: a.ID}, []byte("otro")); !errors.Is(err, ErrRunExists) {
		t.Fatalf("rewriting a run must fail, got %v", err)
	}
	m, err := ReadManifest(a.Dir)
	if err != nil || m.Context.SHA256 != SHA256([]byte("ctx")) || m.Perimeter == "" || m.Schema != SchemaVersion {
		t.Fatalf("manifest: %+v %v", m, err)
	}
	ma, _ := ReadManifest(a.Dir)
	mb, _ := ReadManifest(b.Dir)
	if ma.Context.SHA256 != mb.Context.SHA256 {
		t.Fatal("same bytes must hash the same")
	}
	_ = a.Event("read", map[string]any{"path": "x"})
	_ = a.Event("read", map[string]any{"path": "y"})
	ev, _ := os.ReadFile(filepath.Join(a.Dir, "events.jsonl"))
	if strings.Count(string(ev), "\n") != 2 {
		t.Fatalf("events must append: %s", ev)
	}
}
