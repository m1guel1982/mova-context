// i18n_reload.go — hot-reload for config/lang/lang_active.json.
//
// Design note (honesty, same convention as relevance.go/patcher.go's
// own headers): this uses a lightweight, pure-stdlib POLLING watcher
// (os.Stat every second) instead of a filesystem-event library like
// fsnotify. That is a deliberate trade-off, not an oversight - it
// keeps this package dependency-free (no new entry in go.mod/go.sum,
// no platform-specific watcher backend to vendor), at the cost of up
// to ~1 second of latency between editing lang_active.json and every
// door picking up the change - imperceptible for a person switching
// languages, and the requirement itself only asks for "sin requerir
// el reinicio del proceso", not sub-second latency.
package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// pollInterval is how often lang_active.json's mtime is checked.
const pollInterval = 1 * time.Second

type langActiveFile struct {
	Lang string `json:"lang"`
}

// startHotReload launches the background poller exactly once per
// process (Init may be called more than once in tests; the ticker
// itself is cheap enough that a second one is harmless, but a real
// singleton guard keeps logs/behavior predictable).
var hotReloadStarted bool

func startHotReload() {
	t.mu.Lock()
	already := hotReloadStarted
	hotReloadStarted = true
	t.mu.Unlock()
	if already {
		return
	}
	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for range ticker.C {
			reloadActiveLanguage()
		}
	}()
}

// reloadActiveLanguage re-reads lang_active.json ONLY if its mtime
// changed since the last check, and, if the named language differs
// from what's currently active (or hasn't been loaded yet), (re)loads
// that language's catalog. Every step is best-effort: a missing or
// invalid lang_active.json, or an unreadable/invalid <lang>.json,
// silently keeps the previous state (falling back to English was
// already guaranteed at Init) rather than ever taking the whole
// system's translations offline over one bad edit.
func reloadActiveLanguage() {
	t.mu.RLock()
	root := t.root
	lastMTime := t.activeMTime
	t.mu.RUnlock()
	if root == "" {
		return
	}

	path := filepath.Join(root, "config", "lang", "lang_active.json")
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	mtime := info.ModTime().UnixNano()
	if mtime == lastMTime {
		return // unchanged since last check - nothing to do
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg langActiveFile
	if err := json.Unmarshal(data, &cfg); err != nil || cfg.Lang == "" {
		return
	}

	t.mu.Lock()
	t.activeMTime = mtime
	needsLoad := t.catalogs[cfg.Lang] == nil
	t.mu.Unlock()

	if needsLoad {
		if err := loadCatalog(root, cfg.Lang); err != nil {
			// Could not read/parse the requested language's own file -
			// leave the previous active language in place (which itself
			// falls back to English on any missing key) rather than
			// switching to a language with no catalog at all.
			return
		}
	}

	t.mu.Lock()
	t.active = cfg.Lang
	t.mu.Unlock()
}
