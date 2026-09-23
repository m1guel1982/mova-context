package budget

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"

	"mova.local/core"
	"mova.local/i18n"
)

// buildBudgetFixture mirrors core's fixture builder (kept local to avoid
// an import cycle / cross-package test dependency) — a minimal project
// with agents+skills+prompt+memory, plus an optional focus target.
func buildBudgetFixture(t *testing.T, withFocus bool) (root, projectName string) {
	t.Helper()
	root = t.TempDir()
	write := func(rel, content string) {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("agents/base/dev.md", "# Dev agent\nBe helpful and concise.\n")
	write("skills/base/kiss.md", "# KISS\nKeep it simple, avoid over-engineering.\n")
	write("prompts/base/greet.md", "# Greet\nSay hello to {{PROJECT}}.\n")
	write("projects/fixture/memory.md", "## 2024-01-01\nFirst session notes.\n")
	write("examples/repo/README.md", "# Example repo\nThis is the whole repository content, unfiltered.\n")
	write("examples/repo/notes.txt", repeatLine("Some additional plain text file in the repo, with enough content to clearly outweigh the tiny README when focus is not used to filter it out.\n", 40))

	focusJSON := ""
	if withFocus {
		focusJSON = `,"focus": ["README.md"]`
	}
	write("projects/fixture/project.json", `{
		"project": "fixture",
		"repo": "examples/repo",
		"lang": "en",
		"default_task": "say-hi",
		"agents": {"domain": "base", "use": ["dev"]},
		"skills": {"domain": "base", "use": ["kiss"]},
		"tasks": {"say-hi": {"prompt": "greet"}}`+focusJSON+`
	}`)
	writePrices(t, root, samplePrices)
	return root, "fixture"
}

func TestBuildReport_ComponentBreakdownSumsToTotal(t *testing.T) {
	root, projectName := buildBudgetFixture(t, false)
	adapter := core.NewFileAdapter(root)

	report, err := BuildReport(adapter, root, projectName, "", false)
	if err != nil {
		t.Fatalf("BuildReport: %v", err)
	}
	if len(report.Components) == 0 {
		t.Fatal("expected at least one component in the breakdown")
	}

	sum := 0
	for _, c := range report.Components {
		sum += c.Tokens
	}
	if sum != report.TotalTokens {
		t.Fatalf("component tokens must sum to TotalTokens: sum=%d, TotalTokens=%d", sum, report.TotalTokens)
	}
	if report.TotalTokens <= 0 {
		t.Fatal("expected a positive total token count for a non-trivial fixture")
	}
	if len(report.TotalCosts) != 2 {
		t.Fatalf("expected 2 provider/model cost rows, got %d", len(report.TotalCosts))
	}
}

func TestBuildReport_WithoutFocusFlagHasNoComparison(t *testing.T) {
	root, projectName := buildBudgetFixture(t, false)
	adapter := core.NewFileAdapter(root)

	report, err := BuildReport(adapter, root, projectName, "", false)
	if err != nil {
		t.Fatalf("BuildReport: %v", err)
	}
	if report.Focus != nil {
		t.Fatal("expected Focus to be nil when withFocusComparison=false")
	}
}

func TestBuildReport_FocusComparisonRequiresFocusConfigured(t *testing.T) {
	root, projectName := buildBudgetFixture(t, false) // no "focus" in project.json
	adapter := core.NewFileAdapter(root)

	if _, err := BuildReport(adapter, root, projectName, "", true); err == nil {
		t.Fatal("expected an error asking for --focus on a project with no \"focus\" configured, got nil")
	}
}

func TestBuildReport_FocusComparisonSavingsIsSensible(t *testing.T) {
	root, projectName := buildBudgetFixture(t, true) // focus: ["README.md"]
	adapter := core.NewFileAdapter(root)

	report, err := BuildReport(adapter, root, projectName, "", true)
	if err != nil {
		t.Fatalf("BuildReport: %v", err)
	}
	if report.Focus == nil {
		t.Fatal("expected a focus comparison")
	}
	if report.Focus.TokensWithoutFocus <= report.Focus.TokensWithFocus {
		t.Fatalf("expected the full repo (README.md + notes.txt) to have more tokens than focus (README.md only): without=%d, with=%d",
			report.Focus.TokensWithoutFocus, report.Focus.TokensWithFocus)
	}
	if report.Focus.SavingsPercent <= 0 {
		t.Fatalf("expected positive savings percent, got %v", report.Focus.SavingsPercent)
	}
}

func TestBuildReport_MissingProject(t *testing.T) {
	root, _ := buildBudgetFixture(t, false)
	adapter := core.NewFileAdapter(root)
	if _, err := BuildReport(adapter, root, "no-existe", "", false); err == nil {
		t.Fatal("expected an error for a nonexistent project, got nil")
	}
}

func TestBuildReport_MissingPricesFile(t *testing.T) {
	root, projectName := buildBudgetFixture(t, false)
	os.Remove(PricesPath(root)) // simulate "prices.json inexistente"
	adapter := core.NewFileAdapter(root)
	if _, err := BuildReport(adapter, root, projectName, "", false); err == nil {
		t.Fatal("expected an error when config/prices.json doesn't exist, got nil")
	}
}

func TestRenderMarkdown_IsEnglishAndContainsDisclaimer(t *testing.T) {
	root, projectName := buildBudgetFixture(t, false)
	adapter := core.NewFileAdapter(root)
	report, err := BuildReport(adapter, root, projectName, "", false)
	if err != nil {
		t.Fatalf("BuildReport: %v", err)
	}

	md := RenderMarkdown(report)
	for _, want := range []string{
		"# Mova Budget Report", "## Tokenization", "## Token & Cost Breakdown",
		"tiktoken-go", "estimate", "## Important", "TOTAL",
	} {
		if !contains(md, want) {
			t.Errorf("expected report to contain %q", want)
		}
	}
}

// TestRenderMarkdown_FollowsActiveLanguage is the multi-language
// contract for mova-budget-report.md: switching config/lang's active
// language must switch the ENTIRE report, not just isolated strings —
// see config/lang/{es,en}.json's "budget" section and
// PROJECT_JSON.md/ARTIFACTS.md for the artifacts this applies to.
func TestRenderMarkdown_FollowsActiveLanguage(t *testing.T) {
	root, projectName := buildBudgetFixture(t, false)
	adapter := core.NewFileAdapter(root)
	report, err := BuildReport(adapter, root, projectName, "", false)
	if err != nil {
		t.Fatalf("BuildReport: %v", err)
	}

	// TestMain forced "en" for every other test in this package — flip
	// to "es" via an isolated throwaway catalog copy, render, then
	// restore, so this test doesn't leak language state into whichever
	// test runs after it.
	switchActiveLanguageForTest(t, "es")
	md := RenderMarkdown(report)
	switchActiveLanguageForTest(t, "en")

	for _, want := range []string{
		"# Reporte de Presupuesto de Mova", "## Tokenización", "## Desglose de Tokens y Costo",
		"tiktoken-go", "## Importante", "TOTAL",
	} {
		if !contains(md, want) {
			t.Errorf("expected Spanish report to contain %q, got:\n%s", want, md)
		}
	}
	if contains(md, "# Mova Budget Report") {
		t.Errorf("expected the English title to be GONE once lang=es, got:\n%s", md)
	}
}

// switchActiveLanguageForTest points the i18n package at a FRESH,
// throwaway copy of the real config/lang/ catalog with lang_active.json
// set to lang — same safety rule as TestMain (i18n_test_main_test.go):
// never touch the actual checkout's config/lang/lang_active.json,
// since its hot-reload poller runs as a background goroutine that
// would otherwise race with, and potentially outlive, this one test.
func switchActiveLanguageForTest(t *testing.T, lang string) {
	t.Helper()
	repoRoot := realRepoRootForTest(t)
	if repoRoot == "" {
		t.Skip("real config/lang not found from this checkout — skipping")
	}
	enData, err := os.ReadFile(filepath.Join(repoRoot, "config", "lang", "en.json"))
	if err != nil {
		t.Skip("could not read config/lang/en.json — skipping")
	}
	esData, err := os.ReadFile(filepath.Join(repoRoot, "config", "lang", "es.json"))
	if err != nil {
		t.Skip("could not read config/lang/es.json — skipping")
	}

	tmp := t.TempDir()
	langDir := filepath.Join(tmp, "config", "lang")
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite := func(name string, data []byte) {
		if err := os.WriteFile(filepath.Join(langDir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("en.json", enData)
	mustWrite("es.json", esData)
	mustWrite("lang_active.json", []byte(`{"lang":"`+lang+`"}`))

	if err := i18n.Init(tmp); err != nil {
		t.Fatalf("i18n.Init: %v", err)
	}
}

func realRepoRootForTest(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := goruntime.Caller(0)
	if !ok {
		return ""
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	if _, err := os.Stat(filepath.Join(root, "config", "lang", "es.json")); err != nil {
		return ""
	}
	return root
}

func TestWriteReport_WritesToConfiguredPath(t *testing.T) {
	root, projectName := buildBudgetFixture(t, false)
	adapter := core.NewFileAdapter(root)
	report, err := BuildReport(adapter, root, projectName, "", false)
	if err != nil {
		t.Fatalf("BuildReport: %v", err)
	}
	proj, err := adapter.GetProject(projectName)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	path, err := WriteReport(root, projectName, proj, report)
	if err != nil {
		t.Fatalf("WriteReport: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected report file to exist at %s: %v", path, err)
	}
	if !contains(string(data), "Mova Budget Report") {
		t.Error("written report file doesn't look like a budget report")
	}
}

func repeatLine(line string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += line
	}
	return out
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
