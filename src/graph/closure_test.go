package graph

import (
	"os"
	"path/filepath"
	"testing"

	"mova.local/core"
	"mova.local/core/focus/astfilter"
)

func closureRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	if err := astfilter.Init(repo); err != nil {
		t.Fatal(err)
	}
	write := func(rel, c string) {
		p := filepath.Join(repo, rel)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("src/tarifas.js", "const { tarifaPlanaV1 } = require('./legacy');\nconst { log } = require('./util');\nfunction calcularTarifa(r) { log(r); return r.v1 ? tarifaPlanaV1(r) : 1; }\nmodule.exports = { calcularTarifa };\n")
	write("src/legacy.js", "function tarifaPlanaV1(r) { return 2; }\nmodule.exports = { tarifaPlanaV1 };\n")
	write("src/util.js", "function log(x) { return x; }\nmodule.exports = { log };\n")
	return repo
}

// A focused symbol calling an explicitly excluded symbol is a conflict;
// a callee in the repo but outside the spec is reported as "outside".
func TestClosure_ExcludedDependencyIsConflict(t *testing.T) {
	repo := closureRepo(t)
	r := Closure(repo, "t", "block", []string{"src/tarifas.js::func=calcularTarifa"}, []string{"src/legacy.js"}, nil)
	if !r.Checked || len(r.Conflicts) != 1 || r.Conflicts[0].To != "src/legacy.js::tarifaPlanaV1" || !r.Blocking() {
		t.Fatalf("conflict not detected: %+v", r)
	}
	if len(r.Outside) != 1 || r.Outside[0].To != "src/util.js::log" {
		t.Fatalf("outside dependency not reported: %+v", r.Outside)
	}
	acc := Closure(repo, "t", "block", []string{"src/tarifas.js::func=calcularTarifa"}, []string{"src/legacy.js"},
		[]core.AcceptedDependency{{Symbol: "tarifaPlanaV1", Reason: "catálogo v1 retirado"}})
	if acc.Blocking() || len(acc.Accepted) != 1 || acc.Accepted[0].Reason == "" {
		t.Fatalf("accept_missing not honored: %+v", acc)
	}
	if w := Closure(repo, "t", "warn", []string{"src/tarifas.js::func=calcularTarifa"}, []string{"src/legacy.js"}, nil); w.Blocking() || len(w.Conflicts) != 1 {
		t.Fatalf("warn must report but not block: %+v", w)
	}
}
