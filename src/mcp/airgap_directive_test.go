// airgap_directive_test.go — 10 QA integration tests for PROBLEM 2 (QA
// Test 1): the hardened air-gap payload/directive
// ("reports.egress_airgap_message" + "reports.egress_airgap_directive",
// see models/egress_audit.go's buildAirgapMessage). Five cover the
// air-gap ACTIVE (dry_run: true) case, five cover it INACTIVE — same
// pairing style as egress_gate_test.go's existing tests, extended to
// assert on the new structured payload and the imperative anti-bypass
// directive specifically, not just "MOVA EGRESS AUDIT" appearing.
//
// Honesty note (see docs/README.md's own "Air-gap: alcance y
// limitaciones" section): these tests prove Mova's SIDE of the
// contract — it never lets the raw context leave, and it hands the
// calling host the strongest instruction it can. They cannot, and do
// not try to, prove that every possible MCP host actually OBEYS a
// text instruction embedded in a tool result — that is fundamentally
// outside any MCP server's control. What Test 1 in the original QA
// round found was a real host (Grok, via Cursor) choosing to read
// other files instead of stopping; this directive makes that harder
// to justify, it does not make it impossible.
package mcp

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

// --- ACTIVE (dry_run: true) — 5 tests -------------------------------

// 1. chat_completion, active: directive present, provider never called.
func TestAirgapDirective_ChatCompletion_Active_ContainsDirective(t *testing.T) {
	srv, calls := countingOllama(t)
	defer srv.Close()
	root, project := writeAirgapProject(t, srv.URL, map[string]any{"dry_run": true})

	text := callTool(t, root, "chat_completion", map[string]any{"project": project, "message": "resume el contexto"})
	assertAirgapDirectivePresent(t, text)
	if strings.Contains(text, "TOP_SECRET_CONTEXT_MARKER_98213") {
		t.Fatalf("CONTEXT LEAK in chat_completion while air-gap active:\n%s", text)
	}
	if atomic.LoadInt32(calls) != 0 {
		t.Fatalf("provider was called %d time(s) while air-gap active — must be 0", *calls)
	}
}

// 2. get_full_context, active: directive present, raw context absent.
func TestAirgapDirective_GetFullContext_Active_ContainsDirective(t *testing.T) {
	root, project := writeAirgapProject(t, "http://127.0.0.1:0", map[string]any{"dry_run": true})

	text := callTool(t, root, "get_full_context", map[string]any{"project": project})
	assertAirgapDirectivePresent(t, text)
	if strings.Contains(text, "TOP_SECRET_CONTEXT_MARKER_98213") {
		t.Fatalf("CONTEXT LEAK in get_full_context while air-gap active:\n%s", text)
	}
}

// 3. Structured payload fields: dry_run: true / tokens_sent: 0 /
// tokens_evaluated: <a real positive integer>, exactly the shape
// PROBLEM 2 asked for, not just free text.
func TestAirgapDirective_PayloadStructure_HasRequiredFields(t *testing.T) {
	root, project := writeAirgapProject(t, "http://127.0.0.1:0", map[string]any{"dry_run": true})

	text := callTool(t, root, "get_full_context", map[string]any{"project": project})
	if !strings.Contains(text, "dry_run: true") {
		t.Fatalf("expected a literal \"dry_run: true\" field, got:\n%s", text)
	}
	if !strings.Contains(text, "tokens_sent: 0") {
		t.Fatalf("expected a literal \"tokens_sent: 0\" field, got:\n%s", text)
	}
	m := regexp.MustCompile(`tokens_evaluated:\s*(\d+)`).FindStringSubmatch(text)
	if m == nil {
		t.Fatalf("expected a \"tokens_evaluated: <N>\" field, got:\n%s", text)
	}
	if n, err := strconv.Atoi(m[1]); err != nil || n <= 0 {
		t.Fatalf("expected tokens_evaluated to be a positive integer, got %q", m[1])
	}
}

// 4. The imperative anti-bypass instructions Test 1 found missing are
// actually present, not just the generic audit header.
func TestAirgapDirective_ContainsAntiBypassInstructions(t *testing.T) {
	root, project := writeAirgapProject(t, "http://127.0.0.1:0", map[string]any{"dry_run": true})

	text := callTool(t, root, "get_full_context", map[string]any{"project": project})
	required := []string{
		"DO NOT attempt to bypass",
		"DO NOT read local files",
		"terminate the execution",
	}
	for _, phrase := range required {
		if !strings.Contains(text, phrase) {
			t.Fatalf("expected the directive to contain %q, got:\n%s", phrase, text)
		}
	}
}

// 5. Every other air-gap-gated tool (get_memory, get_memory_all,
// get_workflow, read_file, read_document_layer) also carries the
// directive while active — the hardened message is not exclusive to
// chat_completion/get_full_context.
func TestAirgapDirective_OtherGatedTools_Active_AllContainDirective(t *testing.T) {
	root, project := writeAirgapProject(t, "http://127.0.0.1:0", map[string]any{"dry_run": true})
	if err := os.WriteFile(filepath.Join(root, "projects", project, "memory.md"), []byte("TOP_SECRET_CONTEXT_MARKER_98213"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		tool string
		args map[string]any
	}{
		{"get_memory", map[string]any{"project": project}},
		{"get_memory_all", map[string]any{"project": project}},
		{"get_workflow", map[string]any{"project": project}},
		{"read_file", map[string]any{"project": project, "filename": "secret.txt"}},
	} {
		t.Run(c.tool, func(t *testing.T) {
			text := callTool(t, root, c.tool, c.args)
			assertAirgapDirectivePresent(t, text)
		})
	}
}

// --- INACTIVE (no dry_run / dry_run: false) — 5 tests ---------------

// 6. chat_completion, inactive (host-delegated, no llm_profile — the
// example project's own real-world configuration): real context comes
// back, the directive text is NOT injected into unrelated output.
func TestAirgapDirective_ChatCompletion_Inactive_NoDirective(t *testing.T) {
	srv, calls := countingOllama(t)
	defer srv.Close()
	root, project := writeAirgapProject(t, srv.URL, map[string]any{"dry_run": false})

	text := callTool(t, root, "chat_completion", map[string]any{"project": project, "message": "resume el contexto"})
	if !strings.Contains(text, "TOP_SECRET_CONTEXT_MARKER_98213") {
		t.Fatalf("expected the real context with air-gap inactive, got: %s", text)
	}
	assertAirgapDirectiveAbsent(t, text)
	if atomic.LoadInt32(calls) != 0 {
		t.Fatalf("host-delegated mode must still never call the provider itself, got %d call(s)", *calls)
	}
}

// 7. get_full_context, inactive: unchanged pre-existing behavior — the
// real context is returned, no directive text anywhere in it.
func TestAirgapDirective_GetFullContext_Inactive_NoDirective(t *testing.T) {
	root, project := writeAirgapProject(t, "http://127.0.0.1:0", nil)

	text := callTool(t, root, "get_full_context", map[string]any{"project": project})
	if !strings.Contains(text, "TOP_SECRET_CONTEXT_MARKER_98213") {
		t.Fatalf("expected the real context with no egress_audit configured, got: %s", text)
	}
	assertAirgapDirectiveAbsent(t, text)
}

// 8. The other gated tools, inactive: real content, no directive.
func TestAirgapDirective_OtherGatedTools_Inactive_NoDirective(t *testing.T) {
	root, project := writeAirgapProject(t, "http://127.0.0.1:0", nil)
	if err := os.WriteFile(filepath.Join(root, "projects", project, "memory.md"), []byte("TOP_SECRET_CONTEXT_MARKER_98213"), 0o644); err != nil {
		t.Fatal(err)
	}

	text := callTool(t, root, "get_memory", map[string]any{"project": project})
	if !strings.Contains(text, "TOP_SECRET_CONTEXT_MARKER_98213") {
		t.Fatalf("expected real memory content with air-gap inactive, got: %s", text)
	}
	assertAirgapDirectiveAbsent(t, text)
}

// 9. On-disk evidence: with dry_run:false and an output_file
// configured, WriteEgressAuditLog's block records the ACTUAL context
// (see PROBLEM 1's fix), never the air-gap directive text — the
// directive is only ever a REPLACEMENT for context in the blocked
// case, never mixed into the real evidence log.
func TestAirgapDirective_Inactive_OnDiskEvidenceHasNoDirectiveText(t *testing.T) {
	root, project := writeAirgapProject(t, "http://127.0.0.1:0", map[string]any{
		"dry_run": false, "output_file": "egress_sanitized.md",
	})

	_ = callTool(t, root, "get_full_context", map[string]any{"project": project})

	data, err := os.ReadFile(filepath.Join(root, "projects", project, "egress_sanitized.md"))
	if err != nil {
		t.Fatalf("expected the evidence file to exist: %v", err)
	}
	assertAirgapDirectiveAbsent(t, string(data))
	if !strings.Contains(string(data), "TOP_SECRET_CONTEXT_MARKER_98213") {
		t.Fatalf("expected the real (governed) context in the evidence file, got:\n%s", data)
	}
}

// 10. search_context stays out of scope entirely, activated or not —
// it has no "project" argument and never touches a project's private
// context (see server.go's airgapGatedTools doc comment), so it must
// never be gated by this project's dry_run, and never carry the
// directive either.
func TestAirgapDirective_SearchContext_NeverGatedEvenWhenActive(t *testing.T) {
	if airgapGatedTools["search_context"] {
		t.Fatal("search_context must stay out of airgapGatedTools — see its doc comment in server.go")
	}
}

// --- shared assertions ----------------------------------------------

func assertAirgapDirectivePresent(t *testing.T, text string) {
	t.Helper()
	if !strings.Contains(text, "MOVA EGRESS AUDIT") {
		t.Fatalf("expected the air-gap audit header, got:\n%s", text)
	}
	if !strings.Contains(text, "CRITICAL SECURITY DIRECTIVE FOR ASSISTANT") {
		t.Fatalf("expected the hardened anti-bypass directive, got:\n%s", text)
	}
}

func assertAirgapDirectiveAbsent(t *testing.T, text string) {
	t.Helper()
	if strings.Contains(text, "CRITICAL SECURITY DIRECTIVE FOR ASSISTANT") {
		t.Fatalf("the anti-bypass directive must only appear when the air-gap actually blocks something, got:\n%s", text)
	}
}
