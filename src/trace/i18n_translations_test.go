// i18n_translations_test.go — confirms the PROSE sentences newly wired
// to i18n.T in markdown.go/pdf.go/console.go (see reports.* keys added
// alongside this feature) actually render in Spanish when the active
// language is "es", and in English when it's "en" — not just that
// i18n.T doesn't crash. Table/section headers are deliberately left
// out of scope here (see console.go's own header comment for why they
// stay plain English by design).
package trace

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"

	"mova.local/i18n"
)

// initI18nWithLang loads a COPY of the real config/lang catalog into a
// throwaway temp root with lang forced active, so these tests are
// deterministic regardless of the checkout's own
// config/lang/lang_active.json — same source pattern as
// budget/i18n_test_main_test.go, but parameterized by language and
// callable per-test (i18n.Init can be called repeatedly; see
// i18n/i18n.go's own doc comment) instead of once via TestMain, since
// this file specifically needs to flip between "es" and "en".
func initI18nWithLang(t *testing.T, lang string) {
	t.Helper()
	_, thisFile, _, ok := goruntime.Caller(0)
	if !ok {
		t.Skip("could not locate this test file to find the real config/lang catalog")
	}
	realConfigLang := filepath.Join(filepath.Dir(thisFile), "..", "..", "config", "lang")
	enData, err := os.ReadFile(filepath.Join(realConfigLang, "en.json"))
	if err != nil {
		t.Skipf("could not read the real config/lang/en.json: %v", err)
	}
	esData, err := os.ReadFile(filepath.Join(realConfigLang, "es.json"))
	if err != nil {
		t.Skipf("could not read the real config/lang/es.json: %v", err)
	}

	tmp := t.TempDir()
	langDir := filepath.Join(tmp, "config", "lang")
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(langDir, "en.json"), enData, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(langDir, "es.json"), esData, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(langDir, "lang_active.json"), []byte(`{"lang":"`+lang+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := i18n.Init(tmp); err != nil {
		t.Fatal(err)
	}
}

// minimalData builds the smallest *Data that exercises the
// "no active project.json" branch in all three renderers (the
// simplest, always-present code path to assert translated prose
// against).
func minimalData() *Data {
	return &Data{ExecutionID: "exec-i18n-test", RepoURL: "https://example.com/repo.git"}
}

func TestMarkdownReport_SpanishAndEnglish_TranslatePendingProse(t *testing.T) {
	cases := []struct {
		lang string
		want string
	}{
		{"es", "No hay un `project.json` activo"},
		{"en", "No active `project.json`"},
	}
	for _, c := range cases {
		t.Run(c.lang, func(t *testing.T) {
			initI18nWithLang(t, c.lang)
			out := RenderContextReportMarkdown(minimalData())
			if !strings.Contains(out, c.want) {
				t.Errorf("[%s] expected markdown report to contain %q, got:\n%s", c.lang, c.want, out)
			}
			if !strings.Contains(out, i18n.T("reports.report_disclaimer")) {
				t.Errorf("[%s] expected the translated disclaimer in the markdown report", c.lang)
			}
			if !strings.Contains(out, i18n.T("reports.trace_tagline")) {
				t.Errorf("[%s] expected the translated tagline in the markdown report", c.lang)
			}
		})
	}
}

func TestPDFReport_SpanishAndEnglish_TranslatePendingProse(t *testing.T) {
	cases := []struct {
		lang string
		want string
	}{
		{"es", "No hay un project.json activo"}, // backticks stripped by mdInlineToHTML for PDF/HTML
		{"en", "No active project.json"},
	}
	for _, c := range cases {
		t.Run(c.lang, func(t *testing.T) {
			initI18nWithLang(t, c.lang)
			out := contextReportHTML(minimalData())
			if !strings.Contains(out, c.want) {
				t.Errorf("[%s] expected PDF/HTML report to contain %q, got:\n%s", c.lang, c.want, out)
			}
			if !strings.Contains(out, i18n.T("reports.report_disclaimer")) {
				t.Errorf("[%s] expected the translated disclaimer in the PDF report", c.lang)
			}
			if strings.Contains(out, "**") || strings.Contains(out, "`") {
				t.Errorf("[%s] PDF/HTML output must not contain raw Markdown syntax (mdInlineToHTML should have converted it), got:\n%s", c.lang, out)
			}
		})
	}
}

func TestConsoleOutput_SpanishAndEnglish_TranslatePendingProse(t *testing.T) {
	cases := []struct {
		lang string
		want string
	}{
		{"es", "no se configuró un límite en project.json"},
		{"en", "no limit configured in project.json"},
	}
	for _, c := range cases {
		t.Run(c.lang, func(t *testing.T) {
			initI18nWithLang(t, c.lang)
			out := RenderConsole(minimalData(), nil)
			if !strings.Contains(out, c.want) {
				t.Errorf("[%s] expected local-mode console output to contain %q, got:\n%s", c.lang, c.want, out)
			}
		})
	}
}

func TestConsoleOutput_RemoteMode_SpanishAndEnglish_TranslatePendingProse(t *testing.T) {
	cases := []struct {
		lang string
		want string
	}{
		{"es", "no hay un project.json activo"},
		{"en", "no active project.json"},
	}
	for _, c := range cases {
		t.Run(c.lang, func(t *testing.T) {
			initI18nWithLang(t, c.lang)
			d := minimalData()
			d.IsRemote = true
			out := RenderConsole(d, nil)
			if !strings.Contains(out, c.want) {
				t.Errorf("[%s] expected remote-mode console output to contain %q, got:\n%s", c.lang, c.want, out)
			}
			if !strings.Contains(out, i18n.T("reports.cost_estimates_disclaimer")) {
				t.Errorf("[%s] expected the translated cost-estimates disclaimer in console output", c.lang)
			}
		})
	}
}

// TestReports_TitlesTranslated: los títulos, cabeceras de tabla y etiquetas
// del reporte (md y HTML del PDF) salen en el idioma activo, sin restos del
// otro idioma.
func TestReports_TitlesTranslated(t *testing.T) {
	cases := []struct {
		lang         string
		want, absent []string
	}{
		{"es", []string{"# Reporte de Contexto", "## Composición del contexto", "## Foco", "## Presupuesto", "## Costo de entrada estimado", "Gobernanza de contexto", "| Etapa | Estado |"},
			[]string{"## Focus", "## Budget", "Context composition", "| Stage | Status |", "Estimated input cost"}},
		{"en", []string{"# Context Report", "## Context composition", "## Focus", "## Budget", "## Estimated input cost", "| Stage | Status |"},
			[]string{"Composición", "Presupuesto", "Etapa"}},
	}
	for _, c := range cases {
		t.Run(c.lang, func(t *testing.T) {
			initI18nWithLang(t, c.lang)
			d := minimalData()
			d.Components = []ComponentRow{{Name: "prompt", Tokens: 10}}
			md := RenderContextReportMarkdown(d)
			html := contextReportHTML(d)
			for _, w := range c.want {
				if !strings.Contains(md, w) {
					t.Errorf("md missing %q", w)
				}
			}
			for _, a := range c.absent {
				if strings.Contains(md, a) || strings.Contains(html, a) {
					t.Errorf("unexpected %q", a)
				}
			}
		})
	}
}
