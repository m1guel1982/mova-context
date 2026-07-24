// save_modifiers.go — detects "append instead of overwrite" and
// "overwrite / don't overwrite" intent in plain language, so the same
// control `/save -append`/`-overwrite`/`-no-overwrite` give explicitly in
// mova.local (see cli/chat_save.go) also works just by phrasing a natural-
// language save request a certain way — "agrega al final del reporte",
// "no lo sobreescribas", "reemplaza report.pdf" — the same experience
// DetectSaveIntent (nl_intent.go) already gives for creation itself.
//
// This only decides two booleans (documents.SaveRequest's Append and
// Overwrite/OverwriteExplicit) — it never decides WHERE to save or WHAT
// to save; that's still DetectSaveIntent's job. Callers (cli/nl_save.go,
// mcp/nl_save.go) run both detectors over the same message and combine
// their results.
package documents

import "regexp"

// appendModifierRe: "add to the end" in Spanish and English, several
// common conjugations/phrasings each.
var appendModifierRe = regexp.MustCompile(`(?i)\b(` +
	`agrega\w* al final|añad\w* al final|agregando al final|añadiendo al final|` +
	`al final del archivo|al final del documento|` +
	`append\w*|add(?:ing)? (?:it |this )?to the end` +
	`)\b`)

// overwriteFalseModifierRe: an explicit "do NOT overwrite" — checked
// BEFORE overwriteTrueModifierRe, since e.g. "no sobreescribas" contains
// the same verb stem "sobrescrib-" that a bare positive mention would
// match; the negation must win when both are present in the same phrase.
var overwriteFalseModifierRe = regexp.MustCompile(`(?i)\b(` +
	`no\s+(?:lo\s+|la\s+)?sobre?escrib\w*|` +
	`no\s+(?:lo\s+|la\s+)?reemplac\w*|no\s+(?:lo\s+|la\s+)?remplac\w*|` +
	`sin\s+sobre?escribir|` +
	`(?:don'?t|do\s*not)\s+overwrit\w*|(?:don'?t|do\s*not)\s+replac\w*|` +
	`without\s+overwriting` +
	`)\b`)

// overwriteTrueModifierRe: "overwrite it"/"replace it" as an explicit
// instruction that THIS save should overwrite whatever is already there
// — "sobreescribe", "reemplaza report.pdf", "overwrite it".
var overwriteTrueModifierRe = regexp.MustCompile(`(?i)\b(` +
	`sobre?escrib\w*|reemplaz\w*|remplaz\w*|` +
	`overwrit\w*|replac\w*` +
	`)\b`)

// SaveModifiers is what DetectSaveModifiers found — zero-value means
// "nothing detected, use save's normal defaults" (overwrite, don't append).
type SaveModifiers struct {
	Append         bool
	OverwriteSet   bool // an explicit overwrite/no-overwrite instruction was found
	OverwriteValue bool // only meaningful when OverwriteSet is true
}

// DetectSaveModifiers scans a chat message for append/overwrite intent,
// independent of DetectSaveIntent (which decides the file/directory
// itself). Meant to be run on the SAME message DetectSaveIntent already
// matched — see cli/nl_save.go and mcp/nl_save.go.
func DetectSaveModifiers(text string) SaveModifiers {
	var m SaveModifiers
	switch {
	case overwriteFalseModifierRe.MatchString(text):
		m.OverwriteSet, m.OverwriteValue = true, false
	case overwriteTrueModifierRe.MatchString(text):
		m.OverwriteSet, m.OverwriteValue = true, true
	}
	if appendModifierRe.MatchString(text) {
		m.Append = true
	}
	return m
}
