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

	dirKeywordRe = regexp.MustCompile(`(?i)\b(?:el\s+directorio|la\s+carpeta|un\s+directorio|una\s+carpeta|the\s+directory|the\s+folder|directory|folder|carpeta|directorio)\b\s+([^\s,;]+)`)
	// pathTokenRe matches a path-shaped token ending in a short extension:
	// Windows drive paths (c:/a/b.ext, c:\a\b.ext), Unix absolute paths
	// (/a/b.ext), relative paths with a folder component (a/b.ext), or a
	// bare "name.ext". Kept for backward compatibility with any external
	// caller; extractFileTarget (below) is the robust extractor every
	// detector in this package now actually uses.
	pathTokenRe = regexp.MustCompile(`(?i)([A-Za-z]:[\\/][^\s,;]+\.[A-Za-z0-9]{1,6}|[^\s,;]*/[^\s,;]+\.[A-Za-z0-9]{1,6}|[^\s,;]+\.[A-Za-z0-9]{1,6})`)
	// quotedPathRe matches a target wrapped in matching quotes — always
	// wins over any other extraction, since quotes are an explicit,
	// unambiguous delimiter the person chose themselves (handles spaces,
	// commas, anything).
	quotedPathRe = regexp.MustCompile(`"([^"]+)"|'([^']+)'`)
	// absPathRe matches a Windows drive path (C:\..., D:/...), a UNC
	// network path (\\server\share\...), or a Unix absolute path
	// (/mnt/..., /home/...) — greedy to the next whitespace, so it does
	// NOT stop at a comma the way the old [^\s,;]+ token matchers did
	// (a real bug: "C:\a\pruebas3,4\prueba.py" has a comma INSIDE a
	// legitimate path segment). The Unix alternative requires the "/"
	// to start the token (preceded by whitespace or string-start) — a
	// bare, non-anchored "/" would otherwise match mid-word inside an
	// ordinary RELATIVE path like "carpeta/reporte.pdf" (a real
	// regression this exact anchoring fixes).
	absPathRe = regexp.MustCompile(`(?i)([A-Za-z]:[\\/]\S+|\\\\\S+)|(?:^|\s)(/\S+)`)
	// bareFilenameRe matches a short filename token ending in an
	// extension, with no path separators of its own — "test.cpp",
	// "hola.txt" — used only after any absolute path already found in
	// the clause has been masked out, so it never re-matches a fragment
	// of that same path.
	bareFilenameRe = regexp.MustCompile(`(?i)\b[\w.-]+\.[A-Za-z0-9]{1,6}\b`)
	// relPathRe matches a RELATIVE path with at least one folder
	// component ("carpeta/reporte.pdf", "src/server.js") — tried before
	// bareFilenameRe so the folder isn't silently dropped the way a
	// plain \w-only match would (a real regression caught by this
	// package's own tests while fixing the absolute-path handling
	// above: "carpeta/reporte.pdf" was collapsing to just
	// "reporte.pdf").
	relPathRe = regexp.MustCompile(`(?i)\S+/[\w.-]+\.[A-Za-z0-9]{1,6}\b`)
	// bareTokenRe is the last-resort fallback for a directory-only
	// target with no extension and no separator ("elimina el
	// directorio caca") — same greedy-to-whitespace rule as absPathRe.
	bareTokenRe = regexp.MustCompile(`\S+`)
)

// extractFileTarget is the ONE path/filename extractor every detector in
// this package uses (DetectSaveIntent, DetectEditIntent,
// DetectDeleteIntent, DetectReadIntent, DetectRenameIntent) — replacing
// each detector's own "grab the word right after the keyword" logic,
// which broke on connector words ("que se encuentra en X" captured
// "que"), on multi-word/quoted paths, and on a directory named in a
// SEPARATE clause from the filename ("test.cpp ... en la ruta C:\...").
//
// Strategy: find the best absolute path in the clause (quoted, or a
// Windows/UNC/Unix absolute path) AND, separately, the best bare
// filename (checked in what's left after masking out that absolute
// path, so the two never double-match the same substring) — then
// combine them the way a person actually means:
//   - only a bare filename ("test.cpp")            -> "test.cpp"
//   - only an absolute path that already ends in an
//     extension ("C:\a\test.cpp")                  -> "C:\a\test.cpp"
//   - an absolute DIRECTORY (no extension) PLUS a
//     separately-named filename ("test.cpp" ...
//     "en la ruta C:\a\b")                          -> "C:\a\b\test.cpp"
//   - only an absolute directory, no filename
//     ("el directorio C:\a\b")                       -> "C:\a\b"
func extractFileTarget(clause string) (string, bool) {
	if m := quotedPathRe.FindStringSubmatch(clause); m != nil {
		if m[1] != "" {
			return m[1], true
		}
		return m[2], true
	}

	absMatch := ""
	if m := absPathRe.FindStringSubmatch(clause); m != nil {
		if m[1] != "" {
			absMatch = m[1]
		} else {
			absMatch = m[2]
		}
	}
	remainder := clause
	if absMatch != "" {
		remainder = strings.Replace(clause, absMatch, " ", 1)
	}
	absMatch = strings.TrimRight(absMatch, `.,;:"'`)

	bare := ""
	if m := relPathRe.FindString(remainder); m != "" {
		bare = strings.TrimRight(m, `.,;:"'`)
	} else if m := bareFilenameRe.FindString(remainder); m != "" {
		bare = strings.TrimRight(m, `.,;:"'`)
	}

	switch {
	case absMatch != "" && hasKnownExtension(absMatch):
		return absMatch, true
	case absMatch != "" && bare != "":
		return joinPath(absMatch, bare), true
	case absMatch != "":
		return absMatch, true
	case bare != "":
		return bare, true
	}
	return "", false
}

// hasKnownExtension reports whether p's last path segment already looks
// like "name.ext" — used to tell "an absolute path that IS the file"
// apart from "an absolute path that's just the containing directory".
func hasKnownExtension(p string) bool {
	base := p
	if i := strings.LastIndexAny(base, `/\`); i != -1 {
		base = base[i+1:]
	}
	return bareFilenameRe.MatchString(base)
}

// joinPath combines an absolute directory with a bare filename,
// preserving the directory's own separator style (\ for a Windows/UNC
// path, / otherwise) so a Windows path stays a Windows path.
func joinPath(dir, filename string) string {
	sep := "/"
	if strings.Contains(dir, `\`) {
		sep = `\`
	}
	return strings.TrimRight(dir, `/\`) + sep + filename
}

// DetectSaveIntent scans a chat message for creation intent. Clauses are
// split on " y "/" and " so a single message can both create a directory
// AND ask for a file, e.g. "Crea el directorio X y genera Y". The creation
// verb only needs to appear ONCE anywhere in the message, not once per
// clause — "crea hola.txt y otro chau.txt" has the verb only in the first
// clause, but "otro chau.txt" is still clearly part of the same request;
// requiring the verb per-clause used to silently drop every file after
// the first one in a multi-file message.
func DetectSaveIntent(text string) SaveIntent {
	var out SaveIntent
	if !creationVerbRe.MatchString(text) {
		return out
	}
	for _, clause := range splitClauses(text) {
		if m := dirKeywordRe.FindStringSubmatch(clause); m != nil {
			out.Directories = append(out.Directories, strings.Trim(m[1], `"'`))
			continue
		}
		if target, ok := extractFileTarget(clause); ok {
			out.Files = append(out.Files, target)
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
