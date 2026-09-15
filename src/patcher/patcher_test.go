// patcher_test.go — whole-file and symbol-level apply behavior (see
// patcher.go).
package patcher

import (
	"os"
	"path/filepath"
	"testing"

	"mova.local/documents"
)

func TestApplyBlocks_WholeFile(t *testing.T) {
	dir := t.TempDir()
	blocks := []documents.LabeledCodeBlock{
		{Path: "server.js", Content: "console.log('hi')"},
	}
	applied, err := ApplyBlocks(dir, blocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(applied) != 1 || !applied[0].WholeFile {
		t.Fatalf("expected one whole-file apply, got %+v", applied)
	}
	data, err := os.ReadFile(filepath.Join(dir, "server.js"))
	if err != nil || string(data) != "console.log('hi')" {
		t.Fatalf("file content mismatch: %v %q", err, data)
	}
}

func TestApplyBlocks_SymbolLevelPatchGo(t *testing.T) {
	dir := t.TempDir()
	original := "package auth\n\nfunc Login(u string) error {\n\treturn nil\n}\n\nfunc Other() {}\n"
	path := filepath.Join(dir, "login.go")
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	blocks := []documents.LabeledCodeBlock{
		{Path: "login.go", Symbol: "Login", Content: "func Login(u string) error {\n\treturn fmt.Errorf(\"patched\")\n}"},
	}
	applied, err := ApplyBlocks(dir, blocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(applied) != 1 || applied[0].WholeFile {
		t.Fatalf("expected a symbol-level patch, not a whole-file rewrite: %+v", applied)
	}

	data, _ := os.ReadFile(path)
	content := string(data)
	if !contains(content, "patched") {
		t.Fatalf("expected patched body, got: %s", content)
	}
	if !contains(content, "func Other() {}") {
		t.Fatalf("expected untouched sibling function to survive, got: %s", content)
	}
}

func TestApplyBlocks_SymbolNotFoundFallsBackToWholeFile(t *testing.T) {
	dir := t.TempDir()
	blocks := []documents.LabeledCodeBlock{
		{Path: "new.go", Symbol: "DoesNotExistYet", Content: "func DoesNotExistYet() {}"},
	}
	applied, err := ApplyBlocks(dir, blocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(applied) != 1 || !applied[0].WholeFile {
		t.Fatalf("expected fallback to whole-file write, got %+v", applied)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
