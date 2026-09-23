package models

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"

	"mova.local/i18n"
)

// TestMain loads a COPY of the real config/lang/ catalog (two
// directories up from this package — src/models -> src -> repo root)
// into a throwaway temp directory before any test runs, forcing the
// active language to English — so this package's tests exercise the
// exact same i18n.T lookups the egress air-gap message uses in
// production (see egress_audit.go), with deterministic assertions,
// without ever touching the real config/lang/lang_active.json. Same
// pattern as diagram/diagram_test.go and budget/i18n_test_main_test.go
// — kept in its own file so it's easy to find.
func TestMain(m *testing.M) {
	_, thisFile, _, ok := goruntime.Caller(0)
	if ok {
		realConfigLang := filepath.Join(filepath.Dir(thisFile), "..", "..", "config", "lang")
		if enData, err := os.ReadFile(filepath.Join(realConfigLang, "en.json")); err == nil {
			tmp, tmpErr := os.MkdirTemp("", "mova-models-i18n-*")
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
