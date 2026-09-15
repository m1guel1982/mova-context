// rename_intent.go — detects "rename X to Y" intent in plain language.
package documents

import "regexp"

var renameVerbRe = regexp.MustCompile(`(?i)\b(` +
	`renombra|renombrar|reenombra|cambia(?:le)?\s+el\s+nombre|` +
	`rename|renames` +
	`)\b`)

// renamePairRe splits "X a/to Y" into the source and destination text —
// each half is then resolved through extractFileTarget (nl_intent.go)
// separately, the same robust path extraction every other detector in
// this package uses, instead of assuming each half is already a clean
// single token (broke on absolute paths and connector words).
var renamePairRe = regexp.MustCompile(`(?i)^(?:el\s+archivo|la\s+carpeta|el\s+directorio|the\s+file|the\s+folder|the\s+directory)?\s*(.+?)\s+(?:a|to)\s+(.+)$`)

// DetectRenameIntent scans a chat message for "rename X to Y" intent.
// ok is false when no verb, or no resolvable "X to/a Y" pair, was found.
func DetectRenameIntent(text string) (from, to string, ok bool) {
	for _, clause := range splitClauses(text) {
		if !renameVerbRe.MatchString(clause) {
			continue
		}
		stripped := renameVerbRe.ReplaceAllString(clause, "")
		m := renamePairRe.FindStringSubmatch(stripped)
		if m == nil {
			continue
		}
		fromTarget, fromOK := extractFileTarget(m[1])
		toTarget, toOK := extractFileTarget(m[2])
		if fromOK && toOK {
			return fromTarget, toTarget, true
		}
	}
	return "", "", false
}
