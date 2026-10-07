// gated_context.go — BuildGatedContext factors out the exact sequence
// `mova run` (cli/run_cmd.go) already performed inline: assemble a
// project/task's context (core.BuildContextSections) and apply the
// Budget gate (EnforceLimit) BEFORE returning anything. Reused by the
// multiagent orchestrator, the Job Engine's "tasks" action, and chat
// (see budget_config.go's doc comment on why each of those matters) —
// one function, no second copy of "build then gate" anywhere.
//
// This is also the Context Governance's single chokepoint: every stage
// (Sanitizer → Circuit Breaker) runs here, in this fixed order, BEFORE
// the existing max_tokens gate — see mova.local/sanitize and spend.go.
// A project.json that never opts into any Context Governance field behaves
// byte-for-byte like before this feature existed.
package budget

import (
	"fmt"
	"strings"

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

	// Context Governance results — always populated (even when every stage
	// is a no-op), so a caller can build a report without a nil check.
	Sanitize       sanitize.Stats
	CircuitBreaker CircuitBreakerResult
	// PII: PII Masking stage result — zero value unless the project
	// explicitly opted in (core.PIIMaskingEnabled), see PIIMasking's
	// doc comment on core.BudgetConfig for why this one defaults off.
	PII sanitize.PIIStats
	// Governance: per-block report of secrets redacted/blocked and PII
	// masked (sanitize/govern.go) — copied into the run evidence.
	Governance []sanitize.BlockReport
	// Closure: dependency-closure validation per task in scope
	// (graph/closure.go). A blocking report sets Err (gate "dependency_closure").
	Closure []core.ClosureReport
	// Gate names the gate that set Err ("" when Err is nil): budget,
	// circuit_breaker, dependency_closure, build.
	Gate string
}

// BuildGatedContext assembles project/task's context and runs it
// through the full Context Governance pipeline before the Budget gate.
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

	// [0] Dependency closure — before anything is sanitized or counted:
	// a context whose focused symbols call explicitly EXCLUDED code is
	// incomplete by construction. dependency_policy "block" (default)
	// stops here; "warn" continues and the conflict stays in evidence.
	var closure []core.ClosureReport
	if core.ClosureHook != nil {
		closure = core.ClosureHook(root, proj, scopeTask(proj, task, resolvedTask))
	}
	for _, c := range closure {
		if c.Blocking() {
			return GatedContext{Sections: sections, Closure: closure, Gate: "dependency_closure", Err: closureError(closure)}
		}
	}

	// [1] Sanitizer — cleans sections.Focus/Memory in place, BEFORE
	// anything downstream counts tokens, so every later stage already
	// sees the optimized size. Uses the Context Cache (contextcache.go)
	// when enabled, so unchanged files skip re-sanitizing on repeat runs.
	sanitizeCfg := sanitizeConfigFrom(cfg)
	sanitizeStats := SanitizeCached(root, project, sections, sanitizeCfg, useContextCache(root, project, cfg))

	// [1b] PII Masking — OPTIONAL, off by default (see
	// core.PIIMaskingEnabled), runs right after the Sanitizer and
	// still BEFORE anything is counted or gated, so a project that
	// enables it never sends the original candidate-PII tokens to a
	// model, local or cloud. See mova.local/sanitize/pii.go's header
	// for the technical/legal disclaimer this stage carries.
	piiStats, govReports := applyGovernance(root, proj, sections, cfg)

	text := sections.Full()
	tokens := countTokensRespectingToggle(text, modelHintOf(proj), cfg)

	result := GatedContext{Sections: sections, Text: text, Tokens: tokens, Sanitize: sanitizeStats, PII: piiStats, Governance: govReports, Closure: closure}

	// [2] Circuit Breaker — spend governance, checked BEFORE the
	// content-size gate below so a project that's already over its
	// monthly cap aborts even if this particular context is small.
	cbResult, cbErr := CheckCircuitBreaker(root, project, cfg, tokens)
	result.CircuitBreaker = cbResult
	if cbErr != nil {
		result.Err = cbErr
		result.Gate = "circuit_breaker"
		result.Text = ""
		return result
	}

	// [3] Budget gate — the existing "max_tokens" content-size ceiling,
	// unchanged.
	if gateErr := EnforceLimit(proj, t, tokens); gateErr != nil {
		result.Err = gateErr
		result.Gate = "budget"
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
// applyPIIMasking runs block-aware governance (sanitize/govern.go) over
// sections.Focus/Memory IN PLACE: secrets are ALWAYS enforced per the
// resolved security policy (block or redact the literal only); PII
// masking (field_keys, typed detectors, shape scorer on data blocks)
// only when the project opted in (core.PIIMaskingEnabled). The name is
// kept for its existing callers (estimate.go).
func applyPIIMasking(root string, proj *core.Project, sections *core.ContextSections, cfg *core.BudgetConfig) sanitize.PIIStats {
	stats, _ := applyGovernance(root, proj, sections, cfg)
	return stats
}

// applyGovernance is applyPIIMasking plus the per-block reports, used by
// BuildGatedContext so the run evidence can say what was redacted or
// blocked in which block.
func applyGovernance(root string, proj *core.Project, sections *core.ContextSections, cfg *core.BudgetConfig) (sanitize.PIIStats, []sanitize.BlockReport) {
	if sections == nil {
		return sanitize.PIIStats{}, nil
	}
	opt := GovernanceOptions(root, proj, cfg)
	focus, memory, reports := sanitize.GovernFocus(sections.Focus, sections.Memory, opt)
	sections.Focus, sections.Memory = focus, memory
	var total sanitize.PIIStats
	for _, r := range reports {
		total.TokensMasked += r.FieldValues + r.TypedPII + r.ShapePII + r.KnownValues
	}
	if opt.PIIEnabled {
		total.TokensScanned = len(strings.Fields(focus)) + len(strings.Fields(memory))
	}
	return total, reports
}

// GovernanceOptions resolves the policy (project.json "policies" >
// config/policy.json > built-in defaults) and the PII opt-in for proj.
func GovernanceOptions(root string, proj *core.Project, cfg *core.BudgetConfig) sanitize.GovernOptions {
	req := core.ResolvePolicyRequest(proj, core.OrchestratorPolicySelector(root), nil, nil)
	return sanitize.GovernOptions{
		Policy:     sanitize.LoadPolicySetFor(root, req),
		PIIEnabled: core.PIIMaskingEnabled(cfg),
	}
}

// GovernText governs one piece of text that is about to leave Mova
// outside the context assembly (a read_file result, a tool result in a
// loop Mova controls, a host tool output passed to sanitize_tool_output).
// source names the file (its extension decides code vs data).
func GovernText(root string, proj *core.Project, taskName, source, text string) (string, sanitize.BlockReport) {
	var cfg *core.BudgetConfig
	if proj != nil {
		t := ResolveTask(proj, core.ResolveTaskName(proj, taskName))
		cfg = core.ResolveBudget(proj, t)
	}
	return sanitize.GovernBlock(source, text, GovernanceOptions(root, proj, cfg))
}

func countTokensRespectingToggle(text, modelHint string, cfg *core.BudgetConfig) int {
	if !core.TokenEstimationEnabled(cfg) {
		return len(text) / 4
	}
	tokens, _, _ := CountTokens(text, modelHint)
	return tokens
}

// scopeTask keeps "*" (all tasks) when the caller asked for it, so the
// closure covers every task in scope; otherwise the resolved task.
func scopeTask(proj *core.Project, requested, resolved string) string {
	if core.IsAllTasks(core.NormalizeTaskArg(proj, requested)) {
		return core.TaskAll
	}
	return resolved
}

// closureError formats the dependency-closure gate error in the same
// "ERROR … Suggestion:" shape the Budget gate uses (doors return it as-is).
func closureError(reports []core.ClosureReport) error {
	var b strings.Builder
	b.WriteString("ERROR\n\nDependency closure: el focus depende de código que la misma especificación excluye. No se liberó ningún contexto.\n")
	for _, r := range reports {
		for _, c := range r.Conflicts {
			fmt.Fprintf(&b, "  [%s] %s -> %s (%s, excluido)\n", r.Task, c.From, c.To, c.Kind)
		}
	}
	b.WriteString("\nSuggestion:\n  - incluye la dependencia en \"focus\" o quítala de \"exclude\";\n  - o acéptala con una razón: \"accept_missing\": [{\"symbol\": \"<archivo::nombre>\", \"reason\": \"...\"}];\n  - o usa \"dependency_policy\": \"warn\" para liberar y dejar el conflicto en la evidencia.\n")
	return fmt.Errorf("%s", b.String())
}
