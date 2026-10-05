package patcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mova.local/documents"
)

const jsClass = `'use strict';
const A = 1;

class Planner {
  async _processOrder(vuelo, opts) {
    const x = { a: 1 };
    return x;
  }

  otra() {
    return 2;
  }
}

const normalizar = (p) => {
  return { id: p.id };
};

module.exports = { Planner, normalizar };
`

func write(t *testing.T, dir, rel, body string) string {
	t.Helper()
	p := filepath.Join(dir, rel)
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// El bug original: símbolo no encontrado en un archivo EXISTENTE sobrescribía todo el archivo con el fragmento.
func TestMissingSymbolNeverWipesExistingFile(t *testing.T) {
	dir := t.TempDir()
	p := write(t, dir, "src/a.js", jsClass)
	applied, skipped, err := ApplyBlocksOpts(dir, []documents.LabeledCodeBlock{
		{Path: "src/a.js", Symbol: "noExiste", Content: "function noExiste() {\n  return 1;\n}"},
	}, Options{})
	if err == nil || len(applied) != 0 || len(skipped) != 1 {
		t.Fatalf("must skip: applied=%v skipped=%v err=%v", applied, skipped, err)
	}
	if got, _ := os.ReadFile(p); string(got) != jsClass {
		t.Fatalf("the file must be untouched, got:\n%s", got)
	}
}

func TestPatchesJSClassMethodAndArrowFunction(t *testing.T) {
	dir := t.TempDir()
	p := write(t, dir, "src/a.js", jsClass)
	_, _, err := ApplyBlocksOpts(dir, []documents.LabeledCodeBlock{
		{Path: "src/a.js", Symbol: "_processOrder", Content: "  async _processOrder(vuelo, opts) {\n    return { ...vuelo, etaReal: vuelo.etaReal };\n  }"},
		{Path: "src/a.js", Symbol: "normalizar", Content: "const normalizar = (p) => {\n  return { id: p.id, etaReal: p.etaReal };\n};"},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	s := string(got)
	for _, must := range []string{"etaReal: vuelo.etaReal", "etaReal: p.etaReal", "otra() {", "module.exports", "const A = 1;"} {
		if !strings.Contains(s, must) {
			t.Fatalf("missing %q in:\n%s", must, s)
		}
	}
	if strings.Contains(s, "const x = { a: 1 }") {
		t.Fatalf("old body must be replaced:\n%s", s)
	}
}

func TestRejectsPathsOutsideRepo(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	_ = os.MkdirAll(repo, 0o755)
	outside := filepath.Join(dir, "secreto.txt")
	for _, p := range []string{"../secreto.txt", outside, "src/../../secreto.txt", `..\secreto.txt`} {
		applied, skipped, err := ApplyBlocksOpts(repo, []documents.LabeledCodeBlock{{Path: p, Content: "x"}}, Options{})
		if err == nil || len(applied) != 0 || len(skipped) != 1 {
			t.Fatalf("%q must be rejected: %v %v", p, applied, err)
		}
	}
	if _, err := os.Stat(outside); err == nil {
		t.Fatal("nothing may be written outside the repo")
	}
}

func TestExcludedTargetsAreNotModified(t *testing.T) {
	dir := t.TempDir()
	p := write(t, dir, "src/legacy.js", "function v1() {\n  return 1;\n}\n")
	ex := func(path, sym string) (bool, string) {
		if path == "src/legacy.js" {
			return true, "excluido por el proyecto"
		}
		return false, ""
	}
	applied, skipped, _ := ApplyBlocksOpts(dir, []documents.LabeledCodeBlock{{Path: "src/legacy.js", Content: "boom"}}, Options{Excluded: ex})
	if len(applied) != 0 || len(skipped) != 1 || !strings.Contains(skipped[0].Reason, "excluido") {
		t.Fatalf("excluded file must be skipped: %v %v", applied, skipped)
	}
	if got, _ := os.ReadFile(p); strings.Contains(string(got), "boom") {
		t.Fatal("excluded file was modified")
	}
}

func TestBackupSitsNextToTheOriginal(t *testing.T) {
	dir := t.TempDir()
	p := write(t, dir, "src/a.js", "viejo\n")
	applied, _, err := ApplyBlocksOpts(dir, []documents.LabeledCodeBlock{
		{Path: "src/a.js", Content: "nuevo 1\n"},
		{Path: "src/a.js", Content: "nuevo 2\n"}, // dos bloques sobre el mismo archivo
	}, Options{Backup: true, Stamp: "20261003-153012"})
	if err != nil || len(applied) != 2 {
		t.Fatalf("applied=%v err=%v", applied, err)
	}
	want := p + ".mova-20261003-153012.bak"
	if applied[0].Backup != want {
		t.Fatalf("backup = %q, want %q (same directory, original name + sigla)", applied[0].Backup, want)
	}
	if b, _ := os.ReadFile(want); string(b) != "viejo\n" {
		t.Fatalf("backup must keep the ORIGINAL even after a 2nd block, got %q", b)
	}
	if got, _ := os.ReadFile(p); string(got) != "nuevo 2\n" {
		t.Fatalf("file content %q", got)
	}
	a2, _, _ := ApplyBlocksOpts(dir, []documents.LabeledCodeBlock{{Path: "src/nuevo.js", Content: "x\n"}}, Options{Backup: true})
	if len(a2) != 1 || a2[0].Backup != "" {
		t.Fatalf("a new file has nothing to back up: %+v", a2)
	}
	// backup desactivado: no se crea nada
	p3 := write(t, dir, "src/b.js", "orig\n")
	ApplyBlocksOpts(dir, []documents.LabeledCodeBlock{{Path: "src/b.js", Content: "z\n"}}, Options{Backup: false, Stamp: "s"})
	if _, err := os.Stat(p3 + ".mova-s.bak"); err == nil {
		t.Fatal("backup:false must not create a backup")
	}
}

func TestUnbalancedBraceReplacementIsRejected(t *testing.T) {
	dir := t.TempDir()
	p := write(t, dir, "src/a.js", jsClass)
	_, skipped, err := ApplyBlocksOpts(dir, []documents.LabeledCodeBlock{
		{Path: "src/a.js", Symbol: "otra", Content: "  otra() {\n    return 3;\n"}, // falta la llave de cierre
	}, Options{})
	if err == nil || len(skipped) != 1 {
		t.Fatalf("truncated block must be rejected: %v", err)
	}
	if got, _ := os.ReadFile(p); string(got) != jsClass {
		t.Fatal("file must be untouched")
	}
}
