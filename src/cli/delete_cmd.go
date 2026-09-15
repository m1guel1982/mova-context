// delete_cmd.go — `/delete` for `mova chat` (see chat_cmd.go). Uses the
// SAME unified delete entry point (documents.Delete, see
// documents/delete_service.go) as MCP's "delete_path" tool and HTTP's
// POST /delete — no separate delete logic for the chat door. This file
// only adds the one thing a non-interactive door can't do: read the
// person's y/n from the terminal, one prompt per item, using the exact
// wording documents.FormatDeletePrompt produces everywhere else.
package main

import (
	"strings"

	"mova.local/core"
	"mova.local/documents"
)

// runChatDelete implements `/delete "a.txt" "b.txt" "logs/"`.
// runChatDelete drives the SAME confirmation flow both `/delete` and
// natural-language delete intent (nl_delete.go) use — one path prompt
// per pending item, read via readLine (a plain *bufio.Scanner wrapper
// on `mova chat`'s REPL, a menuChan-backed channel read on `mova ui
// chat` — see chat_helpers.go's readLineFunc and tui_chat.go's
// nlDeleteCmd) so both doors share one implementation instead of two
// that could quietly drift apart.
func runChatDelete(root string, proj *core.Project, rest string, readLine readLineFunc, emit func(string)) {
	if emit == nil {
		emit = consolePrint
	}
	paths := parseDeletePaths(rest)
	if len(paths) == 0 {
		emit("Usage: /delete \"file.txt\" [\"another.txt\" \"dir/\" ...]\n")
		return
	}
	repo := "."
	if proj != nil && proj.Repo != "" {
		repo = proj.Repo
	}

	pending, err := documents.Delete(root, documents.DeleteRequest{Paths: paths, Repo: repo})
	if err != nil {
		emit("Error: " + err.Error() + "\n")
		return
	}

	var confirmed []string
	for _, item := range pending.Items {
		emit(documents.FormatDeletePrompt([]documents.DeleteItem{item}) + " ")
		text, ok := readLine()
		if !ok {
			return
		}
		answer := strings.ToLower(strings.TrimSpace(text))
		if answer == "y" || answer == "yes" || answer == "s" || answer == "si" || answer == "sí" {
			confirmed = append(confirmed, item.Requested)
		}
	}
	if len(confirmed) == 0 {
		emit("Nothing deleted.\n")
		return
	}

	result, err := documents.Delete(root, documents.DeleteRequest{Paths: confirmed, Repo: repo, Confirm: true})
	if err != nil {
		emit("Error: " + err.Error() + "\n")
		return
	}
	emit(result.Message + "\n")
}

// parseDeletePaths splits /delete's argument text into individual path
// tokens — quoted ("a b.txt") or bare, whitespace-separated, same
// convention as parseSaveArgs uses for /save's path.
func parseDeletePaths(rest string) []string {
	var out []string
	var cur strings.Builder
	inQuotes := false
	flush := func() {
		if t := strings.TrimSpace(cur.String()); t != "" {
			out = append(out, t)
		}
		cur.Reset()
	}
	for _, r := range rest {
		switch {
		case r == '"' || r == '\'':
			inQuotes = !inQuotes
		case (r == ' ' || r == '\t') && !inQuotes:
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}
