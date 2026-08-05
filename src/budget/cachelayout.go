// cachelayout.go — the Token Firewall's second stage: lays out the
// system prompt so its FIRST bytes are a stable, byte-identical prefix
// across runs (agents + skills + prompt — curated project files that
// don't change between one task run and the next), with everything
// that changes every time (the "Generated: <timestamp>" header, focus,
// memory) moved AFTER it. This is what lets a Cloud provider's prompt
// caching actually trigger — Anthropic/OpenAI/Gemini all cache based on
// an exact-match prefix, and a single differing byte at the start (like
// a timestamp) defeats it on every single call.
//
// Only used for messages actually sent to a model (see
// cli/chat_helpers.go, cli/tui_chat.go) — `mova run`'s printed output
// and mova-budget-report.md keep core.ContextSections.Full()'s original,
// more readable Header-first order untouched; reordering only matters
// once there's an actual API call whose cache hinges on it.
package budget

import (
	"crypto/sha256"
	"encoding/hex"

	"mova.local/core"
)

// CacheLayout is what LayoutForCache returns.
type CacheLayout struct {
	Text           string // static prefix + dynamic tail, in that order — same bytes as Full(), reordered
	StaticBoundary int    // byte offset in Text where the static prefix ends
	StaticTokens   int
	Hash           string // sha256 of the static prefix — compare across runs to know if the prefix actually stayed stable
}

// LayoutForCache builds the cache-aware ordering. tokenModelHint is
// passed straight to CountTokens (see tokencount.go) for the static
// prefix's token count, shown in the report so a project author can see
// whether it clears a provider's minimum cacheable size (Anthropic:
// 1024 tokens for Claude 3+ models, less for Haiku — see Anthropic's
// docs; this package doesn't hardcode that minimum since it varies by
// model and provider and changes over time).
func LayoutForCache(sections *core.ContextSections, tokenModelHint string) CacheLayout {
	static := sections.Agents + sections.Skills + sections.Prompt
	dynamic := sections.Header + sections.Focus + sections.Memory + sections.Instruction

	tokens, _, _ := CountTokens(static, tokenModelHint)
	sum := sha256.Sum256([]byte(static))

	return CacheLayout{
		Text:           static + dynamic,
		StaticBoundary: len(static),
		StaticTokens:   tokens,
		Hash:           hex.EncodeToString(sum[:])[:16], // 16 hex chars is plenty to notice a change; the report never needs a full 64-char hash
	}
}

// StablePrefix reports whether two LayoutForCache Hash values match —
// a helper for the report/CLI to say "same as last run" or "changed
// since last run (cache will miss)" without either caller reimplementing
// the comparison.
func StablePrefix(previousHash, currentHash string) bool {
	return previousHash != "" && previousHash == currentHash
}
