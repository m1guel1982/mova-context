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
	WholeFile bool   // archivo completo (nuevo, o bloque sin símbolo)
	Backup    string // copia del original antes de sobrescribir ("" si no hubo)
}

// ApplyBlocks mantiene la firma histórica (sin respaldo ni exclusiones):
// ver ApplyBlocksOpts, que es la implementación única que usan Chat, MCP
// y HTTP.
func ApplyBlocks(root string, blocks []documents.LabeledCodeBlock) ([]AppliedFile, error) {
	applied, _, err := ApplyBlocksOpts(root, blocks, Options{})
	return applied, err
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

// symbolPatterns: formas de declaración que reconoce el parcheador para
// un símbolo (%s = nombre exacto). Cubre Go/JS/TS/Java/C ("func"/"function"/
// "def"), métodos de clase JS/TS (`async _processOrder(a, b) {`), y
// funciones flecha / expresiones (`const f = (a) => {`, `const f = function`).
// Las flechas de una sola expresión (sin llaves) NO se parchean: se
// reportan como "no encontrado" en vez de adivinar.
var symbolPatterns = []string{
	`^\s*(?:export\s+)?(?:default\s+)?(?:async\s+)?(?:func|function\*?|def)\s+%s\s*\(`,
	`^\s*(?:(?:public|private|protected|static|async|get|set)\s+)*%s\s*\([^)]*\)\s*(?::\s*[\w<>\[\]| ]+)?\s*\{`,
	`^\s*(?:export\s+)?(?:const|let|var)\s+%s\s*=\s*(?:async\s*)?(?:function\b|\([^)]*\)\s*=>|[A-Za-z_$][\w$]*\s*=>)`,
}

func findSymbolStart(lines []string, symbol string) int {
	var res []*regexp.Regexp
	for _, p := range symbolPatterns {
		res = append(res, regexp.MustCompile(fmt.Sprintf(p, regexp.QuoteMeta(symbol))))
	}
	for i, line := range lines {
		for _, re := range res {
			if re.MatchString(line) {
				return i
			}
		}
	}
	return -1
}

// patchSymbol reemplaza exactamente la función nombrada dentro de path,
// sin tocar ninguna otra línea. Devuelve ok=false (nunca error) si no la
// localiza con certeza — y el llamador YA NO reescribe el archivo entero
// con un fragmento (antes lo hacía y destruía el resto del archivo).
// Rechaza (error) un reemplazo que desbalancee llaves respecto de lo que
// sustituye: señal de un fragmento truncado o mal copiado.
func patchSymbol(path, symbol, newContent string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	lines := strings.Split(string(data), "\n")
	start := findSymbolStart(lines, symbol)
	if start == -1 {
		return false, nil
	}
	end := findFunctionEnd(lines, start)
	if end == -1 {
		return false, nil
	}
	old := strings.Join(lines[start:end+1], "\n")
	if braceNet(old) != braceNet(newContent) {
		return false, fmt.Errorf("el bloque de reemplazo desbalancea llaves respecto de la función original (probable fragmento truncado); no se escribió nada")
	}
	var out []string
	out = append(out, lines[:start]...)
	out = append(out, strings.Split(strings.TrimRight(newContent, "\n"), "\n")...)
	out = append(out, lines[end+1:]...)
	return true, atomicWriteFile(path, strings.Join(out, "\n"))
}

func braceNet(s string) int {
	n := 0
	for _, ch := range s {
		switch ch {
		case '{':
			n++
		case '}':
			n--
		}
	}
	return n
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
