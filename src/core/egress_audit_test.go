package core

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveEgressAudit_AbsentBlock covers the mandatory default from
// the spec: "egress_audit" absent → DryRun false, OutputFile "" —
// today's behavior, unchanged.
func TestResolveEgressAudit_AbsentBlock(t *testing.T) {
	proj := &Project{Project: "p"}
	dryRun, outputFile := ResolveEgressAudit("/root", "p", proj)
	if dryRun {
		t.Fatalf("expected dryRun=false when egress_audit is absent, got true")
	}
	if outputFile != "" {
		t.Fatalf("expected outputFile=\"\" when egress_audit is absent, got %q", outputFile)
	}
}

// TestResolveEgressAudit_NilProject covers the MCP/HTTP door calling
// this before a project was successfully loaded — must never panic.
func TestResolveEgressAudit_NilProject(t *testing.T) {
	dryRun, outputFile := ResolveEgressAudit("/root", "p", nil)
	if dryRun || outputFile != "" {
		t.Fatalf("expected zero values for a nil project, got dryRun=%v outputFile=%q", dryRun, outputFile)
	}
}

// TestResolveEgressAudit_DryRunWithoutOutputFile: dry_run and
// output_file are independent switches — a project can dry-run without
// ever writing an audit log.
func TestResolveEgressAudit_DryRunWithoutOutputFile(t *testing.T) {
	proj := &Project{EgressAudit: &EgressAuditConfig{DryRun: true}}
	dryRun, outputFile := ResolveEgressAudit("/root", "p", proj)
	if !dryRun {
		t.Fatalf("expected dryRun=true")
	}
	if outputFile != "" {
		t.Fatalf("expected outputFile=\"\" when output_file was never set, got %q", outputFile)
	}
}

// TestResolveEgressAudit_RelativeIsUnderProjectDir is THE contract
// requirement: a relative output_file resolves under
// projects/<project>/ — never the process's working directory.
func TestResolveEgressAudit_RelativeIsUnderProjectDir(t *testing.T) {
	proj := &Project{EgressAudit: &EgressAuditConfig{OutputFile: filepath.Join(".mova", "egress_sanitized.log")}}
	_, outputFile := ResolveEgressAudit("/mova-root", "02-pii-compliance-governance", proj)
	want := filepath.Join("/mova-root", "projects", "02-pii-compliance-governance", ".mova", "egress_sanitized.log")
	if outputFile != want {
		t.Fatalf("got %q, want %q", outputFile, want)
	}
}

// TestResolveEgressAudit_AbsolutePaths covers every OS family the spec
// requires (Windows C/D/E, UNC, Unix) via documents.IsAbsCrossPlatform
// — recognized identically regardless of which OS this test runs on,
// so a single Linux CI run genuinely proves all five.
func TestResolveEgressAudit_AbsolutePaths(t *testing.T) {
	cases := []string{
		`C:\audit\egress.log`,
		`D:\logs\egress.log`,
		`E:\data\egress.log`,
		`\\fileserver\share\audit\egress.log`,
		"/var/log/mova/egress.log",
	}
	for _, raw := range cases {
		proj := &Project{EgressAudit: &EgressAuditConfig{OutputFile: raw}}
		_, outputFile := ResolveEgressAudit("/mova-root", "p", proj)
		if strings.Contains(outputFile, filepath.Join("projects", "p")) {
			t.Errorf("absolute path %q must NOT be joined under projects/p/, got %q", raw, outputFile)
		}
		if outputFile == "" {
			t.Errorf("absolute path %q resolved to an empty output file", raw)
		}
	}
}

// TestResolveEgressAudit_DirectoryOnlyGetsDefaultFileName covers "el
// archivo se crea ... por defecto el nombre egress_sanitized.md" for
// an output_file that only names a directory (trailing separator).
func TestResolveEgressAudit_DirectoryOnlyGetsDefaultFileName(t *testing.T) {
	proj := &Project{EgressAudit: &EgressAuditConfig{OutputFile: ".mova/"}}
	_, outputFile := ResolveEgressAudit("/mova-root", "p", proj)
	if filepath.Base(outputFile) != DefaultEgressAuditFileName {
		t.Fatalf("expected default file name %q, got %q", DefaultEgressAuditFileName, outputFile)
	}
}

// TestResolveEgressAudit_BareNameIsUsedAsGiven: a name with no
// trailing separator is trusted as the literal file the person meant
// — even without an extension — since a dotfile-style name like
// ".mova" can't be reliably told apart from an intentional
// extension-less filename (see isDirectoryLikeTarget's doc comment).
func TestResolveEgressAudit_BareNameIsUsedAsGiven(t *testing.T) {
	proj := &Project{EgressAudit: &EgressAuditConfig{OutputFile: ".mova"}}
	_, outputFile := ResolveEgressAudit("/mova-root", "p", proj)
	want := filepath.Join("/mova-root", "projects", "p", ".mova")
	if outputFile != want {
		t.Fatalf("got %q, want %q", outputFile, want)
	}
}
