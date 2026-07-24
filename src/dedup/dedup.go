package dedup

import (
	"regexp"
	"strings"
)

// Normaliza espacios y saltos de línea dentro del párrafo
func normalizeParagraph(p string) string {
	return strings.Join(strings.Fields(p), " ")
}

func Paragraphs(content string, seen map[string]bool) (result string, removedCount int, removedChars int) {
	// Normalizar primero los saltos de línea de Windows (\r\n -> \n)
	cleanContent := strings.ReplaceAll(content, "\r\n", "\n")

	// Separar por 1 o más líneas en blanco (\n\n o más)
	multiBlank := regexp.MustCompile(`\n\s*\n+`)
	paragraphs := multiBlank.Split(cleanContent, -1)

	var kept []string
	for _, p := range paragraphs {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}

		key := normalizeParagraph(trimmed)
		
		// Si ya vimos exactamente esta misma prosa, la descartamos
		if seen[key] {
			removedCount++
			removedChars += len(p)
			continue
		}

		// Marcamos como vista y la conservamos
		seen[key] = true
		kept = append(kept, trimmed)
	}

	return strings.Join(kept, "\n\n"), removedCount, removedChars
}