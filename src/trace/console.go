// console.go — console/terminal/chat text output, replicating the two
// cases from the original spec: a local project with a project.json
// (renderConsoleLocal) and a remote repository or an unconfigured
// local project (renderConsoleRemote). CLI, MCP, HTTP, and Chat all
// print exactly this same text — one presentation function, four
// doors. Table/section HEADERS here stay plain, generic English on
// purpose — this is a dense, tightly fmt.Sprintf-formatted ASCII table
// meant to be scannable at a glance, and a mixed-language header row
// would hurt that more than it would help; see console_governance.go
// for the sibling governance table, which follows the same rule.
// Explanatory PROSE sentences do go through i18n.T like every other
// user-facing surface (see mova.local/i18n) — this file's own two
// calls plus console_governance.go's full box are the current
// coverage; see docs/SOURCE.md § 23 for the full i18n scope note.
package trace

import (
	"fmt"
	"strings"

	"mova.local/i18n"
)

const sep = "----------------------------------"

// RenderConsole picks the right format based on d.IsRemote — the
// single entry point all four doors call (see trace.go's Run).
func RenderConsole(d *Data, outputs []string) string {
	if d.IsRemote {
		return renderConsoleRemote(d, outputs)
	}
	return renderConsoleLocal(d, outputs)
}

func renderConsoleLocal(d *Data, outputs []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Mova Context Trace\n%s\n", sep)
	b.WriteString("INPUT\n")
	fmt.Fprintf(&b, "  project.json         %s\n", d.ProjectJSONPath)
	fmt.Fprintf(&b, "  task                 %s\n\n", d.TaskName)
	fmt.Fprintf(&b, "  agent                %s\n  target model         %s\n  policy author        %s\n\n", orNA(d.AgentClient), orNA(d.TargetModel), orNA(d.PolicyAuthor))

	b.WriteString("CONTEXT\n")
	for _, c := range d.Components {
		fmt.Fprintf(&b, "  %-20s %s tok\n", c.Name, formatInt(c.Tokens))
	}
	fmt.Fprintf(&b, "  %s\n\n", TotalTokensLine(d.TotalTokens, d.Encoding))

	fmt.Fprintf(&b, "FOCUS\n  Included             %d file(s)\n  Excluded             %d file(s)\n\n", d.Focus.Included, d.Focus.Excluded)

	b.WriteString("CONTEXT GOVERNANCE\n")
	b.WriteString(renderFirewallConsole(d.Firewall))
	b.WriteString("\n")

	b.WriteString("BUDGET\n")
	if d.MaxTokens > 0 {
		pct := float64(d.TotalTokens) / float64(d.MaxTokens) * 100
		fmt.Fprintf(&b, "  %s / %s tokens\n  %s\n\n", formatInt(d.TotalTokens), formatInt(d.MaxTokens), formatPct(pct))
	} else {
		b.WriteString("  N/A (no limit configured in project.json)\n\n")
	}

	fmt.Fprintf(&b, "ESTIMATED COST (for the %s tokens actually assembled for this task)\n", formatInt(costBasisTokens(d)))
	b.WriteString(renderCostsConsole(d.Costs))

	b.WriteString("\nOUTPUT\n")
	for _, o := range outputs {
		fmt.Fprintf(&b, "  %s\n", o)
	}
	return b.String()
}

func renderConsoleRemote(d *Data, outputs []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Mova Context Trace (Remote Analysis)\n%s\n", sep)
	b.WriteString("INPUT\n")
	fmt.Fprintf(&b, "  Repository           %s\n", d.RepoURL)
	fmt.Fprintf(&b, "  Branch               %s\n", orNA(d.Branch))
	fmt.Fprintf(&b, "  Task                 %s\n", taskOrNone(d.TaskName))
	fmt.Fprintf(&b, "  Agent                %s\n  Target model         %s\n  Policy author        %s\n", orNA(d.AgentClient), orNA(d.TargetModel), orNA(d.PolicyAuthor))
	if len(d.IgnorePatterns) > 0 {
		fmt.Fprintf(&b, "  %s\n", i18n.T("reports.active_ignore_patterns", map[string]any{"patterns": strings.Join(d.IgnorePatterns, ", ")}))
	}
	b.WriteString("\n")

	fmt.Fprintf(&b, "FOCUS\n  Included             %d file(s)  (repository scope considered, before content-level discovery)\n", d.Focus.Included)
	excludedLine := fmt.Sprintf("  Excluded             %d file(s)", d.Focus.Excluded)
	if len(d.Focus.ExcludedSuggestion) > 0 {
		excludedLine += fmt.Sprintf(" (suggestion: %s)", strings.Join(d.Focus.ExcludedSuggestion, ", "))
	}
	b.WriteString(excludedLine + "\n")
	if d.StateTotals.DiscoveredFiles > 0 && d.StateTotals.DiscoveredFiles != d.Focus.Included {
		gap := d.Focus.Included - d.StateTotals.DiscoveredFiles
		fmt.Fprintf(&b, "  Note: %d of the %d in-scope file(s) could not be read as text/image during discovery (binary/unreadable/empty) and are reported as EXCLUDED below - Focus is the SCOPE considered, Discovery is what was actually analyzed.\n", gap, d.Focus.Included)
	}
	b.WriteString("\n")

	b.WriteString("CONTEXT GOVERNANCE (audit mode - no active project.json)\n")
	b.WriteString(renderFirewallConsoleAudit(d.Firewall))
	b.WriteString("\n")

	if d.StateTotals.DiscoveredFiles > 0 {
		b.WriteString(renderGovernanceConsole(d))
	}

	b.WriteString("BUDGET\n  N/A (no active project.json)\n\n")

	fmt.Fprintf(&b, "  %s\n\n", TotalTokensLine(d.TotalTokens, d.Encoding))

	if len(d.DirBreakdown) > 0 {
		b.WriteString(renderDirBreakdownConsole(d))
		b.WriteString("\n")
	}

	// ESTIMATED COST is computed ONLY over the final sendable context
	// (ALLOWED + SANITIZED tokens) - the subset that WOULD be sent to
	// an LLM after governance, never the raw scanned repository. This
	// path (renderConsoleRemote) only runs in Audit Mode - context-trace
	// against a remote/unconfigured repo never actually sends anything
	// (see the GOVERNANCE box above: "audit mode - nothing written to
	// disk"), so the label must read as a PROJECTION, not a real send.
	fmt.Fprintf(&b, "PROJECTED COST (estimate if the %s candidate tokens were sent - audit mode, nothing sent - not the whole repository)\n", formatInt(costBasisTokens(d)))
	b.WriteString(renderCostsConsole(d.Costs))

	if d.StateTotals.DiscoveredFiles > 0 {
		b.WriteString("\n")
		b.WriteString(renderRelevanceConsole(d))
	}

	b.WriteString("\nOUTPUT\n")
	for _, o := range outputs {
		fmt.Fprintf(&b, "  %s\n", o)
	}

	if d.StateTotals.DiscoveredFiles > 0 {
		b.WriteString("\n")
		b.WriteString(renderExecutionConsole(d))
	}

	if d.SuggestedProjectJSON != "" {
		b.WriteString("\n" + i18n.T("cli.messages.confirm_project_json") + " ")
	}
	return b.String()
}

// renderDirBreakdownConsole is the "plus" table the person asked for:
// which top-level directories are actually consuming the token
// budget, so it's obvious at a glance where excluding a folder (via
// `exclude` in the suggested project.json) would save the most.
func renderDirBreakdownConsole(d *Data) string {
	var b strings.Builder
	b.WriteString("TOP DIRECTORIES BY TOKEN USAGE\n")
	fmt.Fprintf(&b, "  %-28s %14s %8s %8s\n", "Directory", "Tokens", "Files", "% total")
	for _, r := range d.DirBreakdown {
		fmt.Fprintf(&b, "  %-28s %14s %8d %7s\n", truncateDir(r.Dir, 28), formatInt(r.Tokens), r.Files, formatPct(r.Percent))
	}
	fmt.Fprintf(&b, "  %-28s %14s %8d %7s\n", "TOTAL", formatInt(d.TotalTokens), d.Focus.Included, "100.0%")
	return b.String()
}

func truncateDir(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func renderFirewallConsole(f FirewallStatus) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  Sanitizer            %s\n", onOff(f.SanitizerOn))
	piiLine := fmt.Sprintf("  PII Masking          %s", onOff(f.PIIMaskingOn))
	if f.PIIWarning != "" {
		piiLine += "  (" + f.PIIWarning + ")"
	}
	b.WriteString(piiLine + "\n")
	fmt.Fprintf(&b, "  Cache Layout Guard   %s\n", onOff(f.CacheGuardOn))
	fmt.Fprintf(&b, "  Circuit Breaker      %s\n", onOff(f.CircuitBreakerOn))
	return b.String()
}

// renderFirewallConsoleAudit is the discovery-mode ("audit mode - no
// active project.json") twin of renderFirewallConsole. It never
// prints a bare "Sanitizer OFF": in this mode the governance engine
// (see governance.go) still actively classifies and counts files
// under SANITIZED/BLOCKED even though the project.json-driven Token
// Firewall sanitizer stage itself never ran - printing "OFF" next to
// a report that then lists real SANITIZED files is a contradiction a
// reader would reasonably flag as a bug (as this one was).
func renderFirewallConsoleAudit(f FirewallStatus) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  Sanitizer            %s\n", "DISABLED (Audit Mode - see GOVERNANCE below for per-file SANITIZED/BLOCKED decisions)")
	piiLine := fmt.Sprintf("  PII Masking          %s", onOff(f.PIIMaskingOn))
	if f.PIIWarning != "" {
		piiLine += "  (" + f.PIIWarning + ")"
	}
	b.WriteString(piiLine + "\n")
	fmt.Fprintf(&b, "  Cache Layout Guard   %s\n", onOff(f.CacheGuardOn))
	fmt.Fprintf(&b, "  Circuit Breaker      %s\n", onOff(f.CircuitBreakerOn))
	return b.String()
}

func renderCostsConsole(costs []CostRow) string {
	var b strings.Builder
	for _, c := range costs {
		fmt.Fprintf(&b, "  %-24s %s\n", c.Provider+" ("+c.Model+")", FormatCost(c.USD))
	}
	b.WriteString("  Local (Ollama/LM Studio/vLLM)  $0 (local execution)\n")
	b.WriteString("  All figures above are technical ESTIMATES (public price lists + cl100k_base), not absolute costs.\n")
	return b.String()
}
