// egress_store.go — lectura/escritura del archivo de bloques de
// egress_blocks.go: formato, migración del formato antiguo y escritura
// atómica solo cuando algo cambió.
package models

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const egressHeader = "# Egress audit — contexto sanitizado enviado al LLM\n\n" +
	"> Archivo auto-gestionado: un bloque por pieza del contexto. Igual → se omite; " +
	"cambiado → se reemplaza completo; nuevo → se anexa. Nunca se duplica.\n"

var (
	reBlockOpen = regexp.MustCompile("^<!-- egress:block key=\"(.*)\" sha=\"([0-9a-f]+)\" updated=\"([^\"]*)\" -->$")
	egressEnd   = "<!-- egress:end -->"
	egressLocks sync.Map // ruta → *sync.Mutex (chat, MCP y HTTP comparten proceso)
)

func fenceFor(text string) string {
	longest, run := 0, 0
	for _, r := range text {
		if r == '`' {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
	}
	return strings.Repeat("`", max(3, longest+1))
}

func renderEgressFile(blocks []egressBlock) string {
	var b strings.Builder
	b.WriteString(egressHeader)
	for _, bl := range blocks {
		f := fenceFor(bl.Text)
		fmt.Fprintf(&b, "\n<!-- egress:block key=%q sha=%q updated=%q -->\n### %s\n%stext\n%s\n%s\n%s\n",
			bl.Key, bl.Sha, bl.Updated, bl.Key, f, bl.Text, f, egressEnd)
	}
	return b.String()
}

// parseEgressFile lee el formato actual; si el archivo es del formato
// antiguo (un bloque "## Egress audit — execution" por turno) conserva
// SOLO el último contexto, que es lo único que no era repetición.
func parseEgressFile(data string) []egressBlock {
	data = strings.ReplaceAll(data, "\r\n", "\n")
	if !strings.Contains(data, "<!-- egress:block ") {
		return migrateLegacyEgress(data)
	}
	var out []egressBlock
	lines := strings.Split(data, "\n")
	for i := 0; i < len(lines); i++ {
		m := reBlockOpen.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		j := i + 1
		for j < len(lines) && !strings.HasPrefix(lines[j], "```") && !strings.HasPrefix(lines[j], egressEnd) {
			j++ // salta "" y "### clave"
		}
		if j >= len(lines) || strings.HasPrefix(lines[j], egressEnd) {
			continue
		}
		fence := strings.TrimSuffix(lines[j], "text")
		k := j + 1
		var body []string
		for k < len(lines) && lines[k] != fence {
			body = append(body, lines[k])
			k++
		}
		text := strings.Join(body, "\n")
		key := m[1]
		if uq, err := strconv.Unquote(`"` + key + `"`); err == nil {
			key = uq
		}
		out = append(out, egressBlock{Key: key, Sha: shaOf(text), Updated: m[3], Text: text})
		i = k
	}
	return out
}

func migrateLegacyEgress(data string) []egressBlock {
	const marker = "### Sanitized context sent to the LLM\n\n```text\n"
	i := strings.LastIndex(data, marker)
	if i < 0 {
		return nil
	}
	body := data[i+len(marker):]
	if j := strings.LastIndex(body, "\n```"); j >= 0 {
		body = body[:j]
	}
	return splitContextBlocks(body)
}

func lockFor(path string) *sync.Mutex {
	mu, _ := egressLocks.LoadOrStore(path, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

// writeEgressBlocks fusiona ctx en el archivo y lo reescribe solo si hubo
// un cambio real; si no, no toca el disco.
func writeEgressBlocks(outputFile, ctx string) error {
	mu := lockFor(outputFile)
	mu.Lock()
	defer mu.Unlock()
	if dir := filepath.Dir(outputFile); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}
	var existing []egressBlock
	raw, err := os.ReadFile(outputFile)
	switch {
	case err == nil:
		existing = parseEgressFile(string(raw))
	case !os.IsNotExist(err):
		return fmt.Errorf("opening %s: %w", outputFile, err)
	}
	legacy := err == nil && !strings.Contains(string(raw), "<!-- egress:block ")
	merged, changed := mergeBlocks(existing, splitContextBlocks(ctx), time.Now().UTC().Format(time.RFC3339))
	if !changed && !legacy && err == nil {
		return nil
	}
	tmp := fmt.Sprintf("%s.tmp-%d", outputFile, os.Getpid())
	if err := os.WriteFile(tmp, []byte(renderEgressFile(merged)), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outputFile, err)
	}
	if err := os.Rename(tmp, outputFile); err != nil { // Windows: reintento tras borrar
		_ = os.Remove(outputFile)
		if err = os.Rename(tmp, outputFile); err != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("writing %s: %w", outputFile, err)
		}
	}
	return nil
}
