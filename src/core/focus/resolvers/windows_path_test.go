// windows_path_test.go — regression tests for a real reported bug:
// a project.json "repo" using an absolute Windows path with a drive
// letter (e.g. "E:\nuevosProyectos21012026Mova\misProyectos\
// mova_plataforma") could make focus/exclude resolution silently
// break due to drive-letter case mismatches between the path as
// written in project.json and paths returned by Go's filesystem
// APIs. See comparableHostPath (fsutil.go) and relOrBase's own fix.
package resolvers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mova.local/core/focus"
	"mova.local/core/focus/astfilter"
)

func TestComparableHostPath_FoldsOnlyDriveLetterCase(t *testing.T) {
	cases := []struct{ a, b string }{
		{`E:\nuevosProyectos21012026Mova\misProyectos\mova_plataforma`, `e:\nuevosProyectos21012026Mova\misProyectos\mova_plataforma`},
		{`C:\mova-context`, `c:\mova-context`},
	}
	for _, c := range cases {
		if comparableHostPath(c.a) != comparableHostPath(c.b) {
			t.Errorf("comparableHostPath(%q)=%q vs comparableHostPath(%q)=%q - drive letter case must not matter",
				c.a, comparableHostPath(c.a), c.b, comparableHostPath(c.b))
		}
	}
	// The REST of the path (beyond the drive letter) must stay
	// case-sensitive - Windows itself is insensitive there too, but
	// folding the whole string risks silently merging two genuinely
	// different names on a case-sensitive network share.
	if comparableHostPath(`E:\Foo`) == comparableHostPath(`E:\foo`) {
		t.Errorf("comparableHostPath must only fold the drive letter, not the rest of the path")
	}
}

func TestRelOrBase_RepoRootAlwaysResolvesToDot(t *testing.T) {
	// The exact reported symptom: a Windows absolute repo path caused
	// the focus label for the repo root itself to come out as ".."
	// instead of ".". Same drive letter, different case - must still
	// resolve to ".".
	root := `E:\nuevosProyectos21012026Mova\misProyectos\mova_plataforma`
	path := `e:\nuevosProyectos21012026Mova\misProyectos\mova_plataforma`
	if got := relOrBase(root, path); got != "." {
		t.Errorf("relOrBase(root, root-with-different-drive-case) = %q, want \".\"", got)
	}
}

func TestExcludeMatcher_BareNameMatchesUnderWindowsDriveRepo(t *testing.T) {
	repo := `E:\nuevosProyectos21012026Mova\misProyectos\mova_plataforma`
	m := newExcludeMatcher(repo, []string{"node_modules", ".git", "*.pyc"})
	if m == nil {
		t.Fatal("expected a non-nil matcher")
	}
	// This is what actually prunes an entire subtree during a real
	// walk (see skipDirOrExcluded, called once per directory AS the
	// walker encounters it, returning filepath.SkipDir before ever
	// descending) - a bare name match never depends on the repo's
	// absolute path shape, only on the directory's own name.
	if !m.excludesName("node_modules") {
		t.Error("bare name exclusion must work regardless of the repo's absolute path shape")
	}
	// excludesPath is the OTHER check (a single already-resolved file
	// path, e.g. an explicit "focus": ["some/file.py"] target) - it
	// matches by the file's OWN base name or by a configured
	// path/glob, not by any ancestor directory's name (that is
	// excludesName's job, applied per-directory during the walk).
	pycFile := repo + `\mova_ollama\build\cache.pyc`
	if !m.excludesPath(pycFile) {
		t.Error("a *.pyc file must be excluded by its own glob match regardless of the repo's absolute path shape")
	}
}

func TestExcludeMatcher_RepoRelativePatternSurvivesDriveLetterCaseMismatch(t *testing.T) {
	repo := `E:\nuevosProyectos21012026Mova\misProyectos\mova_plataforma`
	m := newExcludeMatcher(repo, []string{`mova_files/private`})
	if m == nil {
		t.Fatal("expected a non-nil matcher")
	}
	// Same location, but the path handed to excludesPath (as if
	// resolved via a slightly different code path/case) uses a
	// lowercase drive letter.
	target := `e:\nuevosProyectos21012026Mova\misProyectos\mova_plataforma\mova_files\private\secret.env`
	if !m.excludesPath(target) {
		t.Error("a repo-relative exclude pattern must match even if the drive letter's case differs between repo and the resolved path")
	}
}

// TestAstSymbolResolver_AbsolutePathTarget is a regression test for a
// real bug found in QA while writing END_TO_END_EXAMPLE.md's
// multi-drive scenario: "file::kind=name" targets whose FILE part was
// an absolute host path (a different Windows drive, a UNC network
// share, or any Unix absolute path outside repo) always fell straight
// through to repoRelativePath (see ast_symbol.go's Resolve), which
// strips what looks like an absolute prefix and re-joins under
// RepoPath — so it could only ever resolve to a nonexistent path
// UNDER the repo, never the real external file. The target was
// reported as "not found" even though the file plainly existed.
// AstSymbolResolver now tries resolveAbsoluteFile FIRST, exactly like
// FileResolver.candidatePath already did — this test would have
// caught the bug before it shipped.
func TestAstSymbolResolver_AbsolutePathTarget(t *testing.T) {
	dir := t.TempDir()
	repoDir := filepath.Join(dir, "repo")
	externalDir := filepath.Join(dir, "external-drive")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(externalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	externalFile := filepath.Join(externalDir, "settings.py")
	if err := os.WriteFile(externalFile, []byte("ADMIN_EMAIL = \"support@company.com\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := astfilter.Init(t.TempDir()); err != nil {
		t.Fatalf("astfilter.Init: %v", err)
	}

	r := NewAstSymbolResolver()
	target := externalFile + "::var=ADMIN_EMAIL"
	ctx := focus.Context{RepoPath: repoDir}
	if !r.Match(ctx, target) {
		t.Fatalf("Match(%q) = false, want true", target)
	}
	blocks, err := r.Resolve(ctx, target)
	if err != nil {
		t.Fatalf("Resolve(%q): %v (this is the bug: an absolute-path AST target must resolve, not report ErrNotFound)", target, err)
	}
	if len(blocks) != 1 || !strings.Contains(blocks[0].Content, "ADMIN_EMAIL") {
		t.Fatalf("Resolve(%q) = %+v, want a block containing ADMIN_EMAIL", target, blocks)
	}
}
