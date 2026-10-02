package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTask_GraphTarget(t *testing.T) {
	cases := map[string]struct {
		spec string
		on   bool
	}{
		`{"graph":"grafo.png"}`:   {"grafo.png", true},
		`{"graph":"  a\\b.svg "}`: {"a\\b.svg", true},
		`{"graph":true}`:          {DefaultGraphFile, true},
		`{"graph":false}`:         {"", false},
		`{"graph":""}`:            {"", false},
		`{"graph":null}`:          {"", false},
		`{"graph":5}`:             {"", false},
		`{}`:                      {"", false},
	}
	for in, want := range cases {
		var task Task
		if err := json.Unmarshal([]byte(in), &task); err != nil {
			t.Fatal(err)
		}
		if spec, on := task.GraphTarget(); spec != want.spec || on != want.on {
			t.Errorf("%s -> (%q,%v), want (%q,%v)", in, spec, on, want.spec, want.on)
		}
	}
}

func TestBuildContextSections_GraphJobForEveryTaskWithGraph(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "projects", "gp")
	_ = os.MkdirAll(dir, 0o755)
	cfg := `{"project":"gp","repo":".","lang":"es","default_task":"b","focus":["global.js"],
	"tasks":{
	  "a":{"prompt":"x","focus":["a.js::func=f"],"exclude":["skip.js"],"graph":"a.png"},
	  "b":{"prompt":"x","graph":true},
	  "c":{"prompt":"x","graph":""},
	  "d":{"prompt":"x"}}}`
	if err := os.WriteFile(filepath.Join(dir, "project.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	var got GraphRequest
	GraphHook = func(r GraphRequest) string { got = r; return "ok" }
	t.Cleanup(func() { GraphHook = nil })

	// Tarea por defecto (b): SOLO se genera el grafo de esa tarea.
	s, err := BuildContextSections(NewFileAdapter(root), root, "gp", "")
	if err != nil {
		t.Fatal(err)
	}
	if s.GraphStatus != "ok" || len(got.Jobs) != 1 {
		t.Fatalf("una tarea → un solo grafo: status=%q jobs=%+v", s.GraphStatus, got.Jobs)
	}
	if b := got.Jobs[0]; b.Task != "b" || b.OutSpec != DefaultGraphFile || len(b.Focus) != 1 || b.Focus[0] != "global.js" { // hereda el focus global
		t.Errorf("job activo = %+v", b)
	}
	// Todas las tareas: un grafo por cada tarea con "graph", orden por nombre.
	if _, err := BuildContextSections(NewFileAdapter(root), root, "gp", TaskAll); err != nil {
		t.Fatal(err)
	}
	if len(got.Jobs) != 2 {
		t.Fatalf("todas las tareas → 2 grafos, got %+v", got.Jobs)
	}
	a2 := got.Jobs[0]
	if a2.Task != "a" || a2.OutSpec != "a.png" || a2.Focus[0] != "a.js::func=f" || a2.Exclude[0] != "skip.js" { // su propio focus/exclude
		t.Errorf("job a = %+v", a2)
	}
}
