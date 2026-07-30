// types.go — shared data structures for Mova Context.
// Single source of truth. No duplication.
package core

import "path/filepath"

// Project maps project.json exactly.
type Project struct {
	Project     string            `json:"project"`
	Description string            `json:"description"`
	Repo        string            `json:"repo"` // the project's single repository — for more than one directory inside it, use "focus" (see ResolveFocus), not a second repo
	Lang        string            `json:"lang"` // "es", "en", "fr", "" (legacy)
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
	Budget      *BudgetConfig     `json:"budget"`  // optional: token ceiling for `mova budget` (see BudgetConfig)
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
	TokenHistoryPath string       `json:"token_history_path,omitempty"`
	Tools            *ToolsConfig `json:"tools"` // optional: lets mova chat / chat_completion call MCP file/document tools mid-conversation (see ToolsConfig)
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

// ToolsConfig turns "mova chat" (and the MCP "chat_completion" tool) into
// a small agent: when enabled, the model can ask Mova — in plain text,
// using a simple marker-based protocol described in
// mova.local/mcp/agent_tools.go — to create directories/files, write a
// .docx/.pdf/.xlsx/.svg, patch an existing file, etc., and keep
// answering using the real result. Works with ANY provider (Ollama,
// Gemini, Claude, GPT...) because it doesn't rely on each API's native
// function-calling format — same "simplicidad" principle as the rest of
// Mova: one plain-text protocol, three doors (CLI/MCP/HTTP).
type ToolsConfig struct {
	Enabled bool     `json:"enabled"`         // default false — opt-in per project
	Allow   []string `json:"allow,omitempty"` // optional whitelist (subset of mova.local/mcp.AgentToolNames()); empty/omitted = all of them allowed
}

// ToolsEnabled reports whether a project turned on chat tool-calling.
func ToolsEnabled(cfg *ToolsConfig) bool {
	return cfg != nil && cfg.Enabled
}

// KnowledgeRef points to agents/skills: domain + list of names.
type KnowledgeRef struct {
	Domain string   `json:"domain"` // e.g. "software", "callcenter", "legal"
	Use    []string `json:"use"`    // file names without extension
	Custom []string `json:"custom"` // custom overrides (optional)
}

// Task defines a single operation within a project.
type Task struct {
	Prompt    string            `json:"prompt"`    // prompt file name, no extension
	Agents    []string          `json:"agents"`    // extra agents for this task
	Skills    []string          `json:"skills"`    // extra skills for this task
	Variables map[string]string `json:"variables"` // task-level variable overrides
	Focus     []string          `json:"focus"`     // task-level focus (overrides global focus if set)
	Budget    *BudgetConfig     `json:"budget"`    // task-level budget ceiling (overrides project-level if set)
}

// ProjectSummary is used by mova list.
type ProjectSummary struct {
	Name        string
	Description string
	Lang        string
	Tasks       []string
}

// SearchResult is returned by mova search and MCP search_context.
type SearchResult struct {
	Kind    string // "agent" | "skill" | "prompt"
	Domain  string
	Lang    string
	Name    string
	Excerpt string
	Score   float64
}

// ArchiveConfig maps project.json "archive" block.
type ArchiveConfig struct {
	Enabled        *bool  `json:"enabled"`          // default true
	RetentionDays  int    `json:"retention_days"`   // default 30
	KeepMemoryOnly bool   `json:"keep_memory_only"` // true = delete archives, keep memory.md
	CleanupPolicy  string `json:"cleanup_policy"`   // "manual" (default) | "auto"
	ConfirmDelete  *bool  `json:"confirm_delete"`   // default true
}

// BudgetConfig sets an optional token ceiling for `mova budget` — a soft
// (actually hard, see EnforceLimit) limit on the ASSEMBLED CONTEXT size
// (agents+skills+prompt+focus+memory), checked by mova.local/budget like
// a linter: "this project's context grew past what you budgeted for".
// This is a completely different knob from the model config's own
// "num_predict" (config/models/<provider>/<config>.json) — that one caps
// how many tokens the MODEL's own REPLY may generate, applied per-request
// by the provider itself, not by Mova. Two different "max size" concepts
// that are easy to confuse because of the similar names:
//
//	budget.max_tokens (this struct, project.json)        → INPUT  ceiling, enforced by Mova BEFORE sending anything
//	model_config.num_predict (config/models/.../*.json)  → OUTPUT ceiling, sent to the provider AS a request parameter
type BudgetConfig struct {
	MaxTokens int `json:"max_tokens"` // 0 = no limit configured, mova budget never flags anything
}

// ResolveBudget decides which BudgetConfig applies to a run: the task's
// own `budget` (if set) wins and REPLACES the project's, same rule as
// ResolveFocus. Returns nil if neither declares one — "no limit
// configured" is not the same as "limit of 0".
func ResolveBudget(proj *Project, task *Task) *BudgetConfig {
	if task.Budget != nil {
		return task.Budget
	}
	return proj.Budget
}

// MemoryDeleteRequest describes a delete operation (CLI → Adapter).
type MemoryDeleteRequest struct {
	All        bool
	Archived   bool
	Date       string
	From       string
	To         string
	KeepActive bool
}

func ArchiveEnabled(cfg *ArchiveConfig) bool {
	if cfg == nil || cfg.Enabled == nil {
		return true
	}
	return *cfg.Enabled
}

func ConfirmDeleteRequired(cfg *ArchiveConfig) bool {
	if cfg == nil || cfg.ConfirmDelete == nil {
		return true
	}
	return *cfg.ConfirmDelete
}

func RetentionDays(cfg *ArchiveConfig) int {
	if cfg == nil || cfg.RetentionDays <= 0 {
		return 30
	}
	return cfg.RetentionDays
}

// LLMProfile controls how the engine formats context for different model capabilities.
// Powerful models (Claude, GPT-4, Gemini) handle rich, dense context well.
// Local models (Llama, Mistral, Phi, Qwen, Gemma, DeepSeek) benefit from
// explicit, sequential, less-ambiguous formatting.
//
// This is the ONLY place where the LLM type influences behavior.
// Agents, Skills, Prompts, and workflow.md never change.
// Single source of truth: "provider" + "config" is a POINTER, nothing
// more — it names config/models/<provider>/<config>.json, the one file
// that holds the actual connection details (base_url, api_key, timeout)
// AND inference parameters (temperature, num_predict, the real model
// tag...) for that model. Nothing about the model is duplicated here.
// (Older projects used "model" + "max_tokens" + "base_url" directly on
// this struct; those fields are gone — "max_tokens" was never wired to
// anything besides this struct itself, since every provider already
// reads its output-token cap from the model config's own "num_predict",
// and "base_url" now lives there too.)
type LLMProfile struct {
	Type     string `json:"type"`               // "powerful" | "local" (default: "powerful") — only knob that changes CONTEXT FORMATTING, see adaptContent. Unrelated to provider identity (see Provider below).
	Provider string `json:"provider,omitempty"` // OPTIONAL. "ollama" | "google" | "anthropic" | "openai" | "lmstudio" | ... — a subfolder of config/models/. When omitted, it is resolved automatically from "config" (see models.ResolveConfigProvider) by locating the one provider folder that has that file — the provider's real identity then comes from that single file's own "type" field (e.g. "google", "anthropic", "openai-compatible", "ollama"), never duplicated here. Set this explicitly only to disambiguate a "config" filename that exists under more than one provider folder.
	Config   string `json:"config"`             // filename (no .json) under config/models/<provider>/ — e.g. "llama3.2.3b", "gemini-2.5-flash"
}

// EmbeddingProfile configures the model used to generate vector embeddings.
// Used for semantic search over agents/skills/prompts and memory.
// Entirely optional — when absent, search falls back to keyword matching.
//
// Typical models:
//   - bge-m3               (multilingual, Ollama — ideal for corpora ES+EN mezclados)
//   - nomic-embed-text      (English-focused, lightweight, Ollama)
//   - text-embedding-3-small (OpenAI)
type EmbeddingProfile struct {
	Provider string `json:"provider"` // "ollama" | "openai" | "openai-compatible"
	Model    string `json:"model"`    // e.g. "bge-m3", "nomic-embed-text"
	BaseURL  string `json:"base_url"` // required for ollama / openai-compatible
	Dims     int    `json:"dims"`     // output dimensions (0 = model default)
}

// RerankerProfile configures a cross-encoder model to rerank retrieval results.
// Applied after embedding search to improve precision.
// Entirely optional — when absent, embedding cosine scores are used as-is.
//
// Typical models:
//   - bge-reranker-v2-m3     (multilingual, best pair for bge-m3)
//   - ms-marco-MiniLM-L-6-v2 (English, very fast)
type RerankerProfile struct {
	Provider string  `json:"provider"`  // "ollama" | "openai-compatible"
	Model    string  `json:"model"`     // e.g. "bge-reranker-v2-m3"
	BaseURL  string  `json:"base_url"`  // endpoint
	MinScore float64 `json:"min_score"` // discard results below this score (0.0–1.0)
}

// isLocal returns true when the profile targets a local/smaller model.
func (p *LLMProfile) IsLocal() bool {
	if p == nil {
		return false
	}
	return p.Type == "local"
}
