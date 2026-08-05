// Package sanitize implements the Token Firewall's first stage: cheap,
// 100%-deterministic noise removal from text BEFORE it's counted or
// sent to a model — no AI, no network call, no new dependency. Every
// rule here is reversible-in-spirit (nothing is summarized or
// reworded, only exact repeats/formatting noise are collapsed), so the
// model never loses information it would have used, only clutter it
// would have skimmed past anyway.
//
// Called from mova.local/budget.BuildGatedContext (see
// budget/gated_context.go) — the single chokepoint every door (mova
// run, mova chat, mova jobs run, mova agents run, the TUI) already
// shares, so this stage reaches all of them automatically without a
// second integration point.
package sanitize

import (
	"regexp"
	"strconv"
	"strings"
)

// Config maps project.json's "budget"."sanitize" object.
type Config struct {
	Enabled       bool `json:"enabled"`
	DedupeLogs    bool `json:"dedupe_logs"`    // collapse runs of near-identical lines (e.g. repeated log lines)
	StripBlank    bool `json:"strip_blank"`    // collapse 3+ consecutive blank lines to 1
	StripComments bool `json:"strip_comments"` // remove large comment blocks (off by default — can remove intent the task needs)
}

// DefaultConfig is what applies when project.json's "budget" doesn't
// declare a "sanitize" object at all — safe, conservative, on for the
// things that are never semantically meaningful (log spam, blank-line
// padding), off for the one rule that could remove real intent.
func DefaultConfig() Config {
	return Config{Enabled: true, DedupeLogs: true, StripBlank: true, StripComments: false}
}

// Stats summarizes what one Text() call actually removed — surfaced in
// mova-budget-report.md's "Sanitizer" section (see budget/report_pipeline.go)
// so the saving is visible, not just silently applied.
type Stats struct {
	LinesRemoved    int
	BlankRemoved    int
	CommentsRemoved int
	CharsRemoved    int
}

var blankRunRe = regexp.MustCompile(`\n{4,}`)

// Text applies every enabled rule to s, returning the cleaned text and
// what was removed. Safe to call on empty/tiny input — never panics,
// never returns less than the caller's minimum expectation (an empty
// string in, an empty string out).
func Text(s string, cfg Config) (string, Stats) {
	if !cfg.Enabled || s == "" {
		return s, Stats{}
	}
	var stats Stats
	before := len(s)

	if cfg.DedupeLogs {
		s, stats.LinesRemoved = dedupeRepeatedLines(s)
	}
	if cfg.StripBlank {
		matches := blankRunRe.FindAllString(s, -1)
		stats.BlankRemoved = len(matches)
		s = blankRunRe.ReplaceAllString(s, "\n\n\n")
	}
	if cfg.StripComments {
		s, stats.CommentsRemoved = stripLargeCommentBlocks(s)
	}

	stats.CharsRemoved = before - len(s)
	return s, stats
}

// leadingTimestampRe matches a common log-line timestamp prefix
// ("2026-08-02 03:00:01 " or "2026-08-02T03:00:01Z ", with or without
// milliseconds) — stripped ONLY for comparison purposes, never from the
// actual output, so a real log (where every line's timestamp differs by
// design) can still be recognized as "the same message, N times".
var leadingTimestampRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(\.\d+)?Z?\s*`)

// withoutTimestamp returns line with any recognized leading timestamp
// removed, for comparison only.
func withoutTimestamp(line string) string {
	return leadingTimestampRe.ReplaceAllString(line, "")
}

// dedupeRepeatedLines collapses 3+ consecutive lines that are identical
// ONCE A LEADING TIMESTAMP IS IGNORED — the classic "50 lines of INFO
// 200 OK" case, which in a real log almost always has a different
// timestamp on every single line (otherwise byte-identical). The first
// occurrence (timestamp included) is kept, followed by a "[×N]" counter;
// everything after it in the run is omitted. Order-preserving, single
// pass, no lookback beyond the immediately previous line so it stays O(n).
func dedupeRepeatedLines(s string) (string, int) {
	lines := strings.Split(s, "\n")
	var out []string
	removed := 0

	i := 0
	for i < len(lines) {
		line := withoutTimestamp(lines[i])
		run := 1
		for i+run < len(lines) && withoutTimestamp(lines[i+run]) == line {
			run++
		}
		if run >= 3 && strings.TrimSpace(line) != "" {
			out = append(out, lines[i], "  [×"+strconv.Itoa(run)+" repeticiones idénticas omitidas]")
			removed += run - 1
		} else {
			for k := 0; k < run; k++ {
				out = append(out, lines[i+k])
			}
		}
		i += run
	}
	return strings.Join(out, "\n"), removed
}

// stripLargeCommentBlocks removes comment-only line runs of 5+ lines
// (// or # style) — off by default (Config.StripComments), since a task
// that's specifically about documentation/comments needs them intact.
func stripLargeCommentBlocks(s string) (string, int) {
	lines := strings.Split(s, "\n")
	var out []string
	removed := 0

	isComment := func(l string) bool {
		t := strings.TrimSpace(l)
		return strings.HasPrefix(t, "//") || strings.HasPrefix(t, "#")
	}

	i := 0
	for i < len(lines) {
		if isComment(lines[i]) {
			j := i
			for j < len(lines) && isComment(lines[j]) {
				j++
			}
			if j-i >= 5 {
				out = append(out, "  [bloque de "+strconv.Itoa(j-i)+" líneas de comentarios omitido]")
				removed += j - i
				i = j
				continue
			}
		}
		out = append(out, lines[i])
		i++
	}
	return strings.Join(out, "\n"), removed
}
