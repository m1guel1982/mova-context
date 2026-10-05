package applyflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mova.local/core"
)

const planJS = `'use strict';
function normalizarPedido(p) {
  return { id: p.id };
}

function otra() {
  return 2;
}
`

const cobrosJS = `'use strict';
function generarCobro(plan) {
  return { llegada: plan.pedido.etaProgramada };
}
module.exports = { generarCobro };
`

const reply = "Propuesta.\n\n```javascript:src/plan.js::normalizarPedido()\nfunction normalizarPedido(p) {\n  return { id: p.id, etaReal: p.etaReal };\n}\n```\n\n```javascript:src/cobros.js::generarCobro()\nfunction generarCobro(plan) {\n  return { llegada: plan.pedido.etaReal || plan.pedido.etaProgramada };\n}\n```\n\n```javascript:src/legacy.js\nmodule.exports = 1;\n```\n\n```javascript:src/nuevo.js\nmodule.exports = {};\n```\n"

func fixture(t *testing.T, applyField string) (root string, proj *core.Project) {
	t.Helper()
	root = t.TempDir()
	repo := filepath.Join(root, "repo")
	for rel, body := range map[string]string{"src/plan.js": planJS, "src/cobros.js": cobrosJS, "src/legacy.js": "old\n"} {
		_ = os.MkdirAll(filepath.Dir(filepath.Join(repo, rel)), 0o755)
		_ = os.WriteFile(filepath.Join(repo, rel), []byte(body), 0o644)
	}
	dir := filepath.Join(root, "projects", "p")
	_ = os.MkdirAll(dir, 0o755)
	pj := `{"project":"p","repo":` + jsonStr(repo) + `,"lang":"es","adapter":"file","default_task":"t",` + applyField + `
	  "tasks":{"t":{"prompt":"x","exclude":["src/legacy.js"]}}}`
	_ = os.WriteFile(filepath.Join(dir, "project.json"), []byte(pj), 0o644)
	var err error
	proj, err = core.NewFileAdapter(root).GetProject("p")
	if err != nil {
		t.Fatal(err)
	}
	return root, proj
}

func jsonStr(s string) string { b, _ := json.Marshal(s); return string(b) }

func read(t *testing.T, p string) string { b, _ := os.ReadFile(p); return string(b) }

func TestApplyDisabledByDefaultBehavesAsBefore(t *testing.T) {
	for _, f := range []string{``, `"apply":false,`, `"apply":{"enabled":false},`} {
		root, proj := fixture(t, f)
		if Build(root, "p", "t", proj, reply) != nil || Offer(root, "p", "t", proj, reply) != "" {
			t.Fatalf("%q: apply off must not propose anything", f)
		}
	}
}

func TestProposalListsAllFilesAndAsksBeforeWriting(t *testing.T) {
	root, proj := fixture(t, `"apply":true,`)
	p := Build(root, "p", "t", proj, reply)
	if p == nil || len(p.Changes) != 4 || p.Files() != 4 {
		t.Fatalf("want 4 changes/4 files, got %+v", p)
	}
	txt := p.Render(false)
	for _, must := range []string{"4 cambio(s)", "src/plan.js::normalizarPedido()", "src/cobros.js::generarCobro()", "src/nuevo.js", "MODIFICAR", "CREAR", "Respaldo activado", "[s]", "[n]", "el archivo está en exclude"} {
		if !strings.Contains(txt, must) {
			t.Fatalf("question text lacks %q:\n%s", must, txt)
		}
	}
	if read(t, filepath.Join(p.Repo, "src/plan.js")) != planJS {
		t.Fatal("asking must not write anything")
	}
}

// Responder «sí»: se modifican TODOS los archivos involucrados (menos lo excluido), con respaldo al lado.
func TestAnswerYesModifiesAllInvolvedFilesWithBackupNextToThem(t *testing.T) {
	root, proj := fixture(t, `"apply":true,`)
	p := Build(root, "p", "t", proj, reply)
	text, wrote := p.Answer(root, "sí")
	if !wrote || !strings.Contains(text, "3 aplicado(s), 1 omitido(s)") {
		t.Fatalf("text:\n%s", text)
	}
	if got := read(t, filepath.Join(p.Repo, "src/plan.js")); !strings.Contains(got, "etaReal: p.etaReal") || !strings.Contains(got, "function otra()") {
		t.Fatalf("plan.js:\n%s", got)
	}
	if got := read(t, filepath.Join(p.Repo, "src/cobros.js")); !strings.Contains(got, "etaReal ||") || !strings.Contains(got, "module.exports") {
		t.Fatalf("cobros.js:\n%s", got)
	}
	if read(t, filepath.Join(p.Repo, "src/nuevo.js")) == "" {
		t.Fatal("new file must be created")
	}
	if read(t, filepath.Join(p.Repo, "src/legacy.js")) != "old\n" {
		t.Fatal("an excluded file must never be modified")
	}
	baks, _ := filepath.Glob(filepath.Join(p.Repo, "src", "*.mova-*.bak"))
	if len(baks) != 2 { // plan.js y cobros.js existían; nuevo.js no; legacy.js no se tocó
		t.Fatalf("want 2 backups next to the originals, got %v", baks)
	}
	for _, b := range baks {
		if !strings.HasPrefix(filepath.Base(b), "plan.js.mova-") && !strings.HasPrefix(filepath.Base(b), "cobros.js.mova-") {
			t.Fatalf("unexpected backup name %s", b)
		}
	}
	for _, b := range baks {
		if strings.Contains(filepath.Base(b), "plan.js") && read(t, b) != planJS {
			t.Fatal("backup must hold the ORIGINAL plan.js")
		}
	}
}

func TestAnswerSubsetNoAndInvalid(t *testing.T) {
	root, proj := fixture(t, `"apply":true,`)
	p := Build(root, "p", "t", proj, reply)
	if text, wrote := p.Answer(root, "no"); wrote || !strings.Contains(text, "No se modificó ningún archivo") {
		t.Fatalf("no: %q", text)
	}
	if text, wrote := p.Answer(root, "tal vez"); wrote || !strings.Contains(text, "No entendí") {
		t.Fatalf("invalid must not write: %q", text)
	}
	if _, wrote := p.Answer(root, "9"); wrote {
		t.Fatal("out-of-range number must not write")
	}
	if _, wrote := p.Answer(root, "2"); !wrote {
		t.Fatal("subset must write")
	}
	if !strings.Contains(read(t, filepath.Join(p.Repo, "src/cobros.js")), "etaReal ||") {
		t.Fatal("#2 (cobros.js) must be applied")
	}
	if strings.Contains(read(t, filepath.Join(p.Repo, "src/plan.js")), "etaReal") {
		t.Fatal("#1 was not selected and must stay untouched")
	}
}

func TestBackupFlagFalseInProjectJSON(t *testing.T) {
	root, proj := fixture(t, `"apply":{"enabled":true,"backup":false},`)
	p := Build(root, "p", "t", proj, reply)
	if p.Backup || !strings.Contains(p.Render(false), "Respaldo desactivado") {
		t.Fatal("backup:false must be reflected")
	}
	p.Answer(root, "s")
	if baks, _ := filepath.Glob(filepath.Join(p.Repo, "src", "*.bak")); len(baks) != 0 {
		t.Fatalf("backup:false must not create backups: %v", baks)
	}
	// objeto sin backup = true por defecto
	root2, proj2 := fixture(t, `"apply":{"enabled":true},`)
	if !Build(root2, "p", "t", proj2, reply).Backup {
		t.Fatal("backup defaults to true")
	}
}

func TestTaskLevelApplyOverridesProject(t *testing.T) {
	root, proj := fixture(t, `"apply":true,`)
	if !core.ApplyEnabled(proj, "t") {
		t.Fatal("project-level true")
	}
	proj.Tasks["t"] = core.Task{Apply: json.RawMessage(`false`), Exclude: []string{"src/legacy.js"}}
	if core.ApplyEnabled(proj, "t") || Build(root, "p", "t", proj, reply) != nil {
		t.Fatal("a task with apply:false must be read-only even if the project enables it")
	}
}

// El mismo flujo cuando la confirmación llega en OTRA llamada (MCP / HTTP).
func TestRemoteFlowAcrossCalls(t *testing.T) {
	root, proj := fixture(t, `"apply":true,`)
	offer := Offer(root, "p", "t", proj, reply)
	if !strings.Contains(offer, "Todavía NO") {
		t.Fatalf("remote question must say nothing was changed yet:\n%s", offer)
	}
	repo := filepath.Join(root, "repo")
	if read(t, filepath.Join(repo, "src/plan.js")) != planJS {
		t.Fatal("offering must not write")
	}
	if _, handled := HandleRemote(root, "p", "es", "explícame mejor el cambio uno por favor, no entiendo la lógica", ""); handled {
		t.Fatal("a new question must not be hijacked as an answer")
	}
	if p, _ := Load(root, "p"); p == nil {
		t.Fatal("the proposal must stay pending after an unrelated message")
	}
	text, handled := HandleRemote(root, "p", "es", "1,2", "")
	if !handled || !strings.Contains(text, "2 aplicado(s)") {
		t.Fatalf("answer: %v %q", handled, text)
	}
	if p, _ := Load(root, "p"); p != nil {
		t.Fatal("answered proposal must be cleared")
	}
	if _, handled := HandleRemote(root, "p", "es", "sí", ""); handled {
		t.Fatal("nothing pending: a bare «sí» is an ordinary message")
	}
	if txt, handled := HandleRemote(root, "p", "es", "", "all"); !handled || !strings.Contains(txt, "No hay cambios pendientes") {
		t.Fatalf("explicit apply_changes without pending: %q", txt)
	}
}

func TestPendingExpires(t *testing.T) {
	root, proj := fixture(t, `"apply":true,`)
	p := Build(root, "p", "t", proj, reply)
	p.Created = "2000-01-01T00:00:00Z"
	_ = Save(root, p)
	if q, expired := Load(root, "p"); q != nil || !expired {
		t.Fatalf("expired proposal must be discarded: %v %v", q, expired)
	}
}

// El caso del prompt «agregar-columnas»: pregunta (Sí/No) al modelo, «sí», y la propuesta no trae destinos.
func TestReformatRequestForClassicYesNoPrompt(t *testing.T) {
	_, proj := fixture(t, `"apply":true,`)
	unlabeled := "Cambio propuesto:\n```diff\n- a\n+ b\n```\n**¿Deseas que aplique estas modificaciones directamente en los archivos fuentes? (Sí/No)**"
	rq, ok := ReformatRequest(proj, "t", "Sí", unlabeled)
	if !ok || !strings.Contains(rq, "bloques de código") || !strings.Contains(rq, "NO afirmes") {
		t.Fatalf("expected reformat request, got %v %q", ok, rq)
	}
	for _, c := range []struct{ line, last string }{
		{"explícame más", unlabeled},           // no es confirmación
		{"sí", "Hola, ¿quieres café? (Sí/No)"}, // sin código
		{"sí", reply + "\n(Sí/No)"},            // ya trae bloques con destino
	} {
		if _, ok := ReformatRequest(proj, "t", c.line, c.last); ok {
			t.Fatalf("must not reformat: %+v", c)
		}
	}
	proj.Apply = nil
	if _, ok := ReformatRequest(proj, "t", "sí", unlabeled); ok {
		t.Fatal("apply off → no reformat")
	}
}

func TestParseSelection(t *testing.T) {
	cases := []struct {
		in   string
		kind Answer
		n    int
	}{{"s", AnswerAll, 4}, {"Sí", AnswerAll, 4}, {"todos", AnswerAll, 4}, {"n", AnswerNone, 0}, {"", AnswerNone, 0},
		{"1,3", AnswerSome, 2}, {"#1, #2", AnswerSome, 2}, {"1 4", AnswerSome, 2}, {"0", AnswerInvalid, 0}, {"5", AnswerInvalid, 0}, {"quizá", AnswerInvalid, 0}}
	for _, c := range cases {
		sel, kind := ParseSelection(c.in, 4)
		if kind != c.kind || len(sel) != c.n {
			t.Errorf("%q → kind=%v n=%d, want %v/%d", c.in, kind, len(sel), c.kind, c.n)
		}
	}
}
