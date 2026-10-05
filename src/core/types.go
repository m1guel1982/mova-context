// types.go — shared data structures for Mova Context.
// Single source of truth. No duplication.
package core

import (
	"encoding/json"
	"path/filepath"
)

// Project maps project.json exactly.
type Project struct {
	Project     string `json:"project"`
	Description string `json:"description"`
	// Author: who authorizes this project's governance policy — audit
	// question "who authorized it?" (see README § Audit Matrix, #3,
	// and core.ResolvePolicyAuthor). Empty resolves to
	// "system:default", never left without an owner.
	Author      string            `json:"author,omitempty"`
	Repo        string            `json:"repo"`        // the project's single repository — for more than one directory inside it, use "focus" (see ResolveFocus), not a second repo
	Lang        string            `json:"lang"`        // "es", "en", "fr", "" (legacy)
	Adapter     string            `json:"adapter"`     // "file" | "db"
	DSN         string            `json:"dsn"`         // database connection string
	LLM         string            `json:"llm"`         // legacy: "claude" | "gpt" | "ollama" (still works)
	LLMProfile  *LLMProfile       `json:"llm_profile"` // optional: full LLM configuration
	Embedding   *EmbeddingProfile `json:"embedding"`   // optional: embedding model for semantic search
	Reranker    *RerankerProfile  `json:"reranker"`    // optional: reranker model for precision boost
	DefaultTask string            `json:"default_task"`
	Variables   map[string]string `json:"variables"`
	Agents      KnowledgeRef      `json:"agents"`
	Skills      KnowledgeRef      `json:"skills"`
	Tasks       map[string]Task   `json:"tasks"`
	Archive     *ArchiveConfig    `json:"archive"` // optional memory management config
	Focus       []string          `json:"focus"`   // files/dirs/symbols to work on — the way to scope to part of "repo", instead of a second repo (see "5. `save` and 9. `save` — Focus" in COMMANDS.md)
	// Exclude: MISMO formato/soporte multiplataforma que "focus"
	// (nombre bare como "node_modules"/".git", ruta relativa como
	// "src/secrets", ruta absoluta del host — "C:\\secrets",
	// "D:\\private", "/mnt/private" — o glob como "*.env"/"**/*.pem")
	// pero para EXCLUSIÓN: cualquier archivo/directorio que matchee un
	// patrón de "exclude" NUNCA se resuelve — ni por un target
	// explícito de "focus", ni al recorrer un directorio/glob — y por
	// lo tanto nunca se agrega a mova-context-cache.json. Ver
	// core.ResolveExclude / core/focus/resolvers/exclude.go.
	Exclude []string      `json:"exclude"`
	Budget  *BudgetConfig `json:"budget"` // optional: token ceiling for `mova budget` (see BudgetConfig)
	// WorkflowPath: where workflow.md lives for this project (see "5./6.
	// workflow.md" in the spec). A single path — once configured, that
	// file is always used: Mova never searches for another workflow.md.
	// See mova.local/budget.LoadWorkflow for the full resolution + Budget
	// gate pipeline.
	WorkflowPath string `json:"workflow_path,omitempty"`
	// BudgetPath: where mova-budget-report.md is written (see "11.
	// budget_path"). Replaces config/prices.json's old "report_path" —
	// see mova.local/budget.BudgetReportPath.
	BudgetPath string `json:"budget_path,omitempty"`
	// TokenHistoryPath: where mova-token-history.json is written (see
	// "10. token_history_path") — see mova.local/budget.HistoryPath.
	TokenHistoryPath string `json:"token_history_path,omitempty"`
	// MemoryPath: where memory.md lives for this project — same
	// resolution rule as BudgetPath/WorkflowPath (absolute path used
	// as-is, relative path joined to MOVA_PROJECT_ROOT). "" (the
	// default) keeps the existing projects/<project>/memory.md
	// location, so every project that doesn't set this is 100%
	// unaffected. See mova.local/core.MemoryPath.
	MemoryPath string `json:"memory_path,omitempty"`
	// Memory: registro automático de memoria — bool o ruta (ver memory_setting.go).
	Memory json.RawMessage `json:"memory,omitempty"`
	// MemoryMaxChars: tope de memoria inyectada (0 = por defecto; ver memory_view.go).
	MemoryMaxChars int `json:"memory_max_chars,omitempty"`
	// Apply: las respuestas del modelo pueden MODIFICAR archivos del repo
	// previa confirmación — bool u objeto {"enabled","backup"} (ver
	// apply_policy.go y mova.local/applyflow). Task.Apply lo anula por tarea.
	Apply json.RawMessage `json:"apply,omitempty"`
	// Debug: when true, every door (chat, mova ui chat, CLI, HTTP API,
	// MCP) prints what it resolved before running a task — repo path;
	// each agent/skill/prompt's name AND resolved file path (or
	// "inline" for free-text entries, see resolveKnowledgeOrLiteral);
	// the exact "focus" and "exclude" entries with their resolved
	// absolute paths. Defaults to false and is intentionally NEVER
	// written into a generated project.json template — see
	// docs/PROJECT.md § debug for the full explanation; this key is
	// documentation-only until a person adds it by hand.
	Debug bool         `json:"debug,omitempty"`
	Tools *ToolsConfig `json:"tools"` // optional: lets mova chat / chat_completion call MCP file/document tools mid-conversation (see ToolsConfig)
	// Diagram: optional visual-diagram preferences (see
	// mova.local/diagram and `mova run <project> --diagram`). Nil/
	// absent = every default (verbose detail, svg export) applies —
	// same "declare nothing, get the safe default" rule every other
	// optional block in this struct follows.
	Diagram *DiagramConfig `json:"diagram,omitempty"`
	// FocusDisplayLimit: cuántos nombres de archivo/directorio de
	// `focus` muestra la línea de estado "[Focus] Selected ..." (`mova
	// chat`, tool MCP/HTTP chat_completion, Mova UI) antes de colapsar
	// el resto en un badge "+N" — ver core.FocusDisplayLimit.
	// 0/ausente = 2 (el default de fábrica). Cualquier número
	// configurado se respeta tal cual: al superarlo SIEMPRE aparece el
	// "+N", sea cual sea el límite.
	FocusDisplayLimit int `json:"focus_display_limit,omitempty"`
	// EgressAudit: optional pre-provider audit log of the SANITIZED
	// context about to leave for an LLM, and/or a dry-run switch that
	// skips the actual provider call — see EgressAuditConfig and
	// core.ResolveEgressAudit. Nil/absent = fully disabled (DryRun
	// false, OutputFile ""), zero behavior change — same "declare
	// nothing, get today's behavior" rule as Diagram/Archive above.
	EgressAudit *EgressAuditConfig `json:"egress_audit,omitempty"`

	// Paths: optional PER-PROJECT override of the same 6 directories
	// config/general/config.json declares globally
	// (agents/skills/prompts/output_dir — see
	// mova.local/mpaths). Nil/absent, or any individual field left ""
	// = falls through to config/general/config.json's own value for
	// that field, which itself falls through to Mova's historical
	// default — same 3-tier "declare nothing, get today's behavior"
	// rule as every other optional block here, just with one extra
	// tier. There is deliberately NO "projects" field here: a project
	// declaring where ALL projects live, inside itself, is circular —
	// that one stays config/general/config.json-only (see
	// mpaths.ProjectsDir). See docs/i18n/{es,en}/PATHS.md § jerarquía
	// por proyecto.
	Paths *ProjectPaths `json:"paths,omitempty"`

	// Policies: optional per-project policy selection that COMPLETELY
	// overrides config/policy.json's own list (see PolicySelector and
	// ResolvePolicyRequest). Nil/absent means this project
	// declares no policy selection of its own — see
	// ResolvePolicyRequest for exactly what that implies.
	Policies *PolicySelector `json:"policies,omitempty"`
}

// PolicySelector maps project.json's (and config/policy.json's)
// "policies" value. Two equivalent JSON shapes are accepted so the
// simple case stays simple and nothing already in the wild breaks:
//
//	"policies": ["security.json", "review.json"]
//	"policies": { "include": ["security.json"], "exclude": ["pii_strict.json"] }
//
// The bare-array form is read as Include with an empty Exclude.
// See ResolvePolicyRequest for resolution/precedence and
// docs/i18n/{es,en}/PROJECT_JSON.md § policies for the user-facing
// contract.
type PolicySelector struct {
	// Include: policy files to load, in order. Each entry is either a
	// bare file name resolved by RECURSIVE search under config/policy/
	// ("pii_strict.json"), a relative path resolved against the Mova
	// root ("config/custom/ventas.json"), or a cross-platform absolute
	// path (Unix "/etc/...", Windows "C:\...", UNC "\\server\share\...").
	Include []string `json:"include"`
	// Exclude: file names (or paths) to drop even when Include — or a
	// recursive match — would otherwise have picked them up. Matched by
	// base file name, case-insensitively, so "pii_strict.json" excludes
	// it no matter which directory it was found in.
	Exclude []string `json:"exclude"`
	// declared records that a "policies" key was physically present in
	// the JSON, which is different from it being present but empty —
	// see UnmarshalJSON and ResolvePolicyRequest.
	declared bool
}

// Declared reports whether a "policies" key was actually present in the
// source JSON (as opposed to absent entirely).
func (p *PolicySelector) Declared() bool { return p != nil && p.declared }

// UnmarshalJSON accepts both documented shapes (bare array / object).
func (p *PolicySelector) UnmarshalJSON(data []byte) error {
	p.declared = true
	var asArray []string
	if err := json.Unmarshal(data, &asArray); err == nil {
		p.Include = asArray
		p.Exclude = nil
		return nil
	}
	var asObject struct {
		Include []string `json:"include"`
		Exclude []string `json:"exclude"`
	}
	if err := json.Unmarshal(data, &asObject); err != nil {
		return err
	}
	p.Include, p.Exclude = asObject.Include, asObject.Exclude
	return nil
}

// EgressAuditConfig maps project.json's optional "egress_audit" object:
//
//	"egress_audit": { "dry_run": true, "output_file": ".mova/egress_sanitized.log" }
//
// See core.ResolveEgressAudit for how OutputFile is resolved to an
// absolute path (always relative to THIS project's own directory, never
// the process's working directory) and models.Session.Send/SendStream
// for where it's actually written and DryRun is honored — one
// implementation shared by CLI/Chat, MCP, and HTTP (see PROJECT_JSON.md
// § egress_audit).
type EgressAuditConfig struct {
	// DryRun: when true, the sanitized context is (optionally) logged
	// but the LLM provider is never called — Send/SendStream return a
	// confirmation reply instead. Defaults to false.
	DryRun bool `json:"dry_run,omitempty"`
	// OutputFile: where to append the sanitized-context audit log.
	// Relative paths resolve under projects/<project>/ (NOT the
	// working directory). "" (the default) disables audit logging
	// entirely, independent of DryRun.
	OutputFile string `json:"output_file,omitempty"`
}

// ProjectPaths maps project.json's optional "paths" object — see
// core.Project.Paths's doc comment for the 3-tier priority
// (project.json > config/general/config.json > Mova's historical
// default) and mova.local/mpaths for the actual resolution (same
// cross-platform rule as "repo": a recognized Windows/UNC/Unix
// absolute path is used as-is; anything else resolves relative to
// Mova's root — see docs/i18n/{es,en}/PATHS.md).
type ProjectPaths struct {
	Agents    string `json:"agents,omitempty"`
	Skills    string `json:"skills,omitempty"`
	Prompts   string `json:"prompts,omitempty"`
	CacheDir  string `json:"cache_dir,omitempty"`
	TempDir   string `json:"temp_dir,omitempty"`
	OutputDir string `json:"output_dir,omitempty"`
}

// forKind returns this project's own directory override for a
// knowledge kind ("agent"/"skill"/"prompt"), nil-safe — p == nil or an
// unrecognized kind both return "", the same as "this project declares
// no override", so every caller can call this unconditionally without
// its own nil check.
func (p *ProjectPaths) forKind(kind string) string {
	if p == nil {
		return ""
	}
	switch kind {
	case "agent":
		return p.Agents
	case "skill":
		return p.Skills
	case "prompt":
		return p.Prompts
	default:
		return ""
	}
}

// DiagramConfig maps project.json's optional "diagram" object — see
// mova.local/diagram.BuildDiagram, which reads DetailLevel (CLI's
// --diagram flag can still override this per-run) and ExportFormats
// (the default format list when `mova run <project> --diagram` is
// given without --export).
type DiagramConfig struct {
	DetailLevel   string   `json:"detail_level,omitempty"`   // "simple" | "verbose" (default "verbose")
	ExportFormats []string `json:"export_formats,omitempty"` // e.g. ["svg"], ["svg","png","pdf"] — default ["svg"] when both this and --export are absent
}

// ResolveWorkflowPath decides which workflow.md file applies to a run of
// this project: an explicit path (typed by the person, e.g.
// "workflow.md <project> <task>" naming a file, or --workflow) always
// wins; otherwise proj.WorkflowPath ("workflow_path" in project.json) is
// used; otherwise a plain "workflow.md" at the Mova root is the default,
// same as today. Once a path is configured, Mova never searches for a
// different file.
func ResolveWorkflowPath(root string, proj *Project, explicit string) string {
	resolve := func(p string) string {
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(root, p)
	}
	if explicit != "" {
		return resolve(explicit)
	}
	if proj != nil && proj.WorkflowPath != "" {
		return resolve(proj.WorkflowPath)
	}
	return filepath.Join(root, "workflow.md")
}

// KnowledgeRef points to agents/skills: domain + list of names.
type KnowledgeRef struct {
	Domain string   `json:"domain"` // e.g. "software", "callcenter", "legal"
	Use    []string `json:"use"`    // file names without extension
	Custom []string `json:"custom"` // custom overrides (optional)
	// Variables: any KEY/value pairs injected into this block's markdown
	// as ${KEY} or {{KEY}} — same engine as task.variables (see inject).
	Variables map[string]string `json:"variables,omitempty"`
}
