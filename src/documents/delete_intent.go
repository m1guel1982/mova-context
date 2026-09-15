// delete_intent.go — detects "delete this file/directory" intent
// written in plain natural language, split out from DetectEditIntent
// (edit_intent.go) to fix a real bug: "elimina"/"borra"/"delete"/
// "remove" used to ALSO match editVerbRe, so "elimina el archivo
// test.cpp" went through the EDIT flow (ask the model for a diff) and
// ended up just emptying the file's content instead of actually
// removing it. Delete intent is now checked FIRST (see
// cli/chat_cmd.go's dispatch order) and routes to the real delete
// confirmation flow (documents.Delete) — the same one `/delete`
// already uses.
//
// A second, real bug found in QA: "elimina el CONTENIDO del archivo X"
// / "elimina el TEXTO que existe en X" were ALSO going through full
// file deletion — the person wanted the file emptied, not removed.
// DetectDeleteIntent now tells the two apart (ClearContentOnly) so the
// caller can route to a content-clear instead of documents.Delete.
package documents

import (
	"regexp"
	"strings"
)

var deleteVerbRe = regexp.MustCompile(`(?i)\b(` +
	`elimina|eliminar|elim[ií]nalo|eliminalo|` +
	`borra|borrar|b[oó]rralo|borralo|` +
	`quita|quitar|qu[ií]talo|quitalo|` +
	`remueve|remover|rem[uú]evelo|remuevelo|` +
	`vac[ií]a|vaciar|` +
	`delete|remove|erase|clear|empty` +
	`)\b`)

// contentOnlyRe: "contenido"/"texto" (or English "content"/"text") in
// the SAME clause as a delete verb means "empty this file", not
// "remove this file" — see this file's header.
var contentOnlyRe = regexp.MustCompile(`(?i)\b(contenido|texto|content|text)\b`)

// deleteKeywordRe requires "el archivo"/"la carpeta"/"the file"/"the
// folder"/"el directorio"/"the directory" in the SAME clause — a bare
// "elimina eso" with no resolvable target isn't enough to safely
// delete anything. Only used to gate whether a delete/clear even
// applies here; the actual TARGET always comes from extractFileTarget
// (nl_intent.go), never from "the word right after this keyword" (a
// real bug: that used to capture connector words like "que" out of
// "el archivo QUE se encuentra en C:\...").
var deleteKeywordRe = regexp.MustCompile(`(?i)\b(?:el\s+archivo|la\s+carpeta|el\s+directorio|the\s+file|the\s+folder|the\s+directory)\b`)

// DeleteIntent is what DetectDeleteIntent found in a chat message.
type DeleteIntent struct {
	VerbDetected     bool
	ClearContentOnly bool     // "elimina el CONTENIDO de X" — empty the file, don't remove it
	Targets          []string // resolved path-shaped tokens, may be empty (caller falls back to chatFileState.lastFile)
}

// DetectDeleteIntent scans a chat message for delete intent, clause by
// clause (same " y "/" and " splitting as the other NL detectors).
func DetectDeleteIntent(text string) DeleteIntent {
	var out DeleteIntent
	for _, clause := range splitClauses(text) {
		if !deleteVerbRe.MatchString(clause) {
			continue
		}
		if !deleteKeywordRe.MatchString(clause) && !extensionOrDirLike(clause) {
			continue
		}
		out.VerbDetected = true
		if contentOnlyRe.MatchString(clause) {
			out.ClearContentOnly = true
		}
		if target, ok := extractFileTarget(clause); ok {
			out.Targets = append(out.Targets, target)
			continue
		}
		if m := dirKeywordRe.FindStringSubmatch(clause); m != nil {
			out.Targets = append(out.Targets, strings.Trim(m[1], `"'`))
		}
	}
	return out
}

// extensionOrDirLike is a cheap check for "does this clause look like
// it names a file/path at all" — used only to avoid a false "delete"
// match on something like "elimina esto" with zero resolvable target
// AND no explicit file/directory keyword either.
func extensionOrDirLike(clause string) bool {
	_, ok := extractFileTarget(clause)
	return ok
}
