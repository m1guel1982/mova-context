package sanitize

import (
	"os"
	"path/filepath"
	"testing"

	"mova.local/core"
)

// policyTree builds config/policy/ with a nested sub-directory —
// mirrors core's own policyTree fixture (core/policy_resolve_test.go)
// since a sanitize-package test can't import core's unexported test
// helpers across packages either way.
func policyTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("config/policy/security.json", `{"block_on_private_key":true,"block_on_api_key":true}`)
	write("config/policy/pii_strict.json", `{"pii_masking":{"min_score":3},"external_model":"deny"}`)
	write("config/policy/ventas/pii_strict_ventas.json", `{"pii_masking":{"min_score":9},"external_model":"allow"}`)
	return root
}

// TestLoadPolicySetFor_CustomNameMergesByContent is the "prioridad por
// nombre" requirement: a custom-named file (pii_strict_ventas.json)
// still merges into the PII dimension, because the dimension is
// detected from the file's CONTENT, not from its name — and it
// replaces, not merges with, the default-named pii_strict.json when
// only the custom one is included.
func TestLoadPolicySetFor_CustomNameMergesByContent(t *testing.T) {
	root := policyTree(t)
	ps := LoadPolicySetFor(root, core.PolicyRequest{
		Include:  []string{"security.json", "pii_strict_ventas.json"},
		Exclude:  []string{"pii_strict.json"},
		Declared: true,
		Origin:   "project.json",
	})
	if ps.PII.MinScore != 9 {
		t.Fatalf("expected the CUSTOM pii file (min_score 9) to win, got %v", ps.PII.MinScore)
	}
	if ps.PIIExternalModel != "allow" {
		t.Fatalf("expected external_model from the custom file, got %q", ps.PIIExternalModel)
	}
	if !ps.Security.BlockOnAPIKey {
		t.Fatal("expected security.json to have merged too")
	}
}

// TestLoadPolicySetFor_UndeclaredKeepsSafeDefaults is the safety
// caveat documented on core.ResolvePolicyRequest: opt-in OFF means "no
// policy FILES", never "no protection at all".
func TestLoadPolicySetFor_UndeclaredKeepsSafeDefaults(t *testing.T) {
	root := policyTree(t)
	ps := LoadPolicySetFor(root, core.PolicyRequest{Declared: false, Origin: "none"})
	if !ps.Security.BlockOnPrivateKey {
		t.Fatal("private-key blocking must stay on even with no policies declared")
	}
	if ps.PII.MinScore <= 0 {
		t.Fatal("built-in PII defaults must still apply")
	}
}
