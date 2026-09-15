// types.go — shared data structures for the mova.local/trace package,
// the single engine behind `context-trace` across its four doors (CLI,
// MCP, HTTP, and Chat/Mova UI Chat — see COMMANDS.md § context-trace).
// Nothing here invents a value: every field is filled from something
// real (project.json, config/prices.json, or an actual repository
// scan) — the same discipline mova.local/diagram.Data already follows
// for `mova run --diagram`.
package trace

import (
	"mova.local/budget"
	"mova.local/sanitize"
)

// FileState re-exports sanitize.FileState so every consumer of this
// package (console.go, markdown_governance.go, cli/trace_cmd.go) can
// refer to trace.FileState without importing mova.local/sanitize
// directly — the governance state machine itself lives in
// mova.local/sanitize (see sanitize/state.go's header for why: it is
// already the single owner of every detection/masking/blocking
// decision, so this alias avoids a second, parallel definition.
type FileState = sanitize.FileState

const (
	StateDiscovered = sanitize.StateDiscovered
	StateCandidate  = sanitize.StateCandidate
	StateAllowed    = sanitize.StateAllowed
	StateSanitized  = sanitize.StateSanitized
	StateBlocked    = sanitize.StateBlocked
	StateExcluded   = sanitize.StateExcluded
)

// RelevanceRankRow is one file's --task relevance score, exposed for
// the "RANKING Y SELECCIÓN" report section (see relevance.go).
type RelevanceRankRow struct {
	Rank   int
	Path   string
	Score  float64
	Tokens int
	Role   string // "producer" | "consumer" | "" — see ast_relevance.go; "" when astfilter doesn't cover this file's language
}

// ExclusionReasonRow is one row of the "WHAT STAYED OUT?" breakdown —
// why a group of files never became CANDIDATE at all (as opposed to
// SecurityFinding, which explains files that DID become CANDIDATE and
// were then BLOCKED/SANITIZED for security reasons).
type ExclusionReasonRow struct {
	Reason string
	Files  int
	Tokens int
}

// ModelCompatRow is one row of the "MODEL COMPATIBILITY" table —
// whether the final sendable context fits a given provider/model's
// context window, straight from config/prices.json's context_window
// field (0 = not configured, rendered as "-").
type ModelCompatRow struct {
	Provider      string
	Model         string
	ContextWindow int
	Fits          bool
}

// CostRow is one estimated-cost row for a provider/model — same shape
// as budget.ModelCost, kept separate so the presentation layer
// (console/markdown/pdf) doesn't need to import more of
// mova.local/budget than AnalyzeLocal/AnalyzeRemote already do.
type CostRow struct {
	Provider string
	Model    string
	USD      float64
}

// ComponentRow is one row of the AGENTS/SKILLS/PROMPT/FOCUS/MEMORY
// breakdown (local project mode only — see Data.Components).
type ComponentRow struct {
	Name   string
	Tokens int
}

// DirRow is one row of the "top directories by token usage" table —
// only populated for a full-repository scan (remote analysis, or a
// local folder with no project.json), where there is no
// Agents/Skills/Prompt/Focus/Memory split to show instead. See
// analyzer.go's directory accounting in AnalyzeRemote.
type DirRow struct {
	Dir     string // top-level path relative to the repo root, or "(repo root)"
	Tokens  int
	Files   int
	Percent float64 // of TotalTokens
}

// FirewallStatus mirrors the Context Governance's status (see
// core/budget_config.go) as it applies to THIS run of context-trace —
// for a local project this is the real configuration read from
// project.json; for a remote repository with no project.json it is an
// informational "audit mode" status (see analyzer.go).
type FirewallStatus struct {
	SanitizerOn      bool
	PIIMaskingOn     bool
	CacheGuardOn     bool
	CircuitBreakerOn bool
	// PIIWarning: a short note shown next to "PII Masking" when the
	// scan found candidate credentials/emails/tokens — see
	// analyzer.go. Empty when nothing was found.
	PIIWarning string
}

// FocusCounts is the real result of the Focus Resolution Engine (see
// core/focus/render.RenderFocusContext) for a local project, or of a
// full repository walk (resolvers.WalkAllFiles) for a remote
// repository with no project.json.
type FocusCounts struct {
	Included int
	Excluded int
	// ExcludedSuggestion: the folder names most often skipped
	// (".git", "node_modules", ...) — only populated in remote mode,
	// feeding the "exclude" suggestion in project_gen.go.
	ExcludedSuggestion []string
}

// ProgressFunc is an optional progress callback: percent is 0-100,
// message is a short, human-readable, generic-English status line
// ("Cloning repository...", "Tokenizing files (1,204/3,139)...").
//
// IMPORTANT: this must stay nil for the MCP stdio door and the HTTP
// door — both write structured protocol bytes to stdout/response
// bodies, and any extra text written there would corrupt the
// response. Only CLI and Chat (real terminals, real humans watching)
// should ever set this.
type ProgressFunc func(percent int, message string)

// Data is the complete result of one context-trace analysis — ready
// to be rendered to console (console.go), Markdown (markdown.go), PDF
// (pdf.go), or PNG (png.go) — a single source of truth for all four
// presentation formats and all four doors.
type Data struct {
	// Origin: "CLI" | "Chat" | "MCP" | "API HTTP" — same purpose as
	// diagram.Data.Origin: which door triggered this analysis,
	// informational only.
	Origin string

	// IsRemote: true when the analysis came from --repo <url> (or the
	// MCP/HTTP/Chat equivalent) instead of a local project.json.
	IsRemote bool

	// ── Local project mode ───────────────────────────────────────────
	ProjectName     string
	ProjectJSONPath string // relative path shown in the report's INPUT section
	TaskName        string
	// IgnorePatterns: the --ignore glob/extglob patterns active for
	// this run (see ignore_glob.go) - propagated into every renderer
	// (console INPUT/FOCUS, PDF, PNG, pii-audit-log.json's
	// execution.ignore_patterns) so the filtered scope is never a
	// silent, unauditable side effect.
	IgnorePatterns []string
	// PrunedDocstringFiles: how many CANDIDATE text files had their
	// comments/docstrings stripped before token-counting because
	// --prune-docstrings was given (see analyzer.go, astfilter.PruneDocs)
	// — 0 when the flag wasn't used, never a silent content change.
	PrunedDocstringFiles int
	HasProjectJSON bool

	// ── Remote repository mode ───────────────────────────────────────
	RepoURL string
	Branch  string
	RepoDir string // resolved temp (or local) directory, used internally

	// Focus / Context Compiler
	Focus FocusCounts

	// Context breakdown (empty in remote mode: there is no
	// project.json declaring agents/skills/prompt/focus/memory
	// separately — only the whole repository is counted, see
	// DirBreakdown instead).
	Components []ComponentRow

	// DirBreakdown: top-level directories ranked by token usage — the
	// "where would I actually save tokens" table, populated whenever
	// a full-repository scan happened (remote mode, or a local folder
	// with no project.json). Empty for a normal local project (its
	// Components table already answers that question).
	DirBreakdown []DirRow

	// Context Governance
	Firewall FirewallStatus

	// Budget — MaxTokens==0 means "no active project.json" (remote
	// mode) or "no budget.max_tokens configured" (local project with
	// no declared limit); both cases render as N/A.
	TotalTokens int
	MaxTokens   int

	// Estimated costs — one row per provider/model from
	// config/prices.json, plus a fixed "local execution, no cost" note
	// (see console.go) that never depends on invented data: a local
	// model genuinely costs nothing to run.
	Costs []CostRow
	// CostsSafeNow: the same cost table, computed over
	// SafeToSendNowTokens (ALLOWED only) instead of FinalSendableTokens
	// (ALLOWED+WOULD_SANITIZE) - see sanitize.StateCounts' doc comments
	// for why these are two different, both-legitimate numbers in
	// audit/discovery mode. Empty for a project.json run (no
	// ambiguity there - the Context Governance already applies sanitization
	// before Costs is computed).
	CostsSafeNow []CostRow

	// Encoding: which tiktoken-go encoding was used — same as
	// budget.Report.Encoding, shown in every long-form report.
	Encoding string

	// Report: the full budget.Report when this analysis came from a
	// local project (nil in remote mode) — write_outputs.go reuses it
	// instead of recomputing anything for budget-report.md.
	Report *budget.Report

	// SuggestedProjectJSON: the already-rendered content of the
	// suggested project.json (see project_gen.go) — only populated in
	// remote mode, when the analysis found enough information to
	// propose one. Empty when not applicable.
	SuggestedProjectJSON string

	// ── Context Governance & Traceability Engine (state machine,
	// policy cascade, security findings) ─────────────────────────────
	// See sanitize/state.go, sanitize/policy_cascade.go,
	// sanitize/findings.go, and analyzer.go's evaluateGovernance.

	// GovernanceStatus: the headline status line — "DISCOVERY ONLY
	// (Default Global Policy)" when no project.json was found (see
	// prompt requirement), or "CONTROLLED" / "PASS WITH CONDITIONS" /
	// "BLOCKED" once a policy cascade actively gated content.
	GovernanceStatus string

	// PolicySource / PolicyVersion: provenance of the policy cascade
	// that decided every ALLOWED/SANITIZED/BLOCKED outcome below (see
	// sanitize.PolicySet.Source/Version) — never invented, always the
	// literal string LoadPolicySet resolved.
	PolicySource  string
	PolicyVersion string

	// ── Audit Matrix — who requested the context, for which model,
	// and under whose authorship (see README § Audit Matrix, questions
	// 3, 10, 11) ────────────────────────────────────────────────────

	// AgentClient: who requested the context — "mova-cli" (CLI/Chat
	// door), the real MCP client name (e.g. "Claude-Code/1.x",
	// "Cursor/0.x") captured during the "initialize" handshake (see
	// mcp/server.go), or "mcp-agent" when the MCP client doesn't
	// declare one. Never left empty in the final report.
	AgentClient string

	// TargetModel: which model was about to receive this context —
	// "<provider>/<model>" taken from project.json → llm_profile (e.g.
	// "anthropic/claude-3-5-sonnet", "ollama/qwen2.5:7b"), or "n/a"
	// when the project declares no llm_profile.
	TargetModel string

	// PolicyAuthor: who authorized the policy applied to this run —
	// hierarchy: project.json/config/policy.json "author" →
	// MOVA_POLICY_AUTHOR environment variable → default
	// "system:default" (see core.ResolvePolicyAuthor).
	PolicyAuthor string

	// StateTotals: file/token counts per governance state — the exact
	// numbers the CLI's ASCII summary and the report's "CONTEXT
	// DECISION" section render (see sanitize.StateCounts).
	StateTotals sanitize.StateCounts

	// Findings: one auditable SecurityFinding per file that reached
	// CANDIDATE and had at least one detected pattern (whether or not
	// it was ultimately sanitized/blocked) — see findings.go's
	// EvaluateFile.
	Findings []sanitize.SecurityFinding

	// SecurityImpact: the aggregated before/after picture across every
	// Findings entry.
	SecurityImpact sanitize.SecurityImpact

	// ExclusionReasons: why files never became CANDIDATE — irrelevance,
	// policy exclusion, generated files, binaries, etc. (distinct from
	// the security BLOCKED state, which only applies to files that DID
	// become CANDIDATE).
	ExclusionReasons []ExclusionReasonRow

	// ExecutionID / CommitHash: forensic traceability for THIS run
	// only (see trace/audit_contract.go's NewExecutionID) - included
	// in every artifact (console/report/diagram/pii-audit-log.json) so
	// two runs can never be confused with each other, and so
	// pii-audit-log.json can prove which execution it belongs to.
	ExecutionID string
	CommitHash  string

	// RelevanceTop: the top-ranked files from a --task run (see
	// relevance.go's RankByTask) with their BM25+structural score and
	// 1-indexed rank - shown in the "RANKING Y SELECCIÓN" / "RELEVANCE"
	// report section so --task's effect is auditable, not a black box.
	// Empty when no task was given.
	RelevanceTop []RelevanceRankRow

	// ModelCompat: one row per config/prices.json model, showing
	// whether StateTotals.FinalSendableTokens() fits its context
	// window.
	ModelCompat []ModelCompatRow

	// SuggestedProjectName: the project name project_gen.go used to
	// build SuggestedProjectJSON (derived from the repository URL) —
	// carried here so a door that asks for confirmation AFTER Run
	// already returned (see cli/trace_cmd.go's promptGenerateProjectJSON)
	// can still call WriteSuggestedProjectJSON with the right name.
	SuggestedProjectName string
}
