// project_paths_test.go — regression coverage for project.json's own
// "paths" object (core.ProjectPaths): a project-level override for
// agents/skills/prompts/output_dir that takes
// PRIORITY over config/general/config.json, which itself falls back
// to Mova's historical default (see mova.local/mpaths's *ForProject
// functions). Exercises the full path end-to-end: project.json ->
// core.BuildContextSections -> core.Adapter.GetKnowledgeWithPathOverride
// -> mpaths — not just the low-level adapter method in isolation.
package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildFixtureProjectWithPaths is buildFixtureProject (engine_test.go)
// plus a SECOND, project-local set of agent/skill/prompt files under
// "custom-agents/custom-skills/custom-prompts" (deliberately different
// content, and a different basename than the global catalog, so a
// match can only come from the override actually being honored) and a
// project.json declaring "paths" to point at them.
func buildFixtureProjectWithPaths(t *testing.T) (root, projectName string) {
	t.Helper()
	root, projectName = buildFixtureProject(t)

	write := func(rel, content string) {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("custom-agents/dev.md", "# Dev agent (PROJECT OVERRIDE)\nOverridden via project.json paths.agents.\n")
	write("custom-skills/kiss.md", "# KISS (PROJECT OVERRIDE)\nOverridden via project.json paths.skills.\n")
	write("custom-prompts/greet.md", "# Greet (PROJECT OVERRIDE)\nOverridden via project.json paths.prompts for {{PROJECT}}.\n")

	write("projects/fixture/project.json", `{
		"project": "fixture",
		"repo": ".",
		"lang": "en",
		"default_task": "say-hi",
		"paths": {
			"agents": "custom-agents",
			"skills": "custom-skills",
			"prompts": "custom-prompts"
		},
		"agents": {"domain": "base", "use": ["dev"]},
		"skills": {"domain": "base", "use": ["kiss"]},
		"tasks": {
			"say-hi": {"prompt": "greet"}
		}
	}`)
	return root, projectName
}

func TestBuildContextSections_ProjectPathsOverride_UsesProjectLocalCatalog(t *testing.T) {
	root, projectName := buildFixtureProjectWithPaths(t)
	adapter := NewFileAdapter(root)

	sections, err := BuildContextSections(adapter, root, projectName, "")
	if err != nil {
		t.Fatalf("BuildContextSections: %v", err)
	}
	full := sections.Full()

	for _, marker := range []string{
		"PROJECT OVERRIDE", // appears in all three overridden files
	} {
		if !strings.Contains(full, marker) {
			t.Errorf("expected the project-local override content (marker %q) in the assembled context, got:\n%s", marker, full)
		}
	}
	if strings.Contains(full, "Be helpful.") || strings.Contains(full, "Keep it simple.") || strings.Contains(full, "Say hello to") {
		t.Errorf("found the GLOBAL catalog's content — the project.json \"paths\" override should have taken priority, got:\n%s", full)
	}
}

func TestGetProject_PathsField_ParsesCorrectly(t *testing.T) {
	root, projectName := buildFixtureProjectWithPaths(t)
	adapter := NewFileAdapter(root)

	proj, err := adapter.GetProject(projectName)
	if err != nil {
		t.Fatal(err)
	}
	if proj.Paths == nil {
		t.Fatal("expected proj.Paths to be non-nil")
	}
	if proj.Paths.Agents != "custom-agents" || proj.Paths.Skills != "custom-skills" || proj.Paths.Prompts != "custom-prompts" {
		t.Errorf("got %+v", proj.Paths)
	}
}

// TestGetProject_NoCache_HotReloadsPathsChanges: editing project.json's
// "paths" between two GetProject calls must be picked up immediately —
// no restart, no cache to invalidate (see mova.local/mpaths's package
// doc comment on why neither config/general/config.json nor
// project.json is ever cached).
func TestGetProject_NoCache_HotReloadsPathsChanges(t *testing.T) {
	root, projectName := buildFixtureProject(t) // no "paths" yet
	adapter := NewFileAdapter(root)

	proj1, err := adapter.GetProject(projectName)
	if err != nil {
		t.Fatal(err)
	}
	if proj1.Paths != nil {
		t.Fatalf("expected no \"paths\" declared yet, got %+v", proj1.Paths)
	}

	// Simulate an operator editing project.json while Mova (e.g. the
	// MCP/HTTP server) keeps running.
	pjPath := filepath.Join(root, "projects", projectName, "project.json")
	updated := `{
		"project": "fixture", "repo": ".", "lang": "en", "default_task": "say-hi",
		"paths": {"agents": "hot-reloaded-agents"},
		"agents": {"domain": "base", "use": ["dev"]},
		"skills": {"domain": "base", "use": ["kiss"]},
		"tasks": {"say-hi": {"prompt": "greet"}}
	}`
	if err := os.WriteFile(pjPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}

	proj2, err := adapter.GetProject(projectName)
	if err != nil {
		t.Fatal(err)
	}
	if proj2.Paths == nil || proj2.Paths.Agents != "hot-reloaded-agents" {
		t.Fatalf("expected the edited \"paths.agents\" to be picked up immediately, got %+v", proj2.Paths)
	}
}

// TestGetKnowledgeWithPathOverride_BlankOverride_MatchesGetKnowledge:
// override == "" must behave EXACTLY like the plain GetKnowledge (2nd
// and 3rd tiers only) — no behavior change for every project that
// doesn't declare "paths" at all.
func TestGetKnowledgeWithPathOverride_BlankOverride_MatchesGetKnowledge(t *testing.T) {
	root, _ := buildFixtureProject(t)
	adapter := NewFileAdapter(root)

	viaPlain, err1 := adapter.GetKnowledge("agent", "base", "en", "dev")
	viaOverride, err2 := adapter.GetKnowledgeWithPathOverride("agent", "base", "en", "dev", "")
	if err1 != nil || err2 != nil {
		t.Fatalf("errs: %v, %v", err1, err2)
	}
	if viaPlain != viaOverride {
		t.Errorf("GetKnowledge and GetKnowledgeWithPathOverride(..., \"\") must match exactly.\nplain:    %q\noverride: %q", viaPlain, viaOverride)
	}
}

// ── Prioridad project.json > config.json > default, con rutas ABSOLUTAS
// fuera del root (el caso de una instalación real: root en C:\appMovaContext,
// projects/agents en C:\opt\...). Se usan rutas absolutas del SO de test,
// que pasan por el mismo camino (IsAbsCrossPlatform) que "C:\\opt\\...".

type prioFixture struct {
	root, opt string
}

func newPrioFixture(t *testing.T, projectPaths, configAgents string) prioFixture {
	t.Helper()
	base := t.TempDir()
	f := prioFixture{root: filepath.Join(base, "app"), opt: filepath.Join(base, "opt")}
	w := func(p, c string) {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	w(filepath.Join(f.root, "workflow.md"), "#")
	w(filepath.Join(f.root, "agents/base/i18n/es/dev.md"), "DEFAULT-AGENT")
	w(filepath.Join(f.root, "skills/base/i18n/es/kiss.md"), "DEFAULT-SKILL")
	w(filepath.Join(f.root, "prompts/base/i18n/es/greet.md"), "DEFAULT-PROMPT")
	cfg := `{"projects":"` + filepath.ToSlash(filepath.Join(f.opt, "proyectos")) + `"`
	if configAgents != "" {
		cfg += `,"agents":"` + filepath.ToSlash(configAgents) + `"`
	}
	w(filepath.Join(f.root, "config/general/config.json"), cfg+`}`)
	paths := ""
	if projectPaths != "" {
		paths = `"paths":` + projectPaths + `,`
	}
	w(filepath.Join(f.opt, "proyectos/p/project.json"), `{"project":"p","repo":".","lang":"es","debug":true,
		"default_task":"t",`+paths+`
		"agents":{"domain":"base","use":["dev"]},"skills":{"domain":"base","use":["kiss"]},
		"tasks":{"t":{"prompt":"greet"}}}`)
	w(filepath.Join(f.opt, "proyectos/p/memory.md"), "m")
	return f
}

func (f prioFixture) write(t *testing.T, rel, content string) string {
	t.Helper()
	p := filepath.Join(f.opt, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func (f prioFixture) build(t *testing.T) *ContextSections {
	t.Helper()
	s, err := BuildContextSections(NewFileAdapter(f.root), f.root, "p", "")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func absJSON(p string) string { return `"` + filepath.ToSlash(p) + `"` }

func TestPriority_ProjectPathsWinOverConfigAndDefault(t *testing.T) {
	tmp := t.TempDir()
	cfgDir := filepath.Join(tmp, "cfg-agents")
	localAgents := filepath.Join(tmp, "local-agents")
	for dir, txt := range map[string]string{cfgDir: "CONFIG-AGENT", localAgents: "PROJECT-AGENT"} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "dev.md"), []byte(txt), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	f := newPrioFixture(t, `{"agents":`+absJSON(localAgents)+`}`, cfgDir)

	s := f.build(t)
	full := s.Full()
	if !strings.Contains(full, "PROJECT-AGENT") || strings.Contains(full, "CONFIG-AGENT") || strings.Contains(full, "DEFAULT-AGENT") {
		t.Fatalf("project.json debe ganar sobre config.json y default:\n%s", full)
	}
	if !strings.Contains(s.DebugLog, filepath.Join(localAgents, "dev.md")) {
		t.Errorf("[debug] debe mostrar la ruta REAL cargada, got:\n%s", s.DebugLog)
	}
	if strings.Contains(s.DebugLog, "agents/base/i18n/es/dev.md") {
		t.Errorf("[debug] no debe mostrar una ruta por defecto inventada:\n%s", s.DebugLog)
	}
}

func TestPriority_ConfigJSONWinsOverDefault_WhenProjectDeclaresNothing(t *testing.T) {
	cfgDir := filepath.Join(t.TempDir(), "cfg-agents")
	os.MkdirAll(cfgDir, 0o755)
	os.WriteFile(filepath.Join(cfgDir, "dev.md"), []byte("CONFIG-AGENT"), 0o644)
	f := newPrioFixture(t, "", cfgDir)

	full := f.build(t).Full()
	if !strings.Contains(full, "CONFIG-AGENT") || strings.Contains(full, "DEFAULT-AGENT") {
		t.Fatalf("config.json debe ganar sobre el default:\n%s", full)
	}
}

func TestPriority_NothingDeclared_UsesDefaultCatalog(t *testing.T) {
	f := newPrioFixture(t, "", "")
	full := f.build(t).Full()
	if !strings.Contains(full, "DEFAULT-AGENT") || !strings.Contains(full, "DEFAULT-SKILL") || !strings.Contains(full, "DEFAULT-PROMPT") {
		t.Fatalf("sin nada declarado debe usar el catálogo de siempre:\n%s", full)
	}
}

// Si el archivo NO existe en la carpeta del proyecto, cae al siguiente
// nivel (config.json → default) en vez de fallar.
func TestPriority_MissingFileInProjectDir_FallsThroughPerFile(t *testing.T) {
	localAgents := filepath.Join(t.TempDir(), "vacio")
	os.MkdirAll(localAgents, 0o755) // existe pero no tiene dev.md
	f := newPrioFixture(t, `{"agents":`+absJSON(localAgents)+`}`, "")

	full := f.build(t).Full()
	if !strings.Contains(full, "DEFAULT-AGENT") {
		t.Fatalf("archivo ausente en paths.agents debe caer al catálogo por defecto:\n%s", full)
	}
}
