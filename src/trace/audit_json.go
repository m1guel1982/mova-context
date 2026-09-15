// audit_json.go — the pii-audit-log.json schema, matching the
// standardized contract given for `mova context-trace` exactly:
// schema_version, execution{}, context{scanned/selection/final},
// decisions{allowed/would_sanitize/blocked/excluded/exclusion_reasons},
// security{mode/indicators/actions/transformation},
// relevance{method/task/top_ranked}, findings[] with nested
// detector/decision/tokens/redaction objects. Split out of
// write_outputs.go purely to keep every file in this package under
// the 300-line limit.
package trace

import (
	"encoding/json"
	"os"

	"mova.local/sanitize"
)

const auditSchemaVersion = "1.0"

type auditLogPayload struct {
	SchemaVersion string          `json:"schema_version"`
	Execution     auditExecution  `json:"execution"`
	Context       auditContext    `json:"context"`
	Decisions     auditDecisions  `json:"decisions"`
	Security      auditSecurity   `json:"security"`
	Relevance     *auditRelevance `json:"relevance,omitempty"`
	Findings      []auditFinding  `json:"findings"`
}

type auditExecution struct {
	ExecutionID    string   `json:"execution_id"`
	Repository     string   `json:"repository"`
	Branch         string   `json:"branch"`
	Commit         string   `json:"commit"`
	Task           *string  `json:"task"`
	IgnorePatterns []string `json:"ignore_patterns,omitempty"`
	// AgentClient/TargetModel/PolicyAuthor: audit questions #10, #11
	// and #3 (see README § Audit Matrix) — never empty, see
	// trace.applyAuditIdentity / core.ResolvePolicyAuthor.
	AgentClient  string `json:"agent_client"`
	TargetModel  string `json:"target_model"`
	PolicyAuthor string `json:"policy_author"`
}

type auditFilesTokens struct {
	Files  int `json:"files"`
	Tokens int `json:"tokens"`
}

type auditContext struct {
	Scanned   auditFilesTokens `json:"scanned"`
	Selection auditSelection   `json:"selection"`
	Final     auditFinal       `json:"final"`
}

type auditSelection struct {
	CandidateFiles  int     `json:"candidate_files"`
	CandidateTokens int     `json:"candidate_tokens"`
	ReductionPct    float64 `json:"reduction_pct"`
}

type auditFinal struct {
	Tokens              int    `json:"tokens"`
	Mode                string `json:"mode"`
	ActuallyTransformed bool   `json:"actually_transformed"`
}

type auditDecisions struct {
	Allowed          auditFilesTokens            `json:"allowed"`
	WouldSanitize    auditFilesTokens            `json:"would_sanitize"`
	Blocked          auditFilesTokens            `json:"blocked"`
	Excluded         auditFilesTokens            `json:"excluded"`
	ExclusionReasons map[string]auditFilesTokens `json:"exclusion_reasons"`
}

type auditSecurity struct {
	Mode           string              `json:"mode"`
	Indicators     auditIndicators     `json:"indicators"`
	Actions        auditActions        `json:"actions"`
	Transformation auditTransformation `json:"transformation"`
}

type auditIndicators struct {
	FilesWithPII          int `json:"files_with_pii"`
	FilesWithSecrets      int `json:"files_with_secrets"`
	FilesWithAnyIndicator int `json:"files_with_any_indicator"`
}

type auditActions struct {
	WouldSanitizeFiles       int `json:"would_sanitize_files"`
	BlockedFiles             int `json:"blocked_files"`
	DetectedNotActionedFiles int `json:"detected_not_actioned_files"`
}

type auditTransformation struct {
	TokensRedacted              int `json:"tokens_redacted"`
	TokensPreventedFromExternal int `json:"tokens_prevented_from_external"`
}

type auditRelevance struct {
	Method    string           `json:"method"`
	Task      string           `json:"task"`
	TopRanked []auditTopRanked `json:"top_ranked"`
}

type auditTopRanked struct {
	Rank  int     `json:"rank"`
	Path  string  `json:"path"`
	Score float64 `json:"score"`
}

type auditFinding struct {
	Path      string             `json:"path"`
	Type      string             `json:"type"`
	Detector  auditDetector      `json:"detector"`
	Decision  auditDecision      `json:"decision"`
	Tokens    auditFindingTokens `json:"tokens"`
	Redaction auditRedaction     `json:"redaction"`
}

type auditDetector struct {
	ID          string   `json:"id"`
	Score       float64  `json:"score"`
	Patterns    []string `json:"patterns"`
	Occurrences int      `json:"occurrences"`
}

type auditDecision struct {
	Action      string `json:"action"`
	RuleID      string `json:"rule_id"`
	PolicyRule  string `json:"policy_rule,omitempty"`
	Transformed bool   `json:"transformed"`
}

type auditFindingTokens struct {
	Before int `json:"before"`
	After  int `json:"after"`
}

type auditRedaction struct {
	CredentialShapedValues int `json:"credential_shaped_values"`
}

// writeAuditLog writes the COMPLETE, unabridged Findings list in the
// standardized schema above - the same []sanitize.SecurityFinding
// values the report's own (capped) table is drawn from, never a
// re-derived or re-scanned copy - scoped to exactly d.ExecutionID, so
// two runs can never be confused.
func writeAuditLog(path string, d *Data) error {
	c := d.StateTotals
	var task *string
	if d.TaskName != "" {
		task = &d.TaskName
	}

	exclusionReasons := map[string]auditFilesTokens{}
	for _, r := range d.ExclusionReasons {
		key := exclusionReasonSlug(r.Reason)
		existing := exclusionReasons[key] // zero value if absent - safe to accumulate into
		exclusionReasons[key] = auditFilesTokens{Files: existing.Files + r.Files, Tokens: existing.Tokens + r.Tokens}
	}

	payload := auditLogPayload{
		SchemaVersion: auditSchemaVersion,
		Execution: auditExecution{
			ExecutionID: d.ExecutionID, Repository: d.RepoURL, Branch: d.Branch,
			Commit: d.CommitHash, Task: task, IgnorePatterns: d.IgnorePatterns,
			AgentClient: d.AgentClient, TargetModel: d.TargetModel, PolicyAuthor: d.PolicyAuthor,
		},
		Context: auditContext{
			Scanned: auditFilesTokens{Files: c.DiscoveredFiles, Tokens: c.DiscoveredTokens},
			Selection: auditSelection{
				CandidateFiles: c.CandidateFiles, CandidateTokens: c.CandidateTokens,
				ReductionPct: roundTo2Local(c.ReductionPercent()),
			},
			Final: auditFinal{Tokens: c.FinalSendableTokens(), Mode: "audit", ActuallyTransformed: false},
		},
		Decisions: auditDecisions{
			Allowed:          auditFilesTokens{Files: c.AllowedFiles, Tokens: c.AllowedTokens},
			WouldSanitize:    auditFilesTokens{Files: c.SanitizedFiles, Tokens: c.SanitizedTokens},
			Blocked:          auditFilesTokens{Files: c.BlockedFiles, Tokens: c.BlockedTokens},
			Excluded:         auditFilesTokens{Files: c.ExcludedFiles, Tokens: c.ExcludedTokens},
			ExclusionReasons: exclusionReasons,
		},
		Security: auditSecurity{
			Mode: "audit",
			Indicators: auditIndicators{
				FilesWithPII: d.SecurityImpact.FilesWithPotentialPII, FilesWithSecrets: d.SecurityImpact.FilesWithPotentialSecrets,
				FilesWithAnyIndicator: d.SecurityImpact.FilesWithAnyIndicator,
			},
			Actions: auditActions{
				WouldSanitizeFiles: c.SanitizedFiles, BlockedFiles: c.BlockedFiles,
				DetectedNotActionedFiles: d.SecurityImpact.DetectedNotActioned,
			},
			Transformation: auditTransformation{
				TokensRedacted: d.SecurityImpact.TokensActuallyRedacted, TokensPreventedFromExternal: d.SecurityImpact.TokensPreventedExternally,
			},
		},
		Relevance: auditRelevanceOf(d),
		Findings:  auditFindingsOf(d.Findings),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// exclusionReasonSlug turns a human-readable (and, since it now goes
// through i18n.T, potentially non-English) exclusion reason into a
// stable, language-independent JSON key - the JSON contract's own
// keys ("not_relevant_to_task", "not_relevant", ...) must never change
// just because the active display language changed.
func exclusionReasonSlug(reason string) string {
	switch {
	case containsFoldLocal(reason, "ignore") || containsFoldLocal(reason, "patrón") || containsFoldLocal(reason, "patron"):
		return "excluded_by_ignore_pattern"
	case containsFoldLocal(reason, "task") || containsFoldLocal(reason, "tarea"):
		return "not_relevant_to_task"
	case containsFoldLocal(reason, "generat") || containsFoldLocal(reason, "generad"):
		return "generated_files"
	case containsFoldLocal(reason, "test"):
		return "tests_not_required"
	case containsFoldLocal(reason, "polic") || containsFoldLocal(reason, "políti"):
		return "policy_exclusion"
	case containsFoldLocal(reason, "binar") || containsFoldLocal(reason, "unsupported") || containsFoldLocal(reason, "soportad"):
		return "unsupported_binary"
	case containsFoldLocal(reason, "secur") || containsFoldLocal(reason, "seguridad"):
		return "security_blocked"
	default:
		return "not_relevant"
	}
}

func containsFoldLocal(s, substr string) bool {
	sl, subl := len(s), len(substr)
	for i := 0; i+subl <= sl; i++ {
		if equalFoldASCII(s[i:i+subl], substr) {
			return true
		}
	}
	return false
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// auditRelevanceOf builds the "relevance" block, or nil when no task
// was given - the JSON contract's own example always has it present
// for a --task run and simply omits it (via omitempty) otherwise.
func auditRelevanceOf(d *Data) *auditRelevance {
	if d.TaskName == "" || len(d.RelevanceTop) == 0 {
		return nil
	}
	top := make([]auditTopRanked, 0, len(d.RelevanceTop))
	for _, r := range d.RelevanceTop {
		top = append(top, auditTopRanked{Rank: r.Rank, Path: r.Path, Score: r.Score})
	}
	return &auditRelevance{Method: "bm25+structural", Task: d.TaskName, TopRanked: top}
}

// auditFindingsOf maps the internal []sanitize.SecurityFinding into
// the contract's nested findings[] shape, applying AuditActionLabel
// (SANITIZED -> WOULD_SANITIZE) along the way - the exact same
// mapping report renderers apply, now also in the JSON itself.
func auditFindingsOf(findings []sanitize.SecurityFinding) []auditFinding {
	out := make([]auditFinding, 0, len(findings))
	for _, f := range findings {
		out = append(out, auditFinding{
			Path: f.Path, Type: f.Type,
			Detector:  auditDetector{ID: f.Detector, Score: f.Score, Patterns: f.PatternsDetected, Occurrences: f.Occurrences},
			Decision:  auditDecision{Action: AuditActionLabel(f.Action, f.Transformed), RuleID: f.RuleID, PolicyRule: f.PolicyRule, Transformed: f.Transformed},
			Tokens:    auditFindingTokens{Before: f.OriginalTokens, After: f.FinalTokens},
			Redaction: auditRedaction{CredentialShapedValues: f.CredentialShapedValues},
		})
	}
	return out
}

func roundTo2Local(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}
