package graph

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mova.local/core"
	"mova.local/core/focus/astfilter"
	"mova.local/i18n"
)

func initEnv(t *testing.T) {
	t.Helper()
	root := filepath.Join("..", "..")
	if err := i18n.Init(root); err != nil {
		t.Fatal(err)
	}
	if err := astfilter.Init(root); err != nil {
		t.Fatal(err)
	}
}

func writeRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

var cycleRepo = map[string]string{
	"lib/a.js": "const B = require('./b');\nconst LIMIT = 10;\nfunction alfa() { return B.beta() + gamma() + LIMIT; }\nfunction gamma() { return 1; }\n",
	"lib/b.js": "const A = require('./a');\nfunction beta() { return A.alfa(); }\n",
	"lib/c.js": "class C { run() { return this.helper() + alfa(); } helper() { return 2; } }\n",
}

func TestBuild_CallsRefsImportsCyclesExcludeAndInference(t *testing.T) {
	initEnv(t)
	dir := writeRepo(t, cycleRepo)
	req := spec{RepoPath: dir, ProjectName: "p", TaskName: "t", Lang: "es",
		Focus:   []string{`lib\a.js::func=alfa`, "b.js::func=beta", "c.js::func=run,missing"}, // ruta con \, nombre suelto y símbolo inexistente
		Exclude: []string{"a.js::func=gamma"}}
	g, st := build(req, locate(req))

	has := func(from, to, kind string, inferred bool) bool {
		for _, e := range g.Edges {
			if e.From == from && e.To == to && e.Kind == kind && e.Inferred == inferred {
				return true
			}
		}
		return false
	}
	for _, c := range []struct {
		from, to, kind string
		inf            bool
	}{
		{"lib/a.js#alfa", "lib/b.js#beta", "call", false},  // B.beta(): binding require
		{"lib/b.js#beta", "lib/a.js#alfa", "call", false},  // ciclo a<->b
		{"lib/a.js#alfa", "lib/a.js#gamma", "call", false}, // llamada local
		{"lib/a.js#alfa", "lib/a.js#LIMIT", "ref", false},  // referencia a constante
		{"lib/c.js#run", "lib/c.js#helper", "call", false}, // this.helper()
		{"lib/c.js#run", "lib/a.js#alfa", "call", true},    // sin import: nombre único
		{"lib/a.js", "lib/b.js", "import", false},
		{"lib/b.js", "lib/a.js", "import", false},
	} {
		if !has(c.from, c.to, c.kind, c.inf) {
			t.Errorf("falta arista %s -> %s (%s, inferred=%v)\nedges=%+v", c.from, c.to, c.kind, c.inf, g.Edges)
		}
	}
	style := map[string]string{}
	for _, n := range g.Nodes {
		style[n.ID] = n.Style
	}
	if style["lib/a.js#gamma"] != "excluded" || style["lib/a.js#alfa"] != "focus" || style["lib/a.js#LIMIT"] != "outside" || style["lib/c.js#helper"] != "outside" {
		t.Errorf("estilos inesperados: %v", style)
	}
	for _, n := range g.Nodes {
		if strings.Contains(n.ID, "#B") || strings.Contains(n.ID, "#A") { // require-bindings no son nodos
			t.Errorf("un binding de require se dibujó como nodo: %s", n.ID)
		}
	}
	if len(g.Clusters) != 3 {
		t.Errorf("clusters = %d, want 3", len(g.Clusters))
	}
	if len(st.warnings) != 1 || !strings.Contains(st.warnings[0], "missing") {
		t.Errorf("warnings = %v, want aviso por 'missing'", st.warnings)
	}
}

func TestResolveOutput(t *testing.T) {
	initEnv(t)
	pd := t.TempDir()
	abs := filepath.Join(t.TempDir(), "x", "g.svg")
	cases := []struct{ spec, want string }{
		{"grafo.png", filepath.Join(pd, "grafo.png")},
		{"sub\\dir\\g.pdf", filepath.Join(pd, "sub", "dir", "g.pdf")},
		{"grafo", filepath.Join(pd, "grafo.png")},
		{"salida/", filepath.Join(pd, "salida", "graph.png")},
		{abs, abs},
	}
	for _, c := range cases {
		got, msg := resolveOutput(pd, c.spec, "es", "t")
		if msg != "" || got != c.want {
			t.Errorf("resolveOutput(%q) = %q (%q), want %q", c.spec, got, msg, c.want)
		}
	}
	if _, msg := resolveOutput(pd, "g.jpg", "es", "t"); !strings.Contains(msg, ".jpg") {
		t.Errorf("extensión inválida debe avisar, got %q", msg)
	}
	if _, msg := resolveOutput(pd, `C:\graficos\g.png`, "es", "t"); msg == "" && filepath.Separator == '/' {
		t.Error("ruta Windows en Linux debe rechazarse con un error claro")
	}
}

func jobReq(root, repo, lang string, jobs ...core.GraphJob) core.GraphRequest {
	return core.GraphRequest{Root: root, RepoPath: repo, ProjectName: "p", Lang: lang, Jobs: jobs}
}

func outDir(root string) string { return filepath.Dir(core.ProjectJSONPath(root, "p")) }

func TestGenerate_FormatsHotReloadAndCache(t *testing.T) {
	initEnv(t)
	repo := writeRepo(t, cycleRepo)
	root := t.TempDir()
	job := core.GraphJob{Task: "t", OutSpec: "g.png", Focus: []string{"a.js::func=alfa"}}
	out := filepath.Join(outDir(root), "g.png")

	s1 := generate(jobReq(root, repo, "en", job))
	first, err := os.ReadFile(out)
	if err != nil || !bytes.HasPrefix(first, []byte("\x89PNG")) || !strings.Contains(s1, "graph generated") {
		t.Fatalf("PNG no generado: err=%v status=%q", err, s1)
	}
	st1, _ := os.Stat(out)
	if s2 := generate(jobReq(root, repo, "en", job)); s2 != s1 {
		t.Errorf("sin cambios: mismo estado, got %q", s2)
	}
	if st2, _ := os.Stat(out); !st2.ModTime().Equal(st1.ModTime()) {
		t.Error("sin cambios no debe regenerar el archivo")
	}

	job.Focus = []string{"a.js::func=alfa", "b.js::func=beta"} // cambio en caliente de project.json
	generate(jobReq(root, repo, "en", job))
	if second, _ := os.ReadFile(out); bytes.Equal(first, second) {
		t.Error("cambiar focus debe regenerar un grafo distinto")
	}
	if s := generate(jobReq(root, repo, "es", job)); !strings.Contains(s, "grafo generado") {
		t.Errorf("lang del proyecto debe traducir el estado, got %q", s)
	}
	entries, _ := os.ReadDir(outDir(root))
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("escritura atómica: quedó un temporal: %s", e.Name())
		}
	}

	for spec, magic := range map[string]string{"g.svg": "<svg", "g.pdf": "%PDF"} {
		j := job
		j.OutSpec = spec
		generate(jobReq(root, repo, "en", j))
		data, err := os.ReadFile(filepath.Join(outDir(root), spec))
		if err != nil || !bytes.HasPrefix(data, []byte(magic)) {
			t.Errorf("%s: err=%v, prefijo %q", spec, err, magic)
		}
	}
}

func TestGenerate_EveryTaskWithGraphAndDuplicateOutput(t *testing.T) {
	initEnv(t)
	repo := writeRepo(t, cycleRepo)
	root := t.TempDir()
	st := generate(jobReq(root, repo, "en",
		core.GraphJob{Task: "analizar", OutSpec: "analizar.png", Focus: []string{"a.js::func=alfa"}},
		core.GraphJob{Task: "columnas", OutSpec: "columnas.svg", Focus: []string{"b.js::func=beta"}},
		core.GraphJob{Task: "repetida", OutSpec: "analizar.png", Focus: []string{"c.js::func=run"}}))
	for _, f := range []string{"analizar.png", "columnas.svg"} {
		if _, err := os.Stat(filepath.Join(outDir(root), f)); err != nil {
			t.Errorf("falta el grafo %s: %v", f, err)
		}
	}
	for _, want := range []string{"analizar:", "columnas:", `task "analizar" already writes`} {
		if !strings.Contains(st, want) {
			t.Errorf("estado sin %q:\n%s", want, st)
		}
	}
}

func TestGenerate_AsyncReturnsImmediatelyThenNotifiesAndFastPath(t *testing.T) {
	initEnv(t)
	core.GraphAsync = true
	done := make(chan string, 4)
	core.GraphNotify = func(m string) { done <- m }
	t.Cleanup(func() { core.GraphAsync, core.GraphNotify = false, nil })

	repo := writeRepo(t, cycleRepo)
	root := t.TempDir()
	req := jobReq(root, repo, "en", core.GraphJob{Task: "t", OutSpec: "g.png", Focus: []string{"a.js::func=alfa", "b.js::func=beta", "c.js::func=run"}})

	t0 := time.Now()
	st := generate(req)
	if d := time.Since(t0); d > 150*time.Millisecond || !strings.Contains(st, "generating in the background") {
		t.Fatalf("en segundo plano debe volver al instante con aviso: %v %q", d, st)
	}
	select {
	case m := <-done:
		if !strings.Contains(m, "graph generated") {
			t.Fatalf("aviso de término inesperado: %q", m)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("el grafo en segundo plano no terminó")
	}
	if _, err := os.Stat(filepath.Join(outDir(root), "g.png")); err != nil {
		t.Fatal(err)
	}
	// camino rápido: con todo igual, cientos de llamadas (un turno de chat cada una) casi no cuestan
	t0 = time.Now()
	for i := 0; i < 300; i++ {
		if s := generate(req); !strings.Contains(s, "graph generated") {
			t.Fatalf("estado en caché inesperado: %q", s)
		}
	}
	if d := time.Since(t0); d > 300*time.Millisecond {
		t.Errorf("300 llamadas sin cambios tardaron %v; el camino rápido debe ser solo stat", d)
	}
	select {
	case m := <-done:
		t.Errorf("sin cambios no debe haber nuevos avisos: %q", m)
	default:
	}
}

func TestGenerate_EmptyDoesNotWrite(t *testing.T) {
	initEnv(t)
	root := t.TempDir()
	st := generate(jobReq(root, t.TempDir(), "en", core.GraphJob{Task: "t", OutSpec: "g.png", Focus: []string{"nada.js::func=x"}}))
	if !strings.Contains(st, "nothing to graph") {
		t.Errorf("status = %q", st)
	}
	if _, err := os.Stat(filepath.Join(outDir(root), "g.png")); err == nil {
		t.Error("sin símbolos no se escribe archivo")
	}
}

func TestGenerate_PersistentCacheSurvivesNewProcess(t *testing.T) {
	initEnv(t)
	repo := writeRepo(t, cycleRepo)
	root := t.TempDir()
	job := core.GraphJob{Task: "t", OutSpec: "g.png", Focus: []string{"a.js::func=alfa"}}
	out := filepath.Join(outDir(root), "g.png")
	generate(jobReq(root, repo, "en", job))
	if _, err := os.Stat(filepath.Join(outDir(root), diskFile)); err != nil {
		t.Fatalf("falta %s: %v", diskFile, err)
	}
	before, _ := os.Stat(out)

	mu.Lock() // proceso nuevo: memoria vacía, disco intacto
	cache, loaded = map[string]*entry{}, map[string]bool{}
	mu.Unlock()
	time.Sleep(20 * time.Millisecond)
	if st := generate(jobReq(root, repo, "en", job)); !strings.Contains(st, "graph generated") {
		t.Fatalf("status = %q", st)
	}
	if after, _ := os.Stat(out); !after.ModTime().Equal(before.ModTime()) {
		t.Error("arranque con fuentes sin cambios no debe volver a renderizar")
	}

	mu.Lock()
	cache, loaded = map[string]*entry{}, map[string]bool{}
	mu.Unlock()
	if err := os.WriteFile(filepath.Join(repo, "lib", "a.js"), []byte(cycleRepo["lib/a.js"]+"function nueva() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	generate(jobReq(root, repo, "en", job))
	if after, _ := os.Stat(out); after.ModTime().Equal(before.ModTime()) {
		t.Error("un archivo fuente modificado debe regenerar el grafo")
	}
}
