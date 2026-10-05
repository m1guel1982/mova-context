package applyflow

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"

	"mova.local/i18n"
)

// TestMain carga una COPIA del catálogo real config/lang/ (src/applyflow →
// src → raíz) para que las pruebas ejerciten los mismos textos que verá la
// persona, sin tocar lang_active.json. Mismo patrón que mcp/budget/models.
func TestMain(m *testing.M) {
	if _, thisFile, _, ok := goruntime.Caller(0); ok {
		realLang := filepath.Join(filepath.Dir(thisFile), "..", "..", "config", "lang")
		if enData, err := os.ReadFile(filepath.Join(realLang, "en.json")); err == nil {
			if tmp, err := os.MkdirTemp("", "mova-applyflow-i18n-*"); err == nil {
				defer os.RemoveAll(tmp)
				langDir := filepath.Join(tmp, "config", "lang")
				_ = os.MkdirAll(langDir, 0o755)
				_ = os.WriteFile(filepath.Join(langDir, "en.json"), enData, 0o644)
				if es, err := os.ReadFile(filepath.Join(realLang, "es.json")); err == nil {
					_ = os.WriteFile(filepath.Join(langDir, "es.json"), es, 0o644)
				}
				_ = os.WriteFile(filepath.Join(langDir, "lang_active.json"), []byte(`{"lang":"en"}`), 0o644)
				_ = i18n.Init(tmp)
			}
		}
	}
	os.Exit(m.Run())
}
