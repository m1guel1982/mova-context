package mpaths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeGeneralConfig(t *testing.T, root, jsonBody string) {
	t.Helper()
	dir := filepath.Join(root, "config", "general")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(jsonBody), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

// TestFallback_NoConfigFile_UsesTodaysDefaults: absent config.json ->
// every accessor falls back to Mova's existing relative-to-root
// default, exactly as before this feature existed.
func TestFallback_NoConfigFile_UsesTodaysDefaults(t *testing.T) {
	root := t.TempDir()

	cases := map[string]func(string) string{
		"agents":   AgentsDir,
		"skills":   SkillsDir,
		"prompts":  PromptsDir,
		"projects": ProjectsDir,
	}
	for name, fn := range cases {
		got := fn(root)
		want := mustAbs(t, filepath.Join(root, name))
		if got != want {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
	}

	if got, want := CacheDir(root), mustAbs(t, filepath.Join(root, ".mova", "cache")); got != want {
		t.Errorf("cache_dir: got %q, want %q", got, want)
	}
	if got, want := TempDir(root), mustAbs(t, filepath.Join(root, ".mova", "temp")); got != want {
		t.Errorf("temp_dir: got %q, want %q", got, want)
	}
	if got, want := OutputDir(root), mustAbs(t, filepath.Join(root, "output")); got != want {
		t.Errorf("output_dir: got %q, want %q", got, want)
	}
	if got := ConfiguredOutputDir(root); got != "" {
		t.Errorf("ConfiguredOutputDir with no config.json: got %q, want \"\" (caller must keep its own default)", got)
	}
}

// TestFallback_BlankOrMissingKey_UsesDefault: config.json exists but a
// field is "" or simply absent from the JSON -> same fallback as no
// file at all, per the requirement's explicit "" / missing-key rule.
func TestFallback_BlankOrMissingKey_UsesDefault(t *testing.T) {
	root := t.TempDir()
	writeGeneralConfig(t, root, `{"agents": "", "skills": "/skills"}`) // "prompts"/"projects" simply absent

	if got, want := AgentsDir(root), mustAbs(t, filepath.Join(root, "agents")); got != want {
		t.Errorf("agents (blank): got %q, want %q", got, want)
	}
	if got, want := PromptsDir(root), mustAbs(t, filepath.Join(root, "prompts")); got != want {
		t.Errorf("prompts (absent key): got %q, want %q", got, want)
	}
	// "/skills" resolves EXACTLY like project.json's "repo" field would
	// (documents.IsAbsCrossPlatform) -> a genuine Unix absolute path,
	// NOT root-relative. See this package's doc comment.
	if got, want := SkillsDir(root), filepath.Clean("/skills"); got != want {
		t.Errorf("skills (configured, /skills): got %q, want %q", got, want)
	}
}

// TestExactSampleConfig_FromRequirement resolves EVERY field exactly
// as given in the requirement's own sample config.json, one by one,
// with the SAME semantics project.json's "repo" field already has.
func TestExactSampleConfig_FromRequirement(t *testing.T) {
	root := t.TempDir()
	writeGeneralConfig(t, root, `{
  "agents": "C://agents",
  "skills": "/skills",
  "prompts": "/prompts",
  "projects": "opt/projects",
  "cache_dir": "/.mova/cache",
  "temp_dir": "/.mova/temp",
  "output_dir": "/output"
}`)

	if runtime.GOOS == "windows" {
		// On Windows, "C://agents" is a real, applicable absolute path.
		if got, want := AgentsDir(root), filepath.Clean(`C:/agents`); got != want {
			t.Errorf("agents (C://agents) on windows: got %q, want %q", got, want)
		}
	} else {
		// On non-Windows, a Windows-style absolute path doesn't apply
		// to this OS -> falls back to the default, per this package's
		// "never break the process" contract.
		if got, want := AgentsDir(root), mustAbs(t, filepath.Join(root, "agents")); got != want {
			t.Errorf("agents (C://agents) on %s: got %q, want fallback %q", runtime.GOOS, got, want)
		}
	}

	// Unix-absolute values ("/skills", "/prompts", "/.mova/cache",
	// "/.mova/temp", "/output") resolve to themselves, literally, same
	// as "repo" already does for an identical value.
	if got, want := SkillsDir(root), filepath.Clean("/skills"); got != want {
		t.Errorf("skills (/skills): got %q, want %q", got, want)
	}
	if got, want := PromptsDir(root), filepath.Clean("/prompts"); got != want {
		t.Errorf("prompts (/prompts): got %q, want %q", got, want)
	}
	if got, want := CacheDir(root), filepath.Clean("/.mova/cache"); got != want {
		t.Errorf("cache_dir (/.mova/cache): got %q, want %q", got, want)
	}
	if got, want := TempDir(root), filepath.Clean("/.mova/temp"); got != want {
		t.Errorf("temp_dir (/.mova/temp): got %q, want %q", got, want)
	}
	if got, want := OutputDir(root), filepath.Clean("/output"); got != want {
		t.Errorf("output_dir (/output): got %q, want %q", got, want)
	}
	if got, want := ConfiguredOutputDir(root), filepath.Clean("/output"); got != want {
		t.Errorf("ConfiguredOutputDir (/output): got %q, want %q", got, want)
	}

	// "opt/projects" has no leading marker at all -> root-relative,
	// exactly like "repo" defaults a bare relative value.
	if got, want := ProjectsDir(root), mustAbs(t, filepath.Join(root, "opt", "projects")); got != want {
		t.Errorf("projects (opt/projects): got %q, want %q", got, want)
	}
}

// TestAbsolutePaths_WindowsDriveAndUNC covers every multiplatform
// absolute example the requirement lists explicitly, run on any host
// OS the same way "repo" already is.
func TestAbsolutePaths_WindowsDriveAndUNC(t *testing.T) {
	root := t.TempDir()
	writeGeneralConfig(t, root, `{
  "skills": "D:/mova/skills",
  "projects": "\\\\servidor\\red\\mova"
}`)

	if runtime.GOOS == "windows" {
		if got := SkillsDir(root); !strings.HasPrefix(got, "D:") {
			t.Errorf("skills (D:/mova/skills) on windows: got %q, expected a D: absolute path", got)
		}
		if got := ProjectsDir(root); !strings.HasPrefix(got, `\\servidor`) {
			t.Errorf("projects (UNC) on windows: got %q, expected a \\\\servidor... UNC path", got)
		}
	} else {
		// Neither applies to this OS -> both fall back to their
		// defaults, same "never break the process" contract.
		if got, want := SkillsDir(root), mustAbs(t, filepath.Join(root, "skills")); got != want {
			t.Errorf("skills (D:/... on %s): got %q, want fallback %q", runtime.GOOS, got, want)
		}
		if got, want := ProjectsDir(root), mustAbs(t, filepath.Join(root, "projects")); got != want {
			t.Errorf("projects (UNC on %s): got %q, want fallback %q", runtime.GOOS, got, want)
		}
	}
}

// TestFolderNameIsArbitrary: "the folder can have any name, Mova must
// be smart enough to retrieve the data via the declared field name" -
// AgentsDir must return whatever path was configured, whatever its
// basename, not assume it's literally named "agents".
func TestFolderNameIsArbitrary(t *testing.T) {
	root := t.TempDir()
	writeGeneralConfig(t, root, `{"agents": "catalogo-de-agentes-custom"}`)

	got := AgentsDir(root)
	want := mustAbs(t, filepath.Join(root, "catalogo-de-agentes-custom"))
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestConfiguredOutputDir_BlankOrMissing_ReturnsEmpty: callers with
// their own historical default (context-trace) must see "" so they
// know to keep it, not accidentally get "<root>/output" injected.
func TestConfiguredOutputDir_BlankOrMissing_ReturnsEmpty(t *testing.T) {
	root := t.TempDir()
	writeGeneralConfig(t, root, `{"output_dir": ""}`)
	if got := ConfiguredOutputDir(root); got != "" {
		t.Errorf("blank output_dir: got %q, want \"\"", got)
	}

	root2 := t.TempDir()
	writeGeneralConfig(t, root2, `{"agents": "/agents"}`) // no output_dir key at all
	if got := ConfiguredOutputDir(root2); got != "" {
		t.Errorf("missing output_dir key: got %q, want \"\"", got)
	}
}

// TestInvalidJSON_FallsBackSilently: a broken config.json must never
// stop Mova from running - same "silent fallback" contract as
// logging.LoadConfig and i18n.T's missing-key fallback elsewhere in
// this codebase.
func TestInvalidJSON_FallsBackSilently(t *testing.T) {
	root := t.TempDir()
	writeGeneralConfig(t, root, `{ this is not valid json`)

	if got, want := AgentsDir(root), mustAbs(t, filepath.Join(root, "agents")); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestToDisplay_UsesForwardSlashes covered above; below: 3-tier
// per-project priority (project.json's "paths.<field>" > config.json >
// default) — see AgentsDirForProject and friends.

func TestForProject_ProjectOverride_WinsOverConfigJSON(t *testing.T) {
	root := t.TempDir()
	writeGeneralConfig(t, root, `{"agents": "config-level-agents"}`)

	got := AgentsDirForProject(root, "./project-level-agents")
	want := mustAbs(t, filepath.Join(root, "project-level-agents"))
	if got != want {
		t.Errorf("got %q, want the project-level override %q (must win over config.json)", got, want)
	}
}

func TestForProject_BlankOverride_FallsThroughToConfigJSON(t *testing.T) {
	root := t.TempDir()
	writeGeneralConfig(t, root, `{"agents": "config-level-agents"}`)

	got := AgentsDirForProject(root, "") // project declares nothing
	want := mustAbs(t, filepath.Join(root, "config-level-agents"))
	if got != want {
		t.Errorf("got %q, want config.json's value %q", got, want)
	}
}

func TestForProject_NoOverrideNoConfigJSON_FallsThroughToDefault(t *testing.T) {
	root := t.TempDir() // no config/general/config.json at all

	got := SkillsDirForProject(root, "")
	want := mustAbs(t, filepath.Join(root, "skills"))
	if got != want {
		t.Errorf("got %q, want the historical default %q", got, want)
	}
}

func TestForProject_OverrideWrongOS_FallsThroughToConfigJSON(t *testing.T) {
	root := t.TempDir()
	writeGeneralConfig(t, root, `{"prompts": "config-level-prompts"}`)

	override := `C:\this-only-applies-on-windows`
	got := PromptsDirForProject(root, override)
	if runtime.GOOS == "windows" {
		if !strings.HasPrefix(got, "C:") {
			t.Errorf("on windows, expected the Windows override to apply, got %q", got)
		}
	} else {
		want := mustAbs(t, filepath.Join(root, "config-level-prompts"))
		if got != want {
			t.Errorf("got %q, want config.json's value %q (override doesn't apply to %s)", got, want, runtime.GOOS)
		}
	}
}

func TestConfiguredOutputDirForProject_ProjectOverride_WinsOverConfigJSON(t *testing.T) {
	root := t.TempDir()
	writeGeneralConfig(t, root, `{"output_dir": "config-level-output"}`)

	got := ConfiguredOutputDirForProject(root, "project-level-output")
	want := mustAbs(t, filepath.Join(root, "project-level-output"))
	if got != want {
		t.Errorf("got %q, want the project-level override %q", got, want)
	}
}

func TestConfiguredOutputDirForProject_NeitherTierConfigured_ReturnsEmpty(t *testing.T) {
	root := t.TempDir() // no config.json at all, no project override
	if got := ConfiguredOutputDirForProject(root, ""); got != "" {
		t.Errorf("got %q, want \"\" (caller must keep its own default)", got)
	}
}

func TestForProject_ArbitraryFolderName_StillWorksAtProjectLevel(t *testing.T) {
	root := t.TempDir()
	got := AgentsDirForProject(root, "un-catalogo-de-agentes-para-este-proyecto")
	want := mustAbs(t, filepath.Join(root, "un-catalogo-de-agentes-para-este-proyecto"))
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestToDisplay_UsesForwardSlashes(t *testing.T) {
	p := filepath.Join("a", "b", "c")
	got := ToDisplay(p)
	if strings.Contains(got, `\`) {
		t.Errorf("ToDisplay(%q) = %q, still contains a backslash", p, got)
	}
}
