package documents

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsAbsCrossPlatform(t *testing.T) {
	cases := map[string]bool{
		"/home/user/carpeta":   true,
		"/etc/config":          true,
		"C:/carpeta/archivo":   true,
		"C:\\carpeta\\archivo": true,
		"D:/otros":             true,
		"\\\\server\\share":    true,
		"carpeta":              false,
		"salida/reportes":      false,
		"":                     false,
	}
	for path, want := range cases {
		if got := IsAbsCrossPlatform(path); got != want {
			t.Errorf("IsAbsCrossPlatform(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestResolveDirectoryPathEmptyDefaultsToRepo(t *testing.T) {
	root := t.TempDir()
	resolved, ambiguous, err := ResolveDirectoryPath(root, "mi-repo", "")
	if err != nil {
		t.Fatalf("ResolveDirectoryPath: %v", err)
	}
	if ambiguous != nil {
		t.Fatalf("expected no ambiguity, got: %v", ambiguous)
	}
	want := filepath.Join(root, "mi-repo")
	if resolved != want {
		t.Errorf("resolved = %q, want %q", resolved, want)
	}
	if _, err := os.Stat(resolved); err != nil {
		t.Errorf("expected repo dir to be created: %v", err)
	}
}

func TestResolveDirectoryPathUnixAbsolute(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "otra-carpeta") // valid absolute path on this host
	resolved, ambiguous, err := ResolveDirectoryPath(root, "mi-repo", abs)
	if err != nil {
		t.Fatalf("ResolveDirectoryPath: %v", err)
	}
	if ambiguous != nil {
		t.Fatalf("expected no ambiguity, got: %v", ambiguous)
	}
	if resolved != filepath.Clean(abs) {
		t.Errorf("resolved = %q, want %q", resolved, abs)
	}
}

func TestResolveDirectoryPathWindowsStyleOnNonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this case only applies on non-Windows hosts")
	}
	root := t.TempDir()
	_, _, err := ResolveDirectoryPath(root, "mi-repo", "C:/carpeta/archivo.txt")
	if err == nil {
		t.Fatal("expected an error for a Windows-style path on a non-Windows host, got nil")
	}
}

func TestResolveDirectoryPathExplicitRelativePath(t *testing.T) {
	root := t.TempDir()
	resolved, ambiguous, err := ResolveDirectoryPath(root, "mi-repo", "salida/reportes")
	if err != nil {
		t.Fatalf("ResolveDirectoryPath: %v", err)
	}
	if ambiguous != nil {
		t.Fatalf("expected no ambiguity, got: %v", ambiguous)
	}
	want := filepath.Join(root, "mi-repo", "salida", "reportes")
	if resolved != want {
		t.Errorf("resolved = %q, want %q", resolved, want)
	}
}

func TestResolveDirectoryPathBareNameNoMatchCreatesNew(t *testing.T) {
	root := t.TempDir()
	repoDir := filepath.Join(root, "mi-repo")
	os.MkdirAll(repoDir, 0o755)

	resolved, ambiguous, err := ResolveDirectoryPath(root, "mi-repo", "nueva-carpeta")
	if err != nil {
		t.Fatalf("ResolveDirectoryPath: %v", err)
	}
	if ambiguous != nil {
		t.Fatalf("expected no ambiguity, got: %v", ambiguous)
	}
	want := filepath.Join(repoDir, "nueva-carpeta")
	if resolved != want {
		t.Errorf("resolved = %q, want %q", resolved, want)
	}
}

func TestResolveDirectoryPathBareNameSingleMatchReusesIt(t *testing.T) {
	root := t.TempDir()
	repoDir := filepath.Join(root, "mi-repo")
	existing := filepath.Join(repoDir, "src", "config")
	os.MkdirAll(existing, 0o755)

	resolved, ambiguous, err := ResolveDirectoryPath(root, "mi-repo", "config")
	if err != nil {
		t.Fatalf("ResolveDirectoryPath: %v", err)
	}
	if ambiguous != nil {
		t.Fatalf("expected no ambiguity, got: %v", ambiguous)
	}
	if resolved != existing {
		t.Errorf("resolved = %q, want %q (should reuse the existing match)", resolved, existing)
	}
}

func TestResolveDirectoryPathBareNameMultipleMatchesAsksWhich(t *testing.T) {
	root := t.TempDir()
	repoDir := filepath.Join(root, "mi-repo")
	first := filepath.Join(repoDir, "src", "config")
	second := filepath.Join(repoDir, "tools", "config")
	os.MkdirAll(first, 0o755)
	os.MkdirAll(second, 0o755)

	resolved, ambiguous, err := ResolveDirectoryPath(root, "mi-repo", "config")
	if err != nil {
		t.Fatalf("ResolveDirectoryPath: %v", err)
	}
	if resolved != "" {
		t.Errorf("expected empty resolved path when ambiguous, got %q", resolved)
	}
	if len(ambiguous) != 2 {
		t.Fatalf("expected 2 ambiguous candidates, got %d: %v", len(ambiguous), ambiguous)
	}
}

func TestResolveFilePathBareFilenameGoesToRepoRoot(t *testing.T) {
	root := t.TempDir()
	resolved, ambiguous, err := ResolveFilePath(root, "mi-repo", "notas.md")
	if err != nil {
		t.Fatalf("ResolveFilePath: %v", err)
	}
	if ambiguous != nil {
		t.Fatalf("expected no ambiguity, got: %v", ambiguous)
	}
	want := filepath.Join(root, "mi-repo", "notas.md")
	if resolved != want {
		t.Errorf("resolved = %q, want %q", resolved, want)
	}
}

func TestResolveFilePathExplicitMultiSegmentPath(t *testing.T) {
	root := t.TempDir()
	resolved, ambiguous, err := ResolveFilePath(root, "mi-repo", "salida/reportes/informe.md")
	if err != nil {
		t.Fatalf("ResolveFilePath: %v", err)
	}
	if ambiguous != nil {
		t.Fatalf("expected no ambiguity, got: %v", ambiguous)
	}
	want := filepath.Join(root, "mi-repo", "salida", "reportes", "informe.md")
	if resolved != want {
		t.Errorf("resolved = %q, want %q", resolved, want)
	}
}

func TestResolveFilePathBareDirNoMatchCreatesNew(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "mi-repo"), 0o755)
	resolved, ambiguous, err := ResolveFilePath(root, "mi-repo", "salida/archivo.txt")
	if err != nil {
		t.Fatalf("ResolveFilePath: %v", err)
	}
	if ambiguous != nil {
		t.Fatalf("expected no ambiguity, got: %v", ambiguous)
	}
	want := filepath.Join(root, "mi-repo", "salida", "archivo.txt")
	if resolved != want {
		t.Errorf("resolved = %q, want %q", resolved, want)
	}
}

func TestResolveFilePathBareDirSingleMatchReusesIt(t *testing.T) {
	root := t.TempDir()
	repoDir := filepath.Join(root, "mi-repo")
	existingDir := filepath.Join(repoDir, "src", "config")
	os.MkdirAll(existingDir, 0o755)

	resolved, ambiguous, err := ResolveFilePath(root, "mi-repo", "config/ajustes.json")
	if err != nil {
		t.Fatalf("ResolveFilePath: %v", err)
	}
	if ambiguous != nil {
		t.Fatalf("expected no ambiguity, got: %v", ambiguous)
	}
	want := filepath.Join(existingDir, "ajustes.json")
	if resolved != want {
		t.Errorf("resolved = %q, want %q (should place the file inside the existing match)", resolved, want)
	}
}

func TestResolveFilePathBareDirMultipleMatchesAsksWhich(t *testing.T) {
	root := t.TempDir()
	repoDir := filepath.Join(root, "mi-repo")
	os.MkdirAll(filepath.Join(repoDir, "src", "config"), 0o755)
	os.MkdirAll(filepath.Join(repoDir, "herramientas", "config"), 0o755)

	resolved, ambiguous, err := ResolveFilePath(root, "mi-repo", "config/ajustes.json")
	if err != nil {
		t.Fatalf("ResolveFilePath: %v", err)
	}
	if resolved != "" {
		t.Errorf("expected empty resolved path when ambiguous, got %q", resolved)
	}
	if len(ambiguous) != 2 {
		t.Fatalf("expected 2 ambiguous candidates, got %d: %v", len(ambiguous), ambiguous)
	}
}

func TestResolveFilePathRejectsEmpty(t *testing.T) {
	root := t.TempDir()
	if _, _, err := ResolveFilePath(root, "mi-repo", ""); err == nil {
		t.Fatal("expected error for empty filename, got nil")
	}
}

func TestCreateDirectoryRecursive(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "a", "b", "c")
	if err := CreateDirectory(target); err != nil {
		t.Fatalf("CreateDirectory: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		t.Fatalf("expected %q to exist as a directory, err=%v", target, err)
	}
}

func TestCreateDirectoryIdempotent(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "ya-existe")
	os.MkdirAll(target, 0o755)
	if err := CreateDirectory(target); err != nil {
		t.Fatalf("expected no error re-creating an existing directory, got: %v", err)
	}
}
