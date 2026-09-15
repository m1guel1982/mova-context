// findings.go — turns raw detection (pii.go, secrets.go) plus the
// active PolicySet into one auditable, explainable decision per file:
// a "potential indicator" is never silently equated with "the file
// was changed". Three actions exist (ALLOWED, SANITIZED, BLOCKED —
// see state.go's FileState), but "SANITIZED" here always means
// "policy WOULD mask/redact this content before sending it" — it does
// NOT mean the file on disk was rewritten. context-trace's own
// analysis never touches the repository it inspects, so Transformed
// is always false on every SecurityFinding it produces; report
// renderers must display SANITIZED as "WOULD_SANITIZE" whenever
// Transformed is false, per the "Audit Mode ≠ Sanitized" requirement
// (see trace/findings_display.go's AuditActionLabel).
package sanitize

// SecurityFinding is the auditable record for ONE inspected file.
// Field names/JSON shape are deliberately aligned with the
// pii-audit-log.json contract: path, type, score, occurrences,
// action, transformed, tokens_before/after, rule_id, detector.
type SecurityFinding struct {
	Path             string   `json:"path"`
	Type             string   `json:"type"`              // e.g. "pii_structural", "secret_api_key"
	Score            float64  `json:"score"`             // 0..1 heuristic confidence — see scoreFor
	PatternsDetected []string `json:"patterns_detected"` // human-readable labels, e.g. "API key pattern"
	Occurrences      int      `json:"occurrences"`
	Action           string   `json:"action"`      // "ALLOWED" | "SANITIZED" | "BLOCKED"
	Transformed      bool     `json:"transformed"` // true only if content was ACTUALLY rewritten on disk (never true from context-trace's own read-only analysis)
	OriginalTokens   int      `json:"tokens_before"`
	FinalTokens      int      `json:"tokens_after"`
	RuleID           string   `json:"rule_id"`     // the SINGLE policy line that decided the action; "" when nothing fired. Always one value — never a "a / b" combination (see PolicyRule).
	PolicyRule       string   `json:"policy_rule"` // a SECOND policy line that ALSO applied on top of RuleID (e.g. a secret rule firing alongside a PII rule); "" when only one rule fired
	Detector         string   `json:"detector"`    // "shannon_entropy_shape" | "regex_pattern_match" | ""
	// CredentialShapedValues: how many credential-shaped values WOULD
	// be redacted if this file were actually sanitized - a count only,
	// never the values themselves (see this file's header: no
	// sensitive value is ever persisted or exposed).
	CredentialShapedValues int `json:"credential_shaped_values"`
}

// SecurityImpact aggregates every SecurityFinding from one run into
// the global before/after picture — the numbers the "SECURITY" section
// of context-report shows.
type SecurityImpact struct {
	FilesWithPotentialPII     int
	FilesWithPotentialSecrets int
	// FilesWithAnyIndicator: unique files with AT LEAST ONE detected
	// pattern (PII and/or secret) - incremented exactly once per
	// finding, unlike FilesWithPotentialPII+FilesWithPotentialSecrets,
	// which double-count a file that has BOTH a PII and a secret
	// pattern. THIS is the number that reconciles exactly against
	// SanitizedFiles+BlockedFiles+DetectedNotActioned - see the
	// "Descalce entre PII Indicators y WOULD_SANITIZE" fix.
	FilesWithAnyIndicator int
	SanitizedFiles        int // files that WOULD be sanitized (see SecurityFinding's header note)
	BlockedFiles          int
	// DetectedNotActioned: files where a pattern WAS detected but no
	// active policy rule modified or blocked it (stays ALLOWED) - see
	// EvaluateFile's "DETECTED but NOT SANITIZED" branch. Exists so
	// reports can reconcile "N files had an indicator" against "M
	// files were WOULD_SANITIZE" without the gap (N-M) looking like an
	// unexplained discrepancy - see the "Descalce entre PII Indicators
	// y WOULD_SANITIZE" fix.
	DetectedNotActioned       int
	PotentialOccurrences      int
	TokensPreventedExternally int // tokens that never entered the final sendable context because of BLOCKED files
	TokensActuallyRedacted    int // always 0 from a read-only analysis - see Transformed's doc comment
}

// Add folds one SecurityFinding into the running SecurityImpact,
// inferring the PII/secret split from the pattern labels EvaluateFile
// already attached (kept here, instead of extra bool parameters, so
// there is exactly one place — EvaluateFile's pattern names — that
// defines what counts as "PII" vs "secret").
func (s *SecurityImpact) Add(f SecurityFinding) {
	hadPII, hadSecret := false, false
	for _, p := range f.PatternsDetected {
		if containsFold(p, "PII") {
			hadPII = true
		} else {
			hadSecret = true
		}
	}
	if hadPII {
		s.FilesWithPotentialPII++
	}
	if hadSecret {
		s.FilesWithPotentialSecrets++
	}
	if len(f.PatternsDetected) > 0 {
		s.FilesWithAnyIndicator++
	}
	s.PotentialOccurrences += f.Occurrences
	switch f.Action {
	case "SANITIZED":
		s.SanitizedFiles++
	case "BLOCKED":
		s.BlockedFiles++
		s.TokensPreventedExternally += f.OriginalTokens
	case "ALLOWED":
		if len(f.PatternsDetected) > 0 {
			s.DetectedNotActioned++
		}
	}
	if f.Transformed {
		s.TokensActuallyRedacted += f.OriginalTokens - f.FinalTokens
	}
}

// EvaluateFile inspects one file's content against ps and returns an
// auditable SecurityFinding, the content that WOULD be used
// downstream, and the resulting FileState (see state.go). This
// function NEVER writes to disk and Transformed is always false on
// its output - see this file's header.
func EvaluateFile(path, content string, tokens int, ps PolicySet) (SecurityFinding, string, FileState) {
	secretHits := DetectSecrets(content)
	_, piiStats := MaskPII(content, ps.PII)

	var patterns []string
	occurrences := 0
	hasPrivateKey, hasBlockingSecret := false, false
	secretType := ""
	for _, h := range secretHits {
		patterns = append(patterns, string(h.Type))
		occurrences += h.Occurrences
		if secretType == "" {
			secretType = string(h.Type)
		}
		if h.Type == SecretPrivateKey {
			hasPrivateKey = true
		}
		if h.Type == SecretAPIKey || h.Type == SecretJWT || h.Type == SecretGenericCred {
			hasBlockingSecret = true
		}
	}
	hadPII := piiStats.TokensMasked > 0
	if hadPII {
		patterns = append(patterns, "structural PII pattern (Shannon entropy + shape rules)")
		occurrences += piiStats.TokensMasked
	}

	findingType, detector := classify(hadPII, secretType)
	score := scoreFor(occurrences, tokens)

	if len(patterns) == 0 {
		f := SecurityFinding{Path: path, Action: "ALLOWED", OriginalTokens: tokens, FinalTokens: tokens}
		return f, content, StateAllowed
	}

	// BLOCK takes priority over everything else — a critical policy
	// match means the file must never reach the final context at all.
	blockRule := ""
	if hasPrivateKey && ps.Security.BlockOnPrivateKey {
		blockRule = "security.block_on_private_key = true"
	} else if hasBlockingSecret && ps.Security.BlockOnAPIKey {
		blockRule = "security.block_on_api_key = true"
	} else if hadPII && ps.PIIExternalModel == "deny" {
		blockRule = "pii.external_model = deny"
	}
	if blockRule != "" {
		f := SecurityFinding{
			Path: path, Type: findingType, Score: score, PatternsDetected: patterns, Occurrences: occurrences,
			Action: "BLOCKED", OriginalTokens: tokens, FinalTokens: 0, RuleID: blockRule, Detector: detector,
		}
		return f, "", StateBlocked
	}

	// A pattern was found, but does any active rule actually MODIFY
	// the content for it? If not (e.g. an IP address alone, or a
	// credential-shaped hit while sanitize_on_* is off), this is the
	// rule 1 requirement made explicit: DETECTED but NOT SANITIZED —
	// the file stays ALLOWED, and report renderers must show
	// PatternsDetected without implying any action was taken.
	willSanitizeSecret := hasBlockingSecret && (ps.Security.SanitizeOnGenericCredential || ps.Security.SanitizeOnJWT)
	if !hadPII && !willSanitizeSecret {
		f := SecurityFinding{
			Path: path, Type: findingType, Score: score, PatternsDetected: patterns, Occurrences: occurrences,
			Action: "ALLOWED", OriginalTokens: tokens, FinalTokens: tokens, Detector: detector,
			RuleID: "detected, no active policy rule modifies this pattern (see config/policy/security.json)",
		}
		return f, content, StateAllowed
	}

	// Otherwise: this WOULD be sanitized (mask PII, redact
	// credential-shaped secrets) if applied - the content returned
	// here reflects that hypothetical result for downstream token
	// accounting, but Transformed stays false: context-trace's own
	// run never writes this back to disk.
	sanitized := content
	if hadPII {
		sanitized, _ = MaskPII(sanitized, ps.PII)
	}
	redacted := 0
	if willSanitizeSecret {
		sanitized, redacted = RedactSecrets(sanitized)
	}

	// RuleID is always a SINGLE policy line, never a combined "a / b"
	// string — each field must represent exactly one thing (see the
	// "rule_id must not bundle two policies" fix). When PII masking
	// and secret redaction BOTH fired for this file, the PII rule
	// (the more privacy-sensitive one) is the primary RuleID and the
	// secret rule that also applied goes in PolicyRule; when only one
	// of the two fired, that one alone is RuleID and PolicyRule stays
	// empty.
	ruleID, policyRule := "", ""
	switch {
	case hadPII && willSanitizeSecret:
		ruleID = "pii_masking.min_score"
		policyRule = secretSanitizeRuleID(ps)
	case hadPII:
		ruleID = "pii_masking.min_score"
	default:
		ruleID = secretSanitizeRuleID(ps)
	}

	f := SecurityFinding{
		Path: path, Type: findingType, Score: score, PatternsDetected: patterns, Occurrences: occurrences,
		Action: "SANITIZED", OriginalTokens: tokens, FinalTokens: tokens, Detector: detector,
		RuleID: ruleID, PolicyRule: policyRule,
		CredentialShapedValues: redacted,
	}
	return f, sanitized, StateSanitized
}

// secretSanitizeRuleID names the SPECIFIC secret.sanitize_on_* switch
// that is actually active, instead of the vague wildcard
// "security.sanitize_on_*" — every rule_id/policy_rule value must
// point at one real config key (see PolicySet.Security in
// policy_cascade.go).
func secretSanitizeRuleID(ps PolicySet) string {
	if ps.Security.SanitizeOnGenericCredential {
		return "security.sanitize_on_generic_credential"
	}
	if ps.Security.SanitizeOnJWT {
		return "security.sanitize_on_jwt"
	}
	return "security.sanitize_on_*"
}

func containsFold(s, substr string) bool {
	sl, subl := len(s), len(substr)
	if subl == 0 {
		return true
	}
	for i := 0; i+subl <= sl; i++ {
		match := true
		for j := 0; j < subl; j++ {
			a, b := s[i+j], substr[j]
			if a >= 'A' && a <= 'Z' {
				a += 'a' - 'A'
			}
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
