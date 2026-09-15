// Package i18n is Mova Context's centralized internationalization
// module: one place that knows how to read config/lang/{es,en}.json
// and config/lang/lang_active.json, and how to look up a translation
// key from ANY door (CLI, MCP, HTTP API, Chat, Chat UI) — see this
// package's header comment in i18n_reload.go for how "lang" changing
// at runtime propagates everywhere without a restart.
//
// Usage:
//
//	i18n.Init(root)                                  // once, at startup
//	i18n.T("cli.messages.done")                       // plain lookup
//	i18n.T("reports.tokens_line", map[string]any{     // with placeholders
//	    "count": 1500, "encoding": "cl100k_base",
//	})
package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// catalog is one language's fully-loaded, flattened key->string map
// ("cli.flags.repo" -> "Local path or remote git URL to analyze") -
// flattened once at load time so T() is a single map lookup, not a
// nested-map walk on every call.
type catalog map[string]string

// translator holds every piece of mutable state this package needs,
// guarded by one RWMutex so concurrent requests (HTTP, MCP, multiple
// chat sessions) can all call T() safely while a reload is in flight.
type translator struct {
	mu          sync.RWMutex
	root        string
	active      string             // e.g. "es"
	catalogs    map[string]catalog // lang code -> flattened catalog
	activeMTime int64              // lang_active.json's mtime, for hot-reload polling
}

var t = &translator{catalogs: map[string]catalog{}}

// fallbackLang is what every door falls back to when the active
// language is missing a key, or its file can't be read at all - see
// the package spec's own requirement: "los mensajes que existen
// actualmente en el sistema... se encuentran todos en ingles".
const fallbackLang = "en"

// Init loads config/lang/ under root once, resolves the active
// language, and starts the hot-reload poller (see i18n_reload.go).
// Safe to call more than once (e.g. in tests) - later calls simply
// re-resolve root and reload.
func Init(root string) error {
	t.mu.Lock()
	t.root = root
	t.activeMTime = 0 // force a fresh reload even if this root's lang_active.json happens to share an mtime with a previous root (e.g. under tests)
	t.mu.Unlock()

	if err := loadCatalog(root, fallbackLang); err != nil {
		return fmt.Errorf("i18n: could not load fallback language %q: %w", fallbackLang, err)
	}
	reloadActiveLanguage() // best-effort; errors here just keep the previous/fallback state
	startHotReload()
	return nil
}

// T looks up key (dot-separated, matching the JSON's nesting, e.g.
// "cli.messages.done") in the active language, falling back to
// English, and finally to the literal key itself so a missing
// translation is visibly wrong rather than a crash or a blank string.
// An optional map[string]any replaces "{{name}}" placeholders.
func T(key string, args ...map[string]any) string {
	t.mu.RLock()
	active := t.active
	activeCat := t.catalogs[active]
	fallbackCat := t.catalogs[fallbackLang]
	t.mu.RUnlock()

	value, ok := activeCat[key]
	if !ok {
		value, ok = fallbackCat[key]
	}
	if !ok {
		return key
	}
	if len(args) > 0 {
		value = interpolate(value, args[0])
	}
	return value
}

// ActiveLanguage returns the currently active language code (e.g.
// "es") - mostly useful for logging/diagnostics.
func ActiveLanguage() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.active
}

func interpolate(s string, args map[string]any) string {
	for k, v := range args {
		s = strings.ReplaceAll(s, "{{"+k+"}}", fmt.Sprintf("%v", v))
	}
	return s
}

// loadCatalog reads config/lang/<lang>.json under root, flattens it,
// and stores it - called for the fallback language once at Init, and
// for the active language on every (re)load.
func loadCatalog(root, lang string) error {
	path := filepath.Join(root, "config", "lang", lang+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid JSON in %s: %w", path, err)
	}
	flat := catalog{}
	flatten("", raw, flat)

	t.mu.Lock()
	t.catalogs[lang] = flat
	t.mu.Unlock()
	return nil
}

// flatten turns a nested JSON object into dot-separated keys,
// skipping the "_comment" convention used throughout config/lang/*.json
// (see en.json/es.json's own header comment).
func flatten(prefix string, node map[string]any, out catalog) {
	for k, v := range node {
		if k == "_comment" {
			continue
		}
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch val := v.(type) {
		case map[string]any:
			flatten(key, val, out)
		case string:
			out[key] = val
		default:
			out[key] = fmt.Sprintf("%v", val)
		}
	}
}
