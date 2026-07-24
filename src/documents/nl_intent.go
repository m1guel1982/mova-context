// nl_intent.go — detects "create a file/directory" intent written in
// plain natural language (Spanish or English), so `/save` is no longer
// the ONLY way to get a file or folder out of a chat — the same
// experience Claude Desktop/Claude Console give: say what you want
// created, and it gets created, without a special command.
//
// Examples this recognizes (see nl_intent_test.go for the full list):
//
//	Genera carpeta/reporte.pdf
//	Genera c:/reportes/salida.pdf
//	Crea c:/proyecto/docs/manual.md
//	Crea el directorio c:/temp/test y genera reporte.pdf
//
// This is intentionally a light heuristic, not a natural-language
// understanding system: it looks for a creation verb (generate/create/
// save, in Spanish or English) paired with either a directory keyword
// ("el directorio"/"la carpeta"/"directory"/"folder") or a path-shaped
// token that ends in a file extension. `/save` keeps working exactly as
// before — this only covers the case where the person never typed it.
package documents

import (
	"regexp"
	"strings"
)

// SaveIntent is what DetectSaveIntent found in a chat message.
type SaveIntent struct {
	Directories []string // e.g. ["c:/temp/test"] — create-only, no content needed
	Files       []string // e.g. ["carpeta/reporte.pdf"] — content comes from the model's reply
}

// HasIntent reports whether anything was detected at all.
func (s SaveIntent) HasIntent() bool {
	return len(s.Directories) > 0 || len(s.Files) > 0
}

var (
	// creationVerbRe recognizes a deliberately long list of creation verbs
	// in Spanish and English — conjugations, imperative/informal forms,
	// and near-synonyms — because people don't all phrase a request for a
	// new file the same way. There is no cap on how many times a verb can
	// appear or how it's phrased; ANY match here, in ANY clause, is
	// enough — precision instead comes from also requiring a directory
	// keyword or an extension-bearing path in that same clause (see
	// dirKeywordRe/pathTokenRe below), not from restricting the verb list.
	creationVerbRe = regexp.MustCompile(`(?i)\b(` +
		// Spanish: generar/crear + common near-synonyms, several
		// conjugations/moods each (indicative, imperative, informal "-me").
		`genera|generar|gener[aá]me|generame|` +
		`crea|crear|cre[aá]me|creame|` +
		`hac[eé]|hace|haz|h[aá]zme|hazme|haceme|` +
		`elabora|elaborar|elab[oó]rame|elaborame|` +
		`escribe|escribir|escr[ií]beme|escribeme|` +
		`redacta|redactar|redact[aá]me|redactame|` +
		`prepara|preparar|prep[aá]rame|preparame|` +
		`arma|armar|[aá]rmame|armame|` +
		`construye|construir|constr[uú]yeme|construyeme|` +
		`produce|producir|` +
		`guarda|guardar|` +
		// English: create/generate + common near-synonyms.
		`create|generate|make|build|write|draft|produce|prepare|put together|save` +
		`)\b`)

	dirKeywordRe   = regexp.MustCompile(`(?i)\b(?:el\s+directorio|la\s+carpeta|un\s+directorio|una\s+carpeta|the\s+directory|the\s+folder|directory|folder|carpeta|directorio)\b\s+([^\s,;]+)`)
	// pathTokenRe matches a path-shaped token ending in a short extension:
	// Windows drive paths (c:/a/b.ext, c:\a\b.ext), Unix absolute paths
	// (/a/b.ext), relative paths with a folder component (a/b.ext), or a
	// bare "name.ext".
	pathTokenRe = regexp.MustCompile(`(?i)([A-Za-z]:[\\/][^\s,;]+\.[A-Za-z0-9]{1,6}|[^\s,;]*/[^\s,;]+\.[A-Za-z0-9]{1,6}|[^\s,;]+\.[A-Za-z0-9]{1,6})`)
)

// DetectSaveIntent scans a chat message for creation intent. Clauses are
// split on " y "/" and " so a single message can both create a directory
// AND ask for a file, e.g. "Crea el directorio X y genera Y".
func DetectSaveIntent(text string) SaveIntent {
	var out SaveIntent
	for _, clause := range splitClauses(text) {
		if !creationVerbRe.MatchString(clause) {
			continue
		}
		if m := dirKeywordRe.FindStringSubmatch(clause); m != nil {
			out.Directories = append(out.Directories, strings.Trim(m[1], `"'`))
			continue
		}
		if m := pathTokenRe.FindStringSubmatch(clause); m != nil {
			out.Files = append(out.Files, strings.Trim(m[1], `"'`))
		}
	}
	return out
}

// splitClauses breaks a message on " y "/" and " (case-insensitive,
// whole-word) — the only conjunction this heuristic needs to handle
// "create a directory AND generate a file" in one sentence.
func splitClauses(text string) []string {
	sep := regexp.MustCompile(`(?i)\s+(?:y|and)\s+`)
	return sep.Split(text, -1)
}
