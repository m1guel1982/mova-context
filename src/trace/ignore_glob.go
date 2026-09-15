// ignore_glob.go — the `--ignore` pattern matcher: glob/extglob-style
// matching against a path RELATIVE to the repository root, applied at
// the earliest possible point (Discovery/Focus, see analyzer.go)
// before any tokenization or PII/security evaluation runs, so ignored
// files never cost a single token of analysis and never enter
// CANDIDATE.
//
// Honesty note (same convention as relevance.go/patcher.go): this is
// a small, pure-Go, hand-written matcher supporting the patterns the
// spec's own examples require - "**" (any number of path segments)
// and a single-level "!(a,b,c)" negation per segment - not a full
// bash extglob implementation (nested extglobs, "?()", "+()", "@()"
// are NOT supported). This covers every example in the spec
// ("docs/**", "docs/!(en)/**", "docs/**/*.png") without vendoring a
// shell-glob library.
package trace

import (
	"path/filepath"
	"strings"
)

// MatchesIgnorePattern reports whether relPath (forward-slash,
// relative to the repo root) matches ANY of patterns.
func MatchesIgnorePattern(relPath string, patterns []string) bool {
	relPath = filepath.ToSlash(relPath)
	for _, p := range patterns {
		if matchGlob(filepath.ToSlash(p), relPath) {
			return true
		}
	}
	return false
}

func matchGlob(pattern, path string) bool {
	return matchSegments(strings.Split(pattern, "/"), strings.Split(path, "/"))
}

// matchSegments recursively matches pattern segments against path
// segments, handling "**" as "zero or more segments" via the two
// classic recursive branches (skip the "**" itself, or consume one
// path segment and retry).
func matchSegments(pat, seg []string) bool {
	if len(pat) == 0 {
		return len(seg) == 0
	}
	if pat[0] == "**" {
		if matchSegments(pat[1:], seg) {
			return true
		}
		if len(seg) > 0 && matchSegments(pat, seg[1:]) {
			return true
		}
		return false
	}
	if len(seg) == 0 {
		return false
	}
	if !matchOneSegment(pat[0], seg[0]) {
		return false
	}
	return matchSegments(pat[1:], seg[1:])
}

// matchOneSegment matches a single path segment against a single
// pattern segment - either extglob negation ("!(en)", "!(en,fr)") or a
// plain filepath.Match-style glob ("*.png", "index.*").
func matchOneSegment(pat, seg string) bool {
	if strings.HasPrefix(pat, "!(") && strings.HasSuffix(pat, ")") {
		inner := pat[2 : len(pat)-1]
		for _, alt := range strings.FieldsFunc(inner, func(r rune) bool { return r == ',' || r == '|' }) {
			if strings.TrimSpace(alt) == seg {
				return false
			}
		}
		return true
	}
	ok, err := filepath.Match(pat, seg)
	return err == nil && ok
}
