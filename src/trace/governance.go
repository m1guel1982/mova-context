// governance.go — the Context Governance & Traceability Engine glue:
// wires sanitize.LoadPolicySet + sanitize.EvaluateFile into the SAME
// file loop AnalyzeRemote already walks (see analyzer.go) — no
// parallel scan, no second file-walking pass. Every DISCOVERED file
// in that loop reports here exactly once; this file only accumulates
// and classifies, it never re-reads a file from disk itself.
package trace

import (
	"path/filepath"
	"sort"
	"strings"

	"mova.local/budget"
	"mova.local/core"
	"mova.local/i18n"
	"mova.local/sanitize"
)

// governanceEngine accumulates one context-trace run's worth of
// state-machine counts, security findings, and exclusion reasons —
// created once per AnalyzeRemote call (see newGovernanceEngine) and
// fed one file at a time from the same loop that already tokenizes
// every file.
type governanceEngine struct {
	policy      sanitize.PolicySet
	counts      sanitize.StateCounts
	findings    []sanitize.SecurityFinding
	impact      sanitize.SecurityImpact
	exclusion   map[string]*ExclusionReasonRow
	tokenBudget int
}

func newGovernanceEngine(root string, req core.PolicyRequest) *governanceEngine {
	ps := sanitize.LoadPolicySetFor(root, req)
	return &governanceEngine{
		policy:      ps,
		exclusion:   map[string]*ExclusionReasonRow{},
		tokenBudget: ps.Compliance.TokenBudget,
	}
}

// recordDiscovered accounts for a path the walk found, before any
// relevance/security decision — every file passes through here
// exactly once (see sanitize/state.go's DISCOVERED stage).
func (g *governanceEngine) recordDiscovered(tokens int) {
	g.counts.DiscoveredFiles++
	g.counts.DiscoveredTokens += tokens
}

// exclusionReasonFor classifies a file the resolvers/focus layer
// already decided to leave out of the walk — a best-effort mapping
// from path shape to a human reason (see review.json's
// generated_file_patterns), used only for the report's "WHAT STAYED
// OUT?" breakdown, never to make a real inclusion/exclusion decision
// (that decision already happened in resolvers.WalkAllFiles).
func (g *governanceEngine) exclusionReasonFor(path string) string {
	slashPath := filepath.ToSlash(path)
	lower := strings.ToLower(slashPath)
	for _, pat := range g.policy.Review.GeneratedFilePatterns {
		p := strings.ToLower(strings.TrimSuffix(pat, "/"))
		if strings.HasSuffix(pat, "/") && strings.Contains(lower, "/"+p+"/") {
			return i18n.T("reports.exclusion_reasons.generated_files")
		}
		if strings.HasPrefix(pat, "*.") && strings.HasSuffix(lower, strings.TrimPrefix(pat, "*")) {
			return i18n.T("reports.exclusion_reasons.generated_files")
		}
	}
	if strings.Contains(lower, "test") {
		return i18n.T("reports.exclusion_reasons.tests_not_required")
	}
	return i18n.T("reports.exclusion_reasons.not_relevant")
}

// addExcluded folds one non-security exclusion into the reason
// breakdown (see ExclusionReasonRow / "WHAT STAYED OUT?").
func (g *governanceEngine) addExcluded(path string, tokens int) {
	g.addExcludedReason(path, tokens, g.exclusionReasonFor(path))
}

// addExcludedReason is addExcluded with an explicit reason instead of
// the best-effort path-shape heuristic - used by the task-relevance
// narrowing step (relevance.go) to record "Not relevant to task" as
// its own distinct, honest reason rather than pretending it was
// "Not relevant" for the same reason a generated/build file is.
func (g *governanceEngine) addExcludedReason(path string, tokens int, reason string) {
	g.counts.ExcludedFiles++
	g.counts.ExcludedTokens += tokens
	row, ok := g.exclusion[reason]
	if !ok {
		row = &ExclusionReasonRow{Reason: reason}
		g.exclusion[reason] = row
	}
	row.Files++
	row.Tokens += tokens
}

// evaluateCandidate runs the security/PII decision for a file that
// DID become a CANDIDATE (relevant, within scope), returning the
// content that should actually be tokenized/sent downstream ("" for
// BLOCKED) and the resulting FileState.
func (g *governanceEngine) evaluateCandidate(path, content string, tokens int) (string, FileState) {
	g.counts.CandidateFiles++
	g.counts.CandidateTokens += tokens

	finding, out, state := sanitize.EvaluateFile(path, content, tokens, g.policy)
	if len(finding.PatternsDetected) > 0 {
		g.findings = append(g.findings, finding)
		g.impact.Add(finding)
	}

	switch state {
	case StateAllowed:
		g.counts.AllowedFiles++
		g.counts.AllowedTokens += tokens
	case StateSanitized:
		g.counts.SanitizedFiles++
		g.counts.SanitizedTokens += tokens
	case StateBlocked:
		g.counts.BlockedFiles++
		g.counts.BlockedTokens += tokens
	}
	return out, state
}

// exclusionRows returns the accumulated "WHAT STAYED OUT?" breakdown,
// in a fixed, deterministic display order.
func (g *governanceEngine) exclusionRows() []ExclusionReasonRow {
	// Return EVERY recorded reason, sorted by token volume descending -
	// a fixed string-matched order list was tried here previously and
	// silently DROPPED any reason whose string didn't match exactly
	// (a real, verified bug: ~1,200 "not relevant to task" files
	// vanished from a report entirely with no warning). Never again:
	// every exclusion this engine recorded is guaranteed to appear.
	rows := make([]ExclusionReasonRow, 0, len(g.exclusion))
	for _, row := range g.exclusion {
		rows = append(rows, *row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Tokens != rows[j].Tokens {
			return rows[i].Tokens > rows[j].Tokens
		}
		return rows[i].Files > rows[j].Files
	})
	return rows
}

// governanceStatus derives the headline STATUS line from the final
// counts — the exact English text of each status now lives in
// config/lang/{es,en}.json's "reports.governance_status_*" keys (see
// i18n.T below) so it follows config/lang/lang_active.json like every
// other user-facing surface; "DISCOVERY ONLY (Default Global Policy)"
// remains the required EN string whenever there is no active
// project.json (see prompt requirement) — that's config/lang/en.json's
// value for reports.governance_status_discovery_only, unchanged.
func governanceStatus(hasProjectJSON bool, c sanitize.StateCounts) string {
	if !hasProjectJSON {
		return i18n.T("reports.governance_status_discovery_only")
	}
	if c.BlockedFiles > 0 && c.AllowedFiles == 0 && c.SanitizedFiles == 0 {
		return i18n.T("reports.governance_status_blocked")
	}
	if c.BlockedFiles > 0 || c.SanitizedFiles > 0 {
		return i18n.T("reports.governance_status_pass_with_conditions")
	}
	return i18n.T("reports.governance_status_controlled")
}

// modelCompatRows builds the "MODEL COMPATIBILITY" table from
// prices.json's own context_window field for every configured
// provider/model — 0 (not configured) always renders as "does not
// fit" is WRONG to assume, so Fits is only ever true when a positive
// context_window is actually >= finalTokens; unknown stays "unknown"
// at the render layer (see markdown_governance.go), never guessed.
func modelCompatRows(prices *budget.PricesConfig, finalTokens int, root string) []ModelCompatRow {
	if prices == nil {
		return nil
	}
	localWindows := localModelContextWindows(root)

	var rows []ModelCompatRow
	for _, providerName := range prices.SortedProviderNames() {
		provider := prices.Providers[providerName]
		names := make([]string, 0, len(provider.Models))
		for name := range provider.Models {
			names = append(names, name)
		}
		sortStrings(names)
		for _, modelName := range names {
			entry := provider.Models[modelName]
			window := entry.ContextWindow
			// For local providers, prices.json's own context_window is
			// only a fallback: the actual, currently-configured window
			// lives in config/models/<provider>/*.json (see
			// local_models.go) - whatever model the person actually
			// pulled/loaded, not a guess.
			if entry.Local {
				if w, ok := localWindows[providerName]; ok && w > 0 {
					window = w
				}
			}
			rows = append(rows, ModelCompatRow{
				Provider: providerName, Model: modelName,
				ContextWindow: window,
				Fits:          window > 0 && finalTokens <= window,
			})
		}
	}
	return rows
}

// sortStrings is a tiny local wrapper so this file only imports
// "sort" if it truly needs to — kept explicit for readability at the
// one call site above.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

func topRelevanceRows(scores map[string]RelevanceScore, files map[string]string, repoDir string, topN int) []RelevanceRankRow {
	if len(scores) == 0 {
		return nil
	}
	paths := make([]string, 0, len(scores))
	for p := range scores {
		paths = append(paths, p)
	}
	sort.Slice(paths, func(i, j int) bool {
		si, sj := scores[paths[i]].Total, scores[paths[j]].Total
		if si != sj {
			return si > sj
		}
		return paths[i] < paths[j]
	})

	// Deduplicate translated documentation pages (docs/<lang>/... all
	// carry near-identical content, so BM25 legitimately scores them
	// almost the same - see the reported issue: a top-10 that is 90%
	// copies of the same page in different languages tells a developer
	// nothing new after the first one). Only the BEST-scoring language
	// variant of a given page is kept; this naturally lets source code
	// and tests surface into the remaining slots instead of being
	// crowded out by duplicate translations.
	rows := make([]RelevanceRankRow, 0, topN)
	seenTailKey := map[string]bool{}
	for _, p := range paths {
		if len(rows) >= topN {
			break
		}
		key := languageStrippedKey(RelativizePath(repoDir, p))
		if seenTailKey[key] {
			continue
		}
		seenTailKey[key] = true
		rows = append(rows, RelevanceRankRow{
			Rank:   len(rows) + 1,
			Path:   RelativizePath(repoDir, p),
			Score:  roundScore(scores[p].Total),
			Tokens: len(tokenize(files[p])),
			Role:   scores[p].ASTRole,
		})
	}
	return rows
}

// languageStrippedKey collapses a translated-docs path
// ("docs/<lang>/docs/tutorial/security/index.md") down to a
// language-independent key ("docs/docs/tutorial/security/index.md")
// so every language variant of the SAME page maps to the same key -
// see topRelevanceRows' dedup step. Any path not matching this exact
// "docs/<lang-code>/..." shape (source code, tests, config) is
// returned unchanged - it was never a duplicate to begin with.
func languageStrippedKey(p string) string {
	parts := strings.Split(p, "/")
	if len(parts) >= 3 && parts[0] == "docs" && looksLikeLangCode(parts[1]) {
		return "docs/" + strings.Join(parts[2:], "/")
	}
	return p
}

// looksLikeLangCode is a narrow, deliberately conservative heuristic:
// only ISO-639-style 2-letter codes ("en", "hi") or hyphenated
// regional codes ("zh-hant", "pt-br") - short enough that it will
// never mistake an ordinary directory name (e.g. "img", "api") for a
// language code.
func looksLikeLangCode(s string) bool {
	if len(s) == 2 {
		for _, c := range s {
			if c < 'a' || c > 'z' {
				return false
			}
		}
		return true
	}
	return len(s) <= 8 && strings.Contains(s, "-")
}

func roundScore(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}
