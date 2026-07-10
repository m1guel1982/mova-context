// stats.go — evidencia real de qué tocó el Focus Resolution Engine al
// recorrer `repo`. Nace de un pedido explícito de transparencia: el
// reporte de compilación no debe ser una caja negra — "reducción del
// 62%" sin más no dice nada; "124 archivos escaneados, 18 incluidos,
// 40 en node_modules/ ignorados, 66 no coincidían con ningún focus, 1
// párrafo duplicado eliminado" sí.
//
// Regla de honestidad: nunca se inventa una categoría de exclusión que no
// corresponda a algo que realmente pasó. Por eso solo existen los motivos
// de exclusión/ajuste reales de este motor: carpeta ignorada, archivo que
// no coincidió con ningún target de `focus`, y párrafo textualmente
// idéntico a uno ya incluido antes en el mismo `contexto.txt` (ver
// render.go — DuplicatesRemoved).
package focus

// ScanStats acumula, para UNA ejecución (`mova run` o `mova compile`,
// todos los targets de `focus` juntos, no uno por uno), cuántos archivos
// vio cualquier resolver que recorrió el filesystem. Deduplica por ruta
// real: si dos targets de `focus` (o dos resolvers en cascada dentro de un
// mismo target) terminan recorriendo el mismo archivo o la misma carpeta
// excluida, cuenta una sola vez — el número reportado es "cuántos
// archivos reales", no "cuántas veces se tocó un archivo".
type ScanStats struct {
	scannedFiles map[string]bool // ruta de archivo -> ya contado
	excludedDirs map[string]bool // ruta de carpeta -> ya contada

	ExcludedByDir map[string]int // nombre de carpeta ignorada -> archivos que contenía
	FilesIncluded int            // lo asigna render.go al final (dedup propio, ver ahí)

	// DuplicatesRemoved cuenta párrafos/secciones cuyo texto (normalizado:
	// espacios colapsados, sin cambiar una sola palabra) es IDÉNTICO a
	// uno que este mismo render ya emitió antes — nunca una
	// reformulación ni un "parecido". Ver render.go (dedupParagraphs).
	// Aplica solo a contenido de prosa (Markdown/legal/texto), nunca a
	// bloques de código, SQL o JSON — esos nunca se tocan.
	DuplicatesRemoved int
}

// FilesScanned es la cuenta final, deduplicada por ruta real.
func (s *ScanStats) FilesScanned() int {
	if s == nil {
		return 0
	}
	return len(s.scannedFiles)
}

// defaultExcludedDirs — carpetas que ningún resolver necesita recorrer
// nunca (metadatos de control de versiones, dependencias instaladas,
// artefactos de build): siempre se ignoran, sin necesidad de configurar
// nada (convención sobre configuración). project.json
// (contextCompiler.focus_exclude) solo puede AMPLIAR esta lista, nunca
// reducirla.
var defaultExcludedDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
	"dist": true, "build": true, "__pycache__": true,
	".venv": true, "venv": true, ".idea": true, ".vscode": true,
}

// SkipDir decide si una carpeta con este nombre debe ignorarse al
// recorrer el repo — por defecto, o porque project.json la agregó a
// ExcludeDirs.
func (c Context) SkipDir(name string) bool {
	if defaultExcludedDirs[name] {
		return true
	}
	for _, d := range c.ExcludeDirs {
		if d == name {
			return true
		}
	}
	return false
}

// RecordScanned / RecordExcluded son no-op seguros cuando Context.Stats es
// nil (el caso normal fuera de `mova compile`/`mova run`, o cuando a nadie
// le interesa la evidencia detallada) — nunca hace falta un nil-check en
// el llamador. Ambos deduplican por ruta física real (ver ScanStats arriba).
func (c Context) RecordScanned(path string) {
	if c.Stats == nil {
		return
	}
	if c.Stats.scannedFiles == nil {
		c.Stats.scannedFiles = map[string]bool{}
	}
	c.Stats.scannedFiles[path] = true
}

func (c Context) RecordExcluded(dirPath, dirName string, filesInside int) {
	if c.Stats == nil || filesInside == 0 {
		return
	}
	if c.Stats.excludedDirs == nil {
		c.Stats.excludedDirs = map[string]bool{}
	}
	if c.Stats.excludedDirs[dirPath] {
		return // ya contada — otro resolver/target ya recorrió esta misma carpeta
	}
	c.Stats.excludedDirs[dirPath] = true
	if c.Stats.ExcludedByDir == nil {
		c.Stats.ExcludedByDir = map[string]int{}
	}
	c.Stats.ExcludedByDir[dirName] += filesInside
}
