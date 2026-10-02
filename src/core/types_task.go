// types_task.go — resto de las estructuras compartidas de core (Task,
// búsqueda, archivo/memoria, perfiles de LLM). Se separó de types.go solo
// para mantener cada archivo ≤300 líneas; el paquete y las firmas no
// cambian.
package core

import "encoding/json"

// Task defines a single operation within a project.
type Task struct {
	Prompt    string            `json:"prompt"`    // prompt file name, no extension
	Agents    []string          `json:"agents"`    // extra agents for this task
	Skills    []string          `json:"skills"`    // extra skills for this task
	Variables map[string]string `json:"variables"` // task-level variables: win over project/agents/skills variables
	Focus     []string          `json:"focus"`     // task-level focus (overrides global focus if set)
	// Exclude: mismo formato que Focus (nombre bare, ruta relativa,
	// ruta absoluta multiplataforma, glob) pero para EXCLUIR — un
	// archivo/directorio que matchea Exclude NUNCA se resuelve, sin
	// importar si Focus lo pide explícitamente. Ver
	// core.ResolveExclude / core/focus/resolvers/exclude.go.
	// Sobreescribe (no combina con) Project.Exclude si viene con al
	// menos un elemento, igual que Focus.
	Exclude []string      `json:"exclude"`
	Budget  *BudgetConfig `json:"budget"` // task-level budget ceiling (overrides project-level if set)
	// Graph: genera un grafo de dependencias AST (sin LLM) de focus/exclude.
	// String = archivo de salida (.png/.svg/.pdf; relativo a la carpeta de
	// project.json, o ruta absoluta/UNC); true = "graph.png" junto a
	// project.json; ""/false/ausente = no se genera. Ver core/graph_hook.go.
	Graph json.RawMessage `json:"graph,omitempty"`
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
	// Path and Line let a caller navigate to the EXACT place a match
	// was found, not just know that it exists — see mova.local/cli's
	// tui_search.go, which opens Path in the same file viewer/jumper
	// tui_fileview.go's ctrl+f already uses, landing directly on Line.
	// Absolute path; Line is 1-indexed, 0 if the match was only in the
	// file's Name (no in-content line to point to).
	Path string
	Line int
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
//
// See core/budget_config.go for BudgetConfig/SanitizeConfig/ResolveBudget
// — split into its own file once the Context Governance's fields pushed
// this one over the 300-line limit.

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
