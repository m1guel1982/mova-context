// spend.go — the Token Firewall's third and last stage: a circuit
// breaker that can stop a run BEFORE anything is sent to a model, based
// on two independent, optional ceilings declared in project.json's
// "budget" (see core.BudgetConfig): "max_tokens_per_run" (this one
// request) and "max_monthly_usd" (this project's running total for the
// current calendar month). Distinct from the existing "max_tokens" +
// EnforceLimit (limit.go) — that one is a hard content-size gate;
// this one is a spend-governance gate, and is the only piece of the
// Token Firewall that persists state between runs.
//
// State lives in its own small file, mova-spend.json, next to
// mova-token-history.json — deliberately NOT merged into that file:
// token-history.json exists purely to calibrate local-vs-real token
// counts (see history.go) and its doc comment promises "only two
// accumulators per provider, forever" — spend tracking is a different
// concern (monthly, resettable, USD-denominated) and mixing the two
// would break that existing promise for a file whose whole design
// point is staying minimal.
package budget

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"mova.local/core"
)

// SpendState is mova-spend.json's exact shape — one project, one
// calendar month, two running totals. Never stores prompts, responses,
// or any project content, same privacy stance as token-history.json.
type SpendState struct {
	Month       string  `json:"month"` // "2026-08" — the month these totals belong to
	TokensSpent int     `json:"tokens_spent_this_month"`
	USDSpent    float64 `json:"usd_spent_this_month"`
}

// SpendPath resolves mova-spend.json — same directory convention as
// mova-token-history.json and mova-budget-report.md (next to
// project.json). No project-level override for this one on purpose —
// it's operational state, not something a project author relocates.
func SpendPath(root, project string) string {
	return filepath.Join(root, "projects", project, "mova-spend.json")
}

func currentMonth() string { return time.Now().Format("2006-01") }

// LoadSpend reads mova-spend.json. A missing file, or one from a
// PREVIOUS month, both return a fresh zero state for the current month
// — spend tracking rolls over automatically, no manual reset needed.
func LoadSpend(path string) (SpendState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return SpendState{Month: currentMonth()}, nil
		}
		return SpendState{}, err
	}
	var s SpendState
	if err := json.Unmarshal(data, &s); err != nil {
		return SpendState{}, err
	}
	if s.Month != currentMonth() {
		return SpendState{Month: currentMonth()}, nil
	}
	return s, nil
}

// RecordSpend adds tokens/usd to the current month's running total and
// saves — called once per REAL API call that actually happened (see
// cli/chat_helpers.go's recordRealUsage and mcp/chat_tool.go), never
// for a dry-run estimate.
func RecordSpend(path string, tokens int, usd float64) error {
	state, err := LoadSpend(path)
	if err != nil {
		return err
	}
	state.TokensSpent += tokens
	state.USDSpent += usd
	state.Month = currentMonth()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// CircuitBreakerResult is what CheckCircuitBreaker returns — always
// non-nil, even when nothing is configured (Checked=false), so callers
// never need a nil check to display it in a report.
type CircuitBreakerResult struct {
	Checked       bool // false = project declares neither ceiling — nothing to report
	RunTokens     int
	RunLimit      int // "max_tokens_per_run"; 0 = not configured
	MonthUSDSpent float64
	MonthUSDLimit float64 // "max_monthly_usd"; 0 = not configured
	RunExceeded   bool
	MonthExceeded bool
	Aborted       bool // true only when OnExceed == "abort" and one of the above tripped
	Message       string
}

// CheckCircuitBreaker evaluates both ceilings for this run. Returns a
// non-nil error ONLY when the configured "on_exceed" is "abort" and a
// ceiling was actually exceeded — every other case (nothing configured,
// within budget, or "on_exceed": "warn") returns nil error with the
// details in CircuitBreakerResult for the caller to display.
func CheckCircuitBreaker(root, project string, cfg *core.BudgetConfig, runTokens int) (CircuitBreakerResult, error) {
	res := CircuitBreakerResult{RunTokens: runTokens}
	if !core.CircuitBreakerEnabled(cfg) {
		return res, nil // mechanism explicitly turned off — ceilings, if any, are ignored
	}
	if cfg == nil || (cfg.MaxTokensPerRun <= 0 && cfg.MaxMonthlyUSD <= 0) {
		return res, nil
	}
	res.Checked = true
	res.RunLimit = cfg.MaxTokensPerRun
	res.MonthUSDLimit = cfg.MaxMonthlyUSD

	if cfg.MaxTokensPerRun > 0 && runTokens > cfg.MaxTokensPerRun {
		res.RunExceeded = true
	}

	if cfg.MaxMonthlyUSD > 0 {
		state, err := LoadSpend(SpendPath(root, project))
		if err != nil {
			return res, nil // an unreadable spend file is never fatal — same "never break on a read error" rule as history.go
		}
		res.MonthUSDSpent = state.USDSpent
		if state.USDSpent >= cfg.MaxMonthlyUSD {
			res.MonthExceeded = true
		}
	}

	if !res.RunExceeded && !res.MonthExceeded {
		return res, nil
	}

	res.Message = circuitBreakerMessage(res)
	onExceed := cfg.OnExceed
	if onExceed == "" {
		onExceed = "warn"
	}
	if onExceed == "abort" {
		res.Aborted = true
		return res, fmt.Errorf("%s", res.Message)
	}
	return res, nil
}

func circuitBreakerMessage(res CircuitBreakerResult) string {
	if res.RunExceeded && res.MonthExceeded {
		return fmt.Sprintf(
			"Circuit breaker: this run uses %d tokens (per-run limit: %d) and monthly spend is already $%.2f (limit: $%.2f).",
			res.RunTokens, res.RunLimit, res.MonthUSDSpent, res.MonthUSDLimit)
	}
	if res.RunExceeded {
		return fmt.Sprintf(
			"Circuit breaker: this run uses %d tokens, exceeding the configured per-run limit (%d).",
			res.RunTokens, res.RunLimit)
	}
	return fmt.Sprintf(
		"Circuit breaker: monthly spend for this project is already $%.2f, reaching or exceeding the configured limit ($%.2f).",
		res.MonthUSDSpent, res.MonthUSDLimit)
}

// EstimateCostFor computes the USD cost of tokens for ONE specific
// provider/model — unlike EstimateCost (which lists every configured
// model for comparison in mova-budget-report.md), this is what the
// circuit breaker and spend recording use: the ONE model this project
// is actually configured to call. Returns (0, false) if that
// provider/model isn't priced in config/prices.json — never a fatal
// error, since an unpriced call should still be allowed to run.
func EstimateCostFor(tokens int, provider, model string, prices *PricesConfig) (usd float64, ok bool) {
	pp, exists := prices.Providers[provider]
	if !exists {
		return 0, false
	}
	entry, exists := pp.Models[model]
	if !exists {
		return 0, false
	}
	divisor := prices.UnitDivisor()
	return (float64(tokens) / divisor) * entry.Input, true
}
