// focus_format.go — línea de estado "[Focus] Selected ..." y helper de
// dedup compartido por el ensamblado de contexto. Movido tal cual desde
// engine.go (que superaba las 300 líneas); sin cambios de comportamiento.
package core

import (
	"fmt"
	"strings"

	corefocus "mova.local/core/focus"
	"mova.local/dedup"
)

// FormatFocusSelection arma la línea de estado "[Focus] Selected ..."
// a partir de los targets ya resueltos — usada por AMBOS `mova chat`
// (cli/chat_helpers.go) y el tool MCP/HTTP chat_completion
// (mcp/chat_tool.go), así CLI, HTTP y MCP muestran exactamente la misma
// línea para el mismo project.json. limit es cuántos nombres listar
// antes de colapsar el resto en un badge "+N" (ver FocusDisplayLimit).
// Devuelve "" cuando items está vacío (sin `focus`/`memory` configurado
// — no imprime nada, igual que antes).
func FormatFocusSelection(items []corefocus.FocusItem, limit int) string {
	if len(items) == 0 {
		return ""
	}
	if limit <= 0 {
		limit = DefaultFocusDisplayLimit
	}

	files, dirs, totalFiles := 0, 0, 0
	names := make([]string, 0, len(items))
	for _, it := range items {
		if it.Kind == "dir" {
			dirs++
		} else {
			files++
		}
		totalFiles += it.Files
		names = append(names, it.Name)
	}

	shownCount := limit
	if shownCount > len(names) {
		shownCount = len(names)
	}
	// Bug real reportado en QA: un target de `focus` literal "." (la
	// raíz del proyecto, ver corefocus.FocusItem.Name) hacía que la
	// lista terminara en "." — al concatenar el "." final fijo del
	// formato de abajo, el mensaje quedaba
	// "[Focus] Selected 1 directory (N file(s) total): ..\n" (dos
	// puntos seguidos), que parece un path truncado o corrupto en vez
	// de la raíz del proyecto. Se muestra "." como "(root)", legible,
	// en vez del carácter crudo de project.json.
	displayNames := make([]string, shownCount)
	for i, n := range names[:shownCount] {
		if n == "." {
			n = "(root)"
		}
		displayNames[i] = n
	}
	list := strings.Join(displayNames, ", ")
	if extra := len(names) - limit; extra > 0 {
		list += fmt.Sprintf(" 📎+%d", extra)
	}

	return fmt.Sprintf("[Focus] Selected %d %s (%d file(s) total): %s.\n",
		len(items), focusKindLabel(files, dirs), totalFiles, list)
}

// focusKindLabel describe la MEZCLA de targets resueltos ("2 files",
// "1 directory", "3 items" cuando hay de ambos tipos) — nunca inventa
// una categoría que no corresponda a lo realmente resuelto.
func focusKindLabel(files, dirs int) string {
	switch {
	case dirs == 0 && files == 1:
		return "file"
	case dirs == 0:
		return "files"
	case files == 0 && dirs == 1:
		return "directory"
	case files == 0:
		return "directories"
	default:
		return "item(s)"
	}
}

// DefaultFocusDisplayLimit: cuántos nombres de target muestra "[Focus]
// Selected ..." antes de colapsar el resto en "+N" cuando project.json
// no configura "focus_display_limit" — ver FocusDisplayLimit.
const DefaultFocusDisplayLimit = 2

// FocusDisplayLimit resuelve cuántos nombres mostrar: el
// "focus_display_limit" de project.json siempre gana; 0/ausente usa
// DefaultFocusDisplayLimit. Cualquier valor configurado (2, 4, 10, ...)
// se respeta tal cual — al superarlo SIEMPRE se colapsa el resto en el
// badge "+N", sin importar cuál sea el número.
func FocusDisplayLimit(proj *Project) int {
	if proj != nil && proj.FocusDisplayLimit > 0 {
		return proj.FocusDisplayLimit
	}
	return DefaultFocusDisplayLimit
}

// dedupSection applies dedup.Paragraphs to one prose chunk and accumulates
// its removed-count/removed-chars into sections — a small helper so every
// call site above stays a single readable line instead of repeating the
// same statements six times.
func dedupSection(text string, seen map[string]bool, sections *ContextSections) string {
	deduped, removed, chars := dedup.Paragraphs(text, seen)
	sections.DuplicatesRemoved += removed
	sections.DuplicatesRemovedChars += chars
	return deduped
}
