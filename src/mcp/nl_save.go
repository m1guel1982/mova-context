// nl_save.go — natural-language file/directory creation for the MCP/HTTP
// "chat_completion" tool, mirroring cli/nl_save.go so a plain message
// like "Genera reporte.pdf" creates a file the same way whether it comes
// from `mova chat`, an MCP client, or a raw HTTP POST — no special
// command needed on any door. Detection itself lives in
// mova.local/documents.DetectSaveIntent, shared by both. Append/overwrite
// modifiers ("agrega al final", "sobreescribe", "no lo sobreescribas" —
// mova.local/documents.DetectSaveModifiers) work here too.
package mcp

import (
	"fmt"
	"strings"

	"mova.local/core"
	"mova.local/documents"
)

// applyNaturalLanguageDirectories creates any directory-creation intent
// found in message BEFORE the message is sent to the model — directories
// never depend on the model's reply, so there's no reason to wait.
func applyNaturalLanguageDirectories(statusLog *strings.Builder, root string, proj *core.Project, message string) documents.SaveIntent {
	intent := documents.DetectSaveIntent(message)
	if len(intent.Directories) == 0 {
		return intent
	}
	repo := ""
	if proj != nil {
		repo = proj.Repo
	}
	for _, dir := range intent.Directories {
		result, err := documents.Save(root, documents.SaveRequest{Directory: dir, Repo: repo})
		if err != nil {
			statusLog.WriteString("[Save] Error: " + err.Error() + "\n")
			continue
		}
		statusLog.WriteString("[Save] " + result.Message + "\n")
	}
	return intent
}

// applyNaturalLanguageFiles saves reply into every file path detected in
// intent (see applyNaturalLanguageDirectories) — called once the model's
// reply is available, since file content comes from that reply. message
// is the original chat message, re-scanned here for append/overwrite
// modifiers (DetectSaveModifiers) — a separate concern from DetectSaveIntent,
// which only decided the path(s) themselves.
func applyNaturalLanguageFiles(statusLog *strings.Builder, root string, proj *core.Project, intent documents.SaveIntent, message, reply string) {
	if len(intent.Files) == 0 {
		return
	}
	repo := ""
	if proj != nil {
		repo = proj.Repo
	}
	modifiers := documents.DetectSaveModifiers(message)
	statusLog.WriteString(fmt.Sprintf("[NL] Detected file creation intent — saving %d file(s) from the reply...\n", len(intent.Files)))
	for _, path := range intent.Files {
		req := documents.SaveRequest{
			Path:    path,
			Content: reply,
			Repo:    repo,
			Append:  modifiers.Append,
		}
		if modifiers.OverwriteSet {
			req.Overwrite = modifiers.OverwriteValue
			req.OverwriteExplicit = true
		}
		result, err := documents.Save(root, req)
		if err != nil {
			statusLog.WriteString("[Save] Error: " + err.Error() + "\n")
			continue
		}
		statusLog.WriteString("[Save] " + result.Message + "\n")
	}
}