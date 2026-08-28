// render_multi_test.go — cubre lo que se corrigió/agregó en esta ronda:
//   - un `focus` con VARIOS targets ("server.js", "backend-test.py")
//     tiene que procesarlos y contarlos TODOS, no solo el primero.
//   - "." recorre TODO el repo recursivamente.
//   - un directorio ("src") se resuelve como su contenido completo.
//   - un target absoluto del host FUERA del repo ("<tmp>/external/…")
//     se resuelve directo, sin tratarlo como relativo al repo.
//   - FocusItem (Kind/Files) distingue "file" de "dir" correctamente,
//     incluso cuando un directorio resuelve a exactamente 1 archivo
//     (caso que una heurística ingenua por cantidad de blocks confunde
//     con "file").
package resolvers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mova.local/core/focus"
)

func writeMultiFixture(t *testing.T) (repo, external string) {
	t.Helper()
	root := t.TempDir()
	repo = filepath.Join(root, "repo")
	external = filepath.Join(root, "external")
	if err := os.MkdirAll(filepath.Join(repo, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(external, 0o755); err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.WriteFile(filepath.Join(repo, "server.js"), []byte("console.log(1)"), 0o644))
	must(os.WriteFile(filepath.Join(repo, "backend-test.py"), []byte("print(1)"), 0o644))
	must(os.WriteFile(filepath.Join(repo, "src", "a.go"), []byte("package a"), 0o644))
	must(os.WriteFile(filepath.Join(external, "archivo.java"), []byte("class X{}"), 0o644))
	return repo, external
}

func TestResolveAll_MultipleFocusTargets_ProcessesEveryOne(t *testing.T) {
	repo, _ := writeMultiFixture(t)

	_, stats := RenderFocusContext(repo, "", []string{"server.js", "backend-test.py"}, nil, nil)

	if len(stats.Items) != 2 {
		t.Fatalf("expected 2 resolved focus items (not stopping at the first), got %d: %+v", len(stats.Items), stats.Items)
	}
	if stats.FilesIncluded != 2 {
		t.Fatalf("expected 2 files included total, got %d", stats.FilesIncluded)
	}
	for _, it := range stats.Items {
		if it.Kind != "file" || it.Files != 1 {
			t.Fatalf("expected each explicit file target to be Kind=file Files=1, got %+v", it)
		}
	}
}

func TestResolveAll_DotRecursesWholeRepo(t *testing.T) {
	repo, _ := writeMultiFixture(t)

	_, stats := RenderFocusContext(repo, "", []string{"."}, nil, nil)

	if stats.FilesIncluded != 3 {
		t.Fatalf("expected \".\" to include every file in the repo (3), got %d", stats.FilesIncluded)
	}
	if len(stats.Items) != 1 || stats.Items[0].Kind != "dir" {
		t.Fatalf("expected \".\" to be reported as a single dir-like item, got %+v", stats.Items)
	}
}

func TestResolveAll_DirectoryTarget_ReportedAsDirEvenWithOneFile(t *testing.T) {
	repo, _ := writeMultiFixture(t)

	// "src" tiene exactamente 1 archivo adentro — una heurística que
	// decide "dir" solo cuando hay >1 block se equivocaría acá.
	_, stats := RenderFocusContext(repo, "", []string{"src"}, nil, nil)

	if len(stats.Items) != 1 {
		t.Fatalf("expected exactly 1 focus item for target \"src\", got %d", len(stats.Items))
	}
	if stats.Items[0].Kind != "dir" {
		t.Fatalf("expected directory target \"src\" (even with a single file inside) to be Kind=dir, got %+v", stats.Items[0])
	}
	if stats.Items[0].Files != 1 {
		t.Fatalf("expected 1 file under src/, got %d", stats.Items[0].Files)
	}
}

func TestResolveAll_AbsoluteHostFile_OutsideRepo(t *testing.T) {
	repo, external := writeMultiFixture(t)
	absFile := filepath.Join(external, "archivo.java")

	text, stats := RenderFocusContext(repo, "", []string{absFile}, nil, nil)

	if stats.FilesIncluded != 1 {
		t.Fatalf("expected the absolute external file to be included, got FilesIncluded=%d", stats.FilesIncluded)
	}
	if len(stats.Items) != 1 || stats.Items[0].Kind != "file" {
		t.Fatalf("expected the absolute file target to resolve as Kind=file, got %+v", stats.Items)
	}
	if !strings.Contains(text, "class X{}") {
		t.Fatalf("expected the external file's content in the rendered focus text, got:\n%s", text)
	}
}

func TestResolveAll_AbsoluteHostDirectory_OutsideRepo(t *testing.T) {
	repo, external := writeMultiFixture(t)

	_, stats := RenderFocusContext(repo, "", []string{external}, nil, nil)

	if stats.FilesIncluded != 1 {
		t.Fatalf("expected the absolute external directory's file to be included, got FilesIncluded=%d", stats.FilesIncluded)
	}
	if len(stats.Items) != 1 || stats.Items[0].Kind != "dir" {
		t.Fatalf("expected the absolute directory target to resolve as Kind=dir, got %+v", stats.Items)
	}
}

// TestResolveAll_LeadingSlash_StaysRepoRelative_WhenNoAbsoluteMatch
// confirma que la convención histórica ("/src" == "<repo>/src") sigue
// intacta cuando NO existe nada absoluto con ese nombre en el host —
// el soporte nuevo de rutas absolutas nunca debe romper project.json
// existentes que ya usaban "/algo" como atajo relativo al repo.
func TestResolveAll_LeadingSlash_StaysRepoRelative_WhenNoAbsoluteMatch(t *testing.T) {
	repo, _ := writeMultiFixture(t)

	_, stats := RenderFocusContext(repo, "", []string{"/src"}, nil, nil)

	if len(stats.Items) != 1 || stats.Items[0].Kind != "dir" {
		t.Fatalf("expected \"/src\" to still resolve as the repo-relative src/ directory, got %+v", stats.Items)
	}
}

// TestResolveAll_DirAndGlob_ExcludeGitAndNodeModules — SEGURIDAD: un
// directorio o glob de `focus` (".", "src", "**/*") NUNCA debe volcar
// el contenido de .git/ (puede tener credenciales en .git/config) o
// node_modules/ (dependencias de terceros, potencialmente enormes) al
// contexto — misma exclusión por defecto que ya aplicaba
// walkFiles/CodeSymbolResolver/etc., ahora también en Directory/Glob.
func TestResolveAll_DirAndGlob_ExcludeGitAndNodeModules(t *testing.T) {
	repo, _ := writeMultiFixture(t)
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "node_modules", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".git", "config"), []byte("[remote]\nurl = https://user:TOKEN_SECRETO@github.com/x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "node_modules", "pkg", "index.js"), []byte("dependencia de terceros"), 0o644); err != nil {
		t.Fatal(err)
	}

	textDot, statsDot := RenderFocusContext(repo, "", []string{"."}, nil, nil)
	if statsDot.FilesIncluded != 3 {
		t.Fatalf("expected \".\" to include only the 3 legitimate repo files (excluding .git/node_modules), got %d", statsDot.FilesIncluded)
	}
	if strings.Contains(textDot, "TOKEN_SECRETO") || strings.Contains(textDot, "dependencia de terceros") {
		t.Fatalf("SECURITY REGRESSION: \".\" leaked .git or node_modules content:\n%s", textDot)
	}

	textGlob, _ := RenderFocusContext(repo, "", []string{"**/*"}, nil, nil)
	if strings.Contains(textGlob, "TOKEN_SECRETO") {
		t.Fatalf("SECURITY REGRESSION: \"**/*\" leaked .git content:\n%s", textGlob)
	}
}

// TestResolveAll_RelativeTraversal_CannotEscapeRepo — SEGURIDAD:
// "../../etc/passwd" (o cualquier "../" que empuje fuera de repoPath)
// NUNCA debe leerse como si fuera un target válido. Esto es DISTINTO
// del soporte nuevo de rutas absolutas (§16.2/fsutil.go), que es
// intencional y exige que el path exista de verdad — acá se prueba que
// un "../" relativo simplemente no encuentra nada, en vez de escapar
// silenciosamente del repo vía filepath.Join.
func TestResolveAll_RelativeTraversal_CannotEscapeRepo(t *testing.T) {
	repo, _ := writeMultiFixture(t)
	outsideDir := filepath.Dir(repo) // el padre de repo/, fuera del repo
	secretPath := filepath.Join(outsideDir, "secret-outside-repo.txt")
	if err := os.WriteFile(secretPath, []byte("SECRETO FUERA DEL REPO"), 0o644); err != nil {
		t.Fatal(err)
	}

	text, stats := RenderFocusContext(repo, "", []string{"../secret-outside-repo.txt"}, nil, nil)

	if stats.FilesIncluded != 0 {
		t.Fatalf("expected \"../secret-outside-repo.txt\" to resolve to NOTHING (blocked traversal), got FilesIncluded=%d", stats.FilesIncluded)
	}
	if strings.Contains(text, "SECRETO FUERA DEL REPO") {
		t.Fatalf("SECURITY REGRESSION: relative path traversal (\"../\") escaped repoPath and leaked file content:\n%s", text)
	}
}

// TestExclude_BareName_BlocksDirectoryAnywhereInTree — el caso pedido
// explícitamente: "exclude": ["node_modules"] bloquea esa carpeta sin
// importar en qué nivel aparezca, igual que el default fijo
// (.git/vendor/...) pero configurable desde project.json.
func TestExclude_BareName_BlocksDirectoryAnywhereInTree(t *testing.T) {
	repo, _ := writeMultiFixture(t)
	if err := os.MkdirAll(filepath.Join(repo, "libs", "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "libs", "node_modules", "dep.js"), []byte("SECRETO_DEP"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "libs", "own.js"), []byte("propio"), 0o644); err != nil {
		t.Fatal(err)
	}

	text, _ := RenderFocusContextWithSeenExclude(repo, []string{"libs"}, []string{"node_modules"})

	if strings.Contains(text, "SECRETO_DEP") {
		t.Fatalf("expected \"exclude\": [\"node_modules\"] to block libs/node_modules entirely, got:\n%s", text)
	}
	if !strings.Contains(text, "propio") {
		t.Fatalf("expected libs/own.js (not excluded) to still be included, got:\n%s", text)
	}
}

// TestExclude_ExplicitFocusFile_IsStillBlocked — el requisito central:
// si algo está en "exclude", "focus" NO LO LEE aunque lo pida por
// nombre exacto.
func TestExclude_ExplicitFocusFile_IsStillBlocked(t *testing.T) {
	repo, _ := writeMultiFixture(t)

	text, stats := RenderFocusContextWithSeenExclude(repo, []string{"server.js"}, []string{"server.js"})

	if stats.FilesIncluded != 0 {
		t.Fatalf("expected an explicit focus target that's also in exclude to resolve to NOTHING, got FilesIncluded=%d", stats.FilesIncluded)
	}
	if strings.Contains(text, "console.log") {
		t.Fatalf("SECURITY: exclude did not block an explicitly-focused file:\n%s", text)
	}
}

// TestExclude_AbsoluteHostPath_Multiplatform — "exclude" soporta la
// MISMA sintaxis multiplataforma que "focus": una ruta absoluta del
// host (aquí, la carpeta de prueba "external/") bloquea todo lo que
// esté debajo, aunque "focus" apunte directo a un archivo adentro.
func TestExclude_AbsoluteHostPath_Multiplatform(t *testing.T) {
	repo, external := writeMultiFixture(t)
	absFile := filepath.Join(external, "archivo.java")

	text, stats := RenderFocusContextWithSeenExclude(repo, []string{absFile}, []string{external})

	if stats.FilesIncluded != 0 {
		t.Fatalf("expected the absolute exclude to block the absolute focus target, got FilesIncluded=%d", stats.FilesIncluded)
	}
	if strings.Contains(text, "class X{}") {
		t.Fatalf("SECURITY: absolute exclude path did not block content:\n%s", text)
	}
}

// TestExclude_Glob_BlocksMatchingExtension confirma que "exclude"
// soporta globs, igual que "focus" (p. ej. "*.env" para nunca incluir
// archivos de variables de entorno sin importar el directorio).
func TestExclude_Glob_BlocksMatchingExtension(t *testing.T) {
	repo, _ := writeMultiFixture(t)
	if err := os.WriteFile(filepath.Join(repo, "src", "secret.env"), []byte("API_KEY=xyz"), 0o644); err != nil {
		t.Fatal(err)
	}

	text, _ := RenderFocusContextWithSeenExclude(repo, []string{"."}, []string{"*.env"})

	if strings.Contains(text, "API_KEY") {
		t.Fatalf("SECURITY: \"exclude\": [\"*.env\"] did not block src/secret.env:\n%s", text)
	}
}

// RenderFocusContextWithSeenExclude es un pequeño wrapper de conveniencia
// para estos tests — evita repetir la firma completa de
// RenderFocusContext con sus parámetros posicionales.
func RenderFocusContextWithSeenExclude(repo string, items, exclude []string) (string, focus.ScanStats) {
	return RenderFocusContext(repo, "", items, nil, exclude)
}

// TestResolveAll_ExtensionGlob_NeverCapturedByCatchAllResolvers confirma
// el fix real de este ciclo: CodeSymbolResolver/SQLResolver/
// MarkdownResolver/LegalResolver/MemoryResolver son "catch-all" (Match
// siempre true) y corren ANTES que GlobResolver en el orden por
// defecto — antes de excluir isGlobPattern() en su Match, un patrón
// como "**/*.go" (o el "." de arriba) podía ser interceptado por su
// pasada LIKE (que hace falso-positivo con símbolos cortos como "."),
// nunca llegando a GlobResolver.
func TestResolveAll_ExtensionGlob_NeverCapturedByCatchAllResolvers(t *testing.T) {
	repo, _ := writeMultiFixture(t)

	_, stats := RenderFocusContext(repo, "", []string{"**/*.go"}, nil, nil)

	if stats.FilesIncluded != 1 {
		t.Fatalf("expected \"**/*.go\" to match exactly src/a.go, got FilesIncluded=%d", stats.FilesIncluded)
	}
	if len(stats.Items) != 1 || stats.Items[0].Kind != "dir" || stats.Items[0].Files != 1 {
		t.Fatalf("expected the glob target to be reported as Kind=dir Files=1, got %+v", stats.Items)
	}
}
