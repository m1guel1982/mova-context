package budget

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"

	"mova.local/i18n"
)

// TestMain loads a COPY of the real config/lang/ catalog (two
// directories up from this package — src/budget -> src -> repo root)
// into a throwaway temp directory before any test runs, forcing the
// active language to English — so this package's tests exercise the
// exact same i18n.T lookups mova-budget-report.md makes in production
// (see report.go/report_pipeline.go) with deterministic, English
// assertions, regardless of what config/lang/lang_active.json happens
// to be set to in this checkout, and without ever touching the real
// one. Exact same pattern as diagram/diagram_test.go's TestMain —
// kept in its own file so it's easy to find. Best-effort: if the
// catalog can't be found, tests still run — they'll just see literal
// untranslated keys instead of real strings, same as before this
// TestMain existed.
func TestMain(m *testing.M) {
	_, thisFile, _, ok := goruntime.Caller(0)
	if ok {
		realConfigLang := filepath.Join(filepath.Dir(thisFile), "..", "..", "config", "lang")
		if enData, err := os.ReadFile(filepath.Join(realConfigLang, "en.json")); err == nil {
			tmp, tmpErr := os.MkdirTemp("", "mova-budget-i18n-*")
			if tmpErr == nil {
				defer os.RemoveAll(tmp)
				langDir := filepath.Join(tmp, "config", "lang")
				_ = os.MkdirAll(langDir, 0o755)
				_ = os.WriteFile(filepath.Join(langDir, "en.json"), enData, 0o644)
				if esData, err := os.ReadFile(filepath.Join(realConfigLang, "es.json")); err == nil {
					_ = os.WriteFile(filepath.Join(langDir, "es.json"), esData, 0o644)
				}
				_ = os.WriteFile(filepath.Join(langDir, "lang_active.json"), []byte(`{"lang":"en"}`), 0o644)
				_ = i18n.Init(tmp)
			}
		}
	}
	os.Exit(m.Run())
}
