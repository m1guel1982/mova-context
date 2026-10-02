// logger_test.go — regression coverage for Open()'s file.path
// resolution now using documents.IsAbsCrossPlatform/NormalizeAbsPath
// (the same rule project.json's "repo" field uses) instead of the
// plain, OS-native filepath.IsAbs it used before.
package logging

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeLoggingConfig(t *testing.T, root, path string) {
	t.Helper()
	dir := filepath.Join(root, "config", "log")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"enabled": true, "file": {"path": "` + path + `", "auto_create": true}}`
	if err := os.WriteFile(filepath.Join(dir, "logging.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestOpen_SampleValue_ResolvesUnderRoot: the shipped default,
// "logs/mova.log", is a bare relative path and resolves under root —
// per requirement 3, exactly as it does today.
func TestOpen_SampleValue_ResolvesUnderRoot(t *testing.T) {
	root := t.TempDir()
	writeLoggingConfig(t, root, "logs/mova.log")

	l := Open(root)
	want := filepath.Join(root, "logs", "mova.log")
	if l.path != want {
		t.Fatalf("got %q, want %q", l.path, want)
	}
}

// TestOpen_NoConfigFile_DefaultsUnderRoot: missing logging.json entirely
// -> still "<root>/logs/mova.log" (LoadConfig's own DefaultConfig).
func TestOpen_NoConfigFile_DefaultsUnderRoot(t *testing.T) {
	root := t.TempDir()

	l := Open(root)
	want := filepath.Join(root, "logs", "mova.log")
	if l.path != want {
		t.Fatalf("got %q, want %q", l.path, want)
	}
}

// TestOpen_BlankPath_DefaultsUnderRoot: file.path present but "" ->
// same default, per requirement 3's explicit blank-path rule.
func TestOpen_BlankPath_DefaultsUnderRoot(t *testing.T) {
	root := t.TempDir()
	writeLoggingConfig(t, root, "")

	l := Open(root)
	want := filepath.Join(root, "logs", "mova.log")
	if l.path != want {
		t.Fatalf("got %q, want %q", l.path, want)
	}
}

// TestOpen_UnixAbsolutePath_UsedAsIs: a genuine Unix absolute path is
// honored exactly, same convention as "repo".
func TestOpen_UnixAbsolutePath_UsedAsIs(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(t.TempDir(), "somewhere-else", "mova.log")
	writeLoggingConfig(t, root, filepath.ToSlash(abs))

	l := Open(root)
	if l.path != filepath.Clean(abs) {
		t.Fatalf("got %q, want the absolute path %q unchanged", l.path, abs)
	}
}

// TestOpen_WindowsDriveLetter_CrossPlatformDetection: a Windows-style
// absolute path is recognized as absolute regardless of host OS
// (documents.IsAbsCrossPlatform) — on Windows it's honored as-is; on
// any other OS it falls back to root-relative instead of being
// silently mis-joined (the bug this fix closes: filepath.IsAbs alone
// would return false for "C:\logs\mova.log" on Linux/macOS, so the
// OLD code joined it as "<root>/C:\logs\mova.log", a broken path).
func TestOpen_WindowsDriveLetter_CrossPlatformDetection(t *testing.T) {
	root := t.TempDir()
	writeLoggingConfig(t, root, `C:\\logs\\mova.log`)

	l := Open(root)
	if runtime.GOOS == "windows" {
		if !filepath.IsAbs(l.path) {
			t.Fatalf("on windows, expected an absolute path, got %q", l.path)
		}
	} else {
		want := filepath.Join(root, "logs", "mova.log") // never a bogus "<root>/C:\..." join
		if l.path != want {
			t.Fatalf("got %q, want the root-relative fallback %q (never a broken join)", l.path, want)
		}
	}
}
