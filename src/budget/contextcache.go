// contextcache.go — Mova's OWN local cache (distinct from the Cache
// Layout Guard in cachelayout.go, which targets a Cloud PROVIDER's
// cache): memoizes the Sanitizer's result for Focus/Memory, keyed by a
// content+config hash, so re-running `mova budget`/a job/the daemon on
// UNCHANGED files with an UNCHANGED sanitize config skips redoing that
// work. Saves wall-clock time only — never tokens or money by itself
// (that's the Sanitizer's and Cache Layout Guard's job respectively);
// this purely avoids repeating already-known work.
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

	// ProjectHash: sha256 de project.json tal como estaba la última vez
	// que este archivo de caché se escribió — ver
	// invalidateOnProjectChange. "" antes de la primera escritura con
	// esta lógica (caches viejas de antes de este campo existir se
	// tratan igual que "cambió", así se auto-sanan en la primera
	// ejecución).
	ProjectHash string `json:"project_hash,omitempty"`
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

// hashText hashes the section text TOGETHER WITH the sanitize.Config
// that will be applied to it. Folding cfg into the hash means any
// change to the config that produced it — e.g. a project.json edit
// that flips a sanitize option on/off — changes the hash immediately,
// so the old entry no longer matches, is treated as stale, gets
// recomputed via apply(), and the new hash+result are written back to
// mova-context-cache.json on the very next run. Without this, a config
// change with unchanged file text would silently keep serving the
// sanitized output produced under the OLD config.
func hashText(s string, cfg sanitize.Config) string {
	h := sha256.New()
	h.Write([]byte(s))
	// Config first controls whether/how sanitize.Apply behaves, so it
	// has to be part of what identifies a cache entry as still valid.
	if cfgBytes, err := json.Marshal(cfg); err == nil {
		h.Write(cfgBytes)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// SanitizeCached is what BuildGatedContext/BuildReport call instead of
// sanitize.Apply directly when core.ContextCacheEnabled(cfg) is true:
// same net effect on sections (Focus/Memory sanitized in place, Stats
// returned), but a cache hit skips re-running the Sanitizer entirely
// for whichever piece hasn't changed — in TEXT or in CONFIG — since
// the last run.
// Guarded end-to-end with withFileLock: two concurrent runs of the same
// project (e.g. a job + a chat, or two overlapping HTTP requests) must
// not read the same cache snapshot, both recompute, and have the second
// write clobber the first — that would only cost a re-sanitize, but
// under heavier concurrency it also risks a torn read of the file mid
// os.WriteFile from another goroutine.
func SanitizeCached(root, project string, sections *core.ContextSections, cfg sanitize.Config, useCache bool) sanitize.Stats {
	if !cfg.Enabled || sections == nil {
		return sanitize.Stats{}
	}
	if !useCache {
		return sanitize.Apply(sections, cfg)
	}

	path := ContextCachePath(root, project)
	var total sanitize.Stats
	_ = withFileLock(path, func() error {
		cacheFile := loadContextCacheFile(path)
		// Actualización en caliente: si project.json cambió desde la
		// última vez que se escribió mova-context-cache.json — se
		// agregó/sacó algo de "focus", se tocó "sanitize"/"budget"/
		// cualquier otro campo — cada entrada vieja se descarta ANTES
		// de comparar hashes de texto individuales. Sin esto, un campo
		// de project.json que no cambia el TEXTO de Focus/Memory (p.
		// ej. reordenar "focus" sin agregar archivos nuevos) podría, en
		// teoría, seguir sirviendo una entrada que ya no corresponde a
		// la configuración vigente. Cubre lo mismo que
		// cli/chat_helpers.go's refreshProjectContext cubre para una
		// sesión de `mova chat` YA abierta, pero acá para cualquier
		// entrada punto (mova run/jobs/HTTP/MCP), sin necesitar una
		// sesión viva.
		changed := invalidateOnProjectChange(root, project, &cacheFile)

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
		return nil
	})
	return total
}

// invalidateOnProjectChange descarta TODAS las entradas cacheadas la
// primera vez que detecta que project.json cambió desde la última
// escritura de mova-context-cache.json (comparando su hash sha256
// completo, ver core.ProjectJSONFingerprint) — no espera a que el
// TEXTO de Focus/Memory difiera, invalida apenas cambia la fuente de
// verdad. ok=false (proyecto sin project.json propio — DB adapter,
// grupo multiagente) es un no-op: no hay nada que vigilar, se sigue
// confiando en el hash de texto+config de siempre (hashText). Devuelve
// true cuando invalidó algo — eso fuerza una escritura inmediata del
// archivo aunque cachedPiece más abajo termine teniendo un hit para
// AMBAS piezas (Focus y Memory), porque el fingerprint en sí ya quedó
// desactualizado y tiene que persistirse.
func invalidateOnProjectChange(root, project string, f *contextCacheFile) bool {
	_, hash, ok := core.ProjectJSONFingerprint(root, project)
	if !ok || hash == f.ProjectHash {
		return false
	}
	f.Entries = map[string]contextCacheEntry{}
	f.ProjectHash = hash
	return true
}

// cachedPiece applies apply() to *text, but only if the hash of
// (*text, cfg) differs from what's cached under key — a hit reuses the
// cached sanitized text and stats instead of recomputing them. Because
// cfg is folded into the hash, a config change alone (no text change)
// is enough to force a recompute and a fresh write to the cache file.
func cachedPiece(text *string, key string, entries map[string]contextCacheEntry, cfg sanitize.Config, apply func(string, sanitize.Config) (string, sanitize.Stats), changed *bool) sanitize.Stats {
	hash := hashText(*text, cfg)
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
