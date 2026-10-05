// preflight.go — qué HARÍA un bloque, sin escribir nada: lo usa la
// pregunta de confirmación para mostrar crear/modificar, líneas +/− y
// avisar de lo que se va a omitir ANTES de que la persona decida.
package patcher

import (
	"os"
	"strings"

	"mova.local/documents"
)

// Plan es el resultado de Preflight.
type Plan struct {
	Action  string // "create" | "modify"
	Added   int
	Removed int
	Skip    string // "" si se aplicará; si no, el motivo por el que se omitirá
}

func countLines(s string) int {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// Preflight evalúa un bloque con las mismas reglas que ApplyBlocksOpts.
func Preflight(root string, b documents.LabeledCodeBlock, opt Options) Plan {
	target, err := SafeTarget(root, b.Path)
	if err != nil {
		return Plan{Skip: err.Error()}
	}
	if opt.Excluded != nil {
		if ex, why := opt.Excluded(b.Path, b.Symbol); ex {
			return Plan{Skip: why}
		}
	}
	data, err := os.ReadFile(target)
	if os.IsNotExist(err) {
		return Plan{Action: "create", Added: countLines(b.Content)}
	}
	if err != nil {
		return Plan{Skip: err.Error()}
	}
	if b.Symbol == "" {
		return Plan{Action: "modify", Added: countLines(b.Content), Removed: countLines(string(data))}
	}
	lines := strings.Split(string(data), "\n")
	start := findSymbolStart(lines, b.Symbol)
	end := -1
	if start >= 0 {
		end = findFunctionEnd(lines, start)
	}
	if start < 0 || end < 0 {
		return Plan{Action: "modify", Skip: "symbol-not-found"}
	}
	if braceNet(strings.Join(lines[start:end+1], "\n")) != braceNet(b.Content) {
		return Plan{Action: "modify", Skip: "unbalanced"}
	}
	return Plan{Action: "modify", Added: countLines(b.Content), Removed: end - start + 1}
}
