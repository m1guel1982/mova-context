// memory_view.go — qué parte de memory.md entra al contexto. memory.md
// crece con cada tarea; inyectarlo entero consumiría el presupuesto de
// tokens. Si cabe en el tope, se inyecta TAL CUAL (sin cambios). Si no:
//
//  1. SIEMPRE se conserva la entrada más reciente de CADA tarea ("fijadas"):
//     el análisis de una tarea nunca se pierde porque otra escribió mucho
//     después;
//  2. el resto, de la más nueva a la más vieja, mientras quepa;
//  3. las que no caben quedan como una línea de título, y se avisa cuántas.
package core

import (
	"fmt"
	"regexp"
	"strings"
)

// DefaultMemoryMaxChars: ≈5k tokens de memoria en el contexto.
const DefaultMemoryMaxChars = 20000

var reEntryTask = regexp.MustCompile(`<!-- mova:entry task=(\S+) `)

type memEntry struct {
	text, task string
	pinned     bool
}

// FormatMemoryForContext aplica el tope maxChars (0 = por defecto).
func FormatMemoryForContext(mem string, maxChars int) string {
	if maxChars <= 0 {
		maxChars = DefaultMemoryMaxChars
	}
	if len(mem) <= maxChars {
		return mem
	}
	var entries []memEntry
	seen := map[string]bool{}
	pins := 0
	for _, raw := range strings.Split(mem, memorySep) {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		e := memEntry{text: strings.TrimSpace(raw)}
		if m := reEntryTask.FindStringSubmatch(raw); m != nil {
			e.task = m[1]
			if !seen[e.task] { // la primera de cada tarea es la más reciente
				seen[e.task], e.pinned = true, true
				pins++
			}
		}
		entries = append(entries, e)
	}
	pinCap := max(1500, maxChars/max(pins, 1))
	left := maxChars
	keepText := make([]string, len(entries))
	for i, e := range entries { // 1) fijadas
		if e.pinned {
			keepText[i] = cutRunes(e.text, pinCap)
			if len(keepText[i]) < len(e.text) {
				keepText[i] += "\n…[entrada recortada; completa en memory.md]"
			}
			left -= len(keepText[i])
		}
	}
	omitted := 0
	for i, e := range entries { // 2) el resto, de nuevo a viejo
		if e.pinned {
			continue
		}
		switch {
		case len(e.text) <= left:
			keepText[i], left = e.text, left-len(e.text)
		default: // 3) solo el título
			title := strings.SplitN(e.text, "\n", 2)[0]
			if len(title) <= left {
				keepText[i], left = title+" …", left-len(title)-2
			}
			omitted++
		}
	}
	var out []string
	for _, t := range keepText {
		if t != "" {
			out = append(out, t)
		}
	}
	res := strings.Join(out, memorySep)
	if omitted > 0 {
		res += fmt.Sprintf("\n\n<!-- %d entrada(s) antigua(s) abreviadas u omitidas por el tope de memoria (%d caracteres); ver memory.md -->", omitted, maxChars)
	}
	return res
}

// cutRunes recorta a n bytes sin partir un carácter UTF-8.
func cutRunes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && n < len(s) && s[n]&0xC0 == 0x80 {
		n--
	}
	return s[:n]
}
