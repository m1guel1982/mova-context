// exclude.go — implementa la clave "exclude" de project.json (y su
// override a nivel task): una lista de targets con la MISMA sintaxis
// multiplataforma que "focus" — nombre bare de carpeta/archivo
// ("node_modules", ".git"), ruta relativa al repo ("src/secrets"), ruta
// absoluta del host ("C:\secrets", "D:\private", "/mnt/private") o glob
// ("*.env", "**/*.pem") — pero para EXCLUSIÓN: cualquier archivo o
// directorio que matchee un patrón de "exclude" NUNCA se resuelve — ni
// siquiera si "focus" lo pide explícitamente por su nombre exacto — y
// por lo tanto nunca llega a mova-context-cache.json (ver
// budget/contextcache.go: solo se cachea lo que SanitizeCached recibe,
// y lo que exclude bloquea nunca sale de aquí).
//
// Se comprueba en DOS niveles, igual que ctx.SkipDir ya hacía para el
// default fijo (.git/node_modules/vendor/...):
//   - por NOMBRE, durante cualquier recorrido recursivo (walkFiles,
//     DirectoryResolver.Resolve, GlobResolver.Resolve) — para que
//     "exclude": ["node_modules"] frene la carpeta apenas se la
//     encuentra, sin necesidad de conocer su ruta completa.
//   - por RUTA YA RESUELTA (absoluta), para los otros tres casos: un
//     archivo puntual ("focus": ["server.js"] pero "server.js" está en
//     exclude), una ruta relativa al repo más específica que un simple
//     nombre ("src/secrets"), o una ruta absoluta del host.
package resolvers

import (
	"path/filepath"
	"strings"

	"mova.local/core/focus"
)

// excludeMatcher precompila los patrones de "exclude" una sola vez por
// llamada a Resolve (nunca por archivo) en las cuatro formas de match
// que necesita — ver el comentario del archivo para el porqué de cada
// una.
type excludeMatcher struct {
	bareNames map[string]bool // "node_modules", ".git", "secret.env" — cualquier nivel, por nombre
	absPaths  []string        // resueltos y normalizados con "/" — "/mnt/private", "c:/secrets"
	repoPaths []string        // filepath.Join(repoPath, ...), absolutos, normalizados con "/"
	globs     []string        // "*.env", "**/*.pem" — normalizados con "/"
}

// newExcludeMatcher interpreta cada patrón crudo de "exclude" con las
// MISMAS reglas de detección que "focus" usa para reconocer una ruta
// absoluta del host (looksAbsoluteHostPath) o un glob (isGlobPattern) —
// un patrón nunca se interpreta de una forma para leer y de otra para
// excluir.
func newExcludeMatcher(repoPath string, patterns []string) *excludeMatcher {
	if len(patterns) == 0 {
		return nil
	}
	m := &excludeMatcher{bareNames: map[string]bool{}}
	for _, raw := range patterns {
		p := strings.TrimSpace(raw)
		if p == "" || p == "." {
			continue // "exclude": ["."] no tiene un significado útil — se ignora en vez de excluir todo el repo por accidente
		}
		switch {
		case isGlobPattern(p):
			m.globs = append(m.globs, normalizeHostPath(p))
		case looksAbsoluteHostPath(p):
			m.absPaths = append(m.absPaths, normalizeHostPath(p))
		case !strings.ContainsAny(p, `/\`):
			m.bareNames[p] = true
		default:
			if rp := repoRelativePath(repoPath, p); rp != "" {
				m.repoPaths = append(m.repoPaths, normalizeHostPath(filepath.Clean(rp)))
			}
		}
	}
	if len(m.bareNames) == 0 && len(m.absPaths) == 0 && len(m.repoPaths) == 0 && len(m.globs) == 0 {
		return nil
	}
	return m
}

// excludesName reporta si una carpeta o archivo, identificado solo por
// su NOMBRE (sin ruta), debe excluirse en cualquier nivel del árbol —
// usado durante el recorrido recursivo, igual que ctx.SkipDir.
func (m *excludeMatcher) excludesName(name string) bool {
	if m == nil {
		return false
	}
	return m.bareNames[name]
}

// excludesPath reporta si una ruta YA RESUELTA (absoluta) debe
// excluirse — por nombre base, por estar dentro de una ruta
// absoluta/relativa-al-repo excluida, o por matchear un glob.
func (m *excludeMatcher) excludesPath(absPath string) bool {
	if m == nil {
		return false
	}
	norm := normalizeHostPath(absPath)
	base := filepath.Base(absPath)
	if m.bareNames[base] {
		return true
	}
	for _, p := range m.absPaths {
		if norm == p || strings.HasPrefix(norm, p+"/") {
			return true
		}
	}
	for _, p := range m.repoPaths {
		if norm == p || strings.HasPrefix(norm, p+"/") {
			return true
		}
	}
	for _, g := range m.globs {
		if matched, _ := filepath.Match(g, base); matched {
			return true
		}
		if matched, _ := filepath.Match(g, norm); matched {
			return true
		}
	}
	return false
}

// skipDirOrExcluded combina ctx.SkipDir (el default fijo:
// .git/node_modules/vendor/dist/build/__pycache__/.venv/venv/.idea/
// .vscode, más "focus_exclude") con el nuevo excludeMatcher — un solo
// punto de verdad para "¿esta carpeta se descarta durante un
// recorrido?", usado por walkFiles/DirectoryResolver/GlobResolver.
func skipDirOrExcluded(ctx focus.Context, m *excludeMatcher, name string) bool {
	return ctx.SkipDir(name) || m.excludesName(name)
}
