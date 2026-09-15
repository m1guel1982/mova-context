// read_intent.go — detects "show me the content of this file" intent,
// so a chat can answer "muestra el contenido de X"/"show me X" without
// the person needing any command.
package documents

import "regexp"

var readVerbRe = regexp.MustCompile(`(?i)\b(` +
	`muestra|muéstrame|muestrame|ense[ñn]a|ens[eé][ñn]ame|ensename|lee|leer|l[eé]eme|leeme|` +
	`show|display|read|print|cat|dump` +
	`)\b`)

// DetectReadIntent scans a chat message for "show me a file's content"
// intent, clause by clause. The target always comes from
// extractFileTarget (nl_intent.go) — a real bug found in QA had
// "muestra el contenido DEL ARCHIVO C:\...\prueba.py" capturing the
// literal word "archivo" instead of the path, because the old regex
// grabbed whatever word immediately followed "el contenido del".
func DetectReadIntent(text string) (target string, ok bool) {
	for _, clause := range splitClauses(text) {
		if !readVerbRe.MatchString(clause) {
			continue
		}
		if t, found := extractFileTarget(clause); found {
			return t, true
		}
	}
	return "", false
}
