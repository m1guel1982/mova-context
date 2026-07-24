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
	prices, err := budget.LoadPrices(root)
	if err != nil {
		consolePrint("Error: " + err.Error() + "\n")
		return
	}
	path, err := budget.WriteReport(root, prices, report)
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
			"  /save -c \"src/app.js\"               saves ONLY source code blocks from the LAST response\n" +
			"  /save -d \"docs/backend\"             creates only the directory\n" +
			"  /save -append \"notes.md\"           appends the LAST response to the end of an existing file\n" +
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
		_, assistant, ok := sess.LastExchange()
		if !ok {
			consolePrint("There is no model response to save yet.\n")
			return
		}

		if flags.onlyCode {
			codeBlocks := extractCodeBlocks(assistant)
			if len(codeBlocks) == 0 {
				consolePrint("[Save] Error: No code blocks (```) found in the last response to extract.\n")
				return
			}
			req.Content = strings.Join(codeBlocks, "\n\n")
		} else {
			req.Content = assistant
		}
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

// saveFlags is what parseSaveArgs extracts from the text after `/save`.
type saveFlags struct {
	isDir          bool
	onlyCode       bool
	appendMode     bool
	overwriteSet   bool // true if -overwrite or -no-overwrite was given at all
	overwriteValue bool // only meaningful when overwriteSet is true
	path           string
}

// parseSaveArgs parses /save's flags (-d, -c, -append, -overwrite, -no-overwrite).
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

// extractCodeBlocks filters out prose/markdown text and returns strictly code contents inside ``` blocks.
func extractCodeBlocks(text string) []string {
	var blocks []string
	lines := strings.Split(text, "\n")
	inBlock := false
	var currentBlock strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if inBlock {
				blocks = append(blocks, strings.TrimRight(currentBlock.String(), "\n"))
				currentBlock.Reset()
				inBlock = false
			} else {
				inBlock = true
			}
			continue
		}
		if inBlock {
			currentBlock.WriteString(line + "\n")
		}
	}
	return blocks
}

// renderMarkdown uses glamour to render syntax-highlighted output in the terminal.
func renderMarkdown(text string) string {
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