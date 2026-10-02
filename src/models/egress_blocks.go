// egress_blocks.go — egress_sanitized.md SIN duplicados.
//
// Antes cada turno de chat anexaba el contexto COMPLETO al final del
// archivo (agents+skills+prompt+focus repetidos N veces: el archivo crecía
// sin aportar nada). Ahora el archivo es un almacén de bloques con clave:
//
//   - un bloque = una pieza del contexto (cabecera, cada agent/skill/prompt,
//     cada FOCUS, memoria, resultados de tareas, instrucción);
//   - misma clave y mismo contenido  → no se toca nada (ni se reescribe el
//     archivo);
//   - misma clave y contenido distinto → el bloque se REEMPLAZA completo;
//   - clave nueva → se anexa al final.
//
// Los bloques quedan en el orden en que aparecieron. Un contexto sin
// marcadores (texto plano) es un único bloque "context".
package models

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

type egressBlock struct{ Key, Sha, Updated, Text string }

var (
	reMarker  = regexp.MustCompile(`^<!-- (core|agent|skill|prompt|result): (.+?) -->$`)
	reSection = regexp.MustCompile(`^## (AGENTS|SKILLS|PROMPT|FOCUS|MEMORY|INSTRUCTION)\s*$`)
)

func shaOf(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:16]
}

// blockKey: clave de un bloque si la línea abre uno nuevo, "" si no.
func blockKey(line string) string {
	switch {
	case strings.HasPrefix(line, "# Mova Context"):
		return "header"
	case strings.HasPrefix(line, "FOCUS:"):
		return "focus:" + strings.TrimSpace(strings.TrimPrefix(line, "FOCUS:"))
	}
	if m := reMarker.FindStringSubmatch(line); m != nil {
		return m[1] + ":" + m[2]
	}
	if m := reSection.FindStringSubmatch(line); m != nil {
		return "section:" + m[1]
	}
	return ""
}

// splitContextBlocks parte un contexto en bloques con clave. Quita la
// línea "Generated: <fecha>" (cambia cada minuto y haría que la cabecera
// pareciera distinta siempre) y los separadores "---" sueltos.
func splitContextBlocks(ctx string) []egressBlock {
	var blocks []egressBlock
	seen := map[string]int{}
	cur, curKey := []string{}, "context"
	flush := func() {
		text := strings.Trim(strings.Join(cur, "\n"), "\n -")
		cur = cur[:0]
		if strings.TrimSpace(text) == "" {
			return
		}
		key := curKey
		if n := seen[key]; n > 0 {
			key = key + "#" + string(rune('1'+n))
		}
		seen[curKey]++
		blocks = append(blocks, egressBlock{Key: key, Sha: shaOf(text), Text: text})
	}
	for _, line := range strings.Split(strings.ReplaceAll(ctx, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "Generated: ") {
			continue
		}
		if k := blockKey(line); k != "" {
			flush()
			curKey = k
		}
		cur = append(cur, line)
	}
	flush()
	return blocks
}

// mergeBlocks aplica los bloques nuevos sobre los existentes: reemplaza
// los cambiados en su sitio, anexa los nuevos, deja intactos los iguales.
// changed indica si hubo alguna diferencia real.
func mergeBlocks(existing, incoming []egressBlock, now string) (out []egressBlock, changed bool) {
	out = append(out, existing...)
	idx := make(map[string]int, len(out))
	for i, b := range out {
		idx[b.Key] = i
	}
	for _, nb := range incoming {
		nb.Updated = now
		i, ok := idx[nb.Key]
		switch {
		case !ok:
			idx[nb.Key] = len(out)
			out = append(out, nb)
			changed = true
		case out[i].Sha != nb.Sha:
			out[i] = nb
			changed = true
		}
	}
	return out, changed
}
