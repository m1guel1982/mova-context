// file_adapter.go — reads knowledge from Markdown files.
// Default adapter. No setup required. 100% backward compatible.
//
// package core (Open Source). Cero dependencias externas — es el adapter
// que cualquier usuario tiene disponible sin instalar nada más.
package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mova.local/mpaths"
)

type fileAdapter struct{ root string }

func NewFileAdapter(root string) *fileAdapter { return &fileAdapter{root} }

func (a *fileAdapter) GetProject(name string) (*Project, error) {
	data, err := os.ReadFile(filepath.Join(mpaths.ProjectsDir(a.root), name, "project.json"))
	if err != nil {
		return nil, fmt.Errorf("project %q not found", name)
	}
	var p Project
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("project.json invalid: %w", err)
	}
	return &p, nil
}

// ProjectJSONPath es dónde vive project.json para un proyecto basado en
// archivos — expuesto para que procesos de vida larga (el REPL de `mova
// chat`, la invalidación en caliente de mova.local/budget/contextcache.go)
// puedan vigilarlo sin reimplementar esta misma ruta. Los proyectos
// respaldados por base de datos (dbAdapter) o los grupos multiagente
// (projects/<group>/config.json) no tienen project.json — ver
// ProjectJSONFingerprint, que devuelve ok=false en ese caso en vez de
// error.
func ProjectJSONPath(root, project string) string {
	return filepath.Join(mpaths.ProjectsDir(root), project, "project.json")
}

// ProjectJSONFingerprint identifica el ESTADO ACTUAL de project.json en
// disco: su mtime y un hash sha256 de su contenido byte a byte. ok es
// false cuando el archivo no existe — nunca un error, porque "este
// proyecto no tiene project.json para vigilar" (DB adapter, grupo
// multiagente) es un caso normal, no una falla. Usado por
// mova.local/budget.SanitizeCached (invalidación del cache al primer
// cambio detectado) y por cli/chat_helpers.go's refreshProjectContext
// (recarga en caliente durante una sesión de `mova chat` ya abierta).
func ProjectJSONFingerprint(root, project string) (modTime time.Time, hash string, ok bool) {
	path := ProjectJSONPath(root, project)
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return info.ModTime(), "", true
	}
	sum := sha256.Sum256(data)
	return info.ModTime(), hex.EncodeToString(sum[:]), true
}

// ListProjects discovers all projects by recursively walking projects/ for project.json files.
// Never uses hardcoded lists. Detects new projects automatically.
func (a *fileAdapter) ListProjects() ([]ProjectSummary, error) {
	projectsDir := mpaths.ProjectsDir(a.root)
	var out []ProjectSummary

	err := filepath.WalkDir(projectsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || d.Name() != "project.json" {
			return nil
		}
		// Extract project name from the directory containing project.json
		dir := filepath.Dir(path)
		name := filepath.Base(dir)

		p, err := a.GetProject(name)
		if err != nil {
			// Try by directory relative to projectsDir
			rel, _ := filepath.Rel(projectsDir, dir)
			p, err = a.getProjectByPath(path)
			if err != nil {
				return nil // skip invalid projects
			}
			name = rel
		}

		tasks := make([]string, 0, len(p.Tasks))
		for k := range p.Tasks {
			tasks = append(tasks, k)
		}
		sort.Strings(tasks)
		out = append(out, ProjectSummary{
			Name: name, Description: p.Description,
			Lang: p.Lang, Tasks: tasks,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// getProjectByPath loads a project directly from a project.json path.
func (a *fileAdapter) getProjectByPath(path string) (*Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Project
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("project.json invalid: %w", err)
	}
	return &p, nil
}

func (a *fileAdapter) Search(query, domain string) ([]SearchResult, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil, nil
	}
	var results []SearchResult
	for _, kind := range []string{"agent", "skill", "prompt"} {
		kd := kindDir(a.root, kind, "") // config/general/config.json or default
		dir := kd
		if domain != "" {
			dir = filepath.Join(dir, domain)
		}
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			content := strings.ToLower(readFile(path))
			name := strings.TrimSuffix(info.Name(), ".md")
			if strings.Contains(name, q) || strings.Contains(content, q) {
				lang, domainFound := extractLangDomain(path, filepath.Dir(kd), filepath.Base(kd))
				results = append(results, SearchResult{
					Kind: kind, Domain: domainFound, Lang: lang,
					Name: name, Excerpt: excerpt(content, q), Score: score(name, content, q),
					Path: path, Line: firstMatchLine(content, q),
				})
			}
			return nil
		})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	return results, nil
}
