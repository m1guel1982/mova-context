// fsutil.go — helpers de filesystem compartidos por los resolvers de este
// paquete. Movido sin cambios de lógica desde
// mova.local/compiler/focus/resolvers (edición Premium) a
// mova.local/core/focus/resolvers (edición Community) — ver
// docs/i18n/{es,en}/focus-engine.md para el porqué del movimiento.
package resolvers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"mova.local/core/focus"
	"mova.local/documents"
)

type dirEntry struct {
	path  string
	isDir bool
}

func listEntries(dir string) []dirEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]dirEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, dirEntry{path: filepath.Join(dir, e.Name()), isDir: e.IsDir()})
	}
	return out
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// listDir devuelve un índice compacto y ordenado de un directorio — nunca
// su contenido. Un directorio es un target de focus legítimo, pero volcar
// cada archivo dentro derrotaría el propósito de "focus".
func listDir(path string) string {
	entries := listEntries(path)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		n := filepath.Base(e.path)
		if e.isDir {
			n += "/"
		}
		names = append(names, n)
	}
	sort.Strings(names)
	return fmt.Sprintf("dir(%d): %s", len(names), strings.Join(names, ", "))
}

// walkFiles recorre dir recursivamente en orden determinista (ordena las
// entradas de cada directorio antes de descender) — nunca depende del
// orden que entregue el sistema operativo. Ignora las carpetas que
// ctx.SkipDir marca (siempre .git/node_modules/vendor/dist/build/
// __pycache__/.venv/venv/.idea/.vscode, más lo que project.json haya
// agregado a "focus_exclude") Y las que la clave "exclude" de
// project.json marque (ctx.Exclude — ver exclude.go, soporta nombres,
// rutas completas y globs, no solo nombres de carpeta), y registra en
// ctx.Stats, si existe, tanto lo que escaneó como lo que excluyó —
// nunca en silencio, para que contexto.report pueda mostrarlo con
// honestidad.
func walkFiles(ctx focus.Context, dir string, fn func(path string)) {
	m := newExcludeMatcher(ctx.RepoPath, ctx.Exclude)
	walkFilesExcluding(ctx, m, dir, fn)
}

func walkFilesExcluding(ctx focus.Context, m *excludeMatcher, dir string, fn func(path string)) {
	entries := listEntries(dir)
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })

	for _, e := range entries {
		name := filepath.Base(e.path)

		// AVALUACIÓN DIRECTA DE LA RUTA DEL DIRECTORIO O ARCHIVO
		if (m != nil && m.excludesPath(e.path)) || skipDirOrExcluded(ctx, m, name) {
			if e.isDir {
				ctx.RecordExcluded(e.path, name, countFiles(e.path))
			} else {
				ctx.RecordExcluded(e.path, name, 1)
			}
			continue // No entra a recorrer este directorio
		}

		if e.isDir {
			walkFilesExcluding(ctx, m, e.path, fn)
			continue
		}

		ctx.RecordScanned(e.path)
		fn(e.path)
	}
}

// countFiles cuenta archivos (no carpetas) dentro de dir, recursivamente —
// usado solo para reportar con un número real cuántos archivos había
// dentro de una carpeta excluida, en vez de solo "1 carpeta ignorada".
func countFiles(dir string) int {
	n := 0
	for _, e := range listEntries(dir) {
		if e.isDir {
			n += countFiles(e.path)
		} else {
			n++
		}
	}
	return n
}

func findByName(ctx focus.Context, dir, name string) string {
	var found string
	walkFiles(ctx, dir, func(p string) {
		if found == "" && filepath.Base(p) == name {
			found = p
		}
	})
	return found
}

// relOrBase da la etiqueta de Source más útil para un path ya resuelto:
// si path vive DENTRO de root, la ruta relativa de siempre; si vive
// FUERA (un target absoluto del host — ver isAbsoluteHostPath en este
// mismo archivo, p. ej. "C:\ejemploPython\x.py" o "/mnt/archivo.java")
// filepath.Rel puede devolver algo técnicamente válido pero inútil como
// "../../../mnt/archivo.java" — en ese caso se usa el path absoluto
// completo, que es la etiqueta clara para algo que está fuera del repo.
func relOrBase(root, path string) string {
	if comparableHostPath(filepath.Clean(root)) == comparableHostPath(filepath.Clean(path)) {
		return "."
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.Base(path)
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return path
	}
	return rel
}

// readFile lee un archivo devolviendo "" en caso de error — nunca panics,
// nunca detiene la resolución de otros targets.
func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

var binaryDocExts = map[string]bool{".docx": true, ".xlsx": true, ".pdf": true}

func readFileText(path string) string {
	if binaryDocExts[strings.ToLower(filepath.Ext(path))] {
		text, err := documents.ReadDocumentLayer(path)
		if err != nil {
			return ""
		}
		return text
	}
	return readFile(path)
}

func WalkAllFiles(ctx focus.Context, root string, fn func(path string)) {
	walkFiles(ctx, root, fn)
}

func ReadFile(path string) string { return readFile(path) }

func RelOrBase(root, path string) string { return relOrBase(root, path) }

// -----------------------------------------------------------------------------
// Rutas absolutas del host, multiplataforma (Windows/Linux/macOS)
// -----------------------------------------------------------------------------

var winDriveRe = regexp.MustCompile(`^[A-Za-z]:[\\/]`)

func isWindowsDriveAbs(target string) bool { return winDriveRe.MatchString(target) }

func isUNCPath(target string) bool {
	return strings.HasPrefix(target, `\\`) || strings.HasPrefix(target, "//")
}

func looksAbsoluteHostPath(target string) bool {
	return isWindowsDriveAbs(target) || isUNCPath(target) || strings.HasPrefix(target, "/")
}

func normalizeHostPath(target string) string {
	return strings.ReplaceAll(target, `\`, "/")
}

func resolveAbsoluteFile(target string) (string, bool) {
	if !looksAbsoluteHostPath(target) {
		return "", false
	}
	norm := normalizeHostPath(target)
	info, err := os.Stat(norm)
	if err != nil || info.IsDir() {
		return "", false
	}
	return norm, true
}

func resolveAbsoluteDir(target string) (string, bool) {
	if !looksAbsoluteHostPath(target) {
		return "", false
	}
	norm := normalizeHostPath(target)
	info, err := os.Stat(norm)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return norm, true
}

func splitAbsoluteGlobRoot(norm string) (root, pattern string, ok bool) {
	idx := strings.IndexAny(norm, "*?[")
	if idx < 0 {
		return "", "", false
	}
	slash := strings.LastIndex(norm[:idx], "/")
	if slash <= 0 {
		return "", "", false
	}
	return norm[:slash], norm[slash+1:], true
}