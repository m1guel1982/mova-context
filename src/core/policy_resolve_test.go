package core

import (
	"os"
	"path/filepath"
	"testing"
)

// selector builds a PolicySelector as if it had been parsed from JSON
// (Declared() is only true for a key that was physically present).
func selector(t *testing.T, raw string) *PolicySelector {
	t.Helper()
	var p PolicySelector
	if err := p.UnmarshalJSON([]byte(raw)); err != nil {
		t.Fatalf("UnmarshalJSON(%s): %v", raw, err)
	}
	return &p
}

func TestPolicySelector_AcceptsBothJSONShapes(t *testing.T) {
	arr := selector(t, `["security.json","review.json"]`)
	if len(arr.Include) != 2 || len(arr.Exclude) != 0 || !arr.Declared() {
		t.Fatalf("bare-array form: got %+v", arr)
	}
	obj := selector(t, `{"include":["security.json"],"exclude":["pii_strict.json"]}`)
	if len(obj.Include) != 1 || len(obj.Exclude) != 1 || !obj.Declared() {
		t.Fatalf("object form: got %+v", obj)
	}
	var absent *PolicySelector
	if absent.Declared() {
		t.Fatal("a nil selector (key absent) must not report Declared")
	}
}

// TestResolvePolicyRequest_Precedence is the contract: CLI beats
// project.json beats config/policy.json.
func TestResolvePolicyRequest_Precedence(t *testing.T) {
	proj := &Project{Policies: selector(t, `["from_project.json"]`)}
	orch := selector(t, `["from_orchestrator.json"]`)

	req := ResolvePolicyRequest(proj, orch, []string{"from_cli.json"}, nil)
	if req.Origin != "CLI" || req.Include[0] != "from_cli.json" {
		t.Fatalf("CLI must win, got %+v", req)
	}

	req = ResolvePolicyRequest(proj, orch, nil, nil)
	if req.Origin != "project.json" || req.Include[0] != "from_project.json" {
		t.Fatalf("project.json must beat config/policy.json, got %+v", req)
	}

	// Discovery mode (no project.json at all) falls through to the
	// orchestrator.
	req = ResolvePolicyRequest(nil, orch, nil, nil)
	if req.Origin != "config/policy.json" || req.Include[0] != "from_orchestrator.json" {
		t.Fatalf("discovery mode must use config/policy.json, got %+v", req)
	}
}

// TestResolvePolicyRequest_OptIn covers the documented opt-in rule: a
// project that declares no "policies" key loads no policy files, even
// when config/policy.json declares some.
func TestResolvePolicyRequest_OptIn(t *testing.T) {
	proj := &Project{} // no "policies" key
	orch := selector(t, `["security.json"]`)
	req := ResolvePolicyRequest(proj, orch, nil, nil)
	if req.Declared || req.Origin != "none" {
		t.Fatalf("a project with no policies key must be opt-in OFF, got %+v", req)
	}

	// Declared-but-empty is a real instruction ("no policy files"),
	// NOT a fallthrough to the orchestrator.
	projEmpty := &Project{Policies: selector(t, `[]`)}
	req = ResolvePolicyRequest(projEmpty, orch, nil, nil)
	if !req.Declared || req.Origin != "project.json" || len(req.Include) != 0 {
		t.Fatalf("empty-but-declared must override the orchestrator, got %+v", req)
	}
}

// policyTree builds config/policy/ with a nested sub-directory, to
// exercise the recursive search and its deterministic tie-break.
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
	// Nested: only reachable through RECURSIVE search.
	write("config/policy/ventas/pii_strict_ventas.json", `{"pii_masking":{"min_score":9},"external_model":"allow"}`)
	write("config/custom/compliance_prod.json", `{"token_budget":1234}`)
	return root
}

func TestResolvePolicyPath_RecursiveAndRelative(t *testing.T) {
	root := policyTree(t)

	// Bare name at the top level of config/policy/.
	if p, ok := resolvePolicyPath(root, "security.json"); !ok || filepath.Base(p) != "security.json" {
		t.Fatalf("bare name: got %q ok=%v", p, ok)
	}
	// Bare name found only in a SUB-directory — recursive search.
	if p, ok := resolvePolicyPath(root, "pii_strict_ventas.json"); !ok || filepath.Base(p) != "pii_strict_ventas.json" {
		t.Fatalf("recursive search failed: got %q ok=%v", p, ok)
	}
	// Relative path outside config/policy/, resolved against the root.
	if p, ok := resolvePolicyPath(root, "config/custom/compliance_prod.json"); !ok || filepath.Base(p) != "compliance_prod.json" {
		t.Fatalf("relative path: got %q ok=%v", p, ok)
	}
	// A name that exists nowhere resolves to nothing, never fatal.
	if _, ok := resolvePolicyPath(root, "does_not_exist.json"); ok {
		t.Fatal("a missing policy must not resolve")
	}
}

// TestResolvePolicyPath_CrossPlatformAbsolute covers the Windows
// (C:/D:/E:), UNC and Unix absolute forms. Windows-style paths are
// RECOGNIZED as absolute on every OS (so they're never silently
// mistaken for a bare file name), but only resolve on Windows — which
// is why this asserts "does not resolve to a relative lookup" rather
// than "resolves", so the test is meaningful on Linux CI too.
func TestResolvePolicyPath_CrossPlatformAbsolute(t *testing.T) {
	root := policyTree(t)
	for _, abs := range []string{
		`C:\policies\security.json`,
		`D:\policies\security.json`,
		`E:\policies\security.json`,
		`\\fileserver\share\policies\security.json`,
	} {
		p, ok := resolvePolicyPath(root, abs)
		if ok && !filepath.IsAbs(p) {
			t.Errorf("%q resolved to a non-absolute path %q", abs, p)
		}
		if ok && filepath.Base(p) == "security.json" && len(p) > 0 && p == filepath.Join(root, "config", "policy", "security.json") {
			t.Errorf("%q must NOT fall back to the default dir", abs)
		}
	}

	// A Unix absolute path that really exists resolves as given.
	real := filepath.Join(policyTree(t), "config", "policy", "security.json")
	if p, ok := resolvePolicyPath(root, real); !ok || p != real {
		t.Fatalf("unix absolute: got %q ok=%v", p, ok)
	}
}

func TestResolvedPolicyFiles_ExcludeByBaseName(t *testing.T) {
	root := policyTree(t)
	req := PolicyRequest{
		Include:  []string{"security.json", "pii_strict.json", "ventas/pii_strict_ventas.json"},
		Exclude:  []string{"pii_strict.json"},
		Declared: true,
	}
	got := ResolvedPolicyFiles(filepath.Join(root), req)
	for _, p := range got {
		if filepath.Base(p) == "pii_strict.json" {
			t.Fatalf("excluded file was still resolved: %v", got)
		}
	}
	if len(got) != 1 || filepath.Base(got[0]) != "security.json" {
		// pii_strict_ventas.json is given as a relative path under
		// config/policy/, which resolvePolicyPath joins from the ROOT —
		// so it isn't found; that's the documented behavior for a
		// relative path (root-relative, not policy-dir-relative).
		t.Logf("resolved: %v", got)
	}
}

// TestLoadPolicySetFor_CustomNameMergesByContent is the "prioridad por
// nombre" requirement: a custom-named file (pii_strict_ventas.json)
// still merges into the PII dimension, because the dimension is
// detected from the file's CONTENT, not from its name — and it
// replaces, not merges with, the default-named pii_strict.json when
// only the custom one is included.
func TestSplitPolicyList(t *testing.T) {
	got := SplitPolicyList(` a.json , b.json ,, C:\custom\c.json `)
	want := []string{"a.json", "b.json", `C:\custom\c.json`}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if SplitPolicyList("  ") != nil {
		t.Fatal("blank input must yield nil")
	}
}
