// nameindex.go — resolución rápida de referencias por nombre o ruta
// parcial ("Gantt.js", "programacion/Gantt.js") para focus y exclude,
// apoyada en focus.FileIndex: una sola caminata del repo por ejecución.
package resolvers

import (
	"path/filepath"
	"strings"

	"mova.local/core/focus"
)

// indexFor devuelve el índice nombre→rutas del repo (construido una vez).
func indexFor(ctx focus.Context) map[string][]string {
	idx := ctx.Index
	if idx == nil {
		idx = &focus.FileIndex{} // sin índice compartido: una caminata por consulta
	}
	return idx.ByName(func(add func(path, base string)) {
		walkFiles(ctx, ctx.RepoPath, func(p string) { add(p, filepath.Base(p)) })
	})
}

// findIndexed devuelve TODOS los archivos indexados a los que puede
// referirse target: un nombre suelto ("Gantt.js") o los últimos
// segmentos de una ruta ("programacion/Gantt.js"), sin distinguir
// mayúsculas. Varias coincidencias se devuelven todas, en orden
// determinista.
func findIndexed(ctx focus.Context, target string) []string {
	t := strings.TrimPrefix(strings.Trim(strings.ReplaceAll(strings.TrimSpace(target), `\`, "/"), "/"), "./")
	if t == "" {
		return nil
	}
	base := t[strings.LastIndex(t, "/")+1:]
	cands := indexFor(ctx)[strings.ToLower(base)]
	if base == t {
		return cands
	}
	suffix := "/" + strings.ToLower(t)
	var out []string
	for _, p := range cands {
		if strings.HasSuffix(strings.ToLower(filepath.ToSlash(p)), suffix) {
			out = append(out, p)
		}
	}
	return out
}

// locateFiles resuelve una referencia a archivo a todos los archivos que
// puede significar, en orden de prioridad: ruta absoluta del host →
// ruta relativa al repo → coincidencia por nombre/ruta parcial. Respeta
// `exclude`. Es el único lugar que decide esto (File, AstSymbol y debug).
func locateFiles(ctx focus.Context, target string) []string {
	target = focus.StripExact(target)
	m := newExcludeMatcher(ctx.RepoPath, ctx.Exclude)
	if abs, ok := resolveAbsoluteFile(target); ok {
		if m.excludesPath(abs) {
			return nil
		}
		return []string{abs}
	}
	if p := repoRelativePath(ctx.RepoPath, target); p != "" && !m.excludesPath(p) && !isDir(p) && readFile(p) != "" {
		return []string{p}
	}
	return findIndexed(ctx, target) // walkFiles ya filtró por exclude
}

// LocateTarget expone, para el log de debug, a qué archivos reales
// apunta una entrada de focus/exclude ("Gantt.js::func=show()" →
// rutas de Gantt.js). Vacío si no es un archivo (directorio, glob...).
func LocateTarget(ctx focus.Context, entry string) []string {
	file, _, _ := strings.Cut(entry, "::")
	if isGlobPattern(file) {
		return nil
	}
	return locateFiles(ctx, strings.TrimSpace(file))
}
