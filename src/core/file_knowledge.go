// file_knowledge.go — búsqueda de agents/skills/prompts en disco (cascada de
// carpetas por dominio e idioma). Movido tal cual desde file_adapter.go
// (≤300 líneas por archivo).
package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"mova.local/mpaths"
)

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
//
// 10. recursive walk: root/             (finds custom/, etc.)
func (a *fileAdapter) GetKnowledge(kind, domain, lang, name string) (string, error) {
	return a.GetKnowledgeWithPathOverride(kind, domain, lang, name, "")
}

// GetKnowledgeWithPathOverride is GetKnowledge with an extra,
// highest-priority directory override — see core.Adapter's doc
// comment and core.ProjectPaths. override == "" behaves identically to
// GetKnowledge.
func (a *fileAdapter) GetKnowledgeWithPathOverride(kind, domain, lang, name, override string) (string, error) {
	content, _, err := a.LocateKnowledge(kind, domain, lang, name, override)
	return content, err
}

// LocateKnowledge is GetKnowledgeWithPathOverride that ALSO reports the
// real file the content came from (used by project.json "debug": true so
// the [debug] lines show the path that actually won, not a made-up
// default one).
//
// Resolution is a per-FILE cascade over up to three catalog directories,
// in strict priority order — the first directory that contains the file
// wins, later ones are only consulted if it is missing there:
//
//  1. project.json "paths.<kind>s"          (override, if declared)
//  2. config/general/config.json "<kind>s"  (or its default if blank)
//  3. <root>/<kind>s                        (historical default)
//
// Inside each directory the exact same 10-step search documented on
// GetKnowledge is used.
func (a *fileAdapter) LocateKnowledge(kind, domain, lang, name, override string) (content, path string, err error) {
	if domain == "" {
		domain = "base"
	}
	if lang == "" {
		lang = "es"
	}
	filename := name + ".md"

	for _, kd := range a.kindDirCascade(kind, override) {
		if c, p := findInKindDir(kd, domain, lang, filename); c != "" {
			return c, p, nil
		}
	}
	return "", "", fmt.Errorf("%s %q not found in domain %q", kind, name, domain)
}

// kindDirCascade returns the ordered, de-duplicated list of catalog
// directories to try for kind (see LocateKnowledge).
func (a *fileAdapter) kindDirCascade(kind, override string) []string {
	candidates := []string{
		kindDir(a.root, kind, override), // tier 1 (or 2/3 when override is blank/inapplicable)
		kindDir(a.root, kind, ""),       // tier 2: config.json / its default
		filepath.Join(a.root, kind+"s"), // tier 3: historical default
	}
	var out []string
	for _, d := range candidates {
		d = filepath.Clean(d)
		dup := false
		for _, o := range out {
			if o == d || (runtime.GOOS == "windows" && strings.EqualFold(o, d)) {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, d)
		}
	}
	return out
}

// findInKindDir runs GetKnowledge's 10-step search inside ONE catalog
// directory and returns the content plus the file it came from.
func findInKindDir(kd, domain, lang, filename string) (content, path string) {
	base := filepath.Join(kd, domain)

	candidates := []string{
		filepath.Join(base, "i18n", lang, filename),
	}
	if lang != "en" {
		candidates = append(candidates, filepath.Join(base, "i18n", "en", filename))
	}
	candidates = append(candidates, filepath.Join(base, lang, filename))
	if lang != "en" {
		candidates = append(candidates, filepath.Join(base, "en", filename))
	}
	candidates = append(candidates,
		filepath.Join(base, filename),
		filepath.Join(kd, filename),
	)
	for _, p := range candidates {
		if c := readFile(p); c != "" {
			return c, p
		}
	}

	walks := []string{filepath.Join(base, "i18n", lang)}
	if lang != "en" {
		walks = append(walks, filepath.Join(base, "i18n", "en"))
	}
	walks = append(walks, base, kd)
	for _, dir := range walks {
		if p := walkFindPath(dir, filename); p != "" {
			if c := readFile(p); c != "" {
				return c, p
			}
		}
	}
	return "", ""
}

// kindDir maps a knowledge kind ("agent"/"skill"/"prompt") to its
// configured directory via mpaths, with FULL 3-tier priority: override
// (a specific project's own project.json "paths.<field>", "" if none)
// > config/general/config.json's "agents"/"skills"/"prompts" > falls
// back to "<root>/<kind>s", the hardcoded default GetKnowledge always
// used before this existed. The folder's own NAME can be anything
// (mpaths might resolve to "catalogo-de-agentes-custom") —
// GetKnowledge only ever cares about the declared field, never the
// directory's literal basename.
func kindDir(root, kind, override string) string {
	switch kind {
	case "agent":
		return mpaths.AgentsDirForProject(root, override)
	case "skill":
		return mpaths.SkillsDirForProject(root, override)
	case "prompt":
		return mpaths.PromptsDirForProject(root, override)
	default:
		// Unknown kind (shouldn't happen — GetKnowledge is only ever
		// called with "agent"/"skill"/"prompt") — same default this
		// function replaces: "<root>/<kind>s".
		return filepath.Join(root, kind+"s")
	}
}

// walkFindPath walks dir recursively and returns the path of the first file named filename.
func walkFindPath(dir, filename string) string {
	var found string
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() == filename {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// walkFind walks dir recursively and returns the content of the first file named filename.
func walkFind(dir, filename string) string {
	if p := walkFindPath(dir, filename); p != "" {
		return readFile(p)
	}
	return ""
}
