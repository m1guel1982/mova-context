// pii_masking_disk_test.go — regression coverage for PROBLEM 1 (QA
// Test 6): with "budget": {"pii_masking": {"enabled": true}}, NO
// on-disk evidence file (egress_audit.output_file) may ever contain
// raw PII, on ANY door that can write one — get_full_context AND
// chat_completion. The bug: mcp/context_tool.go's fullContextTool used
// to call core.BuildContextSections directly, a parallel pipeline that
// never runs budget.applyPIIMasking (that stage only lives inside
// budget.BuildGatedContext) — so get_full_context's own tool result
// AND the audit block WriteEgressAuditLog appended to output_file both
// carried the raw RUT/email untouched, even though
// mova-budget-report.md's separate stats correctly reported N tokens
// masked. See context_tool.go's package comment for the full
// diagnosis (H1 vs H2).
package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mova.local/models"
)

// rawRUT/rawEmail mimic the QA scenario's customers.json fixtures
// (18.234.567-9 / andrea.fuentes@correo-ejemplo.cl) closely enough in
// SHAPE (digit ratio, separators, "@") to clear DefaultPIIPolicy's
// MinScore — these are what must NEVER appear verbatim once PII
// Masking is enabled.
const (
	rawRUT   = "18.234.567-9"
	rawEmail = "andrea.fuentes@correo-ejemplo.cl"
)

// writePIIProject builds a minimal project with
// "budget.pii_masking.enabled": true and an "egress_audit.output_file"
// evidence log, whose only focus file contains rawRUT/rawEmail — the
// same shape as the real 02-pii-compliance-governance example.
//
// It also writes a config/policy/pii_strict.json + config/policy.json
// pair mirroring the REAL example project's own policy (see
// examples/02-pii-compliance-governance/project.json's
// "policies": {"include": ["pii_strict.json"], ...} and
// config/policy/pii_strict.json at the repo root) instead of relying
// on DefaultPIIPolicy's min_score — a plain lowercase, digit-less
// email like rawEmail sits right at DefaultPIIPolicy's threshold
// (score ≈ 0.60 vs MinScore 0.62), which made an earlier version of
// this test flaky/misleading. Using the project's own declared
// "policies" (rather than only the global orchestrator) is also the
// regression coverage for the LoadPIIPolicyForProject fix — a
// project.json opting into a stricter policy must actually see that
// policy applied during masking, not just in context-trace's report.
func writePIIProject(t *testing.T, dryRun bool) (root, project, outputFile string) {
	t.Helper()
	root = t.TempDir()

	// chat_completion's models.NewSession requires an active provider
	// configured on disk even for the "host-delegated" (no
	// llm_profile) path exercised below, which never actually calls
	// it — same minimal stub writeAirgapProject (egress_gate_test.go)
	// already uses.
	modelsDir := filepath.Join(root, "config", "models", "ollama")
	if err := os.MkdirAll(modelsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mc := models.DefaultModelConfig(&models.ModelConfig{Provider: "ollama", Type: "ollama", BaseURL: "http://127.0.0.1:0"})
	if err := models.SaveModelConfig(root, "ollama", "llama3.1", mc); err != nil {
		t.Fatal(err)
	}
	if err := models.SetActiveProvider(root, "ollama"); err != nil {
		t.Fatal(err)
	}
	if err := models.SetActiveModel(root, "llama3.1"); err != nil {
		t.Fatal(err)
	}

	policyDir := filepath.Join(root, "config", "policy")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	strictPolicy := `{
		"pii_masking": {
			"min_score": 0.40,
			"shape_weight": 0.65,
			"entropy_weight": 0.35,
			"min_token_length": 4,
			"hash_length": 8,
			"tag_format": "[PII_%s]",
			"shape_rules": {
				"digit_ratio_threshold": 0.3,
				"digit_ratio_bonus": 0.3,
				"separator_min_count": 2,
				"separator_digit_ratio_threshold": 0.2,
				"separator_bonus": 0.25,
				"at_symbol_bonus": 0.65,
				"long_token_bonus_len": 10,
				"long_mixed_token_bonus": 0.15,
				"upper_run_ratio_threshold": 0.6,
				"upper_run_bonus": 0.15
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(policyDir, "pii_strict.json"), []byte(strictPolicy), 0o644); err != nil {
		t.Fatal(err)
	}

	project = "pii-disk-test"
	projDir := filepath.Join(root, "projects", project)
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	repoDir := filepath.Join(root, "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	customers := "nombre: Andrea Fuentes\nrut: " + rawRUT + "\nemail: " + rawEmail + "\n"
	if err := os.WriteFile(filepath.Join(repoDir, "customers.txt"), []byte(customers), 0o644); err != nil {
		t.Fatal(err)
	}

	pj := map[string]any{
		"project": project, "author": "system:default", "repo": repoDir,
		"lang": "es", "adapter": "file", "default_task": "t",
		"agents": map[string]any{"domain": "software", "use": []string{}},
		"skills": map[string]any{"domain": "software", "use": []string{}},
		"tasks":  map[string]any{"t": map[string]any{"focus": []string{"customers.txt"}}},
		"policies": map[string]any{
			"include": []string{"pii_strict.json"},
		},
		"budget": map[string]any{
			"pii_masking": map[string]any{"enabled": true},
		},
		"egress_audit": map[string]any{
			"dry_run":     dryRun,
			"output_file": "egress_sanitized.md",
		},
	}
	data, _ := json.Marshal(pj)
	if err := os.WriteFile(filepath.Join(projDir, "project.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	outputFile = filepath.Join(projDir, "egress_sanitized.md")
	return root, project, outputFile
}

func assertNoRawPII(t *testing.T, label, text string) {
	t.Helper()
	if strings.Contains(text, rawRUT) {
		t.Fatalf("PII LEAK in %s: raw RUT %q found:\n%s", label, rawRUT, text)
	}
	if strings.Contains(text, rawEmail) {
		t.Fatalf("PII LEAK in %s: raw email %q found:\n%s", label, rawEmail, text)
	}
}

// TestGetFullContext_PIIMasking_NoRawPIIOnDisk is THE regression test
// for PROBLEM 1 / QA Test 6: get_full_context's on-disk audit log must
// never contain raw PII when pii_masking.enabled is true, regardless
// of dry_run.
func TestGetFullContext_PIIMasking_NoRawPIIOnDisk(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		t.Run(map[bool]string{false: "dry_run=false", true: "dry_run=true"}[dryRun], func(t *testing.T) {
			root, project, outputFile := writePIIProject(t, dryRun)

			text := callTool(t, root, "get_full_context", map[string]any{"project": project})
			assertNoRawPII(t, "get_full_context tool result", text)

			data, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("expected %s to exist: %v", outputFile, err)
			}
			assertNoRawPII(t, "on-disk evidence file ("+outputFile+")", string(data))
		})
	}
}

// TestChatCompletion_PIIMasking_NoRawPIIOnDisk is the chat_completion
// counterpart — already correct before this fix (it goes through
// budget.BuildGatedContext), kept here so both doors that can write
// egress_audit.output_file are covered by the same regression suite,
// side by side, instead of only one of them.
func TestChatCompletion_PIIMasking_NoRawPIIOnDisk(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		t.Run(map[bool]string{false: "dry_run=false", true: "dry_run=true"}[dryRun], func(t *testing.T) {
			root, project, outputFile := writePIIProject(t, dryRun)

			text := callTool(t, root, "chat_completion", map[string]any{"project": project, "message": "resume el contexto"})
			assertNoRawPII(t, "chat_completion tool result", text)

			data, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("expected %s to exist: %v", outputFile, err)
			}
			assertNoRawPII(t, "on-disk evidence file ("+outputFile+")", string(data))
		})
	}
}

// TestGetFullContext_PIIMasking_StillMasksSectionsFullNotJustStats is a
// narrower unit-level check disambiguating H1 vs H2 (see
// context_tool.go's package comment): confirms the returned text
// actually contains the deterministic [PII_ pseudonym tag — i.e. the
// buffer itself was substituted, not just counted.
func TestGetFullContext_PIIMasking_StillMasksSectionsFullNotJustStats(t *testing.T) {
	root, project, _ := writePIIProject(t, false)

	text := callTool(t, root, "get_full_context", map[string]any{"project": project})
	if !strings.Contains(text, "[PII_") {
		t.Fatalf("expected a masked [PII_...] pseudonym in get_full_context's result, got:\n%s", text)
	}
}
