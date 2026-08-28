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
		if e.isDir {
			if skipDirOrExcluded(ctx, m, name) {
				ctx.RecordExcluded(e.path, name, countFiles(e.path))
				continue
			}
			walkFilesExcluding(ctx, m, e.path, fn)
			continue
		}
		if m.excludesPath(e.path) {
			ctx.RecordExcluded(e.path, name, 1)
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

// binaryDocExts are the extensions readFile must NOT return as raw
// bytes — .docx/.xlsx/.pdf are ZIP/PDF containers, not text, so dumping
// them verbatim into a focus block previously produced binary garbage
// in the context instead of the document's actual text. Kept as a
// small, explicit set (not a general "is this binary?" heuristic) so
// every plain-text extension (.txt/.md/.json/.log/...) keeps behaving
// exactly as before.
var binaryDocExts = map[string]bool{".docx": true, ".xlsx": true, ".pdf": true}

// readFileText is what FileResolver/GlobResolver call instead of
// readFile for the actual block CONTENT (never for existence checks,
// which only need "is this readable at all" and stay on readFile) —
// for .docx/.xlsx/.pdf it extracts the real text layer via
// mova.local/documents.ReadDocumentLayer (the exact same extraction
// `read_document_layer`/`mova chat` already use), falling back to
// readFile for every other extension. A document that fails to extract
// (corrupted, password-protected, scanned image PDF) returns "" rather
// than raw bytes, same "never less than empty string" contract readFile
// already follows.
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

// WalkAllFiles expone walkFiles fuera del paquete — usado por el
// SemanticResolver de la edición Premium (mova.local/compiler/focus) para
// indexar el mismo conjunto de archivos que ya respeta focus_exclude y las
// carpetas ignoradas por defecto, sin duplicar esta lógica de recorrido.
func WalkAllFiles(ctx focus.Context, root string, fn func(path string)) {
	walkFiles(ctx, root, fn)
}

// ReadFile expone readFile — mismo motivo que WalkAllFiles.
func ReadFile(path string) string { return readFile(path) }

// RelOrBase expone relOrBase — mismo motivo que WalkAllFiles.
func RelOrBase(root, path string) string { return relOrBase(root, path) }

// -----------------------------------------------------------------------------
// Rutas absolutas del host, multiplataforma (Windows/Linux/macOS)
// -----------------------------------------------------------------------------
//
// project.json's "focus"/"memory" siempre trató un target que empieza con
// "/" como relativo a la RAÍZ DEL REPO (ver repoRelativePath más abajo:
// "/src" == "<repo>/src"), nunca como una ruta absoluta del filesystem
// del host — así evita que un project.json ajeno intente leer fuera del
// repo por accidente. Pero hay un caso de uso real y explícito: el
// usuario quiere apuntar `focus` a un archivo o carpeta que vive fuera
// del repo por completo — "C:\ejemploPython\testSentence.py",
// "d:\test\test.py", "/mnt/archivo.java", "/mnt" — y eso tiene que
// funcionar igual en Windows, Linux y macOS sin importar en qué SO
// corre el binario.
//
// Regla de resolución (backward-compatible, nunca rompe project.json
// existentes):
//  1. Una letra de unidad Windows ("C:\...", "d:/...") o una ruta UNC
//     ("\\server\share\...") es INEQUÍVOCAMENTE absoluta — jamás tuvo
//     sentido como target relativo al repo — así que se intenta
//     directo, sin fallback.
//  2. Un "/algo" estilo Unix sigue siendo AMBIGUO con la convención
//     histórica ("/src" == "<repo>/src"): se intenta primero como ruta
//     absoluta real del host (os.Stat); solo si EXISTE así se usa como
//     absoluta. Si no existe como absoluta, cae exactamente al
//     comportamiento de siempre (relativo a la raíz del repo) — ningún
//     project.json existente cambia de comportamiento.
var winDriveRe = regexp.MustCompile(`^[A-Za-z]:[\\/]`)

// isWindowsDriveAbs reporta si target usa notación de unidad de Windows
// ("C:\...", "d:/...") — inequívoco en cualquier SO donde corra Mova.
func isWindowsDriveAbs(target string) bool { return winDriveRe.MatchString(target) }

// isUNCPath reporta si target es una ruta de red UNC de Windows
// ("\\server\share\..." o su variante con "/").
func isUNCPath(target string) bool {
	return strings.HasPrefix(target, `\\`) || strings.HasPrefix(target, "//")
}

// looksAbsoluteHostPath reporta si target TIENE FORMA de ruta absoluta
// del host en cualquier plataforma — no confirma que exista (ver
// resolveAbsoluteFile/Dir, que sí comprueban contra el disco antes de
// usarla).
func looksAbsoluteHostPath(target string) bool {
	return isWindowsDriveAbs(target) || isUNCPath(target) || strings.HasPrefix(target, "/")
}

// normalizeHostPath convierte separadores "\" a "/" — Go's path/filepath
// en Linux/macOS solo reconoce "/" como separador, así que una ruta
// pegada literal de Windows ("C:\a\b.py") necesita normalizarse antes de
// pasarla a os.Stat/os.ReadDir/filepath.WalkDir para que camine
// correctamente sin importar en qué SO corre el binario. En Windows
// mismo, el runtime de Go acepta "/" exactamente igual que "\", así que
// esta normalización es un no-op funcional ahí.
func normalizeHostPath(target string) string {
	return strings.ReplaceAll(target, `\`, "/")
}

// resolveAbsoluteFile intenta target como ARCHIVO absoluto del host
// (ver looksAbsoluteHostPath para qué formas califican). Solo devuelve
// ok=true cuando el path realmente existe en disco y es un archivo —
// nunca "inventa" una ruta que no está ahí, y nunca reclama un
// directorio (eso es trabajo de resolveAbsoluteDir).
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

// resolveAbsoluteDir es el equivalente de resolveAbsoluteFile para
// directorios.
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

// splitAbsoluteGlobRoot separa un patrón glob absoluto normalizado
// ("/mnt/**/*.java", "C:/repos/**/*.go") en la porción de directorio
// SIN metacaracteres ("/mnt", "C:/repos") y el patrón relativo a esa
// raíz ("**/*.java", "**/*.go") — así un glob absoluto puede recorrer
// desde esa raíz externa en vez de desde la raíz del repo. ok=false
// cuando target no tiene ningún metacaracter glob (no es este caso) o
// no tiene un directorio raíz identificable antes del primer
// metacaracter.
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
