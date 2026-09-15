// apply_file_changes.go — the interactive confirmation menu for the
// apply_file_changes tool (see mcp/apply_file_changes.go), used by
// `mova chat` (chat_cmd.go) and `mova ui chat` (tui_chat.go) — the two
// doors that actually have a terminal to show it on. Replaces having
// to type a manual command ("/save -c", etc.) for every file: the
// model proposes every create/modify/delete in ONE apply_file_changes
// call, and this shows ONE menu for all of them — "apply to all",
// "apply to specific files only" (by index, e.g. "1,3"), or "cancel"
// — using the exact i18n keys under tools.file_changes in
// config/lang/{es,en}.json (falls back to English, same as every
// other message in Mova — see i18n.T).
package main

import (
	"fmt"
	"strconv"
	"strings"

	"mova.local/core"
	"mova.local/i18n"
	"mova.local/mcp"
)

// readLineFunc reads one line of input from whatever the caller's
// terminal actually is — chat_cmd.go's REPL passes its existing
// bufio.Scanner-backed reader (see sendWithTools' new parameter); a
// nil readLineFunc (a door with no synchronous terminal to block on,
// e.g. today's mova ui chat event loop) means "the menu cannot be
// shown here", handled the same safe way MCP/HTTP are (see
// mcp.DescribePendingChanges) rather than guessing an answer.
type readLineFunc func() (string, bool)

// confirmAndApplyFileChanges shows the menu, applies whatever the
// person selected (reusing mcp.ApplyFileChange — the SAME "save"/
// "delete_path" code /save and the delete_path tool already run), and
// returns the TOOL_RESULT text to feed back to the model. emit prints
// to the terminal exactly like the rest of chat_helpers.go's emit
// convention (nil means print directly).
func confirmAndApplyFileChanges(adapter core.Adapter, root string, changes []mcp.FileChange, readLine readLineFunc, emit func(string)) string {
	if emit == nil {
		emit = consolePrint
	}
	if readLine == nil {
		// No synchronous terminal available on this call path (see
		// this file's header) — same safe, non-auto-applying summary
		// MCP/HTTP use, never a silent guess.
		return mcp.DescribePendingChanges(changes)
	}

	emit(i18n.T("tools.file_changes.prompt_confirm_header") + "\n")
	for i, c := range changes {
		emit(fmt.Sprintf("  %d. [%s] %s\n", i+1, mcp.ActionLabel(c.Action), c.Path))
	}
	emit(i18n.T("tools.file_changes.options_menu") + "\n")
	emit(i18n.T("tools.file_changes.prompt_select"))

	line, ok := readLine()
	if !ok {
		return i18n.T("tools.file_changes.status_cancelled")
	}
	selected, cancelled := parseChangeSelection(strings.TrimSpace(line), len(changes))
	if cancelled {
		emit(i18n.T("tools.file_changes.status_cancelled") + "\n")
		return i18n.T("tools.file_changes.status_cancelled")
	}

	var results []string
	for i, c := range changes {
		if !selected[i+1] {
			continue
		}
		if _, err := mcp.ApplyFileChange(adapter, root, c); err != nil {
			line := i18n.T("tools.file_changes.status_error", map[string]any{"path": c.Path, "error": err.Error()})
			emit(line + "\n")
			results = append(results, line)
			continue
		}
		line := i18n.T("tools.file_changes.status_applied", map[string]any{"path": c.Path})
		emit(line + "\n")
		results = append(results, line)
	}
	return strings.Join(results, "\n")
}

// parseChangeSelection interprets the menu answer: "a"/"all" (every
// change), "c"/"cancel" (none, cancelled=true), or a comma-separated
// list of 1-indexed positions ("1", "1,3"). Anything unrecognized is
// treated as "cancel" — an ambiguous answer must never accidentally
// write a file.
func parseChangeSelection(answer string, total int) (selected map[int]bool, cancelled bool) {
	selected = map[int]bool{}
	lower := strings.ToLower(strings.TrimSpace(answer))
	switch lower {
	case "a", "all", "todos", "todo":
		for i := 1; i <= total; i++ {
			selected[i] = true
		}
		return selected, false
	case "", "c", "cancel", "cancelar", "n", "no":
		return selected, true
	}
	found := false
	for _, part := range strings.Split(lower, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(part)); err == nil && n >= 1 && n <= total {
			selected[n] = true
			found = true
		}
	}
	if !found {
		return map[int]bool{}, true
	}
	return selected, false
}
