// compiler_focus.go — Context Compiler, Fase 2: "Poda quirúrgica".
//
// When project.json defines "focus", resolves each focus item (file / dir /
// symbol) against "repo" and extracts only the relevant fragment instead of
// sending whole files. Uses lightweight structural heuristics (brace/indent
// matching, heading hierarchy, date blocks) — not a full compiler-grade AST,
// but enough to reliably avoid sending unrelated content. Falls back to a
// bounded excerpt when no structural match is found; never silently sends
// a full, unrelated file when focus is set.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// resolveRepoPath applies the workflow.md "repo" resolution rules.
func resolveRepoPath(root, repo string) string {
	if repo == "" || repo == "." {
		return root
	}
	if filepath.IsAbs(repo) {
		return repo
	}
	return filepath.Join(root, repo)
}

// buildFocusContext resolves every focus item and returns a compact,
// machine-oriented block. Never dumps unrelated full files.
func buildFocusContext(root string, proj *Project, focus []string) string {
	if len(focus) == 0 {
		return ""
	}
	repoPath := resolveRepoPath(root, proj.Repo)
	var sb strings.Builder
	for _, item := range focus {
		sb.WriteString("FOCUS:" + item + "\n")
		sb.WriteString(resolveFocusItem(repoPath, item))
		sb.WriteString("\n")
	}
	return sb.String()
}

// resolveFocusItem classifies a focus item and extracts its content.
// Resolution order follows workflow.md FOCUS: absolute/relative path,
// bare name searched recursively, then symbol (function/class/method,
// legal heading, or chronological keyword).
func resolveFocusItem(repoPath, item string) string {
	hasParens := strings.HasSuffix(item, "()")
	symbol := strings.TrimSuffix(item, "()")

	if !hasParens {
		path := item
		if !filepath.IsAbs(path) {
			path = filepath.Join(repoPath, item)
		}
		if isDir(path) {
			return listDir(path)
		}
		content := readFile(path)
		if content == "" && !strings.ContainsAny(item, `/\`) {
			if found := findByName(repoPath, item); found != "" {
				content = readFile(found)
			}
		}
		if content != "" {
			return extractFragment(content, "")
		}
		// no file/dir matched — fall through and try it as a symbol
	}

	match, matchPath := findSymbolInRepo(repoPath, symbol)
	if match == "" {
		return "  not found: " + item
	}
	return fmt.Sprintf("  (%s)\n%s", relOrBase(repoPath, matchPath), match)
}

// extractFragment tries, in order: code symbol, document heading,
// chronological block. query == "" means "the whole file is the requested
// scope" — returned as-is, since it was explicitly named by the user (not
// an unrelated full file the compiler decided to include).
func extractFragment(content, query string) string {
	if query == "" {
		return content
	}
	if frag, ok := extractCodeSymbol(content, query); ok {
		return frag
	}
	if frag, ok := extractDocSection(content, query); ok {
		return frag
	}
	if frag, ok := extractChronological(content, query); ok {
		return frag
	}
	return boundedExcerpt(content, query)
}

// findSymbolInRepo walks repoPath and returns the first structural match
// (code symbol first, then document heading).
func findSymbolInRepo(repoPath, symbol string) (fragment, path string) {
	var candidates []string
	walkFiles(repoPath, func(p string) { candidates = append(candidates, p) })
	sort.Strings(candidates)
	for _, p := range candidates {
		if content := readFile(p); content != "" {
			if frag, ok := extractCodeSymbol(content, symbol); ok {
				return frag, p
			}
		}
	}
	for _, p := range candidates {
		if frag, ok := extractDocSection(readFile(p), symbol); ok {
			return frag, p
		}
	}
	return "", ""
}

// codeDeclPattern matches a function/class/method/interface declaration line
// for the requested symbol across common brace languages, plus Python.
func codeDeclPattern(symbol string) *regexp.Regexp {
	name := regexp.QuoteMeta(symbol)
	return regexp.MustCompile(
		`(?i)^\s*(func|def|class|interface|type|public|private|protected|static|export|async)?[\s\w<>\[\],.*]*\b` + name + `\b\s*[(<:{]`)
}

// extractCodeSymbol extracts a function/class/method body via brace or
// indentation matching. Heuristic — not a real parser — but reliably scopes
// to one declaration for typical code style.
func extractCodeSymbol(content, symbol string) (string, bool) {
	lines := strings.Split(content, "\n")
	decl := codeDeclPattern(symbol)
	for i, line := range lines {
		if !decl.MatchString(line) {
			continue
		}
		if strings.Contains(line, "{") {
			return extractBraceBlock(lines, i), true
		}
		if strings.TrimSpace(line) != "" {
			if frag := extractIndentBlock(lines, i); frag != "" {
				return frag, true // Python-style / brace-less header
			}
		}
	}
	return "", false
}

// extractBraceBlock returns lines from start to the matching closing brace.
func extractBraceBlock(lines []string, start int) string {
	depth := 0
	end := start
	started := false
	for i := start; i < len(lines); i++ {
		depth += strings.Count(lines[i], "{") - strings.Count(lines[i], "}")
		if strings.Contains(lines[i], "{") {
			started = true
		}
		end = i
		if started && depth <= 0 {
			break
		}
	}
	return strings.Join(lines[start:end+1], "\n")
}

// extractIndentBlock returns the header line plus every deeper-indented line
// that follows (Python def/class style).
func extractIndentBlock(lines []string, start int) string {
	base := indentOf(lines[start])
	end := start
	for i := start + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			end = i
			continue
		}
		if indentOf(lines[i]) <= base {
			break
		}
		end = i
	}
	if end == start {
		return ""
	}
	return strings.Join(lines[start:end+1], "\n")
}

func indentOf(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

// docHeadingPattern matches legal/manual hierarchy headings (Título, Capítulo,
// Sección, Artículo, Inciso) plus generic Markdown headings.
var docHeadingPattern = regexp.MustCompile(`(?i)^\s*(#{1,6}\s+.*|(t[íi]tulo|cap[íi]tulo|secci[óo]n|art[íi]culo|inciso)\s+\S.*)$`)

// extractDocSection returns the span from the matching heading up to (but
// excluding) the next heading — the hierarchical extraction described in
// FASE 2 for laws, contracts, manuals and procedures.
func extractDocSection(content, query string) (string, bool) {
	lines := strings.Split(content, "\n")
	q := strings.ToLower(query)
	start := -1
	for i, line := range lines {
		if docHeadingPattern.MatchString(line) && strings.Contains(strings.ToLower(line), q) {
			start = i
			break
		}
	}
	if start == -1 {
		return "", false
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if docHeadingPattern.MatchString(lines[i]) {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n"), true
}

// datedLinePattern matches lines starting a chronological entry (log, call
// record, transaction, clinical note...): a date followed by content.
var datedLinePattern = regexp.MustCompile(`^\s*\**\[?(\d{4}-\d{2}-\d{2}|\d{2}/\d{2}/\d{4})\]?\**`)

// extractChronological returns dated blocks whose text contains the query —
// the Categoría/Fecha/Evento/Datos extraction described in FASE 2 for logs,
// call center, finance and health histories.
func extractChronological(content, query string) (string, bool) {
	lines := strings.Split(content, "\n")
	q := strings.ToLower(query)
	var blocks []string
	var cur []string
	flush := func() {
		if len(cur) > 0 && strings.Contains(strings.ToLower(strings.Join(cur, " ")), q) {
			blocks = append(blocks, strings.Join(cur, "\n"))
		}
		cur = nil
	}
	for _, line := range lines {
		if datedLinePattern.MatchString(line) {
			flush()
		}
		cur = append(cur, line)
	}
	flush()
	if len(blocks) == 0 {
		return "", false
	}
	return strings.Join(blocks, "\n---\n"), true
}

// boundedExcerpt is the last-resort fallback: a small window of context
// around the first textual occurrence of query. Deliberately short — the
// point of "focus" is to avoid sending unrelated content.
func boundedExcerpt(content, query string) string {
	idx := strings.Index(strings.ToLower(content), strings.ToLower(query))
	if idx == -1 {
		return "  no structural or textual match for: " + query
	}
	lines := strings.Split(content, "\n")
	pos, count := 0, 0
	for i, l := range lines {
		count += len(l) + 1
		if count >= idx {
			pos = i
			break
		}
	}
	s, e := pos-10, pos+10
	if s < 0 {
		s = 0
	}
	if e > len(lines) {
		e = len(lines)
	}
	return "  [no exact structural match — nearby excerpt]\n" + strings.Join(lines[s:e], "\n")
}

// ── small fs helpers (compiler-only; distinct from file_adapter's) ─────────

type dirEntry struct {
	path  string
	isDir bool
}

func listEntries(dir string) []dirEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]dirEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, dirEntry{path: filepath.Join(dir, e.Name()), isDir: e.IsDir()})
	}
	return out
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// listDir returns a compact, sorted index of a directory — never its
// contents. Directories are legitimate focus targets, but dumping every
// file inside would defeat the purpose of "focus".
func listDir(path string) string {
	entries := listEntries(path)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		n := filepath.Base(e.path)
		if e.isDir {
			n += "/"
		}
		names = append(names, n)
	}
	sort.Strings(names)
	return fmt.Sprintf("  dir(%d): %s", len(names), strings.Join(names, ", "))
}

func walkFiles(dir string, fn func(path string)) {
	for _, e := range listEntries(dir) {
		if e.isDir {
			walkFiles(e.path, fn)
		} else {
			fn(e.path)
		}
	}
}

func findByName(dir, name string) string {
	var found string
	walkFiles(dir, func(p string) {
		if found == "" && filepath.Base(p) == name {
			found = p
		}
	})
	return found
}

func relOrBase(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		return rel
	}
	return filepath.Base(path)
}
