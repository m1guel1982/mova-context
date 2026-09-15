// patch_apply.go — parsing support for Mova Chat's "Auto-Apply" flow
// (see mova.local/patcher): detecting an affirmative confirmation
// after a "Direct Application" proposal, and extracting the labeled
// code blocks (```lang:path``` or ```lang:path::symbol()```) a
// proposal response uses to say which file (and, optionally, which
// function) each block belongs to. Lives here, not in cli/ or mcp/,
// for the exact same reason DetectSaveIntent does (see nl_save.go's
// header): CLI and MCP must detect identically, so detection has
// exactly one implementation both doors call.
package documents

import (
	"regexp"
	"strings"
)

// LabeledCodeBlock is one fenced code block whose fence tag named an
// explicit target: ```javascript:server.js``` (whole-file) or
// ```javascript:server.js::validateToken()``` (function-level - see
// mova.local/patcher for how Symbol is used).
type LabeledCodeBlock struct {
	Lang    string
	Path    string
	Symbol  string // "" for a whole-file block
	Content string
}

// confirmationRe implements the exact pattern from the spec: sí/si/
// yes/s/y, or one or more comma-separated #N references - the ENTIRE
// line must match (anchored), so a message that merely CONTAINS "si"
// as a word inside a longer sentence is never treated as a
// confirmation.
var confirmationRe = regexp.MustCompile(`(?i)^\s*(sí|si|yes|s|y|(?:#\d+(?:\s*,\s*#\d+)*))\s*$`)
var idRe = regexp.MustCompile(`#(\d+)`)

// DetectApplyConfirmation reports whether line is a bare confirmation
// ("sí", "si", "yes", "s", "y") or a set of "#N" references, and, for
// the latter, which IDs were named (1-indexed references into the
// previous proposal's findings table - interpretation is the caller's
// job, this only parses the text).
func DetectApplyConfirmation(line string) (ok bool, ids []string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || !confirmationRe.MatchString(trimmed) {
		return false, nil
	}
	for _, m := range idRe.FindAllStringSubmatch(trimmed, -1) {
		ids = append(ids, m[1])
	}
	return true, ids
}

// directApplyMarkers are phrases a proposal response uses to signal
// "these are ready-to-apply changes, not just discussion" - matched
// case-insensitively, in either language since Mova Chat itself may
// answer in the person's language even though context-trace's own
// output stays English-only (see COMMANDS.md's i18n note - that rule
// is specific to context-trace, not chat in general).
var directApplyMarkers = []string{"aplicación directa", "direct application", "aplicacion directa"}

// LooksLikeDirectApplyProposal reports whether assistantText is the
// kind of response DetectApplyConfirmation's "yes" should actually
// act on: it either says so explicitly, or it already contains at
// least one labeled code block (a proposal that names its own target
// file is unambiguous regardless of wording).
func LooksLikeDirectApplyProposal(assistantText string) bool {
	lower := strings.ToLower(assistantText)
	for _, marker := range directApplyMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return len(ExtractLabeledCodeBlocks(assistantText)) > 0
}

// fenceTagRe recognizes a fenced block's opening tag with an explicit
// target: ```<lang>:<path>``` or ```<lang>:<path>::<symbol>()```. The
// language segment may be empty (":server.js" alone is also valid).
var fenceTagRe = regexp.MustCompile(`^([A-Za-z0-9_+-]*):([^\s:]+(?:/[^\s:]+)*)(?:::([A-Za-z_][A-Za-z0-9_]*)\(\))?$`)

// ExtractLabeledCodeBlocks scans text for every fenced code block
// whose tag matches fenceTagRe, in order - blocks without a
// recognized "lang:path" tag are skipped here (see
// ExtractCodeBlocks for the untagged, single-file /save behavior this
// deliberately does not change).
func ExtractLabeledCodeBlocks(text string) []LabeledCodeBlock {
	var blocks []LabeledCodeBlock
	lines := strings.Split(text, "\n")
	inBlock := false
	var current LabeledCodeBlock
	var body strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if inBlock {
				current.Content = strings.TrimRight(body.String(), "\n")
				if strings.TrimSpace(current.Content) != "" {
					blocks = append(blocks, current)
				}
				body.Reset()
				inBlock = false
				continue
			}
			tag := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			m := fenceTagRe.FindStringSubmatch(tag)
			if m == nil {
				continue // untagged or malformed - not a labeled block
			}
			inBlock = true
			current = LabeledCodeBlock{Lang: m[1], Path: m[2], Symbol: m[3]}
			continue
		}
		if inBlock {
			body.WriteString(line + "\n")
		}
	}
	return blocks
}
