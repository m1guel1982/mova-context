// contextcache_project_test.go — cubre la actualización en caliente
// pedida para contextcache.go: apenas project.json cambia (se
// agrega/saca algo de "focus", se toca cualquier otro campo),
// mova-context-cache.json se invalida de inmediato en la siguiente
// llamada a SanitizeCached, sin esperar a que el TEXTO de Focus/Memory
// difiera por sí solo.
package budget

import (
	"os"
	"path/filepath"
	"testing"

	"mova.local/core"
	"mova.local/sanitize"
)

func writeProjectJSON(t *testing.T, root, project, contents string) {
	t.Helper()
	dir := filepath.Join(root, "projects", project)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "project.json"), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidateOnProjectChange_NoProjectJSON_IsNoop(t *testing.T) {
	root := t.TempDir()
	f := contextCacheFile{Entries: map[string]contextCacheEntry{"focus": {Hash: "x"}}}

	if invalidateOnProjectChange(root, "does-not-exist", &f) {
		t.Fatalf("expected no-op (false) when project.json doesn't exist")
	}
	if len(f.Entries) != 1 {
		t.Fatalf("expected entries untouched when there's nothing to watch, got %+v", f.Entries)
	}
}

func TestInvalidateOnProjectChange_FirstRun_RecordsHashWithoutLoss(t *testing.T) {
	root := t.TempDir()
	writeProjectJSON(t, root, "demo", `{"repo":"."}`)
	f := contextCacheFile{Entries: map[string]contextCacheEntry{}}

	changed := invalidateOnProjectChange(root, "demo", &f)

	if !changed {
		t.Fatalf("expected the very first observation of project.json to report changed=true (so ProjectHash gets persisted)")
	}
	if f.ProjectHash == "" {
		t.Fatalf("expected ProjectHash to be recorded after the first run")
	}
}

func TestInvalidateOnProjectChange_UnchangedProjectJSON_NoInvalidation(t *testing.T) {
	root := t.TempDir()
	writeProjectJSON(t, root, "demo", `{"repo":"."}`)
	f := contextCacheFile{Entries: map[string]contextCacheEntry{}}
	invalidateOnProjectChange(root, "demo", &f) // primera pasada: fija ProjectHash

	f.Entries["focus"] = contextCacheEntry{Hash: "keepme"}
	changed := invalidateOnProjectChange(root, "demo", &f)

	if changed {
		t.Fatalf("expected no invalidation when project.json didn't change between runs")
	}
	if _, ok := f.Entries["focus"]; !ok {
		t.Fatalf("expected the existing entry to survive when project.json is unchanged")
	}
}

// TestInvalidateOnProjectChange_FocusEdited_WipesStaleEntries es el caso
// central pedido: "al agregar o sacar algo del focus" — cualquier
// edición de project.json, sin importar si el TEXTO de Focus ya
// resuelto cambió, invalida las entradas de inmediato.
func TestInvalidateOnProjectChange_FocusEdited_WipesStaleEntries(t *testing.T) {
	root := t.TempDir()
	writeProjectJSON(t, root, "demo", `{"repo":".","focus":["a.go"]}`)
	f := contextCacheFile{Entries: map[string]contextCacheEntry{}}
	invalidateOnProjectChange(root, "demo", &f)
	f.Entries["focus"] = contextCacheEntry{Hash: "stale", SanitizedText: "old focus text"}

	// El usuario agrega un segundo archivo a "focus".
	writeProjectJSON(t, root, "demo", `{"repo":".","focus":["a.go","b.go"]}`)

	changed := invalidateOnProjectChange(root, "demo", &f)

	if !changed {
		t.Fatalf("expected editing project.json's focus list to be detected as a change")
	}
	if len(f.Entries) != 0 {
		t.Fatalf("expected every cached entry to be wiped after a project.json edit, got %+v", f.Entries)
	}
}

// TestSanitizeCached_HotInvalidation_EndToEnd ejercita el flujo
// completo: SanitizeCached escribe mova-context-cache.json, se edita
// project.json, y la siguiente llamada a SanitizeCached descarta la
// entrada vieja en vez de servir el texto sanitizado bajo la
// configuración anterior.
func TestSanitizeCached_HotInvalidation_EndToEnd(t *testing.T) {
	root := t.TempDir()
	writeProjectJSON(t, root, "demo", `{"repo":"."}`)
	cfg := sanitize.Config{Enabled: true, StripComments: true}

	sections := &core.ContextSections{Focus: "// comentario\ncontenido real"}
	SanitizeCached(root, "demo", sections, cfg, true)

	cachePath := ContextCachePath(root, "demo")
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("expected mova-context-cache.json to be written, got err=%v", err)
	}

	// Cambia project.json (p. ej. se agrega una config nueva) — sin
	// tocar el texto de Focus en sí.
	writeProjectJSON(t, root, "demo", `{"repo":".","focus_display_limit":4}`)

	before := loadContextCacheFile(cachePath)
	if len(before.Entries) == 0 {
		t.Fatalf("expected a cached entry to exist before the second run")
	}

	sections2 := &core.ContextSections{Focus: "// comentario\ncontenido real"}
	SanitizeCached(root, "demo", sections2, cfg, true)

	after := loadContextCacheFile(cachePath)
	if after.ProjectHash == before.ProjectHash {
		t.Fatalf("expected ProjectHash to move forward after project.json changed")
	}
}
