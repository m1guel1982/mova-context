// estimate_helpers.go — helper functions for BuildReport (estimate.go):
// historical accuracy lookup, per-model cost estimation, the
// whole-repo-vs-focus comparison, and currency defaulting. Split out
// once estimate.go grew past the 300-line limit.
package budget

import (
	"fmt"
	"sort"

	"mova.local/core"
	"mova.local/core/focus"
	"mova.local/core/focus/resolvers"
	"mova.local/documents"
	"mova.local/sanitize"
)

func buildHistoricalAccuracy(root, projectName string, proj *core.Project, prices *PricesConfig) []ProviderAccuracy {
	history, err := LoadHistory(HistoryPath(root, projectName, proj))
	if err != nil {
		history = TokenHistory{}
	}
	var out []ProviderAccuracy
	for _, providerName := range prices.SortedProviderNames() {
		percent, ok := history.DeviationPercent(providerName)
		out = append(out, ProviderAccuracy{Provider: providerName, DeviationPercent: percent, HasData: ok})
	}
	return out
}

// EstimateCost cross-references tokens against every provider/model in
// prices.json. Adding a new provider or model in that JSON shows up here
// automatically — this function never needs to change.
func EstimateCost(tokens int, prices *PricesConfig) []ModelCost {
	divisor := prices.UnitDivisor()
	var out []ModelCost
	for _, providerName := range prices.SortedProviderNames() {
		modelNames := make([]string, 0, len(prices.Providers[providerName].Models))
		for name := range prices.Providers[providerName].Models {
			modelNames = append(modelNames, name)
		}
		sort.Strings(modelNames)
		for _, modelName := range modelNames {
			entry := prices.Providers[providerName].Models[modelName]
			usd := (float64(tokens) / divisor) * entry.Input
			clp := usd * prices.ExchangeRateCLP
			out = append(out, ModelCost{Provider: providerName, Model: modelName, USD: usd, CLP: clp})
		}
	}
	return out
}

// compareFocus compares the token count of the WHOLE repo (no focus
// filtering at all — resolvers.WalkAllFiles, the same walking utility the
// focus engine uses internally) against the already-computed FOCUS
// section text (focusText) — the same content the real context would
// use, never tokenized twice with different logic. Focus is never
// automatic compression — it's the developer deciding what context to send.
func compareFocus(root string, proj *core.Project, task *core.Task, focusText, modelHint string) (*FocusComparison, error) {
	items := core.ResolveFocus(proj, task)
	if len(items) == 0 {
		return nil, fmt.Errorf("project %q (task %q) has no \"focus\" configured — nothing to compare. Add \"focus\" to project.json or to the task", proj.Project, task.Prompt)
	}

	repoDir, _, err := documents.ResolveDirectoryPath(root, proj.Repo, "")
	if err != nil {
		return nil, err
	}

	var fullText []byte
	ctx := focus.Context{RepoPath: repoDir}
	resolvers.WalkAllFiles(ctx, repoDir, func(path string) {
		fullText = append(fullText, resolvers.ReadFile(path)...)
		fullText = append(fullText, '\n')
	})

	tokensWithoutFocus, _, err := CountTokens(string(fullText), modelHint)
	if err != nil {
		return nil, fmt.Errorf("could not tokenize the whole repository: %w", err)
	}
	tokensWithFocus, _, err := CountTokens(focusText, modelHint)
	if err != nil {
		return nil, fmt.Errorf("could not tokenize the focus content: %w", err)
	}

	savings := 0.0
	if tokensWithoutFocus > 0 {
		savings = (1 - float64(tokensWithFocus)/float64(tokensWithoutFocus)) * 100
	}

	return &FocusComparison{
		TokensWithoutFocus: tokensWithoutFocus,
		TokensWithFocus:    tokensWithFocus,
		SavingsPercent:     savings,
	}, nil
}

func orDefaultCurrency(c string) string {
	if c == "" {
		return "USD"
	}
	return c
}

// populateDetailedReport fills in the before/after comparison and the
// per-Focus-file breakdown — only called when
// core.DetailedReportsEnabled(cfg) is true (see estimate.go's
// BuildReport), since it costs one extra tokenization pass over the
// unsanitized text purely for this comparison.
func populateDetailedReport(report *Report, rawFocus, rawMemory string, sections *core.ContextSections, modelHint string, prices *PricesConfig) {
	rawTokens := report.TotalTokens - report.tokensForFocusAndMemory() // start from the already-sanitized total...
	if rawFocus != "" {
		n, _, _ := CountTokens(rawFocus, modelHint)
		rawTokens += n
	}
	if rawMemory != "" {
		n, _, _ := CountTokens(rawMemory, modelHint)
		rawTokens += n
	}
	report.RawTokens = rawTokens
	report.RawCosts = EstimateCost(rawTokens, prices)

	if rawTokens > 0 && rawTokens > report.TotalTokens {
		saved := rawTokens - report.TotalTokens
		report.MemorySavingsPercent = float64(saved) / float64(rawTokens) * 100
		report.CostSavingsPercent = report.MemorySavingsPercent // same ratio: cost scales linearly with tokens for a fixed model
	}

	if sections.Focus == "" {
		return
	}
	for _, block := range sanitize.SplitFocusBlocks(sections.Focus) {
		n, _, _ := CountTokens(block.Content, modelHint)
		report.FileBreakdown = append(report.FileBreakdown, ComponentBreakdown{
			Name: block.Name, Tokens: n, Costs: EstimateCost(n, prices),
		})
	}
}

// tokensForFocusAndMemory reads back the already-computed Focus+Memory
// token counts from Components, so populateDetailedReport can build the
// "before" total by swapping in the RAW measurement for just those two
// pieces without re-deriving the other components (Agents/Skills/
// Prompt/Engine overhead never change size due to sanitizing).
func (r *Report) tokensForFocusAndMemory() int {
	total := 0
	for _, c := range r.Components {
		if c.Name == "Focus" || c.Name == "Memory" {
			total += c.Tokens
		}
	}
	return total
}
