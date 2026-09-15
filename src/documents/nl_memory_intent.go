// nl_memory_intent.go — detects "remember this" / "save this to
// memory" intent written in plain natural language (Spanish or
// English), so `/memory` is no longer the ONLY way to persist the
// last exchange into memory.md — same philosophy as nl_intent.go's
// file-creation detector, and deliberately built the same way: a
// light heuristic (a save/remember verb, optionally paired with the
// word "memoria"/"memory"), not a natural-language understanding
// system. `/memory` keeps working exactly as before.
package documents

import "regexp"

// memoryIntentRe recognizes a deliberately wide set of phrasings.
// "guarda"/"save"/"recuerda"/"remember" alone are ALSO enough — a bare
// "recuérdalo" or "remember that" is common phrasing for exactly this
// intent — so the word "memoria"/"memory" is not required, only
// sufficient to disambiguate from an unrelated "guarda" (e.g. "guarda
// silencio" would still incorrectly match "guarda" alone, which is
// why callers should treat this as a hint on a SHORT, otherwise
// commandless message — see cli/nl_save.go's caller for that guard —
// not as license to intercept every message containing these words).
var memoryIntentRe = regexp.MustCompile(`(?i)\b(` +
	// Spanish
	`recuerda|recu[eé]rdalo|recu[eé]rdame|acu[eé]rdate|` +
	`guarda(?:lo)?\s+(?:esto\s+)?(?:la\s+)?(?:en\s+(?:la\s+)?)?memoria|` +
	`guarda\s+(?:esto|la\s+conversaci[oó]n|el\s+chat)|memoriza(?:lo)?|` +
	// English
	`remember\s+(?:this|that)|save\s+(?:this\s+to\s+memory|the\s+chat|the\s+conversation)|` +
	`memorize\s+this|keep\s+this\s+in\s+memory` +
	`)\b`)

// DetectMemoryIntent reports whether text expresses "save/remember
// the last exchange" intent.
func DetectMemoryIntent(text string) bool {
	return memoryIntentRe.MatchString(text)
}
