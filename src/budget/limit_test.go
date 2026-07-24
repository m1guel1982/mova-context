package budget

import (
	"strings"
	"testing"

	"mova.local/core"
)

func TestCheckLimit_NoConfigMeansNoLimit(t *testing.T) {
	proj := &core.Project{}
	task := &core.Task{}
	maxTokens, over := CheckLimit(proj, task, 999999)
	if maxTokens != 0 || over {
		t.Fatalf("expected no limit and over=false, got maxTokens=%d over=%v", maxTokens, over)
	}
}

func TestCheckLimit_TaskOverridesProject(t *testing.T) {
	proj := &core.Project{Budget: &core.BudgetConfig{MaxTokens: 1000}}
	task := &core.Task{Budget: &core.BudgetConfig{MaxTokens: 50}}
	maxTokens, over := CheckLimit(proj, task, 60)
	if maxTokens != 50 || !over {
		t.Fatalf("expected task's limit (50) to win and over=true, got maxTokens=%d over=%v", maxTokens, over)
	}
}

func TestCheckLimit_WithinBudget(t *testing.T) {
	proj := &core.Project{Budget: &core.BudgetConfig{MaxTokens: 8000}}
	_, over := CheckLimit(proj, &core.Task{}, 5000)
	if over {
		t.Fatal("expected over=false when tokens are within the configured limit")
	}
}

func TestEnforceLimit_ReturnsNilWhenWithinBudget(t *testing.T) {
	proj := &core.Project{Budget: &core.BudgetConfig{MaxTokens: 8000}}
	if err := EnforceLimit(proj, &core.Task{}, 5000); err != nil {
		t.Fatalf("expected nil error within budget, got: %v", err)
	}
}

func TestEnforceLimit_ExactErrorFormat(t *testing.T) {
	proj := &core.Project{Budget: &core.BudgetConfig{MaxTokens: 8000}}
	err := EnforceLimit(proj, &core.Task{}, 14250)
	if err == nil {
		t.Fatal("expected an error when over budget")
	}
	got := err.Error()
	for _, want := range []string{
		"ERROR", "Current context (14,250 tokens) exceeds the configured limit (8,000).",
		"Suggestion:", "--focus",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected error message to contain %q, got:\n%s", want, got)
		}
	}
}

func TestFormatThousands(t *testing.T) {
	cases := map[int]string{
		0: "0", 5: "5", 999: "999",
		1000: "1,000", 14250: "14,250", 1234567: "1,234,567",
	}
	for in, want := range cases {
		if got := formatThousands(in); got != want {
			t.Errorf("formatThousands(%d) = %q, want %q", in, got, want)
		}
	}
}
