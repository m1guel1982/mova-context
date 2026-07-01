// file_helpers.go — shared file utilities (UTF-8, scoring, excerpts).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if isUTF8(data) {
		return string(data)
	}
	r := make([]rune, len(data))
	for i, b := range data {
		r[i] = rune(b)
	}
	return string(r)
}

func isUTF8(data []byte) bool {
	for i := 0; i < len(data); {
		b := data[i]
		var sz int
		switch {
		case b < 0x80:
			sz = 1
		case b < 0xC2:
			return false
		case b < 0xE0:
			sz = 2
		case b < 0xF0:
			sz = 3
		case b < 0xF5:
			sz = 4
		default:
			return false
		}
		if i+sz > len(data) {
			return false
		}
		for j := 1; j < sz; j++ {
			if data[i+j]&0xC0 != 0x80 {
				return false
			}
		}
		i += sz
	}
	return true
}

func parseEntryDate(entry string) time.Time {
	lines := strings.SplitN(entry, "\n", 2)
	if len(lines) == 0 {
		return time.Time{}
	}
	line := strings.TrimPrefix(strings.TrimSpace(lines[0]), "## ")
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return time.Time{}
	}
	t, _ := time.Parse("2006-01-02", parts[0])
	return t
}

func extractLangDomain(path, root, kindPlural string) (lang, domain string) {
	rel := strings.TrimPrefix(path, filepath.Join(root, kindPlural)+string(os.PathSeparator))
	parts := strings.Split(rel, string(os.PathSeparator))
	if len(parts) >= 3 {
		return parts[1], parts[0] // domain/lang/file.md
	}
	if len(parts) >= 2 {
		return "", parts[0] // domain/file.md
	}
	return "", ""
}

func excerpt(content, q string) string {
	idx := strings.Index(content, q)
	if idx < 0 {
		if len(content) > 100 {
			return content[:100] + "..."
		}
		return content
	}
	s := idx - 30
	if s < 0 {
		s = 0
	}
	e := idx + len(q) + 70
	if e > len(content) {
		e = len(content)
	}
	return "..." + content[s:e] + "..."
}

func score(name, content, q string) float64 {
	s := 0.0
	if strings.Contains(name, q) {
		s += 0.6
	}
	c := strings.Count(content, q)
	if c > 0 {
		s += 0.4 * float64(c) / float64(c+5)
	}
	return s
}

// DeleteMemory removes memory entries matching the request.
// Returns the number of entries deleted.
// File mode: rewrites memory.md and/or removes archive files.
func (a *fileAdapter) DeleteMemory(project string, req MemoryDeleteRequest) (int, error) {
	memPath := filepath.Join(a.root, "projects", project, "memory.md")
	archDir := filepath.Join(a.root, "projects", project, "memory-archive")

	// Delete only archived files, keep memory.md
	if req.Archived || req.KeepActive {
		entries, err := os.ReadDir(archDir)
		if err != nil {
			return 0, nil // no archive dir = nothing to do
		}
		count := 0
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				if err := os.Remove(filepath.Join(archDir, e.Name())); err == nil {
					count++
				}
			}
		}
		return count, nil
	}

	// Delete everything (active + archives)
	if req.All {
		count := 0
		if err := os.WriteFile(memPath, []byte(""), 0644); err == nil {
			count++
		}
		entries, _ := os.ReadDir(archDir)
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				if err := os.Remove(filepath.Join(archDir, e.Name())); err == nil {
					count++
				}
			}
		}
		return count, nil
	}

	// Delete a specific day or date range from memory.md
	if req.Date != "" || (req.From != "" && req.To != "") {
		content := readFile(memPath)
		entries := strings.Split(content, "\n\n---\n\n")
		var keep []string
		deleted := 0
		for _, e := range entries {
			e = strings.TrimSpace(e)
			if e == "" {
				continue
			}
			d := parseEntryDate(e)
			remove := false
			if !d.IsZero() {
				ds := d.Format("2006-01-02")
				if req.Date != "" && ds == req.Date {
					remove = true
				}
				if req.From != "" && req.To != "" && ds >= req.From && ds <= req.To {
					remove = true
				}
			}
			if remove {
				deleted++
			} else {
				keep = append(keep, e)
			}
		}
		var sb strings.Builder
		for i, e := range keep {
			sb.WriteString(e)
			if i < len(keep)-1 {
				sb.WriteString("\n\n---\n\n")
			}
		}
		os.WriteFile(memPath, []byte(sb.String()), 0644)
		return deleted, nil
	}

	return 0, fmt.Errorf("no delete criteria specified")
}
