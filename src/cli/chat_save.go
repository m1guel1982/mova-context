// chat_save.go — `/memory`, `/budget`, and `/save` command
// implementations for `mova chat` (see chat_cmd.go). Split into its own
// file so no single file in cli/ grows past 300 lines.
package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"mova.local/budget"
	"mova.local/core"
	"mova.local/documents"
	"mova.local/models"
)

// runChatMemory saves the last exchange intelligently into memory.md.
// Extracts ```memory``` blocks if present, or builds a compact summary.
func runChatMemory(adapter core.Adapter, project string, sess *models.Session) {
	if project == "" || adapter == nil {
		consolePrint("/memory requires starting the chat with a project: mova chat <project> [task]\n")
		return
	}

	user, assistant, ok := sess.LastExchange()
	if !ok {
		consolePrint("There is no exchange to save yet.\n")
		return
	}

	var contentToSave string
	isStructuredBlock := false

	// 1. Intentar extraer ÚNICAMENTE el bloque ```memory```
	block, err := core.ExtractMemoryBlock(assistant)
	if err == nil && block != "" {
		contentToSave = block
		isStructuredBlock = true
	} else {
		// 2. Fallback: Compactar la respuesta si viene en texto plano/extenso
		contentToSave = summarizeContent(user, assistant)
	}

	// 3. Persistir en memory.md
	if err := adapter.AppendMemory(project, contentToSave); err != nil {
		consolePrint("Error saving memory: " + err.Error() + "\n")
		return
	}

	if isStructuredBlock {
		consolePrint("[Memory] Saved extracted ```memory``` block to memory.md (" + project + ")\n")
	} else {
		consolePrint("[Memory] Saved compact summary to memory.md (" + project + ")\n")
	}
}

// summarizeContent extracts the core task and main lines of the assistant response.
func summarizeContent(user, assistant string) string {
	// 1. Sanitizar y acortar la tarea del usuario
	userTask := strings.TrimSpace(user)
	if idx := strings.Index(userTask, "\n"); idx != -1 {
		userTask = userTask[:idx]
	}
	if len(userTask) > 80 {
		userTask = userTask[:80] + "..."
	}

	// 2. Extraer las primeras 3 líneas útiles de la respuesta
	lines := strings.Split(assistant, "\n")
	var keyLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Ignorar líneas vacías, tablas, separadores y headers pesados
		if trimmed == "" || strings.HasPrefix(trimmed, "|") || strings.HasPrefix(trimmed, "---") || strings.HasPrefix(trimmed, "##") {
			continue
		}

		keyLines = append(keyLines, trimmed)
		if len(keyLines) >= 3 {
			break
		}
	}

	compactAssistant := strings.Join(keyLines, "\n")
	if compactAssistant == "" {
		compactAssistant = "Review completed."
	}

	return fmt.Sprintf("**task:** %s\n**summary:**\n%s\n", userTask, compactAssistant)
}

// runChatBudget generates budget report.
func runChatBudget(root string, adapter core.Adapter, project, task string) {
	if project == "" || adapter == nil {
		consolePrint("/budget requires starting the chat with a project: mova chat <project> [task]\n")
		return
	}
	consolePrint("[Budget] Estimating context size...\n")
	report, err := budget.BuildReport(adapter, root, project, task, false)
	if err != nil {
		consolePrint("Error: " + err.Error() + "\n")
		return
	}
	consolePrint(fmt.Sprintf("[Budget] Estimated context size: %d tokens.\n", report.TotalTokens))
	if len(report.TotalCosts) > 0 {
		consolePrint(fmt.Sprintf("[Budget] Estimated cost: %s %.4f (%s %s).\n",
			report.Currency, report.TotalCosts[0].USD, report.TotalCosts[0].Provider, report.TotalCosts[0].Model))
	}
	proj, err := adapter.GetProject(project)
	if err != nil {
		consolePrint("Error: " + err.Error() + "\n")
		return
	}
	path, err := budget.WriteReport(root, project, proj, report)
	if err != nil {
		consolePrint("Error: " + err.Error() + "\n")
		return
	}
	consolePrint("[Report] Budget report generated successfully: " + path + "\n")
}

// runChatSave implements `/save`.
func runChatSave(adapter core.Adapter, root string, proj *core.Project, sess *models.Session, rest string, state *chatFileState) {
	if proj == nil || adapter == nil {
		consolePrint("/save requires starting the chat with a project: mova chat <project> [task]\n")
		return
	}
	flags := parseSaveArgs(rest)
	if flags.path == "" {
		consolePrint("Usage:\n" +
			"  /save \"docs/readme.md\"                saves the LAST model response there\n" +
			"  /save -c \"src/app.js\"               saves ONLY source code blocks from the selected response(s)\n" +
			"  /save -text \"notes.md\"               saves ONLY prose, code blocks excluded\n" +
			"  /save -all \"transcript.md\"           saves the FULL conversation so far\n" +
			"  /save -range 2-4 \"excerpt.md\"        saves exchanges 2 through 4 (1-indexed)\n" +
			"  /save -d \"docs/backend\"             creates only the directory\n" +
			"  /save -append \"notes.md\"           appends the selected content to the end of an existing file\n" +
			"  /save -overwrite \"report.pdf\"      forces overwrite if the file already exists\n" +
			"  /save -no-overwrite \"report.pdf\"   fails instead of overwriting if the file already exists\n")
		return
	}

	req := documents.SaveRequest{Repo: proj.Repo, Append: flags.appendMode}
	if flags.overwriteSet {
		req.Overwrite = flags.overwriteValue
		req.OverwriteExplicit = true
	}
	if flags.isDir {
		req.Directory = flags.path
	} else {
		req.Path = flags.path
		content, cErr := documents.SelectContent(chatTurns(sess.History), flags.mode(), flags.rangeStart, flags.rangeEnd, flags.onlyCode, flags.textOnly)
		if cErr != nil {
			consolePrint("[Save] Error: " + cErr.Error() + "\n")
			return
		}
		req.Content = content
	}

	result, err := documents.Save(root, req)
	if err != nil {
		consolePrint("Error: " + err.Error() + "\n")
		return
	}
	consolePrint("[Save] " + result.Message + "\n")
	if !flags.isDir {
		state.lastFile = result.Path
	}
}

// chatTurns adapts Session.History (models.ChatMessage) into
// []documents.ChatTurn — the transport-agnostic shape SelectContent
// works on, so documents (which models never imports) has no dependency
// on the CLI's session type. MCP/HTTP build the same []documents.ChatTurn
// directly from a "history" argument in the request body — see
// mcp/documents_tool.go's "save" case.
func chatTurns(history []models.ChatMessage) []documents.ChatTurn {
	turns := make([]documents.ChatTurn, len(history))
	for i, m := range history {
		turns[i] = documents.ChatTurn{Role: m.Role, Content: m.Content}
	}
	return turns
}

// saveFlags is what parseSaveArgs extracts from the text after `/save`.
type saveFlags struct {
	isDir          bool
	onlyCode       bool
	textOnly       bool // -text: only prose, code blocks stripped out (complement of onlyCode)
	appendMode     bool
	overwriteSet   bool // true if -overwrite or -no-overwrite was given at all
	overwriteValue bool // only meaningful when overwriteSet is true
	all            bool // -all: the full conversation, not just the last response
	rangeSet       bool // -range N-M: a range of exchanges (1-indexed, inclusive)
	rangeStart     int
	rangeEnd       int
	path           string
}

// mode maps saveFlags' -all/-range booleans onto documents.SelectionMode.
func (f saveFlags) mode() documents.SelectionMode {
	switch {
	case f.all:
		return documents.ModeAll
	case f.rangeSet:
		return documents.ModeRange
	default:
		return documents.ModeCurrent
	}
}

// parseSaveArgs parses /save's flags (-d, -c, -text, -all, -range,
// -append, -overwrite, -no-overwrite).
func parseSaveArgs(rest string) saveFlags {
	var f saveFlags
	rest = strings.TrimSpace(rest)
loop:
	for {
		switch {
		case rest == "-d" || strings.HasPrefix(rest, "-d ") || strings.HasPrefix(rest, "-d\t"):
			f.isDir = true
			rest = strings.TrimSpace(strings.TrimPrefix(rest, "-d"))
		case rest == "-c" || strings.HasPrefix(rest, "-c ") || strings.HasPrefix(rest, "-c\t"):
			f.onlyCode = true
			rest = strings.TrimSpace(strings.TrimPrefix(rest, "-c"))
		case rest == "-text" || strings.HasPrefix(rest, "-text ") || strings.HasPrefix(rest, "-text\t"):
			f.textOnly = true
			rest = strings.TrimSpace(strings.TrimPrefix(rest, "-text"))
		case rest == "-all" || strings.HasPrefix(rest, "-all ") || strings.HasPrefix(rest, "-all\t"):
			f.all = true
			rest = strings.TrimSpace(strings.TrimPrefix(rest, "-all"))
		case rest == "-range" || strings.HasPrefix(rest, "-range ") || strings.HasPrefix(rest, "-range\t"):
			remainder := strings.TrimSpace(strings.TrimPrefix(rest, "-range"))
			token, after := splitFirstToken(remainder)
			f.rangeSet = true
			f.rangeStart, f.rangeEnd = documents.ParseRangeToken(token)
			rest = after
		case rest == "-append" || strings.HasPrefix(rest, "-append ") || strings.HasPrefix(rest, "-append\t"):
			f.appendMode = true
			rest = strings.TrimSpace(strings.TrimPrefix(rest, "-append"))
		case rest == "-no-overwrite" || strings.HasPrefix(rest, "-no-overwrite ") || strings.HasPrefix(rest, "-no-overwrite\t"):
			f.overwriteSet, f.overwriteValue = true, false
			rest = strings.TrimSpace(strings.TrimPrefix(rest, "-no-overwrite"))
		case rest == "-overwrite" || strings.HasPrefix(rest, "-overwrite ") || strings.HasPrefix(rest, "-overwrite\t"):
			f.overwriteSet, f.overwriteValue = true, true
			rest = strings.TrimSpace(strings.TrimPrefix(rest, "-overwrite"))
		default:
			break loop
		}
	}
	f.path = strings.Trim(rest, `"'`)
	return f
}

// splitFirstToken splits s into its first whitespace-delimited token and
// everything after it — used by -range to take exactly "N-M" (or "N")
// and leave the rest (the path) for the normal parsing loop above.
func splitFirstToken(s string) (token, rest string) {
	s = strings.TrimSpace(s)
	idx := strings.IndexAny(s, " \t")
	if idx == -1 {
		return s, ""
	}
	return s[:idx], strings.TrimSpace(s[idx+1:])
}

// renderMarkdown uses glamour to render syntax-highlighted output in the
// terminal — code fences without an explicit language tag are first
// auto-tagged by documents.AutoTagCodeFences (see highlight.go), the same
// shared language-detection MCP/HTTP responses use, so highlighting
// behaves identically regardless of which door produced the text.
func renderMarkdown(text string) string {
	text = documents.AutoTagCodeFences(text)
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return text
	}
	out, err := r.Render(text)
	if err != nil {
		return text
	}
	return strings.TrimSpace(out)
}