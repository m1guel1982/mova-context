// markdown_governance.go — the Context Governance & Traceability
// Engine sections of context-report.md: STATUS, CONTEXT DECISION,
// SECURITY FINDINGS & ACTIONS, WHAT ENTERED / WHAT STAYED OUT, POLICY,
// and MODEL COMPATIBILITY (see reporte.md for the shape this mirrors).
// Split out of markdown.go purely to respect the 300-line-per-file
// limit — RenderContextReportMarkdown (markdown.go) calls
// RenderGovernanceMarkdown once, right after the composition section.
package trace

import (
	"fmt"
	"strings"

	"mova.local/i18n"
)

// RenderGovernanceMarkdown returns "" when there is nothing to show
// (a project.json run that has not been wired into the per-file state
// machine yet — see AnalyzeLocal's doc comment) so callers can just
// concatenate its result unconditionally.
func RenderGovernanceMarkdown(d *Data) string {
	if d.StateTotals.DiscoveredFiles == 0 {
		return ""
	}
	var b strings.Builder
	c := d.StateTotals
	imp := d.SecurityImpact

	fmt.Fprintf(&b, "\n## Status\n\n**%s**\n\nExecution ID: `%s` · Repository state: `%s` · Task: `%s`\n\n", d.GovernanceStatus, d.ExecutionID, orNA(d.CommitHash), taskOrNone(d.TaskName))
	fmt.Fprintf(&b, "%s: `%s` · %s: `%s` · %s: `%s`\n\n", i18n.T("reports.agent_client_label"), orNA(d.AgentClient), i18n.T("reports.target_model_label"), orNA(d.TargetModel), i18n.T("reports.policy_author_label"), orNA(d.PolicyAuthor))

	b.WriteString("## Context decision\n\n")
	fmt.Fprintf(&b, "| Stage | Files | Tokens |\n|---|---:|---:|\n")
	fmt.Fprintf(&b, "| DISCOVERED (scanned) | %s | %s |\n", formatInt(c.DiscoveredFiles), formatInt(c.DiscoveredTokens))
	fmt.Fprintf(&b, "| CANDIDATE | %s | %s |\n", formatInt(c.CandidateFiles), formatInt(c.CandidateTokens))
	fmt.Fprintf(&b, "| ALLOWED | %s | %s |\n", formatInt(c.AllowedFiles), formatInt(c.AllowedTokens))
	fmt.Fprintf(&b, "| WOULD_SANITIZE (audit mode - no file was rewritten) | %s | %s |\n", formatInt(c.SanitizedFiles), formatInt(c.SanitizedTokens))
	fmt.Fprintf(&b, "| BLOCKED | %s | %s |\n", formatInt(c.BlockedFiles), formatInt(c.BlockedTokens))
	fmt.Fprintf(&b, "| EXCLUDED | %s | %s |\n\n", formatInt(c.ExcludedFiles), formatInt(c.ExcludedTokens))

	if d.Focus.Included != c.DiscoveredFiles {
		gap := d.Focus.Included - c.DiscoveredFiles
		fmt.Fprintf(&b, "_Focus (%s files) is the repository SCOPE considered before content-level discovery; Discovery (%s files) is what was actually analyzed as text/image. The %s-file gap could not be read (binary/unreadable/empty) and is counted under EXCLUDED above._\n\n", formatInt(d.Focus.Included), formatInt(c.DiscoveredFiles), formatInt(gap))
	}

	b.WriteString("### Sendable context breakdown (audit mode - nothing written to disk)\n\n")
	fmt.Fprintf(&b, "| | Tokens |\n|---|---:|\n")
	fmt.Fprintf(&b, "| Repository context (scanned) | %s |\n", formatInt(c.DiscoveredTokens))
	fmt.Fprintf(&b, "| Policy allowed (safe now) | %s |\n", formatInt(c.AllowedTokens))
	fmt.Fprintf(&b, "| Would require sanitization | %s |\n", formatInt(c.SanitizedTokens))
	fmt.Fprintf(&b, "| Actually transformed/redacted | %s |\n\n", formatInt(imp.TokensActuallyRedacted))
	fmt.Fprintf(&b, "**Safe to send RIGHT NOW: %s tok (%s reduction)** - only ALLOWED content.\n\n", formatInt(c.SafeToSendNowTokens()), formatPct(c.SafeReductionPercent()))
	fmt.Fprintf(&b, "Sendable AFTER sanitization is actually applied: %s tok (%s reduction) - WOULD_SANITIZE content is **not** safe to send as-is until a real sanitization pass runs (e.g. via a `project.json` Context Governance).\n\n", formatInt(c.FinalSendableTokens()), formatPct(c.ReductionPercent()))

	totalIndicatorFiles := imp.FilesWithAnyIndicator
	fmt.Fprintf(&b, "_Files with a possible PII/secret indicator: **%s** = would_sanitize (%s) + blocked (%s) + detected-but-allowed (%s, no active rule modifies that pattern - see `config/policy/security.json`)._\n\n",
		formatInt(totalIndicatorFiles), formatInt(c.SanitizedFiles), formatInt(c.BlockedFiles), formatInt(imp.DetectedNotActioned))

	if d.TaskName == "" {
		b.WriteString("_No task was provided (see `--task`): CANDIDATE equals every DISCOVERED file that passed the existing focus/ignore rules, so this reduction reflects security/policy exclusions only, not task relevance._\n\n")
	} else {
		fmt.Fprintf(&b, "_Task: **%s** — CANDIDATE was narrowed by BM25 relevance ranking (see the Ranking & Selection section below)._\n\n", d.TaskName)
	}

	b.WriteString(renderExclusionMarkdown(d))
	b.WriteString(renderPolicyMarkdown(d))
	b.WriteString(renderRelevanceMarkdown(d))
	b.WriteString(renderModelCompatMarkdown(d))
	b.WriteString(renderNextStepsMarkdown(d))
	return b.String()
}

// renderRelevanceMarkdown is the "Ranking & Selection" block —
// transparent, auditable exposure of --task's BM25 ranking (or an
// explicit "not applied" note with no task) so the effect of --task
// is never a black box.
func renderRelevanceMarkdown(d *Data) string {
	var b strings.Builder
	b.WriteString("## Ranking & selection\n\n")
	if d.TaskName == "" || len(d.RelevanceTop) == 0 {
		b.WriteString("Task-specific ranking: **NOT APPLIED** - no task or query intent was provided.\n\n")
		return b.String()
	}
	fmt.Fprintf(&b, "%s", i18n.T("reports.task_label", map[string]any{"task": d.TaskName}))
	b.WriteString("| Rank | File | Score |\n|---:|---|---:|\n")
	for _, r := range d.RelevanceTop {
		fmt.Fprintf(&b, "| %d | %s | %.2f |\n", r.Rank, r.Path, r.Score)
	}
	b.WriteString("\n_Score = BM25 lexical relevance + a small structural/filename boost - see `docs/i18n/en/context-trace.md`'s FAQ for the exact formula. This is deterministic and explainable, not a semantic/embedding search._\n\n")
	return b.String()
}

// renderNextStepsMarkdown is context-report's "Actionable Next Steps"
// section: dynamic, data-driven suggestions (never invented) computed
// straight from d.DirBreakdown - the same table the console's "TOP
// DIRECTORIES BY TOKEN USAGE" and the suggested project.json's
// `exclude` list already use (see project_gen.go), so all three can
// never disagree about which directory is the biggest opportunity.
func renderNextStepsMarkdown(d *Data) string {
	if len(d.DirBreakdown) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Actionable next steps\n\n")
	shown := 0
	for _, r := range d.DirBreakdown {
		if r.Percent < 1.0 || shown >= 5 {
			continue
		}
		fmt.Fprintf(&b, "- Excluding `%s/` would save an estimated **%s** of the current context (%s tokens, %d file(s)).\n", r.Dir, formatPct(r.Percent), formatInt(r.Tokens), r.Files)
		shown++
	}
	if shown == 0 {
		b.WriteString("- No single directory concentrates enough tokens to suggest an exclusion.\n")
	}
	b.WriteString("\n")
	return b.String()
}

func renderExclusionMarkdown(d *Data) string {
	if len(d.ExclusionReasons) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## What stayed out?\n\n")
	b.WriteString("| Reason | Files | Tokens |\n|---|---:|---:|\n")
	for _, r := range d.ExclusionReasons {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", r.Reason, formatInt(r.Files), formatInt(r.Tokens))
	}
	b.WriteString("\n")
	return b.String()
}

func renderPolicyMarkdown(d *Data) string {
	var b strings.Builder
	b.WriteString("## Policy\n\n")
	fmt.Fprintf(&b, "- Policy source: `%s`\n", orNA(d.PolicySource))
	fmt.Fprintf(&b, "- Policy version: %s\n", orNA(d.PolicyVersion))
	b.WriteString("- Decision authority: Mova Context Policy Engine\n\n")
	return b.String()
}

func renderModelCompatMarkdown(d *Data) string {
	if len(d.ModelCompat) == 0 {
		return ""
	}
	var b strings.Builder
	final := d.StateTotals.FinalSendableTokens()
	fmt.Fprintf(&b, "## Model compatibility\n\nFinal context: %s tokens\n\n", formatInt(final))
	b.WriteString("| Provider | Model | Context window | Fits |\n|---|---|---:|---|\n")
	for _, r := range d.ModelCompat {
		window := "-"
		fits := "unknown (context_window not configured)"
		if r.ContextWindow > 0 {
			window = formatInt(r.ContextWindow)
			if r.Fits {
				fits = "yes"
			} else {
				fits = "no"
			}
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", r.Provider, r.Model, window, fits)
	}
	b.WriteString("\n_Provider/model configuration may impose additional limits._\n\n")
	return b.String()
}
