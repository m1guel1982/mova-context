// file_memory.go — memory.md del adaptador de archivos: lectura, anexado
// (más reciente primero), lectura con archivos y archivado mensual.
// Movido desde file_adapter.go (≤300 líneas por archivo) con tres
// correcciones: respeta la ruta configurada también al archivar (antes
// archivaba SIEMPRE projects/<p>/memory.md), escribe con bloqueo + de
// forma atómica, y informa de una ruta inutilizable en vez de crear un
// archivo fantasma.
package core

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

const memorySep = "\n\n---\n\n"

func (a *fileAdapter) GetMemory(project string) (string, error) {
	return readFile(MemoryPath(a.root, project)), nil
}

func (a *fileAdapter) GetMemoryAll(project string) (string, error) {
	active, _ := a.GetMemory(project)
	archDir := memoryArchiveDir(MemoryPath(a.root, project))
	entries, err := os.ReadDir(archDir)
	if err != nil {
		return active, nil
	}
	parts := []string{}
	if active != "" {
		parts = append(parts, active)
	}
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			if c := readFile(filepath.Join(archDir, e.Name())); c != "" {
				parts = append(parts, "<!-- archive: "+e.Name()+" -->\n"+c)
			}
		}
	}
	return strings.Join(parts, memorySep), nil
}

func (a *fileAdapter) AppendMemory(project, entry string) error {
	path, err := ResolveMemoryTarget(a.root, project)
	if err != nil {
		return err
	}
	return withMemoryLock(path, func() error {
		updated := strings.TrimSpace(entry) + memorySep + readFile(path)
		return writeFileAtomic(path, []byte(updated))
	})
}

func (a *fileAdapter) ArchiveMemory(project string, keepDays int) error {
	memPath := MemoryPath(a.root, project)
	archDir := memoryArchiveDir(memPath)
	return withMemoryLock(memPath, func() error {
		content := readFile(memPath)
		if content == "" {
			return nil
		}
		cutoff := time.Now().AddDate(0, 0, -keepDays)
		var keep []string
		byMonth := map[string][]string{}
		for _, e := range strings.Split(content, memorySep) {
			e = strings.TrimSpace(e)
			if e == "" {
				continue
			}
			d := parseEntryDate(e)
			if d.IsZero() || d.After(cutoff) {
				keep = append(keep, e)
			} else {
				byMonth[d.Format("2006-01")] = append(byMonth[d.Format("2006-01")], e)
			}
		}
		if len(byMonth) == 0 {
			return nil
		}
		if err := os.MkdirAll(archDir, 0o755); err != nil {
			return err
		}
		for month, items := range byMonth {
			path := filepath.Join(archDir, month+".md")
			out := strings.Join(items, memorySep) + memorySep + readFile(path)
			if err := writeFileAtomic(path, []byte(out)); err != nil {
				return err
			}
		}
		return writeFileAtomic(memPath, []byte(strings.Join(keep, memorySep)))
	})
}
