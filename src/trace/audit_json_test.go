// audit_json_test.go — validates the pii-audit-log.json schema
// against the standardized contract (see audit_json.go).
package trace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"mova.local/sanitize"
)

func TestWriteAuditLog_MatchesContractSchema(t *testing.T) {
	d := &Data{
		ExecutionID: "exec-123", RepoURL: "https://github.com/fastapi/fastapi",
		Branch: "master", CommitHash: "50113da", TaskName: "fix OAuth authentication",
		IgnorePatterns: []string{"docs/!(en)/**"},
		StateTotals: sanitize.StateCounts{
			DiscoveredFiles: 2955, DiscoveredTokens: 9098672,
			CandidateFiles: 509, CandidateTokens: 1280661,
			AllowedFiles: 314, AllowedTokens: 362222,
			SanitizedFiles: 195, SanitizedTokens: 918439,
			BlockedFiles: 0, BlockedTokens: 0,
			ExcludedFiles: 2630, ExcludedTokens: 7818011,
		},
		SecurityImpact: sanitize.SecurityImpact{
			FilesWithPotentialPII: 195, FilesWithPotentialSecrets: 56, FilesWithAnyIndicator: 195,
		},
		ExclusionReasons: []ExclusionReasonRow{
			{Reason: "Not relevant to task", Files: 2446, Tokens: 7818011},
		},
		RelevanceTop: []RelevanceRankRow{
			{Rank: 1, Path: "docs/ru/docs/tutorial/dependencies/index.md", Score: 13.70},
		},
		Findings: []sanitize.SecurityFinding{
			{
				Path: ".github/DISCUSSION_TEMPLATE/questions.yml", Type: "pii_structural",
				Score: 0.01, PatternsDetected: []string{"structural PII pattern"}, Occurrences: 1,
				Action: "SANITIZED", Transformed: false, OriginalTokens: 1348, FinalTokens: 1348,
				RuleID: "pii_masking.min_score", Detector: "shannon_entropy_shape",
			},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "pii-audit-log.json")
	if err := writeAuditLog(path, d); err != nil {
		t.Fatalf("writeAuditLog failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if got["schema_version"] != "1.0" {
		t.Errorf("schema_version = %v, want 1.0", got["schema_version"])
	}
	exec, _ := got["execution"].(map[string]any)
	if exec["execution_id"] != "exec-123" || exec["commit"] != "50113da" || exec["task"] != "fix OAuth authentication" {
		t.Errorf("execution block wrong: %+v", exec)
	}

	ctx, _ := got["context"].(map[string]any)
	scanned, _ := ctx["scanned"].(map[string]any)
	if scanned["files"].(float64) != 2955 {
		t.Errorf("context.scanned.files = %v, want 2955", scanned["files"])
	}
	final, _ := ctx["final"].(map[string]any)
	if final["actually_transformed"] != false {
		t.Errorf("context.final.actually_transformed must be false, got %v", final["actually_transformed"])
	}

	decisions, _ := got["decisions"].(map[string]any)
	wouldSanitize, _ := decisions["would_sanitize"].(map[string]any)
	if wouldSanitize["files"].(float64) != 195 {
		t.Errorf("decisions.would_sanitize.files = %v, want 195", wouldSanitize["files"])
	}

	findings, _ := got["findings"].([]any)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	f0, _ := findings[0].(map[string]any)
	decision, _ := f0["decision"].(map[string]any)
	if decision["action"] != "WOULD_SANITIZE" {
		t.Errorf("finding action = %v, want WOULD_SANITIZE (never raw SANITIZED)", decision["action"])
	}
	if decision["transformed"] != false {
		t.Errorf("finding transformed must be false, got %v", decision["transformed"])
	}
	tokens, _ := f0["tokens"].(map[string]any)
	if tokens["before"].(float64) != tokens["after"].(float64) {
		t.Errorf("audit mode must never change tokens.before vs tokens.after: %+v", tokens)
	}

	relevance, _ := got["relevance"].(map[string]any)
	if relevance == nil || relevance["method"] != "bm25+structural" {
		t.Errorf("relevance block missing or wrong: %+v", relevance)
	}
}

func TestWriteAuditLog_ExclusionReasonsNeverLoseData(t *testing.T) {
	// Regression test for a real bug found while testing against
	// fastapi/fastapi with an active Spanish locale: two DIFFERENT
	// reason strings ("No relevante para la tarea" and "No relevante")
	// both fell through exclusionReasonSlug's old substring matching
	// into the SAME "not_relevant" key, and the map-building loop
	// OVERWROTE rather than accumulated - silently discarding 1,197
	// files' worth of exclusion accounting from the JSON.
	d := &Data{
		ExecutionID: "e1",
		StateTotals: sanitize.StateCounts{DiscoveredFiles: 10},
		ExclusionReasons: []ExclusionReasonRow{
			{Reason: "No relevante para la tarea", Files: 1197, Tokens: 5430256},
			{Reason: "No relevante", Files: 87, Tokens: 0},
		},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "pii-audit-log.json")
	if err := writeAuditLog(path, d); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var got map[string]any
	json.Unmarshal(data, &got)
	decisions, _ := got["decisions"].(map[string]any)
	reasons, _ := decisions["exclusion_reasons"].(map[string]any)

	taskRow, _ := reasons["not_relevant_to_task"].(map[string]any)
	if taskRow == nil || taskRow["files"].(float64) != 1197 {
		t.Fatalf("not_relevant_to_task must show 1197 files, got %+v (full: %+v)", taskRow, reasons)
	}
	genericRow, _ := reasons["not_relevant"].(map[string]any)
	if genericRow == nil || genericRow["files"].(float64) != 87 {
		t.Fatalf("not_relevant must show 87 files, got %+v", genericRow)
	}
}

func TestWriteAuditLog_NoTaskOmitsRelevance(t *testing.T) {
	d := &Data{ExecutionID: "e1", StateTotals: sanitize.StateCounts{DiscoveredFiles: 1}}
	dir := t.TempDir()
	path := filepath.Join(dir, "pii-audit-log.json")
	if err := writeAuditLog(path, d); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var got map[string]any
	json.Unmarshal(data, &got)
	if _, ok := got["relevance"]; ok {
		t.Errorf("relevance must be omitted entirely when no task was given, got %v", got["relevance"])
	}
	exec, _ := got["execution"].(map[string]any)
	if exec["task"] != nil {
		t.Errorf("execution.task must be JSON null when no task was given, got %v", exec["task"])
	}
}
