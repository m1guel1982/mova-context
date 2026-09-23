// i18n_reload.go — hot-reload for config/lang/*.json, in TWO
// independent dimensions, both on the same 1s poll:
//
//  1. WHICH language is active — config/lang/lang_active.json's own
//     "lang" value (reloadActiveLanguage, unchanged from before).
//  2. WHAT a given language's messages actually SAY — each loaded
//     catalog's own <lang>.json file content (reloadCatalogFileIfChanged,
//     tracked per-language in translator.fileMTimes). This is what
//     lets a person or an admin EDIT a message's text — e.g.
//     config/lang/es.json's "reports.egress_airgap_message", the
//     egress air-gap security directive documented in
//     PROJECT_JSON.md § egress_audit — and have every door (CLI, MCP,
//     HTTP, Chat) pick up the new wording within ~1s, no restart,
//     independent of whether the active LANGUAGE also changed. Only
//     the active language and the fallback ("en") are watched this
//     way — an unloaded, inactive language's file is only ever read
//     when something actually switches to it (reloadActiveLanguage's
//     existing needsLoad path), so this poll never grows unbounded as
//     more languages are added to config/lang/.
//
// Design note (honesty, same convention as relevance.go/patcher.go's
// own headers): this uses a lightweight, pure-stdlib POLLING watcher
// (os.Stat every second) instead of a filesystem-event library like
// fsnotify. That is a deliberate trade-off, not an oversight - it
// keeps this package dependency-free (no new entry in go.mod/go.sum,
// no platform-specific watcher backend to vendor), at the cost of up
// to ~1 second of latency between editing a lang file and every door
// picking up the change - imperceptible for a person switching
// languages or tuning a message, and the requirement itself only asks
// for "sin requerir el reinicio del proceso" / "tomar los cambios en
// caliente", not sub-second latency.
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
			reloadCatalogFileIfChanged(fallbackLang)
			reloadCatalogFileIfChanged(ActiveLanguage())
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

// reloadCatalogFileIfChanged re-reads config/lang/<lang>.json's
// CONTENT if its own mtime changed since it was last loaded —
// independent of reloadActiveLanguage's lang_active.json check above.
// lang == "" (ActiveLanguage() before anything ever loaded) is a
// harmless no-op. Best-effort like every other step in this file: a
// stat/read/parse failure just keeps the previous, already-loaded
// catalog in place rather than ever taking translations offline over
// one bad edit (e.g. someone saving es.json mid-edit with invalid
// JSON — the next successful save 1s later picks it up normally).
func reloadCatalogFileIfChanged(lang string) {
	if lang == "" {
		return
	}
	t.mu.RLock()
	root := t.root
	lastMTime, known := t.fileMTimes[lang]
	t.mu.RUnlock()
	if root == "" {
		return
	}

	path := filepath.Join(root, "config", "lang", lang+".json")
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	mtime := info.ModTime().UnixNano()
	if known && mtime == lastMTime {
		return // unchanged since last load - nothing to do
	}
	_ = loadCatalog(root, lang) // best-effort; on error, previous catalog for lang stays active
}
