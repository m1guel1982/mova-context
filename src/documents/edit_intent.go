// edit_intent.go — detects "modify an EXISTING file" intent written in
// plain natural language (Spanish or English): "Modifica la función X",
// "Cambia el texto de report.md", "Fix the login bug in auth.go", "Update
// server.js". This is the counterpart to nl_intent.go's DetectSaveIntent
// (which is about NEW files) — see cli/nl_edit.go and mcp/nl_edit.go for
// how the two are told apart (mainly: does the target already exist on
// disk).
package documents

import (
	"regexp"
	"strings"
)

// editVerbRe recognizes edit/modify verbs in Spanish and English —
// conjugations and near-synonyms, same open-ended philosophy as
// creationVerbRe in nl_intent.go: no cap on phrasing, precision comes from
// requiring a resolvable, EXISTING target file (see cli/nl_edit.go).
var editVerbRe = regexp.MustCompile(`(?i)\b(` +
	`modifica|modificar|modif[ií]cale|modificale|` +
	`edita|editar|ed[ií]tale|editale|` +
	`cambia|cambiar|c[aá]mbiale|cambiale|` +
	`actualiza|actualizar|actual[ií]zale|actualizale|` +
	`corrige|corregir|corr[ií]gele|corrigele|` +
	`arregla|arreglar|arr[eé]glale|arreglale|` +
	`repara|reparar|` +
	`reemplaza|reemplazar|reempl[aá]zale|reemplazale|` +
	`ajusta|ajustar|aj[uú]stale|ajustale|` +
	`revisa|revisar|rev[ií]sale|revisale|` +
	`agrega|agregar|a[ñn]ade|a[ñn]adir|quita|quitar|elimina|eliminar|borra|borrar|` +
	`modify|edit|change|update|fix|correct|repair|replace|adjust|revise|alter|rewrite|refactor` +
	`)\b`)

// EditIntent is what DetectEditIntent found in a chat message.
type EditIntent struct {
	VerbDetected bool     // an edit/modify verb was found somewhere in the message
	Files        []string // explicit path-shaped tokens found alongside that verb; may be empty (caller falls back to "last touched file" context — see cli/nl_edit.go's chatFileState)
}

// DetectEditIntent scans a chat message for modify/edit intent, clause by
// clause (same " y "/" and " splitting as DetectSaveIntent). It does NOT
// check whether the file exists — that's deliberately left to the caller
// (cli/nl_edit.go, mcp/nl_edit.go), which is also what tells this apart
// from "create a new file" intent: an edit only makes sense against
// something that already exists.
func DetectEditIntent(text string) EditIntent {
	var out EditIntent
	for _, clause := range splitClauses(text) {
		if !editVerbRe.MatchString(clause) {
			continue
		}
		out.VerbDetected = true
		for _, m := range pathTokenRe.FindAllString(clause, -1) {
			out.Files = append(out.Files, strings.Trim(m, `"'`))
		}
	}
	return out
}
