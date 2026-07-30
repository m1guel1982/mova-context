// save_selection.go — the SINGLE implementation behind "which text does
// /save actually save": the last response (default), a 1-indexed range
// of exchanges, or the full conversation (see "2. Guardado completo del
// chat" in the spec this implements), each optionally narrowed to only
// code blocks or only prose. Chat (cli/chat_save.go), the "save" MCP
// tool, and POST /save all call SelectContent instead of keeping their
// own copy — Chat converts its Session.History into []ChatTurn first
// (see cli/chat_save.go), MCP/HTTP pass "history" in the request body
// the same shape a client already holds client-side.
package documents

import (
	"fmt"
	"strconv"
	"strings"
)

// ChatTurn is the minimal, transport-agnostic shape of one conversation
// message — deliberately NOT models.ChatMessage, so this package (which
// core/models never import) has no dependency on the CLI's session type.
type ChatTurn struct {
	Role    string // "user" | "assistant"
	Content string
}

// Exchange is one user/assistant pair.
type Exchange struct {
	User      string
	Assistant string
}

// GroupExchanges pairs up consecutive user→assistant turns, in order —
// "/save -range 2-4" refers to the 2nd, 3rd, and 4th pair, 1-indexed.
func GroupExchanges(turns []ChatTurn) []Exchange {
	var out []Exchange
	for i := 0; i+1 < len(turns); i++ {
		if turns[i].Role == "user" && turns[i+1].Role == "assistant" {
			out = append(out, Exchange{User: turns[i].Content, Assistant: turns[i+1].Content})
			i++
		}
	}
	return out
}

// SelectionMode is /save's scope: "" or "current" (the default: just the
// last response, unchanged historical behavior), "all" (the full
// conversation), or "range" (RangeStart-RangeEnd, 1-indexed inclusive).
type SelectionMode string

const (
	ModeCurrent SelectionMode = ""
	ModeAll     SelectionMode = "all"
	ModeRange   SelectionMode = "range"
)

// SelectContent picks the text /save works on: the default (last
// exchange's assistant response, exactly as before), the full
// conversation transcript, or a range of exchanges. onlyCode/textOnly
// are then applied on top of that, in that order of precedence, so
// every combination (e.g. mode=all + onlyCode) behaves consistently.
func SelectContent(turns []ChatTurn, mode SelectionMode, rangeStart, rangeEnd int, onlyCode, textOnly bool) (string, error) {
	var content string

	switch mode {
	case ModeAll:
		exchanges := GroupExchanges(turns)
		if len(exchanges) == 0 {
			return "", fmt.Errorf("there is no conversation to save yet")
		}
		content = TranscriptText(exchanges)

	case ModeRange:
		exchanges := GroupExchanges(turns)
		start, end := rangeStart, rangeEnd
		if start < 1 {
			start = 1
		}
		if end > len(exchanges) {
			end = len(exchanges)
		}
		if len(exchanges) == 0 || start > end {
			return "", fmt.Errorf("no exchanges found in range %d-%d (this conversation has %d so far)",
				rangeStart, rangeEnd, len(exchanges))
		}
		content = TranscriptText(exchanges[start-1 : end])

	default:
		last := lastExchange(turns)
		if last == nil {
			return "", fmt.Errorf("there is no model response to save yet")
		}
		content = last.Assistant
	}

	switch {
	case onlyCode:
		blocks := ExtractCodeBlocks(content)
		if len(blocks) == 0 {
			return "", fmt.Errorf("no code blocks (```) found to extract")
		}
		return strings.Join(blocks, "\n\n"), nil
	case textOnly:
		return StripCodeBlocks(content), nil
	default:
		return content, nil
	}
}

func lastExchange(turns []ChatTurn) *Exchange {
	exchanges := GroupExchanges(turns)
	if len(exchanges) == 0 {
		return nil
	}
	return &exchanges[len(exchanges)-1]
}

// TranscriptText renders a slice of exchanges as a readable transcript —
// used by mode=all and mode=range (mode=current keeps returning just the
// raw assistant text, unchanged, for backward compatibility).
func TranscriptText(exchanges []Exchange) string {
	var b strings.Builder
	for i, ex := range exchanges {
		b.WriteString(fmt.Sprintf("### You\n%s\n\n### Assistant\n%s\n", ex.User, ex.Assistant))
		if i < len(exchanges)-1 {
			b.WriteString("\n---\n\n")
		}
	}
	return b.String()
}

// ExtractCodeBlocks returns strictly the contents of every fenced ```
// code block in text, in order — independent of language ("go", "python",
// "yaml", or no tag at all), used by `/save -c` ("únicamente código").
func ExtractCodeBlocks(text string) []string {
	var blocks []string
	lines := strings.Split(text, "\n")
	inBlock := false
	var cur strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if inBlock {
				blocks = append(blocks, strings.TrimRight(cur.String(), "\n"))
				cur.Reset()
				inBlock = false
			} else {
				inBlock = true
			}
			continue
		}
		if inBlock {
			cur.WriteString(line + "\n")
		}
	}
	return blocks
}

// StripCodeBlocks removes every fenced ``` code block and keeps
// everything else — the complement of ExtractCodeBlocks, used by
// `/save -text` ("únicamente texto").
func StripCodeBlocks(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	inBlock := false
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inBlock = !inBlock
			continue
		}
		if !inBlock {
			out = append(out, line)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// ParseRangeToken parses "N-M" or a single "N" (meaning N-N) into
// 1-indexed start/end exchange numbers — shared by every door's own
// argument parsing (cli's `/save -range`, MCP/HTTP's "range": "N-M").
// Malformed input becomes 1-0, an empty range, so SelectContent reports
// a clear error instead of silently saving something unintended.
func ParseRangeToken(token string) (start, end int) {
	token = strings.TrimSpace(token)
	if a, b, ok := strings.Cut(token, "-"); ok {
		start, _ = strconv.Atoi(strings.TrimSpace(a))
		end, _ = strconv.Atoi(strings.TrimSpace(b))
		return start, end
	}
	n, err := strconv.Atoi(token)
	if err != nil {
		return 1, 0
	}
	return n, n
}
