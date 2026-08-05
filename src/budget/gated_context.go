// gated_context.go — BuildGatedContext factors out the exact sequence
// `mova run` (cli/run_cmd.go) already performed inline: assemble a
// project/task's context (core.BuildContextSections) and apply the
// Budget gate (EnforceLimit) BEFORE returning anything. Reused by the
// multiagent orchestrator, the Job Engine's "tasks" action, and chat
// (see budget_config.go's doc comment on why each of those matters) —
// one function, no second copy of "build then gate" anywhere.
//
// This is also the Token Firewall's single chokepoint: every stage
// (Sanitizer → Circuit Breaker) runs here, in this fixed order, BEFORE
// the existing max_tokens gate — see mova.local/sanitize and spend.go.
// A project.json that never opts into any Token Firewall field behaves
// byte-for-byte like before this feature existed.
package budget

import (
	"mova.local/core"
	"mova.local/sanitize"
)

// GatedContext is what BuildGatedContext returns: either Text (every
// gate passed) or a non-nil Err (the failing gate's own formatted
// message), following the "print nothing on failure" rule every caller
// of EnforceLimit already follows.
type GatedContext struct {
	Sections *core.ContextSections
	Text     string
	Tokens   int
	Err      error // set when a gate rejected this context — Text is "" in that case

	// Token Firewall results — always populated (even when every stage
	// is a no-op), so a caller can build a report without a nil check.
	Sanitize       sanitize.Stats
	CircuitBreaker CircuitBreakerResult
}

// BuildGatedContext assembles project/task's context and runs it
// through the full Token Firewall pipeline before the Budget gate.
// Callers should check Err before using Text.
func BuildGatedContext(adapter core.Adapter, root, project, task string) GatedContext {
	proj, err := adapter.GetProject(project)
	if err != nil {
		return GatedContext{Err: err}
	}
	sections, err := core.BuildContextSections(adapter, root, project, task)
	if err != nil {
		return GatedContext{Err: err}
	}
	resolvedTask := core.ResolveTaskName(proj, task)
	t := ResolveTask(proj, resolvedTask)
	cfg := core.ResolveBudget(proj, t)

	// [1] Sanitizer — cleans sections.Focus/Memory in place, BEFORE
	// anything downstream counts tokens, so every later stage already
	// sees the optimized size. Uses the Context Cache (contextcache.go)
	// when enabled, so unchanged files skip re-sanitizing on repeat runs.
	sanitizeCfg := sanitizeConfigFrom(cfg)
	sanitizeStats := SanitizeCached(root, project, sections, sanitizeCfg, core.ContextCacheEnabled(cfg))
	text := sections.Full()
	tokens := countTokensRespectingToggle(text, modelHintOf(proj), cfg)

	result := GatedContext{Sections: sections, Text: text, Tokens: tokens, Sanitize: sanitizeStats}

	// [2] Circuit Breaker — spend governance, checked BEFORE the
	// content-size gate below so a project that's already over its
	// monthly cap aborts even if this particular context is small.
	cbResult, cbErr := CheckCircuitBreaker(root, project, cfg, tokens)
	result.CircuitBreaker = cbResult
	if cbErr != nil {
		result.Err = cbErr
		result.Text = ""
		return result
	}

	// [3] Budget gate — the existing "max_tokens" content-size ceiling,
	// unchanged.
	if gateErr := EnforceLimit(proj, t, tokens); gateErr != nil {
		result.Err = gateErr
		result.Text = ""
		return result
	}
	return result
}

// sanitizeConfigFrom converts core.SanitizeConfig (project.json's JSON
// shape) into sanitize.Config (the package's own, core-independent
// type) — nil means "use the conservative default", never "disabled".
func sanitizeConfigFrom(cfg *core.BudgetConfig) sanitize.Config {
	if cfg == nil || cfg.Sanitize == nil {
		return sanitize.DefaultConfig()
	}
	s := cfg.Sanitize
	return sanitize.Config{
		Enabled:       s.Enabled,
		DedupeLogs:    s.DedupeLogs,
		StripBlank:    s.StripBlank,
		StripComments: s.StripComments,
	}
}

func modelHintOf(proj *core.Project) string {
	if proj != nil && proj.LLMProfile != nil {
		return proj.LLMProfile.Config
	}
	return ""
}

// countTokensRespectingToggle uses the real tiktoken tokenizer unless
// the project explicitly set "token_estimation": false — a pure
// performance trade-off (see core.TokenEstimationEnabled's doc comment)
// for a very large Focus set where an exact count isn't needed on
// every single run. The approximation (chars/4) is the same rough rule
// of thumb documented throughout this codebase's own comments.
func countTokensRespectingToggle(text, modelHint string, cfg *core.BudgetConfig) int {
	if !core.TokenEstimationEnabled(cfg) {
		return len(text) / 4
	}
	tokens, _, _ := CountTokens(text, modelHint)
	return tokens
}
