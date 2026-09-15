// patcher.go — writes mova.local/documents.LabeledCodeBlock values to
// disk: the Core Engine behind Mova Chat's "Auto-Apply" flow (see
// cli/chat_apply.go, mcp/chat_tool.go, http/server.go's "auto_patch"
// handling - all three call ApplyBlocks instead of keeping their own
// copy, the same one-implementation-three-doors discipline
// mova.local/documents.SelectContent already established for /save).
//
// Symbol-level patching (3.1 in the spec: "server.js::validateToken()")
// is a deliberately lightweight, brace/indent-based function-boundary
// finder — NOT a real parser for nine languages (see
// trace/relevance.go's header for the same honesty note about why: a
// true embedded multi-language AST is a multi-week vendoring effort,
// commonly needs CGO, and is out of scope here). When the named symbol
// isn't found with confidence, ApplyBlocks always falls back to a safe,
// atomic whole-file write, exactly as the spec requires (3.1: "Si...el
// selector AST no encuentra la firma exacta, realiza la actualización
// atómica del archivo completo").
package patcher

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"mova.local/documents"
)

// AppliedFile is one file this run actually wrote — the shape
// cli/chat_apply.go's "[Auto-Apply] Archivo actualizado: <ruta>" line
// and the HTTP "files_modified" array are both built from.
type AppliedFile struct {
	Path      string
	Symbol    string // "" for a whole-file write
	WholeFile bool   // true if the symbol patch fell back to a full rewrite
}

// ApplyBlocks writes every block to disk under root, one file at a
// time, and returns what was actually written, in order. A write
// failure for one block does not stop the rest — every block is
// independent (multi-file, multi-symbol support per spec 3.2) — but
// is returned as an error so the caller can report exactly which
// file(s) failed.
func ApplyBlocks(root string, blocks []documents.LabeledCodeBlock) ([]AppliedFile, error) {
	var applied []AppliedFile
	var errs []string

	for _, b := range blocks {
		target := resolveTargetPath(root, b.Path)
		if b.Symbol == "" {
			if err := atomicWriteFile(target, b.Content); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", b.Path, err))
				continue
			}
			applied = append(applied, AppliedFile{Path: b.Path, WholeFile: true})
			continue
		}

		ok, err := patchSymbol(target, b.Symbol, b.Content)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s::%s(): %v", b.Path, b.Symbol, err))
			continue
		}
		if ok {
			applied = append(applied, AppliedFile{Path: b.Path, Symbol: b.Symbol})
			continue
		}
		// Symbol not found with confidence - safe fallback: atomic
		// whole-file write (see this file's header).
		if err := atomicWriteFile(target, b.Content); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", b.Path, err))
			continue
		}
		applied = append(applied, AppliedFile{Path: b.Path, Symbol: b.Symbol, WholeFile: true})
	}

	if len(errs) > 0 {
		return applied, fmt.Errorf("failed to apply %d block(s): %s", len(errs), strings.Join(errs, "; "))
	}
	return applied, nil
}

func resolveTargetPath(root, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(root, p)
}

// atomicWriteFile writes content to a temp file in the same
// directory, then renames it over path - a rename is atomic on every
// OS Mova Context targets, so a crash mid-write never leaves a
// half-written source file behind.
func atomicWriteFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("could not create directory: %w", err)
	}
	tmp := path + ".mova-apply.tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return fmt.Errorf("could not write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("could not finalize write: %w", err)
	}
	return nil
}

// funcHeaderPattern matches the most common function/method
// declaration shapes across the language families Mova Context deals
// with (Go/JS/TS/Java/C-family: "func"/"function"; Python: "def").
// %s is substituted with the exact symbol name being searched for.
const funcHeaderPattern = `^\s*(?:export\s+)?(?:async\s+)?(?:func|function|def)\s+%s\s*\(`

// patchSymbol replaces exactly the body of the named function inside
// path with newContent, preserving every other line untouched
// (imports, other functions, comments, global variables - spec 3.3's
// "Trazabilidad y Estilo" requirement). Returns ok=false (never an
// error) when the symbol cannot be located with confidence, so the
// caller falls back to a whole-file write instead of guessing.
func patchSymbol(path, symbol, newContent string) (bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil // nothing to patch into yet - fall back to whole-file create
	}
	if err != nil {
		return false, err
	}
	lines := strings.Split(string(data), "\n")

	headerRe := regexp.MustCompile(fmt.Sprintf(funcHeaderPattern, regexp.QuoteMeta(symbol)))
	start := -1
	for i, line := range lines {
		if headerRe.MatchString(line) {
			start = i
			break
		}
	}
	if start == -1 {
		return false, nil
	}

	end := findFunctionEnd(lines, start)
	if end == -1 {
		return false, nil
	}

	var out []string
	out = append(out, lines[:start]...)
	out = append(out, strings.Split(strings.TrimRight(newContent, "\n"), "\n")...)
	out = append(out, lines[end+1:]...)

	return true, atomicWriteFile(path, strings.Join(out, "\n"))
}

// findFunctionEnd locates the last line of the function that starts
// at lines[start], using brace-balance for brace-delimited languages
// and indentation for Python's "def" - a well-understood, deterministic
// heuristic, not a parser (see this file's header).
func findFunctionEnd(lines []string, start int) int {
	trimmedStart := strings.TrimSpace(lines[start])
	if strings.HasPrefix(trimmedStart, "def ") {
		baseIndent := len(lines[start]) - len(strings.TrimLeft(lines[start], " \t"))
		for i := start + 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "" {
				continue
			}
			indent := len(lines[i]) - len(strings.TrimLeft(lines[i], " \t"))
			if indent <= baseIndent {
				return i - 1
			}
		}
		return len(lines) - 1
	}

	depth := 0
	seenBrace := false
	for i := start; i < len(lines); i++ {
		for _, ch := range lines[i] {
			switch ch {
			case '{':
				depth++
				seenBrace = true
			case '}':
				depth--
			}
		}
		if seenBrace && depth <= 0 {
			return i
		}
	}
	return -1
}
