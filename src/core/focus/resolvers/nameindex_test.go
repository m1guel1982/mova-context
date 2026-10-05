package resolvers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mova.local/core/focus"
	"mova.local/core/focus/astfilter"
)

// fixtureRepo: "Gantt.js" existe en dos portales, más un archivo único.
func fixtureRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"portal-a/www/main/schedule/Gantt.js": "function show() { return 'a'; }\nfunction other() {}\n",
		"portal-b/www/main/schedule/Gantt.js": "function show() { return 'b'; }\n",
		"portal-a/www/main/Unico.js":              "function solo() {}\n",
	}
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestFileResolver_BareNameReturnsEveryMatch(t *testing.T) {
	root := fixtureRepo(t)
	ctx := focus.Context{RepoPath: root, Index: &focus.FileIndex{}}
	blocks, err := NewFileResolver().Resolve(ctx, "Gantt.js")
	if err != nil || len(blocks) != 2 {
		t.Fatalf("want 2 blocks (one per portal), got %d, err=%v", len(blocks), err)
	}
}

func TestFileResolver_PartialPathNarrowsMatches(t *testing.T) {
	root := fixtureRepo(t)
	ctx := focus.Context{RepoPath: root, Index: &focus.FileIndex{}}
	blocks, err := NewFileResolver().Resolve(ctx, `portal-a\www\main\schedule\Gantt.js`)
	if err != nil || len(blocks) != 1 || !strings.Contains(filepath.ToSlash(blocks[0].Source), "portal-a") {
		t.Fatalf("want only the portal-a file, got %+v err=%v", blocks, err)
	}
	blocks, _ = NewFileResolver().Resolve(ctx, "main/schedule/gantt.js") // sufijo, sin distinguir mayúsculas
	if len(blocks) != 2 {
		t.Fatalf("trailing-segments match must find both portals, got %d", len(blocks))
	}
}

func TestAstSymbol_BareFileNameWithFunc(t *testing.T) {
	if err := astfilter.Init(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	root := fixtureRepo(t)
	ctx := focus.Context{RepoPath: root, Index: &focus.FileIndex{}}
	blocks, err := NewAstSymbolResolver().Resolve(ctx, "Gantt.js::func=show()")
	if err != nil || len(blocks) != 2 {
		t.Fatalf("want show() from both Gantt.js, got %d err=%v", len(blocks), err)
	}
	for _, b := range blocks {
		if strings.Contains(b.Content, "other") {
			t.Errorf("only show() must be extracted, got %q", b.Content)
		}
	}
}

func TestExclude_BareAndPartialNames(t *testing.T) {
	root := fixtureRepo(t)
	// nombre suelto: excluye Gantt.js en TODAS las carpetas
	ctx := focus.Context{RepoPath: root, Exclude: []string{"Gantt.js"}, Index: &focus.FileIndex{}}
	if _, err := NewFileResolver().Resolve(ctx, "Gantt.js"); err == nil {
		t.Fatal("bare-name exclude must hide every Gantt.js")
	}
	// ruta parcial: solo la de portal-b
	ctx = focus.Context{RepoPath: root, Exclude: []string{"portal-b/www/main/schedule/Gantt.js"}, Index: &focus.FileIndex{}}
	blocks, err := NewFileResolver().Resolve(ctx, "Gantt.js")
	if err != nil || len(blocks) != 1 || !strings.Contains(filepath.ToSlash(blocks[0].Source), "portal-a") {
		t.Fatalf("partial-path exclude must leave only portal-a, got %+v err=%v", blocks, err)
	}
}

func TestIndex_BuiltOnce(t *testing.T) {
	root := fixtureRepo(t)
	stats := &focus.ScanStats{}
	ctx := focus.Context{RepoPath: root, Index: &focus.FileIndex{}, Stats: stats}
	for _, target := range []string{"Gantt.js", "Unico.js", "main/Unico.js"} {
		if !NewFileResolver().Match(ctx, target) {
			t.Fatalf("%s must match", target)
		}
	}
	if got := stats.FilesScanned(); got != 3 {
		t.Fatalf("index must walk the repo once (3 files), scanned=%d", got)
	}
}

func TestLocateTarget_StripsSymbolSuffix(t *testing.T) {
	root := fixtureRepo(t)
	ctx := focus.Context{RepoPath: root, Index: &focus.FileIndex{}}
	if got := LocateTarget(ctx, "Gantt.js::func=show(),other"); len(got) != 2 {
		t.Fatalf("want 2 paths, got %v", got)
	}
}
