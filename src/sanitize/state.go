// state.go — the formal, immutable governance state machine every
// file (and, by extension, every token budget it carries) passes
// through during `mova context-trace` (see trace/analyzer.go). Lives
// in mova.local/sanitize — not a new "internal/governance" package —
// because sanitize is already the single owner of every
// detection/masking/blocking decision in this codebase (pii.go,
// sanitize.go); a second package would be exactly the kind of
// parallel governance layer this feature is explicitly not allowed to
// create. mova.local/trace aliases FileState (see trace/types.go) so
// callers outside sanitize never need to know where it actually lives.
package sanitize

// FileState is one of the six explicit, mutually exclusive stages a
// discovered path can be in by the time context-trace finishes. Every
// transition is one-directional (see IsValidTransition) — a file
// never moves "backwards" (e.g. from BLOCKED back to CANDIDATE) within
// a single run.
type FileState string

const (
	// StateDiscovered: found during the initial repository/local scan
	// (resolvers.WalkAllFiles or the focus resolution engine), before
	// any relevance or policy decision has been made.
	StateDiscovered FileState = "DISCOVERED"

	// StateCandidate: pre-selected as potentially relevant within the
	// project's scope/budget (task relevance, dependency, explicit
	// policy, or supporting documentation — see FindingReason).
	StateCandidate FileState = "CANDIDATE"

	// StateAllowed: explicitly permitted by the active governance
	// policy cascade with no modification required.
	StateAllowed FileState = "ALLOWED"

	// StateSanitized: content was structurally modified (PII masked,
	// secret-like values redacted) before entering the final context.
	StateSanitized FileState = "SANITIZED"

	// StateBlocked: excluded by force because it violates a critical
	// security/compliance policy (e.g. "pii.external_model = deny").
	StateBlocked FileState = "BLOCKED"

	// StateExcluded: left out for reasons unrelated to security —
	// irrelevance, binaries, ignored patterns, or token-budget
	// overflow.
	StateExcluded FileState = "EXCLUDED"
)

// terminal reports whether a state is a final resting state (no
// further transition happens to it within the same run).
func (s FileState) terminal() bool {
	switch s {
	case StateAllowed, StateSanitized, StateBlocked, StateExcluded:
		return true
	default:
		return false
	}
}

// validNext maps every state to the set of states it may legally move
// to next — encodes the pipeline's one-directional shape:
// DISCOVERED -> CANDIDATE -> {ALLOWED, SANITIZED, BLOCKED, EXCLUDED}.
var validNext = map[FileState]map[FileState]bool{
	StateDiscovered: {StateCandidate: true, StateExcluded: true},
	StateCandidate:  {StateAllowed: true, StateSanitized: true, StateBlocked: true, StateExcluded: true},
}

// IsValidTransition reports whether moving a file from "from" to "to"
// is a legal step in the state machine. Terminal states never accept
// a further transition; DISCOVERED and CANDIDATE only accept the
// transitions listed in validNext.
func IsValidTransition(from, to FileState) bool {
	if from.terminal() {
		return false
	}
	next, ok := validNext[from]
	if !ok {
		return false
	}
	return next[to]
}

// StateCounts tallies how many files (and how many tokens they
// represent) ended in each state during one context-trace run — the
// exact numbers the CLI's ASCII governance summary and
// context-report.{md,pdf} render (see trace/console.go,
// trace/markdown_governance.go).
type StateCounts struct {
	DiscoveredFiles int
	CandidateFiles  int
	AllowedFiles    int
	SanitizedFiles  int
	BlockedFiles    int
	ExcludedFiles   int

	DiscoveredTokens int
	CandidateTokens  int
	AllowedTokens    int
	SanitizedTokens  int
	BlockedTokens    int
	ExcludedTokens   int
}

// FinalSendableTokens is the token count AFTER a hypothetical
// sanitization pass has actually been applied: everything ALLOWED
// plus everything that WOULD be masked (SANITIZED). This is the
// right number for "what would the final context cost if sanitization
// were actually run" - it is NOT the same as "safe to send right now"
// during a read-only audit run, where sanitization was only
// evaluated, never applied - see SafeToSendNowTokens for that.
func (c StateCounts) FinalSendableTokens() int {
	return c.AllowedTokens + c.SanitizedTokens
}

// SafeToSendNowTokens is what is ACTUALLY safe to send AS-IS, with no
// further action: only ALLOWED tokens. In audit/discovery mode (see
// governance.go), SANITIZED never means "already masked on disk" -
// see sanitize/findings.go's header - so those tokens must NOT be
// counted as already safe. Report renderers use THIS number, not
// FinalSendableTokens, whenever they say something is ready to send
// without qualification (see the "Ambigüedad en FINAL SENDABLE
// CONTEXT" fix).
func (c StateCounts) SafeToSendNowTokens() int {
	return c.AllowedTokens
}

// ReductionPercent is how much smaller the final sendable context is
// than the raw scanned repository — the headline number in
// "CONTEXT DECISION" (see reporte.md's own "Context reduction" line).
func (c StateCounts) ReductionPercent() float64 {
	if c.DiscoveredTokens == 0 {
		return 0
	}
	final := c.FinalSendableTokens()
	return (1 - float64(final)/float64(c.DiscoveredTokens)) * 100
}

// SafeReductionPercent is the reduction against what is ACTUALLY safe
// to send right now (SafeToSendNowTokens), as opposed to
// ReductionPercent's "after a hypothetical sanitization pass" framing
// - the more conservative, audit-mode-honest number.
func (c StateCounts) SafeReductionPercent() float64 {
	if c.DiscoveredTokens == 0 {
		return 0
	}
	return (1 - float64(c.SafeToSendNowTokens())/float64(c.DiscoveredTokens)) * 100
}
