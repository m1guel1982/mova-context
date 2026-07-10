// file_adapter.go — reads knowledge from Markdown files.
// Default adapter. No setup required. 100% backward compatible.
//
// package core (Open Source). Cero dependencias externas — es el adapter
// que cualquier usuario tiene disponible sin instalar nada más.
package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type fileAdapter struct{ root string }

func NewFileAdapter(root string) *fileAdapter { return &fileAdapter{root} }

// GetKnowledge resolves a knowledge file by kind/domain/lang/name.
//
// Search order (first match wins):
//  1. domain/i18n/lang/name.md          (new i18n structure, exact)
//  2. domain/i18n/en/name.md            (en fallback, exact)
//  3. domain/lang/name.md               (legacy flat, exact)
//  4. domain/en/name.md                 (legacy en fallback, exact)
//  5. domain/name.md                    (no-lang legacy)
//  6. root/name.md                      (legacy root-level)
//  7. recursive walk: domain/i18n/lang/ (handles subdirs like engineering/)
//  8. recursive walk: domain/i18n/en/   (en fallback, recursive)
//  9. recursive walk: domain/           (any subdir under domain)
// 10. recursive walk: root/             (finds custom/, etc.)
func (a *fileAdapter) GetKnowledge(kind, domain, lang, name string) (string, error) {
	base := filepath.Join(a.root, kind+"s", domain)
	filename := name + ".md"

	// Exact path candidates (fast)
	candidates := []string{}
	if lang != "" {
		candidates = append(candidates, filepath.Join(base, "i18n", lang, filename))
		if lang != "en" {
			candidates = append(candidates, filepath.Join(base, "i18n", "en", filename))
		}
		candidates = append(candidates, filepath.Join(base, lang, filename))
		if lang != "en" {
			candidates = append(candidates, filepath.Join(base, "en", filename))
		}
	}
	candidates = append(candidates,
		filepath.Join(base, filename),
		filepath.Join(a.root, kind+"s", filename),
	)
	for _, path := range candidates {
		if c := readFile(path); c != "" {
			return c, nil
		}
	}

	// Recursive walk (handles arbitrary subdirectories)
	if lang != "" {
		if c := walkFind(filepath.Join(base, "i18n", lang), filename); c != "" {
			return c, nil
		}
		if lang != "en" {
			if c := walkFind(filepath.Join(base, "i18n", "en"), filename); c != "" {
				return c, nil
			}
		}
	}
	if c := walkFind(base, filename); c != "" {
		return c, nil
	}
	// Global fallback: searches all of agents/skills/prompts (finds custom/, etc.)
	if c := walkFind(filepath.Join(a.root, kind+"s"), filename); c != "" {
		return c, nil
	}

	return "", fmt.Errorf("%s %q not found in domain %q", kind, name, domain)
}

// walkFind walks dir recursively and returns the content of the first file named filename.
func walkFind(dir, filename string) string {
	var found string
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() == filename {
			found = readFile(path)
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func (a *fileAdapter) GetProject(name string) (*Project, error) {
	data, err := os.ReadFile(filepath.Join(a.root, "projects", name, "project.json"))
	if err != nil {
		return nil, fmt.Errorf("project %q not found", name)
	}
	var p Project
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("project.json invalid: %w", err)
	}
	return &p, nil
}

// ListProjects discovers all projects by recursively walking projects/ for project.json files.
// Never uses hardcoded lists. Detects new projects automatically.
func (a *fileAdapter) ListProjects() ([]ProjectSummary, error) {
	projectsDir := filepath.Join(a.root, "projects")
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

func (a *fileAdapter) GetMemory(project string) (string, error) {
	return readFile(filepath.Join(a.root, "projects", project, "memory.md")), nil
}

func (a *fileAdapter) GetMemoryAll(project string) (string, error) {
	active, _ := a.GetMemory(project)
	archDir := filepath.Join(a.root, "projects", project, "memory-archive")
	entries, err := os.ReadDir(archDir)
	if err != nil {
		return active, nil
	}
	parts := []string{}
	if active != "" {
		parts = append(parts, active)
	}
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			c := readFile(filepath.Join(archDir, e.Name()))
			if c != "" {
				parts = append(parts, "<!-- archive: "+e.Name()+" -->\n"+c)
			}
		}
	}
	return strings.Join(parts, "\n\n---\n\n"), nil
}

func (a *fileAdapter) AppendMemory(project, entry string) error {
	path := filepath.Join(a.root, "projects", project, "memory.md")
	existing := readFile(path)
	updated := strings.TrimSpace(entry) + "\n\n---\n\n" + existing
	return os.WriteFile(path, []byte(updated), 0644)
}

func (a *fileAdapter) ArchiveMemory(project string, keepDays int) error {
	memPath := filepath.Join(a.root, "projects", project, "memory.md")
	archDir := filepath.Join(a.root, "projects", project, "memory-archive")
	content := readFile(memPath)
	if content == "" {
		return nil
	}
	cutoff := time.Now().AddDate(0, 0, -keepDays)
	entries := strings.Split(content, "\n\n---\n\n")
	var keep []string
	byMonth := map[string][]string{}

	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		d := parseEntryDate(e)
		if d.IsZero() || d.After(cutoff) {
			keep = append(keep, e)
		} else {
			m := d.Format("2006-01")
			byMonth[m] = append(byMonth[m], e)
		}
	}
	if len(byMonth) == 0 {
		return nil
	}
	if err := os.MkdirAll(archDir, 0755); err != nil {
		return err
	}
	for month, items := range byMonth {
		path := filepath.Join(archDir, month+".md")
		existing := readFile(path)
		var sb strings.Builder
		for _, item := range items {
			sb.WriteString(item)
			sb.WriteString("\n\n---\n\n")
		}
		sb.WriteString(existing)
		if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
			return err
		}
	}
	var sb strings.Builder
	for i, e := range keep {
		sb.WriteString(e)
		if i < len(keep)-1 {
			sb.WriteString("\n\n---\n\n")
		}
	}
	return os.WriteFile(memPath, []byte(sb.String()), 0644)
}

func (a *fileAdapter) Search(query, domain string) ([]SearchResult, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil, nil
	}
	var results []SearchResult
	for _, kind := range []string{"agent", "skill", "prompt"} {
		dir := filepath.Join(a.root, kind+"s")
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
				lang, domainFound := extractLangDomain(path, a.root, kind+"s")
				results = append(results, SearchResult{
					Kind: kind, Domain: domainFound, Lang: lang,
					Name: name, Excerpt: excerpt(content, q), Score: score(name, content, q),
				})
			}
			return nil
		})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	return results, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────
