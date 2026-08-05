// dedupe.go — the Sanitizer's second rule: when Focus pulls in several
// files that share an identical leading block (a license header, a
// common import group), keep it once and reference the rest — the same
// "exact repeats only, never summarized" rule sanitize.go's other rules
// follow. Deliberately narrow in scope (only the leading block of a
// file, not arbitrary mid-file blocks) — matching arbitrary duplicate
// blocks anywhere risks colliding with legitimately-repeated code
// (e.g. two similar test cases), which is exactly the kind of surprise
// a zero-AI, deterministic tool must never spring on a developer.
package sanitize

import "strings"

// FileBlock is one file's content, tagged with its display name (as it
// already appears in Focus output, e.g. "FOCUS: src/checkout.js").
type FileBlock struct {
	Name    string
	Content string
}

// DedupeResult is what DedupeLeadingBlocks returns: the same files,
// with any file whose leading block matched an earlier file's leading
// block having that block replaced by a one-line reference.
type DedupeResult struct {
	Blocks  []FileBlock
	Removed int // how many files had a block deduplicated
}

// DedupeLeadingBlocks compares each file's first minLines non-empty
// lines against every earlier file's — an identical match (imports,
// license headers, boilerplate preambles) is replaced with a pointer
// back to the first file that had it, instead of repeating the full
// text again for every subsequent file.
func DedupeLeadingBlocks(files []FileBlock, minLines int) DedupeResult {
	if minLines <= 0 {
		minLines = 3
	}
	seen := map[string]string{} // leading block text -> file name that first had it
	out := make([]FileBlock, len(files))
	removed := 0

	for i, f := range files {
		block, rest := leadingBlock(f.Content, minLines)
		out[i] = f
		if block == "" {
			continue
		}
		if firstFile, ok := seen[block]; ok {
			out[i].Content = "  [encabezado idéntico al de " + firstFile + " — omitido]\n" + rest
			removed++
			continue
		}
		seen[block] = f.Name
	}
	return DedupeResult{Blocks: out, Removed: removed}
}

// leadingBlock returns the first minLines non-blank lines of content
// (joined back with "\n") and everything after them, so a match can be
// replaced without losing the rest of the file.
func leadingBlock(content string, minLines int) (block, rest string) {
	lines := strings.Split(content, "\n")
	count, cut := 0, 0
	for cut = 0; cut < len(lines) && count < minLines; cut++ {
		if strings.TrimSpace(lines[cut]) != "" {
			count++
		}
	}
	if count < minLines {
		return "", content // file is too short to have a meaningful leading block
	}
	return strings.Join(lines[:cut], "\n"), strings.Join(lines[cut:], "\n")
}
