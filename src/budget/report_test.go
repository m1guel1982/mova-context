package budget

import (
	"os"
	"path/filepath"
	"testing"

	"mova.local/core"
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

func TestWriteReport_WritesToConfiguredPath(t *testing.T) {
	root, projectName := buildBudgetFixture(t, false)
	adapter := core.NewFileAdapter(root)
	report, err := BuildReport(adapter, root, projectName, "", false)
	if err != nil {
		t.Fatalf("BuildReport: %v", err)
	}
	prices, err := LoadPrices(root)
	if err != nil {
		t.Fatalf("LoadPrices: %v", err)
	}
	path, err := WriteReport(root, prices, report)
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
