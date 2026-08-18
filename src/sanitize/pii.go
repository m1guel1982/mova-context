// pii.go — Technical/Structural PII Masking: an OPTIONAL Token Firewall
// stage (see budget/gated_context.go) that replaces candidate-PII
// tokens in Focus/Memory with a deterministic [PII_xxxxxxxx] pseudonym
// BEFORE anything is counted or sent to a model.
//
// IMPORTANT — read before enabling this in a real project: this is a
// heuristic, structural mitigation, not a legal anonymization or
// compliance guarantee. It does not, by itself, make a project comply
// with Ley 21.719, GDPR, or any other data-protection regulation — see
// docs/i18n/{es,en}/COMMANDS.md § PII Masking for the full disclaimer.
//
// Design constraints this file follows (see the project's own
// requirements doc for the PII Masking feature):
//   - Agnostic to language: no word lists, no dictionaries, no
//     grammar/spelling rules for Spanish/English/any language. Every
//     signal below is either a MATHEMATICAL property of the token's
//     characters (Shannon entropy) or a STRUCTURAL property of its
//     shape (digit ratio, separators, case, length) — the same
//     features would fire on a RUT, a phone number, or an email
//     regardless of what language the surrounding sentence is in.
//   - Deterministic: MaskPII(x) always produces the same pseudonym for
//     the same input token (FNV-1a is not cryptographic, and isn't
//     meant to be — determinism, not secrecy, is the goal: seeing the
//     same [PII_xxxxxxxx] twice tells you it was the same original
//     value, without ever reconstructing what that value was from the
//     tag alone).
//   - Off by default, config-driven: see core.PIIMaskingEnabled
//     (project.json's opt-in) and LoadPIIPolicy (config/policy.json's
//     thresholds) — nothing here is a magic number.
package sanitize

import (
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"unicode"
)

// PIIStats summarizes what one MaskPII call replaced — surfaced in
// mova-budget-report.md's "PII Masking" section (see
// budget/report_pipeline.go), same "show what changed, don't just
// silently apply it" rule sanitize.Stats already follows.
type PIIStats struct {
	TokensScanned int // total whitespace-delimited tokens examined
	TokensMasked  int // tokens whose score reached policy.MinScore
}

// MaskPII scans s token by token (whitespace-delimited) and replaces
// any token whose combined word-shape + entropy score reaches
// policy.MinScore with a deterministic pseudonym tag. Tokens under
// policy.MinTokenLength are never masked (too short for either signal
// to be meaningful — matches policy.json's own documented rationale).
// Safe on empty input; never panics.
func MaskPII(s string, policy PIIPolicy) (string, PIIStats) {
	if s == "" {
		return s, PIIStats{}
	}
	var stats PIIStats
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return s, stats
	}

	replacer := make([]string, 0, len(fields)*2)
	rest := s
	for _, tok := range fields {
		idx := strings.Index(rest, tok)
		if idx < 0 {
			// Should not happen (tok came from strings.Fields(rest's
			// origin)), but never let a scan mismatch corrupt output.
			replacer = append(replacer, tok)
			continue
		}
		replacer = append(replacer, rest[:idx])
		rest = rest[idx+len(tok):]

		stats.TokensScanned++
		masked, wasMasked := maskToken(tok, policy)
		replacer = append(replacer, masked)
		if wasMasked {
			stats.TokensMasked++
		}
	}
	replacer = append(replacer, rest)
	return strings.Join(replacer, ""), stats
}

// maskToken evaluates ONE whitespace-delimited token: strips leading/
// trailing punctuation into prefix/core/suffix (so "juan@test.cl," only
// scores "juan@test.cl", keeping the trailing comma intact), scores the
// core, and — if it clears policy.MinScore — replaces the core with a
// deterministic pseudonym while preserving prefix/suffix untouched.
func maskToken(tok string, policy PIIPolicy) (string, bool) {
	prefix, core, suffix := trimEdges(tok)
	if len([]rune(core)) < policy.MinTokenLength {
		return tok, false
	}
	if isAlreadyTagged(core, policy) {
		return tok, false // idempotent: a previously-masked tag is never re-scored
	}

	score := wordShapeScore(core, policy.ShapeRules) * policy.ShapeWeight
	score += entropyScore(core) * policy.EntropyWeight

	if score < policy.MinScore {
		return tok, false
	}
	return prefix + pseudonym(core, policy) + suffix, true
}

// trimEdges splits a raw whitespace-delimited token into its leading
// punctuation, alphanumeric-ish core, and trailing punctuation — a
// STRUCTURAL split (Unicode letter/digit/@ vs. everything else), not a
// language rule. "@" stays part of the core deliberately: it is a
// structural signal (see wordShapeScore's AtSymbolBonus), not a letter.
func trimEdges(tok string) (prefix, core, suffix string) {
	runes := []rune(tok)
	isCore := func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '@' || r == '-' || r == '.' || r == '_' || r == '+'
	}
	start := 0
	for start < len(runes) && !isCore(runes[start]) {
		start++
	}
	end := len(runes)
	for end > start && !isCore(runes[end-1]) {
		end--
	}
	// Trailing "." is almost always sentence punctuation, not part of an
	// identifier's shape (unlike a "." inside a RUT/email) — pull ONE
	// trailing "." back out if the core would otherwise end with it and
	// isn't itself digit-heavy right before it (a crude but purely
	// structural heuristic, no language rule involved).
	if end > start && runes[end-1] == '.' && !(end-1 > start && unicode.IsDigit(runes[end-2])) {
		end--
	}
	return string(runes[:start]), string(runes[start:end]), string(runes[end:])
}

func isAlreadyTagged(core string, policy PIIPolicy) bool {
	openTag := strings.SplitN(policy.TagFormat, "%s", 2)[0]
	return openTag != "" && strings.HasPrefix(core, openTag)
}

// wordShapeScore returns a 0..1 structural score: how much this
// token's SHAPE (digit density, separators, case, length, presence of
// "@") resembles a formatted identifier (RUT, phone, email, code) —
// never what the characters spell. Every bonus/threshold is read from
// rules (config/policy.json), never hardcoded here.
func wordShapeScore(core string, rules PIIShapeRules) float64 {
	runes := []rune(core)
	n := len(runes)
	if n == 0 {
		return 0
	}
	var digits, uppers, seps int
	hasAt := false
	for _, r := range runes {
		switch {
		case unicode.IsDigit(r):
			digits++
		case r == '@':
			hasAt = true
		case unicode.IsUpper(r):
			uppers++
		case r == '.' || r == '-' || r == '_' || r == '/':
			seps++
		}
	}
	digitRatio := float64(digits) / float64(n)
	upperRatio := float64(uppers) / float64(n)

	var score float64
	if hasAt {
		score += rules.AtSymbolBonus
	}
	if digitRatio > rules.DigitRatioThreshold {
		score += rules.DigitRatioBonus
	}
	if seps >= rules.SeparatorMinCount && digitRatio > rules.SeparatorDigitRatioThreshold {
		score += rules.SeparatorBonus
	}
	if n >= rules.LongTokenBonusLen && digits > 0 && digits < n {
		score += rules.LongMixedTokenBonus
	}
	if digits == 0 && upperRatio > rules.UpperRunRatioThreshold && n >= 3 {
		score += rules.UpperRunBonus
	}
	return clamp01(score)
}

// entropyScore returns the token's Shannon entropy normalized against
// the theoretical maximum for its own length (log2(min(n, 26)) — a
// purely mathematical property of the character distribution, agnostic
// to what alphabet/language those characters belong to. A short,
// high-diversity token (e.g. a mixed-case alphanumeric ID) scores near
// 1; a long run of one repeated character scores near 0.
func entropyScore(core string) float64 {
	runes := []rune(core)
	n := len(runes)
	if n <= 1 {
		return 0
	}
	counts := make(map[rune]int, n)
	for _, r := range runes {
		counts[r]++
	}
	var h float64
	for _, c := range counts {
		p := float64(c) / float64(n)
		h -= p * math.Log2(p)
	}
	maxBits := math.Log2(math.Min(float64(n), 26))
	if maxBits <= 0 {
		return 0
	}
	return clamp01(h / maxBits)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// pseudonym computes the deterministic, cero-allocation-per-word tag
// for core using FNV-1a (non-cryptographic, chosen for speed — see this
// file's header). Case-insensitive on purpose: "Juan.Perez" and
// "juan.perez" are the same underlying value and must collapse to the
// same tag.
func pseudonym(core string, policy PIIPolicy) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(strings.ToLower(core)))
	sum := h.Sum64()
	hex := fmt.Sprintf("%016x", sum)
	n := policy.HashLength
	if n <= 0 || n > len(hex) {
		n = len(hex)
	}
	format := policy.TagFormat
	if format == "" {
		format = "[PII_%s]"
	}
	return fmt.Sprintf(format, hex[:n])
}
