// Package mpaths centralizes EVERY path Mova Context resolves from
// config/general/config.json — agents/skills/prompts/projects,
// cache_dir/temp_dir/output_dir — AND their optional per-project
// override in a project's own project.json ("paths": {...}, see
// core.Project.Paths) — so no other package hand-rolls its own "root +
// hardcoded folder name" join. Zero mova.local dependencies except
// documents (whose IsAbsCrossPlatform/NormalizeAbsPath are reused
// verbatim, see below), so any package (core, budget, cli, mcp,
// orchestrator, trace, logging...) can import mpaths with no risk of
// an import cycle — documents itself imports nothing else from this
// codebase. mpaths does NOT import core (core imports mpaths) — a
// project-level override is always passed in as a plain string by the
// caller (e.g. proj.Paths.Agents), never as a *core.Project, precisely
// to keep this package cycle-free.
//
// # Three-tier priority (agents/skills/prompts/cache_dir/temp_dir/output_dir)
//
// Every *DirForProject function in this file resolves with:
//
//  1. The CURRENT project's own project.json "paths.<field>", when
//     declared and non-blank (e.g. "paths": {"agents": "./mis-agentes"}).
//  2. config/general/config.json's own "<field>", when declared and
//     non-blank.
//  3. Mova's historical, hardcoded default (e.g. "<root>/agents").
//
// The plain *Dir functions (AgentsDir, SkillsDir, ...) implement ONLY
// tiers 2-3 — for callers with no specific project in scope (a project
// can't sensibly override where "projects" itself lives — see
// ProjectsDir, which has no *ForProject counterpart at all: it's
// circular by nature and stays config/general/config.json-only). Every
// tier uses the EXACT SAME cross-platform resolution rule (see below),
// just checked in a different order.
//
// # Resolution rule — IDENTICAL to project.json's own "repo" field
//
// Every field, at every tier, is resolved with the EXACT SAME
// cross-platform rule project.json's "repo" field already uses for
// write_file/create_directory/generate_* (see
// documents/pathresolve.go's IsAbsCrossPlatform + NormalizeAbsPath),
// on purpose — one convention, not several:
//
//  1. A recognized absolute path — Unix (leading "/"), Windows drive
//     letter ("C:\...", "D:/...", "C://..."), or UNC ("\\server\share")
//     — is honored EXACTLY as given, regardless of which OS Mova
//     itself is running on right now, via
//     documents.IsAbsCrossPlatform + documents.NormalizeAbsPath.
//  2. A Windows-style absolute path declared while Mova runs on a
//     non-Windows host (NormalizeAbsPath's own explicit error, e.g. a
//     shared config.json/project.json used by a mixed Windows/Linux
//     team) falls through to the NEXT tier instead of failing the
//     whole process — same "never break the process" philosophy as a
//     blank/missing field.
//  3. Anything else — a bare relative path with no leading marker at
//     all (e.g. "opt/projects", "agents", "./custom") — is joined
//     under Mova's root, exactly like "repo" defaults to project.json's
//     own directory today.
//
// # Hot reload — no restart, on every door (CLI, MCP, HTTP, Chat)
//
// Neither config/general/config.json NOR project.json is ever cached
// by this package or by core.Adapter.GetProject — both are re-read
// from disk on every single call (load() below; core/file_adapter.go's
// GetProject). This is a deliberate trade-off (both files are tiny),
// and it's exactly what makes editing either file, while a long-lived
// MCP/HTTP server is running, take effect on the very next request —
// no restart, no cache-invalidation logic to write or get wrong. The
// same trade-off config/lang/*.json already makes (see
// i18n/i18n_reload.go) and config/log/logging.json's LoadConfig.
//
// Concretely, against the sample config.json this feature ships with:
//
//	"agents": "C://agents"      -> Windows absolute, used as-is (on Windows);
//	                                falls back to "<root>/agents" elsewhere
//	"skills": "/skills"         -> Unix absolute "/skills" (same as "repo"
//	                                would resolve it) — NOT root-relative
//	"prompts": "/prompts"       -> Unix absolute "/prompts", same rule
//	"projects": "opt/projects"  -> no leading marker -> "<root>/opt/projects"
//	"cache_dir": "/.mova/cache" -> Unix absolute "/.mova/cache"
//
// This is a deliberate, explicit choice: config/general/config.json's
// (and project.json's "paths") fields behave exactly like "repo"
// (which real Mova users already know), not like a bespoke new
// convention — see docs/i18n/{es,en}/PATHS.md for the fully
// worked-through explanation and what a leading "/" actually means on
// each OS.
//
// Only ONE environment variable participates in path resolution
// anywhere in Mova: MOVA_PROJECT_ROOT (plus MOVA_PROJECT_PATH, its
// direct-path sibling) — both already existed in runtime/root.go
// before this feature, and both apply to finding Mova's ROOT itself,
// not to any of these fields. This package intentionally does NOT add
// MOVA_AGENTS_DIR/MOVA_SKILLS_DIR/etc.: config/general/config.json and
// project.json's own "paths" are the one and only places these fields
// are configured, exactly as requested.
package mpaths

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"mova.local/documents"
)

// GeneralConfig maps config/general/config.json exactly. Every field
// is optional — a blank value, a missing key, or the whole file being
// absent all fall back identically to Mova's existing default (see
// each accessor function below), never an error.
type GeneralConfig struct {
	Agents    string `json:"agents"`
	Skills    string `json:"skills"`
	Prompts   string `json:"prompts"`
	Projects  string `json:"projects"`
	CacheDir  string `json:"cache_dir"`
	TempDir   string `json:"temp_dir"`
	OutputDir string `json:"output_dir"`
}

// ConfigPath returns config/general/config.json under root.
func ConfigPath(root string) string {
	return filepath.Join(root, "config", "general", "config.json")
}

// load reads config/general/config.json under root. Any error
// (missing file, invalid JSON) yields nil — every caller below treats
// nil exactly like "no field configured", i.e. falls back to Mova's
// existing default. Intentionally NOT cached: this file is tiny, and
// re-reading it on every call is what makes it hot-reloadable (edit
// it, the next path resolved picks up the change — no restart), the
// same trade-off already made for config/lang/*.json (see
// i18n/i18n_reload.go), applied here for consistency.
func load(root string) *GeneralConfig {
	data, err := os.ReadFile(ConfigPath(root))
	if err != nil {
		return nil
	}
	var cfg GeneralConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	return &cfg
}

// AgentsDir resolves config.json's "agents", falling back to
// "<root>/agents" (today's default) when unconfigured. The folder's
// own NAME can be anything — Mova only ever looks it up via this
// field, never by assuming a literal "agents" basename.
func AgentsDir(root string) string { return dirFor(root, "agents", func(c *GeneralConfig) string { return c.Agents }) }

// SkillsDir resolves config.json's "skills", falling back to
// "<root>/skills" (today's default) when unconfigured.
func SkillsDir(root string) string { return dirFor(root, "skills", func(c *GeneralConfig) string { return c.Skills }) }

// PromptsDir resolves config.json's "prompts", falling back to
// "<root>/prompts" (today's default) when unconfigured.
func PromptsDir(root string) string { return dirFor(root, "prompts", func(c *GeneralConfig) string { return c.Prompts }) }

// ProjectsDir resolves config.json's "projects", falling back to
// "<root>/projects" (today's default — every project.json, memory.md,
// mova-budget-report.md etc. live under here) when unconfigured.
func ProjectsDir(root string) string {
	return dirFor(root, "projects", func(c *GeneralConfig) string { return c.Projects })
}

// CacheDir resolves config.json's "cache_dir", falling back to
// "<root>/.mova/cache" when unconfigured. General-purpose cache
// location — distinct from a project's own
// projects/<project>/mova-context-cache.json, which stays exactly
// where it is today (that file is project DATA, colocated with
// project.json on purpose, not a candidate for this general setting).
func CacheDir(root string) string {
	return dirFor(root, filepath.Join(".mova", "cache"), func(c *GeneralConfig) string { return c.CacheDir })
}

// TempDir resolves config.json's "temp_dir", falling back to
// "<root>/.mova/temp" when unconfigured.
func TempDir(root string) string {
	return dirFor(root, filepath.Join(".mova", "temp"), func(c *GeneralConfig) string { return c.TempDir })
}

// OutputDir resolves config.json's "output_dir", falling back to
// "<root>/output" when unconfigured. Most callers that generate
// reports/diagrams want ConfiguredOutputDir instead (see below) —
// this unconditional variant is for a caller with no other sensible
// default of its own.
func OutputDir(root string) string {
	return dirFor(root, "output", func(c *GeneralConfig) string { return c.OutputDir })
}

// ConfiguredOutputDir returns the resolved "output_dir" ONLY when it
// is actually configured and non-blank (and applies to the current
// OS); returns "" otherwise. This is what a command with its own
// historical "--output" fallback (context-trace's
// context-report.*/context-diagram.png — its only real consumer
// today) should check FIRST: "" means "config.json declares nothing
// usable here, keep your own existing default exactly as before" —
// config.json is only ever a PRIORITY override, never a replacement
// for a command's default when the operator hasn't configured one
// (see PROJECT_JSON.md and docs/i18n/{es,en}/PATHS.md).
//
// "output_dir" is intentionally NOT one of the fields the shipped
// config/general/config.json ships with a real value for (unlike
// agents/skills/prompts/projects/cache_dir/temp_dir, which affect many
// features and reproduce today's existing default when set to their
// factory value) — since it only affects one command's --output
// fallback, shipping it with a value would silently change that one
// command's everyday behavior for everyone. An operator can still add
// the key by hand at any time; this function (and the rest of the
// resolution rule) works exactly the same either way.
func ConfiguredOutputDir(root string) string {
	cfg := load(root)
	if cfg == nil {
		return ""
	}
	v := strings.TrimSpace(cfg.OutputDir)
	if v == "" {
		return ""
	}
	resolved, ok := resolveConfigured(root, v)
	if !ok {
		return ""
	}
	return resolved
}

// AgentsDirForProject resolves "agents" with the FULL 3-tier
// priority: projectOverride (project.json's own "paths.agents" — pass
// "" when the project declares nothing) > config/general/config.json's
// "agents" > "<root>/agents" (today's default). See dirForProject.
func AgentsDirForProject(root, projectOverride string) string {
	return dirForProject(root, projectOverride, "agents", func(c *GeneralConfig) string { return c.Agents })
}

// SkillsDirForProject is AgentsDirForProject for "skills"/"paths.skills".
func SkillsDirForProject(root, projectOverride string) string {
	return dirForProject(root, projectOverride, "skills", func(c *GeneralConfig) string { return c.Skills })
}

// PromptsDirForProject is AgentsDirForProject for "prompts"/"paths.prompts".
func PromptsDirForProject(root, projectOverride string) string {
	return dirForProject(root, projectOverride, "prompts", func(c *GeneralConfig) string { return c.Prompts })
}

// CacheDirForProject is AgentsDirForProject for "cache_dir"/"paths.cache_dir".
func CacheDirForProject(root, projectOverride string) string {
	return dirForProject(root, projectOverride, filepath.Join(".mova", "cache"), func(c *GeneralConfig) string { return c.CacheDir })
}

// TempDirForProject is AgentsDirForProject for "temp_dir"/"paths.temp_dir".
func TempDirForProject(root, projectOverride string) string {
	return dirForProject(root, projectOverride, filepath.Join(".mova", "temp"), func(c *GeneralConfig) string { return c.TempDir })
}

// ConfiguredOutputDirForProject is ConfiguredOutputDir's 3-tier
// counterpart: project.json's own "paths.output_dir" takes priority
// over config/general/config.json's "output_dir", which in turn is
// the only tier ConfiguredOutputDir itself checks. Returns "" only
// when NEITHER tier configures a usable value — same contract as
// ConfiguredOutputDir, so callers with their own historical default
// (context-trace) don't need to change how they interpret "".
func ConfiguredOutputDirForProject(root, projectOverride string) string {
	if v := strings.TrimSpace(projectOverride); v != "" {
		if resolved, ok := resolveConfigured(root, v); ok {
			return resolved
		}
		// project.json's override doesn't apply to this OS -> fall
		// through to config/general/config.json's own tier, same
		// "never break the process" philosophy as everywhere else in
		// this package.
	}
	return ConfiguredOutputDir(root)
}

// dirForProject layers projectOverride (tier 1) on top of dirFor's
// existing 2-tier config.json/default resolution (tiers 2/3) — the
// shared implementation behind every *DirForProject accessor above.
func dirForProject(root, projectOverride, defaultRelative string, get func(*GeneralConfig) string) string {
	if v := strings.TrimSpace(projectOverride); v != "" {
		if resolved, ok := resolveConfigured(root, v); ok {
			return resolved
		}
		// project.json's override doesn't apply to this OS (e.g. a
		// Windows drive letter declared while running on Linux/macOS)
		// -> fall through to config.json/default, same "never break
		// the process" philosophy as dirFor's own fallback below.
	}
	return dirFor(root, defaultRelative, get)
}

// dirFor is the shared implementation behind every *Dir accessor:
// config.json's field if present, non-blank, and valid for the
// current OS; otherwise defaultRelative — both go through the exact
// same absolute/relative rule (resolveConfigured), so a configured
// value and the fallback default behave identically (predictable, one
// code path, no special-casing).
func dirFor(root, defaultRelative string, get func(*GeneralConfig) string) string {
	if cfg := load(root); cfg != nil {
		if v := strings.TrimSpace(get(cfg)); v != "" {
			if resolved, ok := resolveConfigured(root, v); ok {
				return resolved
			}
			// Configured but not valid for this OS (e.g. a Windows
			// drive letter declared while running on Linux/macOS) —
			// fall through to defaultRelative, same as "not
			// configured at all", per this package's fallback
			// philosophy (see the package doc comment).
		}
	}
	resolved, _ := resolveConfigured(root, defaultRelative) // defaultRelative is always a valid bare relative path
	return resolved
}

// resolveConfigured applies "repo"'s own cross-platform rule (see the
// package doc comment) to one raw config.json value or built-in
// default fragment. ok is false only when raw recognizably targets a
// different OS than the one Mova is running on right now (a Windows
// drive letter on non-Windows) — callers fall back to their default
// in that case, they never propagate the error, matching this
// package's "always resolve to something, never break the process"
// contract.
func resolveConfigured(root, raw string) (path string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "." {
		return absClean(root), true
	}
	if documents.IsAbsCrossPlatform(raw) {
		normalized, err := documents.NormalizeAbsPath(raw)
		if err != nil {
			return "", false
		}
		return absClean(normalized), true
	}
	return absClean(filepath.Join(root, filepath.FromSlash(raw))), true
}

// absClean runs filepath.Abs (falling back to filepath.Clean alone if
// Abs ever errors, e.g. an unreadable cwd) so every path this package
// returns is absolute and fully normalized for the current OS. A
// value documents.NormalizeAbsPath already recognized as absolute
// (Unix "/skills", or a Windows drive/UNC path on Windows) is passed
// through filepath.Abs too, purely for filepath.Clean's normalization
// side effects (removing "..", duplicate separators, etc.) — Abs is a
// no-op on an already-absolute path other than that cleanup.
func absClean(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return filepath.Clean(p)
}

// ToDisplay renders p with forward slashes for cross-platform-
// readable console/report output (e.g. "mova context trace"'s status
// lines, or a generated project.json's own path fields) — the same
// normalized form regardless of which OS produced it.
func ToDisplay(p string) string {
	return filepath.ToSlash(p)
}
