// secrets.go — structural, deterministic secret-indicator detection:
// the same "no dictionaries, no language rules" discipline pii.go
// already follows, applied to credential-shaped tokens instead of
// personal-data-shaped ones. Every pattern here is a well-known,
// PUBLIC shape (a JWT has three dot-separated base64url segments by
// specification; a PEM key block has a fixed header/footer; cloud
// vendors publish their own key-prefix conventions) — nothing here is
// a secret itself, only a recognizer for the SHAPE a secret takes.
package sanitize

import "regexp"

// SecretPatternType names a family of secret-shaped indicators — kept
// as a small closed set so findings.go and every report renderer can
// group occurrences without inventing new categories.
type SecretPatternType string

const (
	SecretAPIKey      SecretPatternType = "API key pattern"
	SecretJWT         SecretPatternType = "JWT-like token"
	SecretPrivateKey  SecretPatternType = "Private key block"
	SecretGenericCred SecretPatternType = "Generic credential assignment"
	SecretIPAddress   SecretPatternType = "IP address"
)

// secretPattern pairs a SecretPatternType with the regexp that
// recognizes it.
type secretPattern struct {
	kind SecretPatternType
	re   *regexp.Regexp
}

// secretPatterns is intentionally small and vendor-shape based, not an
// attempt at an exhaustive secret-scanning product — see
// docs/i18n/en/COMMANDS.md § context-trace for the "possible
// indicators, not proof" disclaimer every caller must keep attached to
// these results.
var secretPatterns = []secretPattern{
	{SecretAPIKey, regexp.MustCompile(`\b(?:sk-[A-Za-z0-9]{16,}|AKIA[0-9A-Z]{16}|ghp_[A-Za-z0-9]{20,}|xox[baprs]-[A-Za-z0-9-]{10,})\b`)},
	{SecretJWT, regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}\b`)},
	{SecretPrivateKey, regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`)},
	{SecretGenericCred, regexp.MustCompile(`(?i)\b(?:api[_-]?key|secret|token|password|passwd)\b\s*[:=]\s*["']?[A-Za-z0-9_\-/+]{12,}["']?`)},
	{SecretIPAddress, regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4]\d|1?\d?\d)\.){3}(?:25[0-5]|2[0-4]\d|1?\d?\d)\b`)},
}

// SecretHit is one recognized pattern family and how many times it
// occurred in a given piece of content.
type SecretHit struct {
	Type        SecretPatternType
	Occurrences int
}

// DetectSecrets scans content for every known secret-shaped pattern
// and returns one SecretHit per family that actually matched — never
// panics, safe on empty input. This is DETECTION only: it never
// modifies content (see RedactSecrets for the sanitizing step).
func DetectSecrets(content string) []SecretHit {
	if content == "" {
		return nil
	}
	var hits []SecretHit
	for _, p := range secretPatterns {
		matches := p.re.FindAllString(content, -1)
		if len(matches) > 0 {
			hits = append(hits, SecretHit{Type: p.kind, Occurrences: len(matches)})
		}
	}
	return hits
}

// RedactSecrets replaces every recognized secret-shaped match with a
// fixed, non-reversible marker ("[REDACTED]") and reports how many
// substitutions were made in total. IP addresses are treated as a
// lower-severity indicator and are NOT redacted by this function —
// only credential-shaped patterns are (see findings.go for how policy
// decides whether IP-only hits still warrant SANITIZED vs ALLOWED).
func RedactSecrets(content string) (string, int) {
	if content == "" {
		return content, 0
	}
	out := content
	total := 0
	for _, p := range secretPatterns {
		if p.kind == SecretIPAddress {
			continue
		}
		out, total = redactOne(out, p.re, total)
	}
	return out, total
}

func redactOne(content string, re *regexp.Regexp, runningTotal int) (string, int) {
	matches := re.FindAllString(content, -1)
	if len(matches) == 0 {
		return content, runningTotal
	}
	return re.ReplaceAllString(content, "[REDACTED]"), runningTotal + len(matches)
}
