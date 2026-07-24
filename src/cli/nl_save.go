// nl_save.go — natural-language file/directory creation for `mova chat`,
// the same experience Claude Desktop/Claude Console give: say what you
// want created in plain language and it gets created, without typing
// `/save` first. Detection lives in mova.local/documents (DetectSaveIntent
// — shared with the MCP door, see mcp/nl_save.go), so both stay in sync.
//
// Examples: "Genera carpeta/reporte.pdf", "Crea c:/proyecto/docs/manual.md",
// "Crea el directorio c:/temp/test y genera reporte.pdf".
//
// Append/overwrite modifiers ("agrega al final", "sobreescribe",
// "no lo sobreescribas", "append", "overwrite", "don't overwrite" — see
// mova.local/documents.DetectSaveModifiers) work in the same message too,
// the natural-language equivalent of `/save -append`/`-overwrite`/
// `-no-overwrite` (see chat_save.go) — for every format `save` supports.
//
// `/save` keeps working exactly as before — this only covers messages
// where the person never typed it.
package main

import (
	"fmt"

	"mova.local/core"
	"mova.local/documents"
	"mova.local/models"
)

// handleNaturalLanguageSave inspects a chat line for creation intent. It
// returns true when it handled the message itself (so the caller must
// NOT also send it through the ordinary chat turn). Behavior:
//
//   - Directory-only intent ("Crea el directorio X"): created immediately,
//     no model call needed.
//   - File intent ("Genera reporte.pdf"): the message is still sent to
//     the model as an ordinary turn (so it can produce the content), and
//     the reply is then saved to each detected path automatically —
//     the same content `/save` would have saved, just without the
//     explicit command.
//   - Both in one message ("Crea el directorio X y genera Y"): the
//     directory is created first, then the file flow runs as above.
func handleNaturalLanguageSave(adapter core.Adapter, root string, proj *core.Project, sess *models.Session, project, line string, state *chatFileState) bool {
	intent := documents.DetectSaveIntent(line)
	if !intent.HasIntent() {
		return false
	}

	repo := ""
	if proj != nil {
		repo = proj.Repo
	}

	for _, dir := range intent.Directories {
		result, err := documents.Save(root, documents.SaveRequest{Directory: dir, Repo: repo})
		if err != nil {
			consolePrint("[Save] Error: " + err.Error() + "\n")
			continue
		}
		consolePrint("[Save] " + result.Message + "\n")
	}

	if len(intent.Files) == 0 {
		return true
	}

	consolePrint(fmt.Sprintf("[NL] Detected file creation intent — generating content for %d file(s)...\n", len(intent.Files)))
	runChatTurn(sess, adapter, proj, root, project, line)

	_, reply, ok := sess.LastExchange()
	if !ok {
		return true
	}

	// Detectar modificadores de lenguaje natural (append, overwrite, no-overwrite)
	modifiers := documents.DetectSaveModifiers(line)
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
			consolePrint("[Save] Error: " + err.Error() + "\n")
			continue
		}
		consolePrint("[Save] " + result.Message + "\n")
		state.lastFile = result.Path
	}
	return true
}