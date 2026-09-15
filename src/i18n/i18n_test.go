// i18n_test.go — fallback behavior, placeholder interpolation, and
// hot-reload (see i18n.go/i18n_reload.go).
package i18n

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestRoot(t *testing.T) string {
	root := t.TempDir()
	langDir := filepath.Join(root, "config", "lang")
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(langDir, "en.json"), `{"cli":{"hello":"Hello, {{name}}!"},"only_en":{"key":"english only"}}`)
	writeFile(t, filepath.Join(langDir, "es.json"), `{"cli":{"hello":"¡Hola, {{name}}!"}}`)
	writeFile(t, filepath.Join(langDir, "lang_active.json"), `{"lang":"es"}`)
	return root
}

func writeFile(t *testing.T, path, content string) {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestT_ActiveLanguageAndPlaceholders(t *testing.T) {
	root := setupTestRoot(t)
	if err := Init(root); err != nil {
		t.Fatal(err)
	}
	// Give the initial reload a moment (Init already calls it synchronously,
	// but be defensive against future async changes).
	time.Sleep(10 * time.Millisecond)

	got := T("cli.hello", map[string]any{"name": "Ana"})
	if got != "¡Hola, Ana!" {
		t.Fatalf("expected Spanish translation with placeholder, got %q", got)
	}
}

func TestT_FallsBackToEnglishWhenKeyMissing(t *testing.T) {
	root := setupTestRoot(t)
	if err := Init(root); err != nil {
		t.Fatal(err)
	}
	got := T("only_en.key")
	if got != "english only" {
		t.Fatalf("expected fallback to English, got %q", got)
	}
}

func TestT_ReturnsKeyItselfWhenNowhereFound(t *testing.T) {
	root := setupTestRoot(t)
	if err := Init(root); err != nil {
		t.Fatal(err)
	}
	got := T("does.not.exist")
	if got != "does.not.exist" {
		t.Fatalf("expected the literal key back, got %q", got)
	}
}

func TestHotReload_SwitchesLanguageWithoutRestart(t *testing.T) {
	root := setupTestRoot(t)
	if err := Init(root); err != nil {
		t.Fatal(err)
	}
	if got := T("cli.hello", map[string]any{"name": "Ana"}); got != "¡Hola, Ana!" {
		t.Fatalf("expected Spanish before switch, got %q", got)
	}

	// Simulate an operator editing lang_active.json while the process
	// keeps running - no restart, no re-Init call.
	writeFile(t, filepath.Join(root, "config", "lang", "lang_active.json"), `{"lang":"en"}`)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if ActiveLanguage() == "en" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	got := T("cli.hello", map[string]any{"name": "Ana"})
	if got != "Hello, Ana!" {
		t.Fatalf("expected English after hot-reload, got %q (active=%s)", got, ActiveLanguage())
	}
}
