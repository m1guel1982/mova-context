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
	"sort"
	"strings"

	"mova.local/core/focus"
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
// agregado) y registra en ctx.Stats, si existe, tanto lo que escaneó como
// lo que excluyó — nunca en silencio, para que contexto.report pueda
// mostrarlo con honestidad.
func walkFiles(ctx focus.Context, dir string, fn func(path string)) {
	entries := listEntries(dir)
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })
	for _, e := range entries {
		if e.isDir {
			name := filepath.Base(e.path)
			if ctx.SkipDir(name) {
				ctx.RecordExcluded(e.path, name, countFiles(e.path))
				continue
			}
			walkFiles(ctx, e.path, fn)
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

func relOrBase(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		return rel
	}
	return filepath.Base(path)
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
