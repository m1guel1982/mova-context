// types.go — shared data structures for Mova Context.
// Single source of truth. No duplication.
package main

// Project maps project.json exactly.
type Project struct {
	Project     string            `json:"project"`
	Description string            `json:"description"`
	Repo        string            `json:"repo"`
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
	Kind    string  // "agent" | "skill" | "prompt"
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

// MemoryDeleteRequest describes a delete operation (CLI → Adapter).
type MemoryDeleteRequest struct {
	All        bool
	Archived   bool
	Date       string
	From       string
	To         string
	KeepActive bool
}

func archiveEnabled(cfg *ArchiveConfig) bool {
	if cfg == nil || cfg.Enabled == nil {
		return true
	}
	return *cfg.Enabled
}

func confirmDeleteRequired(cfg *ArchiveConfig) bool {
	if cfg == nil || cfg.ConfirmDelete == nil {
		return true
	}
	return *cfg.ConfirmDelete
}

func retentionDays(cfg *ArchiveConfig) int {
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
type LLMProfile struct {
	Type      string `json:"type"`       // "powerful" | "local" (default: "powerful")
	Provider  string `json:"provider"`   // "claude" | "gpt" | "gemini" | "ollama" | "openai-compatible"
	Model     string `json:"model"`      // e.g. "claude-sonnet-4-6", "llama3.1", "mistral"
	MaxTokens int    `json:"max_tokens"` // 0 = no limit applied
	BaseURL   string `json:"base_url"`   // for ollama / openai-compatible endpoints
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
func (p *LLMProfile) isLocal() bool {
	if p == nil {
		return false
	}
	return p.Type == "local"
}
