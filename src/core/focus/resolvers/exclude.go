package resolvers

import (
	"path/filepath"
	"strings"

	"mova.local/core/focus"
)

type excludeMatcher struct {
	bareNames map[string]bool
	absPaths  []string
	repoPaths []string
	globs     []string
}

func normalizePath(p string) string {
	p = strings.ReplaceAll(p, `\`, "/")
	p = strings.TrimSpace(p)
	return strings.ToLower(p)
}

func newExcludeMatcher(repoPath string, patterns []string) *excludeMatcher {
	if len(patterns) == 0 {
		return nil
	}
	m := &excludeMatcher{bareNames: map[string]bool{}}
	cleanRepo := normalizePath(repoPath)

	for _, raw := range patterns {
		p := strings.TrimSpace(raw)
		if p == "" || p == "." {
			continue
		}

		// Quitar la barra inicial si viene como "/mova_print/..." para normalizar a ruta relativa del repo
		pClean := strings.TrimPrefix(p, "/")
		pClean = strings.TrimPrefix(pClean, `\`)

		switch {
		case isGlobPattern(p):
			m.globs = append(m.globs, normalizePath(p))
		case looksAbsoluteHostPath(p):
			m.absPaths = append(m.absPaths, normalizePath(p))
		case !strings.ContainsAny(p, `/\`):
			m.bareNames[strings.ToLower(p)] = true
		default:
			full := normalizePath(filepath.Join(cleanRepo, pClean))
			m.repoPaths = append(m.repoPaths, full)
		}
	}

	if len(m.bareNames) == 0 && len(m.absPaths) == 0 && len(m.repoPaths) == 0 && len(m.globs) == 0 {
		return nil
	}
	return m
}

func (m *excludeMatcher) excludesName(name string) bool {
	if m == nil {
		return false
	}
	return m.bareNames[strings.ToLower(name)]
}

func (m *excludeMatcher) excludesPath(absPath string) bool {
	if m == nil {
		return false
	}
	norm := normalizePath(absPath)
	base := strings.ToLower(filepath.Base(absPath))

	// 1. Coincidencia por nombre simple ("node_modules", "docs", etc.)
	if m.bareNames[base] {
		return true
	}

	// 2. Coincidencia si algún bareName existe como directorio en la ruta
	for name := range m.bareNames {
		if strings.Contains(norm, "/"+name+"/") || strings.HasSuffix(norm, "/"+name) {
			return true
		}
	}

	// 3. Coincidencia por ruta del repositorio ("mova_print/frontend/node_modules")
	for _, p := range m.repoPaths {
		if norm == p || strings.HasPrefix(norm, p+"/") {
			return true
		}
	}

	// 4. Coincidencia por ruta absoluta
	for _, p := range m.absPaths {
		if norm == p || strings.HasPrefix(norm, p+"/") {
			return true
		}
	}

	// 5. Coincidencia por Globs
	for _, g := range m.globs {
		// Probar contra el nombre base (ej. "imagen.jpg" contra "*.jpg")
		if matched, _ := filepath.Match(g, base); matched {
			return true
		}
		// Probar extensión directamente si el glob empieza con "*."
		if strings.HasPrefix(g, "*.") {
			ext := strings.TrimPrefix(g, "*")
			if strings.HasSuffix(base, ext) {
				return true
			}
		}
	}
	return false
}

func skipDirOrExcluded(ctx focus.Context, m *excludeMatcher, name string) bool {
	return ctx.SkipDir(name) || (m != nil && m.excludesName(name))
}