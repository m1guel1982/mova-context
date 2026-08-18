// budget_config.go — BudgetConfig, SanitizeConfig, and ResolveBudget,
// split out of types.go once the Token Firewall's new fields pushed
// that file over the 300-line limit. Conceptually still part of
// Project's "budget" field (see types.go's Project struct) — just
// filed separately for size, the same reason job/multiagent types got
// their own files earlier in this codebase's history.
package core

// BudgetConfig maps project.json's (or a task's) "budget" object.
// Two different "max size" concepts that are easy to confuse because
// of the similar names:
//
//	budget.max_tokens (this struct, project.json)        → INPUT  ceiling, enforced by Mova BEFORE sending anything
//	model_config.num_predict (config/models/.../*.json)  → OUTPUT ceiling, sent to the provider AS a request parameter
type BudgetConfig struct {
	MaxTokens int `json:"max_tokens"` // 0 = no limit configured, mova budget never flags anything

	// ── Token Firewall (see mova.local/budget's gated_context.go,
	// spend.go, cachelayout.go) — all optional, all off unless declared,
	// zero behavior change for a project.json that doesn't set them. ──

	// MaxTokensPerRun: circuit breaker ceiling for a SINGLE run/request
	// — distinct from MaxTokens, which is a content-size gate checked
	// against the assembled context regardless of history. 0 = unset.
	MaxTokensPerRun int `json:"max_tokens_per_run"`

	// MaxMonthlyUSD: circuit breaker ceiling for this project's total
	// spend in the current calendar month, tracked in mova-spend.json
	// (see budget.SpendState). 0 = unset.
	MaxMonthlyUSD float64 `json:"max_monthly_usd"`

	// OnExceed: what the circuit breaker does when either ceiling above
	// is hit. "warn" (default) reports it but still runs; "abort" stops
	// BEFORE anything is sent to a model.
	OnExceed string `json:"on_exceed"`

	// Sanitize: the noise-removal stage (see mova.local/sanitize).
	// nil = DefaultConfig() (enabled, conservative — logs/blank-line
	// cleanup on, comment stripping off).
	Sanitize *SanitizeConfig `json:"sanitize"`

	// CacheHint: when true (default — nil means true, see
	// core.CacheGuardEnabled), chat sends (mova chat, the TUI's chat
	// screen, MCP's chat_completion) reorder the system prompt into a
	// stable static-prefix + dynamic-tail layout and, for Anthropic,
	// mark the prefix as cacheable — see budget.LayoutForCache. There
	// is no real downside to leaving this on (see docs), so it
	// defaults on like every other Token Firewall stage; set it to
	// `false` explicitly to opt out.
	CacheHint *bool `json:"cache_hint"`

	// CircuitBreaker: enables/disables the spend-governance MECHANISM
	// itself (nil/true = enabled, matching every other Token Firewall
	// stage). This is independent from whether a ceiling is actually
	// configured — MaxTokensPerRun/MaxMonthlyUSD being 0 already means
	// "no ceiling"; this flag lets you keep ceilings configured but
	// temporarily switch enforcement off (e.g. while debugging) without
	// deleting the numbers.
	CircuitBreaker *bool `json:"circuit_breaker"`

	// TokenEstimation: when true (default), every count in this
	// pipeline uses the real tiktoken tokenizer (see tokencount.go).
	// Set to `false` to use a fast, rough chars/4 approximation instead
	// — a pure performance trade-off (skips real BPE encoding) for
	// projects with very large Focus sets where exact counts aren't
	// needed on every run; the Budget gate still works either way, just
	// with a less precise number.
	TokenEstimation *bool `json:"token_estimation"`

	// DetailedReports: when true (default), mova-budget-report.md
	// includes the full Token Firewall breakdown (per-file tokens,
	// Sanitizer/Cache Layout/Circuit Breaker sections, before/after
	// cost comparison). Set to `false` for just the totals — useful
	// for a project that wants a short report.
	DetailedReports *bool `json:"detailed_reports"`

	// ContextCache: local memoization of the Sanitizer+tokenizer's
	// result for Focus/Memory content, keyed by a content hash — skips
	// redoing that work when nothing changed since the last run (see
	// mova.local/budget's contextcache.go). Distinct from CacheHint
	// (which is about a Cloud PROVIDER's cache): this one is Mova's own
	// local cache, saves wall-clock time, never tokens or money by
	// itself. Default enabled; state lives in mova-context-cache.json.
	ContextCache *bool `json:"context_cache"`

	// PIIMasking: optional technical/structural pseudonymization stage
	// (see mova.local/sanitize's pii.go + config/policy.json) that
	// replaces candidate-PII tokens in Focus/Memory with deterministic
	// [TAG_HASH] pseudonyms BEFORE counting/sending anything, using
	// only Shannon entropy + word-shape structure — no word lists, no
	// language-specific rules. UNLIKE every other Token Firewall stage
	// above, this one defaults OFF (nil/absent = disabled): it changes
	// the actual content sent to the model, not just cosmetic noise, so
	// it must be an explicit, informed opt-in per project. See
	// docs/i18n/{es,en}/COMMANDS.md § PII Masking for the full
	// technical/legal disclaimer — this is NOT an anonymization or Ley
	// 21.719/GDPR compliance guarantee, only a heuristic mitigation.
	PIIMasking *PIIMaskingConfig `json:"pii_masking"`
}

// PIIMaskingConfig maps project.json's (or a task's)
// "budget"."pii_masking" object — a single on/off switch. Every
// threshold/tag/weight the algorithm itself uses lives in
// config/policy.json (see mova.local/sanitize.LoadPIIPolicy), never
// here and never hardcoded in Go — project.json only decides WHETHER
// this project wants the stage to run, not HOW it scores tokens.
type PIIMaskingConfig struct {
	Enabled bool `json:"enabled"` // default false — must be explicitly turned on, see BudgetConfig.PIIMasking's doc comment
}

// SanitizeConfig mirrors sanitize.Config's JSON shape — kept in core
// (not the sanitize package) so core.Project never has to import
// sanitize, the same Adapter/adapters-style rule this codebase already
// follows everywhere else (see e.g. core.JobSpec's doc comment).
type SanitizeConfig struct {
	Enabled       bool `json:"enabled"`
	DedupeLogs    bool `json:"dedupe_logs"`
	StripBlank    bool `json:"strip_blank"`
	StripComments bool `json:"strip_comments"`
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

// ── Token Firewall toggles — every one of these defaults to enabled
// (an absent field, or an explicit `true`, both mean "on"); only an
// explicit `false` turns a stage off. This is the opposite default
// from core.ToolsEnabled (tools default OFF) because these stages are
// meant to be safe-by-default protections, not opt-in extras — see
// docs/SOURCE.md § Token Firewall for the reasoning. ──────────────────

// CacheGuardEnabled reports whether the Cache Layout Guard should run.
func CacheGuardEnabled(cfg *BudgetConfig) bool {
	if cfg == nil {
		return true
	}
	return cfg.CacheHint == nil || *cfg.CacheHint
}

// CircuitBreakerEnabled reports whether the spend-governance mechanism
// itself is active (independent of whether a ceiling is configured).
func CircuitBreakerEnabled(cfg *BudgetConfig) bool {
	if cfg == nil {
		return true
	}
	return cfg.CircuitBreaker == nil || *cfg.CircuitBreaker
}

// TokenEstimationEnabled reports whether real tokenization should run
// (false = fast chars/4 approximation instead).
func TokenEstimationEnabled(cfg *BudgetConfig) bool {
	if cfg == nil {
		return true
	}
	return cfg.TokenEstimation == nil || *cfg.TokenEstimation
}

// DetailedReportsEnabled reports whether mova-budget-report.md should
// include the full Token Firewall breakdown.
func DetailedReportsEnabled(cfg *BudgetConfig) bool {
	if cfg == nil {
		return true
	}
	return cfg.DetailedReports == nil || *cfg.DetailedReports
}

// ContextCacheEnabled reports whether Mova's own local memoization of
// Sanitizer+tokenizer results should be used.
func ContextCacheEnabled(cfg *BudgetConfig) bool {
	if cfg == nil {
		return true
	}
	return cfg.ContextCache == nil || *cfg.ContextCache
}

// PIIMaskingEnabled reports whether the technical/structural PII
// pseudonymization stage should run — the ONE Token Firewall stage
// that defaults OFF (nil/absent = false), unlike every helper above.
// See PIIMasking's doc comment for why.
func PIIMaskingEnabled(cfg *BudgetConfig) bool {
	return cfg != nil && cfg.PIIMasking != nil && cfg.PIIMasking.Enabled
}

// SanitizerEnabled reports whether the noise-removal stage should run
// — mirrors sanitize.DefaultConfig()'s own default (true) so callers
// have one place to ask "is the Sanitizer on for this project?"
// without reaching into cfg.Sanitize themselves.
func SanitizerEnabled(cfg *BudgetConfig) bool {
	if cfg == nil || cfg.Sanitize == nil {
		return true
	}
	return cfg.Sanitize.Enabled
}
