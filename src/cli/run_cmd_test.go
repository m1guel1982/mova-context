// run_cmd_test.go — regression coverage for the gap found while
// auditing "policy integration across every door" (see run_cmd.go's
// package comment): `mova run` used to print the fully-governed
// context straight to stdout with NO egress air-gap check at all —
// "egress_audit": {"dry_run": true} was silently ignored by this one
// door even though models/egress_audit.go's own package comment
// already claimed `mova run` as a participant. These tests exercise
// runProject exactly as the CLI binary would.
package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"

	"mova.local/core"
	"mova.local/i18n"
)

// TestMain loads a copy of the real config/lang/ catalog so i18n.T
// (used by the air-gap message/directive) resolves real text instead
// of bare keys — same pattern as mcp/i18n_test_main_test.go.
func TestMain(m *testing.M) {
	_, thisFile, _, ok := goruntime.Caller(0)
	if ok {
		realConfigLang := filepath.Join(filepath.Dir(thisFile), "..", "..", "config", "lang")
		if enData, err := os.ReadFile(filepath.Join(realConfigLang, "en.json")); err == nil {
			tmp, tmpErr := os.MkdirTemp("", "mova-cli-i18n-*")
			if tmpErr == nil {
				defer os.RemoveAll(tmp)
				langDir := filepath.Join(tmp, "config", "lang")
				_ = os.MkdirAll(langDir, 0o755)
				_ = os.WriteFile(filepath.Join(langDir, "en.json"), enData, 0o644)
				_ = os.WriteFile(filepath.Join(langDir, "lang_active.json"), []byte(`{"lang":"en"}`), 0o644)
				_ = i18n.Init(tmp)
			}
		}
	}
	os.Exit(m.Run())
}

// writeRunCmdProject builds a minimal project with an obvious secret
// marker in its focus file, optionally with an "egress_audit" block.
func writeRunCmdProject(t *testing.T, egressAudit map[string]any) (root, project string) {
	t.Helper()
	root = t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "workflow.md"), []byte("# stub"), 0o644); err != nil {
		t.Fatal(err)
	}
	project = "run-cmd-test"
	projDir := filepath.Join(root, "projects", project)
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	repoDir := filepath.Join(root, "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "secret.txt"), []byte("TOP_SECRET_CONTEXT_MARKER_98213"), 0o644); err != nil {
		t.Fatal(err)
	}
	pj := map[string]any{
		"project": project, "author": "system:default", "repo": repoDir,
		"lang": "en", "adapter": "file", "default_task": "t",
		"agents": map[string]any{"domain": "software", "use": []string{}},
		"skills": map[string]any{"domain": "software", "use": []string{}},
		"tasks":  map[string]any{"t": map[string]any{"focus": []string{"secret.txt"}}},
	}
	if egressAudit != nil {
		pj["egress_audit"] = egressAudit
	}
	data, _ := json.Marshal(pj)
	if err := os.WriteFile(filepath.Join(projDir, "project.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return root, project
}

// captureStdout redirects os.Stdout for the duration of fn (consolePrint
// writes straight to os.Stdout on non-Windows — see console_other.go)
// and returns everything written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string)
	go func() {
		data, _ := io.ReadAll(r)
		done <- string(data)
	}()

	fn()

	w.Close()
	os.Stdout = orig
	return <-done
}

// TestRunProject_NoEgressAudit_PrintsRealContext is the negative
// control: with no "egress_audit" configured, behavior is unchanged —
// the real context is printed.
func TestRunProject_NoEgressAudit_PrintsRealContext(t *testing.T) {
	root, project := writeRunCmdProject(t, nil)
	adapter := core.NewFileAdapter(root)

	out := captureStdout(t, func() { runProject(root, adapter, project, "") })
	if !strings.Contains(out, "TOP_SECRET_CONTEXT_MARKER_98213") {
		t.Fatalf("expected the real context with no egress_audit configured, got:\n%s", out)
	}
}

// TestRunProject_DryRunTrue_BlocksContextAndShowsDirective is THE
// regression test: `mova run` must honor "egress_audit": {"dry_run":
// true} exactly like get_full_context/chat_completion do — never
// print the raw context, and show the hardened air-gap directive.
func TestRunProject_DryRunTrue_BlocksContextAndShowsDirective(t *testing.T) {
	root, project := writeRunCmdProject(t, map[string]any{"dry_run": true})
	adapter := core.NewFileAdapter(root)

	out := captureStdout(t, func() { runProject(root, adapter, project, "") })
	if strings.Contains(out, "TOP_SECRET_CONTEXT_MARKER_98213") {
		t.Fatalf("CONTEXT LEAK: `mova run` printed the raw context despite dry_run:\n%s", out)
	}
	if !strings.Contains(out, "MOVA EGRESS AUDIT") {
		t.Fatalf("expected the air-gap audit header, got:\n%s", out)
	}
	if !strings.Contains(out, "CRITICAL SECURITY DIRECTIVE FOR ASSISTANT") {
		t.Fatalf("expected the hardened anti-bypass directive, got:\n%s", out)
	}
}

// TestRunProject_DryRunTrue_WritesEvidenceFile confirms the on-disk
// audit trail (egress_audit.output_file) is written here too, exactly
// like every other door — this is what closes the "todo integrado"
// gap for `mova run` specifically.
func TestRunProject_DryRunTrue_WritesEvidenceFile(t *testing.T) {
	root, project := writeRunCmdProject(t, map[string]any{"dry_run": true, "output_file": "egress_sanitized.md"})
	adapter := core.NewFileAdapter(root)

	_ = captureStdout(t, func() { runProject(root, adapter, project, "") })

	data, err := os.ReadFile(filepath.Join(root, "projects", project, "egress_sanitized.md"))
	if err != nil {
		t.Fatalf("expected the evidence file to exist: %v", err)
	}
	if !strings.Contains(string(data), "TOP_SECRET_CONTEXT_MARKER_98213") {
		t.Fatalf("expected the real (governed) context in the evidence file, got:\n%s", data)
	}
}
