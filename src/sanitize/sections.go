// sections.go — the Sanitizer's entry point: applies every rule in
// this package to a *core.ContextSections in place, targeting Focus
// and Memory specifically (the two sections most likely to carry real
// file/log content — Agents/Skills/Prompt are curated project files,
// rarely noisy in the way this package targets). Called once, from
// mova.local/budget.BuildGatedContext, before token counting.
package sanitize

import (
	"strings"

	"mova.local/core"
)

// Apply sanitizes sections.Focus and sections.Memory in place and
// returns combined Stats. A Config with Enabled=false (or the zero
// value) is a guaranteed no-op — sections is returned byte-for-byte
// unchanged, so a project that never opts in sees zero behavior change.
func Apply(sections *core.ContextSections, cfg Config) Stats {
	if !cfg.Enabled || sections == nil {
		return Stats{}
	}
	var total Stats
	if sections.Focus != "" {
		cleaned, stats := ApplyFocus(sections.Focus, cfg)
		sections.Focus = cleaned
		total = addStats(total, stats)
	}
	if sections.Memory != "" {
		cleaned, stats := ApplyMemory(sections.Memory, cfg)
		sections.Memory = cleaned
		total = addStats(total, stats)
	}
	return total
}

// ApplyFocus runs both Focus-specific rules (cross-file leading-block
// dedup, then the general Text rules) on a raw Focus string — exported
// so mova.local/budget's Context Cache (contextcache.go) can call the
// exact same logic per-piece when its cache misses, instead of a second
// copy of "how Focus gets sanitized". Preserves core/engine.go's
// "\n\n---\n## FOCUS\n" section header exactly (see splitFocusBlocks) —
// sanitizing never removes structure the rest of Mova depends on to
// find where Focus starts.
func ApplyFocus(focus string, cfg Config) (string, Stats) {
	preamble, blocks := splitFocusBlocks(focus)
	dedupeResult := DedupeLeadingBlocks(blocks, 3)
	rebuilt := joinFocusBlocks(preamble, dedupeResult.Blocks)

	cleaned, stats := Text(rebuilt, cfg)
	stats.LinesRemoved += dedupeResult.Removed // count of duplicated headers collapsed
	return cleaned, stats
}

// ApplyMemory runs the general Text rules on a raw Memory string — no
// cross-file dedup (memory.md is already one file), but exported for
// the same Context Cache reuse reason as ApplyFocus.
func ApplyMemory(memory string, cfg Config) (string, Stats) {
	return Text(memory, cfg)
}

func addStats(a, b Stats) Stats {
	return Stats{
		LinesRemoved:    a.LinesRemoved + b.LinesRemoved,
		BlankRemoved:    a.BlankRemoved + b.BlankRemoved,
		CommentsRemoved: a.CommentsRemoved + b.CommentsRemoved,
		CharsRemoved:    a.CharsRemoved + b.CharsRemoved,
	}
}

// SplitFocusBlocks parses render.go's "FOCUS:<path>\n<content>" marker
// convention back into individual FileBlocks, discarding the
// "\n\n---\n## FOCUS\n" section header core/engine.go prepends —
// exported for mova.local/budget's per-file report (which only needs
// the files, not the header) to reuse the exact same parsing
// ApplyFocus relies on internally, instead of a second copy of it.
func SplitFocusBlocks(focus string) []FileBlock {
	_, blocks := splitFocusBlocks(focus)
	return blocks
}

// splitFocusBlocks is ApplyFocus's internal version: unlike
// SplitFocusBlocks, it ALSO returns the preamble (everything before the
// first "FOCUS:" marker — core/engine.go's "\n\n---\n## FOCUS\n"
// header), which joinFocusBlocks needs to reconstruct sections.Focus
// without silently dropping that header.
func splitFocusBlocks(focus string) (preamble string, blocks []FileBlock) {
	idx := strings.Index(focus, "FOCUS:")
	if idx < 0 {
		return focus, nil // no recognizable markers — nothing to split, treat it all as an opaque preamble
	}
	preamble = focus[:idx]
	rest := focus[idx:]

	parts := strings.Split(rest, "\nFOCUS:")
	for i, p := range parts {
		if i == 0 {
			p = strings.TrimPrefix(p, "FOCUS:")
		}
		nl := strings.IndexByte(p, '\n')
		if nl < 0 {
			continue
		}
		blocks = append(blocks, FileBlock{Name: p[:nl], Content: p[nl+1:]})
	}
	return preamble, blocks
}

// joinFocusBlocks reassembles preamble+blocks back into the exact
// format splitFocusBlocks parsed, so downstream consumers of
// sections.Focus (reports, the assembled prompt) see no format change
// — the "## FOCUS" header is always preserved exactly as core/engine.go
// wrote it.
func joinFocusBlocks(preamble string, blocks []FileBlock) string {
	var b strings.Builder
	b.WriteString(preamble)
	for _, blk := range blocks {
		b.WriteString("FOCUS:" + blk.Name + "\n" + blk.Content)
	}
	return b.String()
}
