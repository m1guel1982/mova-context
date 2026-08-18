// model.go — the data Diagram rendering (svg.go/png.go/pdf.go) draws
// from. Every field here is filled by build.go from something real:
// project.json, a group's config.json (mova.local/orchestrator), or a
// live orchestrator.Count result (the exact same engine `mova run
// --count`/`mova agents run` already use — see README at the top of
// build.go). Nothing in this package invents a number or a stage that
// isn't actually configured; a field left at its zero value simply
// isn't drawn (see svg.go's "only draw what's real" rule).
package diagram

import "mova.local/budget"

// DetailLevel controls how much of the pipeline the SVG renders.
type DetailLevel string

const (
	DetailSimple  DetailLevel = "simple"  // collapsed: one box per major stage
	DetailVerbose DetailLevel = "verbose" // expanded: every sub-stage, every agent's skills/prompts/tasks
)

// Data is everything one diagram render needs. One Data value = one
// project OR one multiagent group, never a mix.
type Data struct {
	ProjectName string
	Description string
	Lang        string // "es" | "en" | "" — from project.json's own "lang", used to pick node labels
	IsGroup     bool
	DetailLevel DetailLevel
	// Origin: which door triggered this render — "CLI", "Chat", "MCP",
	// or "API HTTP" (see build.go's ValidOrigins). Set by the CALLER
	// (cli/diagram_cmd.go, cli/chat_cmd.go, mcp/diagram_tool.go,
	// http/server.go's /diagram) — BuildDiagram has no way to infer
	// this on its own, and defaults to "MCP" only because that's the
	// lowest-level shared entry point every other door already funnels
	// through (see mcp/diagram_tool.go's own doc comment on why HTTP
	// still passes its own explicit "API HTTP" rather than relying on
	// that default).
	Origin string

	Sources    []SourceRef // from Focus (or the union of every agent's Focus, when IsGroup)
	Compiler   []string    // Context Compiler stage names actually exercised (see build.go's compilerStages)
	Firewall   Firewall
	Agents     []AgentNode // exactly one entry when !IsGroup (this project itself); one per group agent otherwise
	Jobs       []JobNode
	Interfaces []string // CLI/Chat/HTTP/MCP — always the same four doors, informational (not project-specific) — see Origin for WHICH one triggered this specific render
	Metrics    *Metrics // nil when no report could be built (e.g. adapter/read error) — never fabricated
}

// SourceRef is one Focus entry.
type SourceRef struct {
	Path string
	Kind string // "file" | "dir" | "glob" | "symbol" | "" — best-effort, from core/focus's own resolver naming
}

// Firewall mirrors core.BudgetConfig's Token Firewall fields exactly —
// see core/budget_config.go's *Enabled helpers, which this reads
// through rather than re-deciding defaults itself.
type Firewall struct {
	SanitizerOn      bool
	DedupeLogsOn     bool
	StripBlankOn     bool
	StripCommentsOn  bool
	PIIMaskingOn     bool // off by default — see core.PIIMaskingEnabled
	CacheGuardOn     bool
	CircuitBreakerOn bool
	MaxTokensPerRun  int
	MaxMonthlyUSD    float64
}

// AgentNode is one project (single-project mode) or one group member
// (multiagent mode) — same shape either way, since a group agent is
// itself an ordinary project (see mova.local/orchestrator's package doc).
type AgentNode struct {
	Name        string // display name — the agent's own subdirectory name in group mode, ProjectName otherwise
	Description string
	AgentRoles  []string // project.json's agents.use
	Skills      []string // project.json's skills.use
	Tasks       []string // project.json's tasks keys
	Provider    string   // resolved llm_profile.provider (config/models/<provider>/ folder), or "" if unset
	ModelConfig string   // llm_profile.config — the config FILE name (e.g. "llama3.2.3b")
	ModelName   string   // the REAL model tag from config/models/<provider>/<config>.json's own "model" field (e.g. "llama3.2:3b", "gemini-3-flash-preview") — see build.go's resolveModel. Falls back to ModelConfig if the file couldn't be read.
	IsLocal     bool     // derived from the RESOLVED model config's own "type" (see build.go's resolveModel) — NOT from llm_profile.type, which most projects never set
	PIIMasking  bool     // this SPECIFIC agent's own budget.pii_masking.enabled — see core.PIIMaskingEnabled. Independent of Data.Firewall.PIIMaskingOn, which is only "representative" for a group (see build.go)
}

// JobNode is one project.json "jobs" entry.
type JobNode struct {
	Schedule       string // raw cron expression, exactly as project.json wrote it
	ScheduleHuman  string // best-effort human-readable form (e.g. "Daily at 02:00") — see build.go's humanizeCron; falls back to Schedule verbatim for patterns it doesn't recognize, never a guessed/wrong translation
	Tasks          []string
	Save           string
}

// Metrics carries a real orchestrator.Count result — token counts and
// costs are exactly what `mova run --count`/`mova budget` would show
// for this same project/group right now, never estimated separately.
type Metrics struct {
	TotalTokens int
	Costs       []budget.ModelCost
	// PerAgent has one entry per AgentNode (empty for a single project's
	// own totals, which already equal TotalTokens/Costs above).
	PerAgent []AgentMetrics
}

// AgentMetrics is one group member's own token/firewall numbers.
type AgentMetrics struct {
	Name             string
	Tokens           int
	SanitizedLines   int // sanitize.Stats.RepeatedLinesCollapsed-equivalent — see build.go
	PIIScanned       int
	PIIMasked        int
	CircuitTriggered bool
	// Costs and IsLocal — see #12: cost display is conditional on
	// whether THIS agent's model is local or cloud (matched to its
	// AgentNode by Name in build.go's metricsFrom). Costs is nil/empty
	// AND ignored by svg.go whenever IsLocal is true, on purpose — a
	// dollar figure next to a model that never leaves the machine would
	// be misleading, not just unnecessary.
	Costs   []budget.ModelCost
	IsLocal bool

	// ── Token reduction pipeline (#18) — all read straight from
	// budget.Report, never recomputed here. See build.go's
	// metricsFrom for exactly which Report field feeds each one.

	// Components: per-resource token breakdown (Agents/Skills/Prompt/
	// Focus/Memory/Engine overhead) — budget.Report.Components,
	// already the same numbers `mova budget`'s own report table shows.
	Components []budget.ComponentBreakdown

	// DuplicatesRemoved: identical paragraphs collapsed across Agents+
	// Skills+Prompt+Focus+Memory — budget.Report.DuplicatesRemoved.
	DuplicatesRemoved int

	// RawTokens: the token count BEFORE Sanitizer/PII Masking ran —
	// budget.Report.RawTokens. Zero means "not available" (detailed
	// reports were turned off for this project — core.
	// DetailedReportsEnabled — not that raw and final happened to be
	// identical), so svg.go only draws the reduction-pipeline section
	// when this is > 0.
	RawTokens int
	// RawCosts: what RawTokens would have cost on each cloud provider —
	// budget.Report.RawCosts. Used only for the "money saved" line, and
	// only when IsLocal is false (see #12's own rule).
	RawCosts []budget.ModelCost
	// SavingsPercent: budget.Report.MemorySavingsPercent — the exact
	// percentage `mova budget`'s own report already computes, not a
	// second, possibly-inconsistent calculation.
	SavingsPercent float64

	// FocusComparison: only present when the Budget & Focus layer was
	// actually measured (same as `mova budget --focus`) —
	// budget.Report.Focus. Nil when not computed; svg.go skips that one
	// layer of the pipeline rather than draw a guess.
	FocusComparison *budget.FocusComparison
}
