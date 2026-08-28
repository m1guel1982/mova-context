// render_test.go — la deduplicación de párrafos en sí misma (texto
// idéntico normalizado, casos límite) se prueba en mova.local/dedup, que
// es donde vive esa lógica ahora. Acá se prueba el punto de integración
// específico de este paquete: RenderFocusContextWithSeen debe compartir
// el mapa "seen" con lo que el llamador ya vio ANTES de invocar focus —
// exactamente lo que core.BuildContextSections necesita para deduplicar
// AGENTS+SKILLS+PROMPT contra FOCUS, no solo FOCUS contra sí mismo.
package resolvers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixtureRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "Shared warning paragraph that repeats elsewhere.\n\nUnique repo content."
	if err := os.WriteFile(filepath.Join(repo, "notes.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestRenderFocusContextWithSeen_DedupsAgainstExternalSeen(t *testing.T) {
	root := writeFixtureRepo(t)

	// Simula que "Shared warning paragraph..." YA se emitió antes (p.ej.
	// en un agent) — antes de siquiera llamar a focus.
	externalSeen := map[string]bool{
		"Shared warning paragraph that repeats elsewhere.": true,
	}

	text, stats := RenderFocusContextWithSeen(root, "repo", []string{"notes.md"}, nil, nil, externalSeen)

	if strings.Contains(text, "Shared warning paragraph") {
		t.Fatalf("expected the paragraph already in externalSeen to be removed from focus output, got:\n%s", text)
	}
	if !strings.Contains(text, "Unique repo content.") {
		t.Fatalf("expected the non-duplicate paragraph to still be present, got:\n%s", text)
	}
	if stats.DuplicatesRemoved != 1 {
		t.Fatalf("expected DuplicatesRemoved=1, got %d", stats.DuplicatesRemoved)
	}
}

func TestRenderFocusContext_StillWorksWithoutExternalSeen(t *testing.T) {
	root := writeFixtureRepo(t)

	// RenderFocusContext (sin "WithSeen") no debe cambiar de
	// comportamiento — sigue creando su propio seen interno, como
	// siempre.
	text, _ := RenderFocusContext(root, "repo", []string{"notes.md"}, nil, nil)
	if !strings.Contains(text, "Shared warning paragraph") {
		t.Fatalf("expected RenderFocusContext (no external seen) to include all content normally, got:\n%s", text)
	}
}
