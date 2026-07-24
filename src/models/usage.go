// usage.go — token usage display shared by every door into Mova Context
// (CLI `mova chat`, MCP chat_completion, and HTTP, which is a thin
// wrapper over the same MCP tool — see http/server.go). Mirrors the
// minimal usage line tools like Cline show after each request: how many
// tokens THIS request used, and the model's maximum context window, so
// the person chatting can see how close they are to the limit without
// running `mova budget` separately.
//
// This is purely informational (never blocks anything) — the hard gate
// that actually stops execution before a request is sent lives in
// mova.local/budget.EnforceLimit, driven by project.json's "budget"
// block, not by the model's raw context window.
package models

import "fmt"

// UsageInfo is the minimal pair of numbers this feature promises: how
// many tokens the last request used, and the model's maximum context
// window (0 if the active model config does not declare one).
type UsageInfo struct {
	UsedTokens    int
	ContextWindow int // from ModelConfig.ContextWindow; 0 = not declared for this model
}

// FormatLine renders UsageInfo as a single line, e.g.:
//
//	[Tokens] 1,842 used / 131,072 max context window (1.4%)
//
// or, when the model config does not declare context_window:
//
//	[Tokens] 1,842 used / context window not configured for this model
func (u UsageInfo) FormatLine() string {
	used := formatThousandsInt(u.UsedTokens)
	if u.ContextWindow <= 0 {
		return fmt.Sprintf("[Tokens] %s used / context window not configured for this model\n", used)
	}
	percent := (float64(u.UsedTokens) / float64(u.ContextWindow)) * 100
	return fmt.Sprintf("[Tokens] %s used / %s max context window (%.1f%%)\n",
		used, formatThousandsInt(u.ContextWindow), percent)
}

// UsageFor builds UsageInfo for the last turn of sess against mc.
// usedTokens prefers the real usage the provider reported
// (sess.LastUsage.PromptTokens); if the provider does not report usage
// (Usage{} stays zero), it falls back to fallbackEstimate — a local
// tiktoken-go count of the same text, so the line is never just blank.
func UsageFor(sess *Session, mc *ModelConfig, fallbackEstimate int) UsageInfo {
	used := sess.LastUsage.PromptTokens
	if used <= 0 {
		used = fallbackEstimate
	}
	contextWindow := 0
	if mc != nil {
		contextWindow = mc.ContextWindow
	}
	return UsageInfo{UsedTokens: used, ContextWindow: contextWindow}
}

// formatThousandsInt renders an int with thousands separators (14250 ->
// "14,250"). Small, dependency-free, deliberately duplicated from
// budget.formatThousands (unexported there) rather than adding a shared
// package for one loop.
func formatThousandsInt(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	return string(out)
}
