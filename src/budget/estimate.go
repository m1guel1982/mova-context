// estimate.go orchestrates `mova budget`: builds the SAME context that
// `mova run`/MCP get_full_context/chat_completion produce
// (core.BuildContextSections — a single implementation, see
// core/engine.go), counts each piece declared in project.json (agents,
// skills, prompt, focus, memory) separately with tiktoken-go, and
// cross-references each one against config/prices.json — so it's clear
// WHERE the token budget is going, not just one opaque total.
//
// With --focus, it also walks the entire repo WITHOUT any filter
// (resolvers.WalkAllFiles, the same utility the Premium edition uses) to
// compare the cost of the whole repository against only what `focus`
// selects.
//
// The whole calculation is 100% local: no line of the repo or the
// context is ever sent anywhere, and no LLM or external API is called —
// only the embedded tiktoken-go tokenizer and config/prices.json's
// arithmetic. Because the estimate uses exactly what project.json
// declares (the same agents/skills/prompt/focus that end up in the real
// context), the number reflects the project as configured TODAY —
// trimming agents/skills/focus in project.json before running
// `mova budget` is how you optimize the reported cost.
package budget

import (
	"fmt"

	"mova.local/core"
	"mova.local/sanitize"
)

// ModelCost is one estimated-cost row for a given provider/model.
type ModelCost struct {
	Provider string
	Model    string
	USD      float64
	CLP      float64
}

// ComponentBreakdown is ONE piece declared in project.json (or produced
// by the engine): how many tokens it used, and what it would cost under
// each provider/model in config/prices.json — so you can see, in both
// tokens and dollars, exactly what's worth optimizing first.
type ComponentBreakdown struct {
	Name   string // "Agents", "Skills", "Prompt", "Focus", "Memory", "Engine overhead"
	Tokens int
	Costs  []ModelCost
}

// FocusComparison is only present when --focus was requested.
type FocusComparison struct {
	TokensWithoutFocus int
	TokensWithFocus    int
	SavingsPercent     float64
}

// ProviderAccuracy is one row of the "Historical Token Accuracy" section:
// how far off the local tiktoken-go estimate has been, on average, from
// the real usage this provider's API has reported for this project.
// HasData is false ("No historical data") until at least one real call
// has been recorded (see history.go).
type ProviderAccuracy struct {
	Provider         string
	DeviationPercent float64
	HasData          bool
}

// Report is the full result of `mova budget`, ready to render to
// Markdown (see report.go). TotalTokens/TotalCosts are the exact sum of
// Components — the total is never a separate number, always
// reconcilable by adding up the breakdown.
type Report struct {
	ProjectName string
	TaskName    string
	Encoding    string // e.g. "cl100k_base" — which tiktoken encoding was used
	Components  []ComponentBreakdown
	TotalTokens int
	TotalCosts  []ModelCost
	Currency    string
	CLPRate     float64
	Focus       *FocusComparison // nil unless --focus was requested

	// DuplicatesRemoved: identical paragraphs automatically removed by
	// dedup.Paragraphs across AGENTS+SKILLS+PROMPT+FOCUS+MEMORY (see
	// core.BuildContextSections) — a real optimization, not a suggestion.
	DuplicatesRemoved int

	// MaxTokens/OverBudget: see "budget": {"max_tokens": N} in
	// project.json/task (CheckLimit). MaxTokens=0 means no limit configured.
	MaxTokens  int
	OverBudget bool

	// HistoricalAccuracy: one row per provider configured in
	// config/prices.json, comparing the local estimate against real
	// Cloud API usage recorded for this project (see history.go).
	HistoricalAccuracy []ProviderAccuracy

	// Token Firewall — see budget_config.go, gated_context.go,
	// spend.go, cachelayout.go, contextcache.go, mova.local/sanitize.
	// All zero-valued when a project never opts into any of it.
	SanitizeStats  sanitize.Stats
	CircuitBreaker CircuitBreakerResult
	CacheLayout    *CacheLayout // nil unless Cache Guard is enabled (default) — see core.CacheGuardEnabled

	// PIIStats: PII Masking stage result — zero value unless the
	// project explicitly opted in (core.PIIMaskingEnabled). See
	// mova.local/sanitize/pii.go's header for the disclaimer this
	// stage carries; report_pipeline.go's piiMaskingSection renders
	// this only when TokensMasked > 0.
	PIIStats sanitize.PIIStats

	// RawTokens/RawCosts: the total BEFORE the Sanitizer ran — only
	// populated when core.DetailedReportsEnabled(cfg), since it costs
	// one extra tokenization pass over the unsanitized text purely for
	// this comparison. TotalTokens/TotalCosts (above) are always the
	// AFTER numbers — every existing caller of this struct keeps
	// meaning exactly what it always meant.
	RawTokens int
	RawCosts  []ModelCost

	// FileBreakdown: one row per Focus file, tokens after sanitizing —
	// only populated when core.DetailedReportsEnabled(cfg).
	FileBreakdown []ComponentBreakdown

	// MemorySavingsPercent/CostSavingsPercent: (Raw-Total)/Raw*100 —
	// 0 when RawTokens is 0 (not computed, or genuinely nothing to
	// compare against).
	MemorySavingsPercent float64
	CostSavingsPercent   float64
}

// BuildReport builds the full Report: reads project.json (via
// core.BuildContextSections — the same path `mova run`, MCP
// get_full_context, and chat_completion use), counts tokens for each
// declared component, and cross-references each against
// config/prices.json. withFocusComparison adds the whole-repo-vs-focus
// comparison (see FocusComparison).
func BuildReport(adapter core.Adapter, root, projectName, taskName string, withFocusComparison bool) (*Report, error) {
	proj, err := adapter.GetProject(projectName)
	if err != nil {
		return nil, err
	}
	if taskName == "" {
		taskName = proj.DefaultTask
	}
	task, ok := proj.Tasks[taskName]
	if !ok {
		return nil, fmt.Errorf("task %q not found in project %q", taskName, projectName)
	}

	sections, err := core.BuildContextSections(adapter, root, projectName, taskName)
	if err != nil {
		return nil, err
	}

	// Token Firewall stage [1]: sanitize BEFORE counting, so every
	// number below (per-component and total) already reflects it —
	// the report shows the real, optimized cost, not a "before" number
	// with a separate "savings" footnote bolted on. The raw (pre-
	// sanitize) text is kept only long enough to measure it for the
	// before/after comparison below, when detailed reports are on.
	cfg := core.ResolveBudget(proj, &task)
	detailed := core.DetailedReportsEnabled(cfg)
	rawFocus, rawMemory := sections.Focus, sections.Memory

	sanitizeStats := SanitizeCached(root, projectName, sections, sanitizeConfigFrom(cfg), core.ContextCacheEnabled(cfg))

	// PII Masking preview — same optional, off-by-default stage
	// BuildGatedContext runs (see gated_context.go's applyPIIMasking);
	// `mova budget` never sends anything to a model, but the report
	// should show real numbers when the project has this enabled.
	piiStats := applyPIIMasking(root, sections, cfg)

	prices, err := LoadPrices(root)
	if err != nil {
		return nil, err
	}

	modelHint := ""
	if proj.LLMProfile != nil {
		modelHint = proj.LLMProfile.Config
	}

	report := &Report{ProjectName: proj.Project, TaskName: taskName, Currency: orDefaultCurrency(prices.Currency), CLPRate: prices.ExchangeRateCLP}

	// Overhead = Header + Instruction: fixed engine boilerplate, not
	// declared in project.json, but real — reported separately so the
	// total always reconciles with the sum of the breakdown.
	pieces := []struct {
		name string
		text string
	}{
		{"Agents", sections.Agents},
		{"Skills", sections.Skills},
		{"Prompt", sections.Prompt},
		{"Focus", sections.Focus},
		{"Memory", sections.Memory},
		{"Engine overhead", sections.Header + sections.Instruction},
	}

	var encoding string
	for _, piece := range pieces {
		tokens := 0
		if piece.text != "" {
			n, enc, err := CountTokens(piece.text, modelHint)
			if err != nil {
				return nil, fmt.Errorf("could not tokenize %s: %w", piece.name, err)
			}
			tokens = n
			encoding = enc
		}
		report.Components = append(report.Components, ComponentBreakdown{
			Name: piece.name, Tokens: tokens, Costs: EstimateCost(tokens, prices),
		})
		report.TotalTokens += tokens
	}
	if encoding == "" {
		encoding = "cl100k_base"
	}
	report.Encoding = encoding
	report.TotalCosts = EstimateCost(report.TotalTokens, prices)
	report.DuplicatesRemoved = sections.DuplicatesRemoved
	report.MaxTokens, report.OverBudget = CheckLimit(proj, &task, report.TotalTokens)
	report.HistoricalAccuracy = buildHistoricalAccuracy(root, projectName, proj, prices)
	report.SanitizeStats = sanitizeStats
	report.PIIStats = piiStats

	// Token Firewall stages [2]/[3] preview — `mova budget` never sends
	// anything to a model, so a Circuit Breaker "abort" here is purely
	// informational (the error, if any, is intentionally discarded);
	// the actual enforcement happens in BuildGatedContext.
	cbResult, _ := CheckCircuitBreaker(root, projectName, cfg, report.TotalTokens)
	report.CircuitBreaker = cbResult

	if core.CacheGuardEnabled(cfg) {
		layout := LayoutForCache(sections, modelHint)
		report.CacheLayout = &layout
	}

	if detailed {
		populateDetailedReport(report, rawFocus, rawMemory, sections, modelHint, prices)
	}

	if withFocusComparison {
		comparison, err := compareFocus(root, proj, &task, sections.Focus, modelHint)
		if err != nil {
			return nil, err
		}
		report.Focus = comparison
	}

	return report, nil
}

// buildHistoricalAccuracy returns one ProviderAccuracy row per provider
// configured in config/prices.json, using mova-token-history.json's
// accumulators (see history.go). A provider with no recorded API calls
// yet gets HasData=false ("No historical data") — never an error.
