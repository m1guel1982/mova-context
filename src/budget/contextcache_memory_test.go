package budget

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mova.local/core"
	"mova.local/sanitize"
)

// memory.md cambia con cada tarea: el caché debe servir el contexto nuevo (por hash de texto),
// sin devolver NUNCA una memoria vieja.
func TestSanitizeCached_MemoryChangeIsAlwaysSeen(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "projects", "p"), 0o755)
	cfg := sanitize.DefaultConfig()
	focus := "\n\n---\n## FOCUS\nFOCUS: a.js\nconst a = 1\n"
	s1 := &core.ContextSections{Focus: focus, Memory: "\n\n---\n## MEMORY\nmemoria v1\n"}
	SanitizeCached(root, "p", s1, cfg, true)
	f1, _ := os.ReadFile(ContextCachePath(root, "p"))
	if !strings.Contains(string(f1), "memoria v1") {
		t.Fatalf("cache should hold the sanitized memory:\n%s", f1)
	}
	s2 := &core.ContextSections{Focus: focus, Memory: "\n\n---\n## MEMORY\nmemoria v2 nueva\n"}
	SanitizeCached(root, "p", s2, cfg, true)
	if !strings.Contains(s2.Memory, "memoria v2 nueva") || strings.Contains(s2.Memory, "memoria v1") {
		t.Fatalf("stale memory served from cache: %q", s2.Memory)
	}
	f2, _ := os.ReadFile(ContextCachePath(root, "p"))
	if strings.Contains(string(f2), "memoria v1") || !strings.Contains(string(f2), "memoria v2 nueva") {
		t.Fatalf("cache must be updated in place:\n%s", f2)
	}
}

// Con PII activo el caché (que guarda texto antes del enmascarado) no se usa y el viejo se borra.
func TestUseContextCache_PIIMaskingDisablesAndWipesCache(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "projects", "p"), 0o755)
	path := ContextCachePath(root, "p")
	_ = os.WriteFile(path, []byte(`{"memory":{"sanitized_text":"correo juan@acme.com"}}`), 0o644)
	var off, on core.BudgetConfig
	_ = json.Unmarshal([]byte(`{}`), &off)
	_ = json.Unmarshal([]byte(`{"pii_masking":{"enabled":true}}`), &on)
	if !useContextCache(root, "p", &off) {
		t.Fatal("cache should be usable without PII masking")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("cache untouched when PII masking is off")
	}
	if useContextCache(root, "p", &on) {
		t.Fatal("cache must be disabled when pii_masking is on")
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("stale unmasked cache must be deleted")
	}
}
