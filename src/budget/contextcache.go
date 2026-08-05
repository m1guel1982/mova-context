// contextcache.go — Mova's OWN local cache (distinct from the Cache
// Layout Guard in cachelayout.go, which targets a Cloud PROVIDER's
// cache): memoizes the Sanitizer's result for Focus/Memory, keyed by a
// content hash, so re-running `mova budget`/a job/the daemon on
// UNCHANGED files skips redoing that work. Saves wall-clock time only
// — never tokens or money by itself (that's the Sanitizer's and Cache
// Layout Guard's job respectively); this purely avoids repeating
// already-known work.
//
// State lives in mova-context-cache.json, next to mova-spend.json and
// mova-token-history.json — same "small, single-purpose file, never
// project content beyond what's already being sent to a model anyway"
// convention as those two.
package budget

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"mova.local/core"
	"mova.local/sanitize"
)

type contextCacheEntry struct {
	Hash          string         `json:"hash"`
	SanitizedText string         `json:"sanitized_text"`
	Stats         sanitize.Stats `json:"stats"`
}

type contextCacheFile struct {
	Entries map[string]contextCacheEntry `json:"entries"`
}

// ContextCachePath resolves mova-context-cache.json for a project.
func ContextCachePath(root, project string) string {
	return filepath.Join(root, "projects", project, "mova-context-cache.json")
}

func loadContextCacheFile(path string) contextCacheFile {
	data, err := os.ReadFile(path)
	if err != nil {
		return contextCacheFile{Entries: map[string]contextCacheEntry{}}
	}
	var f contextCacheFile
	if err := json.Unmarshal(data, &f); err != nil || f.Entries == nil {
		return contextCacheFile{Entries: map[string]contextCacheEntry{}}
	}
	return f
}

func saveContextCacheFile(path string, f contextCacheFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func hashText(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// SanitizeCached is what BuildGatedContext/BuildReport call instead of
// sanitize.Apply directly when core.ContextCacheEnabled(cfg) is true:
// same net effect on sections (Focus/Memory sanitized in place, Stats
// returned), but a cache hit skips re-running the Sanitizer entirely
// for whichever piece hasn't changed since the last run.
func SanitizeCached(root, project string, sections *core.ContextSections, cfg sanitize.Config, useCache bool) sanitize.Stats {
	if !cfg.Enabled || sections == nil {
		return sanitize.Stats{}
	}
	if !useCache {
		return sanitize.Apply(sections, cfg)
	}

	path := ContextCachePath(root, project)
	cacheFile := loadContextCacheFile(path)
	changed := false
	var total sanitize.Stats

	if sections.Focus != "" {
		stats := cachedPiece(&sections.Focus, "focus", cacheFile.Entries, cfg, sanitize.ApplyFocus, &changed)
		total.LinesRemoved += stats.LinesRemoved
		total.BlankRemoved += stats.BlankRemoved
		total.CommentsRemoved += stats.CommentsRemoved
		total.CharsRemoved += stats.CharsRemoved
	}
	if sections.Memory != "" {
		stats := cachedPiece(&sections.Memory, "memory", cacheFile.Entries, cfg, sanitize.ApplyMemory, &changed)
		total.LinesRemoved += stats.LinesRemoved
		total.BlankRemoved += stats.BlankRemoved
		total.CommentsRemoved += stats.CommentsRemoved
		total.CharsRemoved += stats.CharsRemoved
	}

	if changed {
		_ = saveContextCacheFile(path, cacheFile) // a failed write never blocks the run — same rule every other Token Firewall state file follows
	}
	return total
}

// cachedPiece applies apply() to *text, but only if its hash differs
// from what's cached under key — a hit reuses the cached sanitized
// text and stats instead of recomputing them.
func cachedPiece(text *string, key string, entries map[string]contextCacheEntry, cfg sanitize.Config, apply func(string, sanitize.Config) (string, sanitize.Stats), changed *bool) sanitize.Stats {
	hash := hashText(*text)
	if entry, ok := entries[key]; ok && entry.Hash == hash {
		*text = entry.SanitizedText
		return entry.Stats
	}
	cleaned, stats := apply(*text, cfg)
	entries[key] = contextCacheEntry{Hash: hash, SanitizedText: cleaned, Stats: stats}
	*text = cleaned
	*changed = true
	return stats
}
