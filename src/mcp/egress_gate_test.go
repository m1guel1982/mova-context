// egress_gate_test.go — integration coverage for the air-gap contract
// (see egress_audit.go's package comment and PROJECT_JSON.md §
// egress_audit): "egress_audit": {"dry_run": true} must block BOTH
// chat_completion AND get_full_context from ever returning the actual
// assembled context, on every door. These tests call mcp.Process
// directly — the exact same JSON-RPC entry point stdio (MCP) and
// http/server.go (HTTP) both call — so a pass here is a real guarantee
// for MCP and HTTP at once, not just a unit-level assertion.
package mcp

import (
	"archive/zip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"mova.local/core"
	"mova.local/models"
)

// countingOllama is fakeToolCallingOllama's sibling: same shape, but
// exposes a call counter so these tests can assert the provider was
// NEVER reached — the one fact the whole air-gap feature exists to
// guarantee.
func countingOllama(t *testing.T) (*httptest.Server, *int32) {
	t.Helper()
	var calls int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"message":           map[string]string{"role": "assistant", "content": "SECRET_MODEL_REPLY_MUST_NEVER_APPEAR"},
			"done":              true,
			"prompt_eval_count": 10,
			"eval_count":        5,
		})
	})
	return httptest.NewServer(mux), &calls
}

// writeAirgapProject builds a minimal, self-contained project whose
// repo contains an obviously-secret marker string — the ledger of
// truth these tests check for ABSENCE in every "blocked" response.
func writeAirgapProject(t *testing.T, baseURL string, egressAudit map[string]any) (root, project string) {
	t.Helper()
	root = t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "workflow.md"), []byte("# stub"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "config", "models", "ollama")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	mc := models.DefaultModelConfig(&models.ModelConfig{Provider: "ollama", Type: "ollama", BaseURL: baseURL})
	if err := models.SaveModelConfig(root, "ollama", "llama3.1", mc); err != nil {
		t.Fatal(err)
	}
	if err := models.SetActiveProvider(root, "ollama"); err != nil {
		t.Fatal(err)
	}
	if err := models.SetActiveModel(root, "llama3.1"); err != nil {
		t.Fatal(err)
	}

	project = "airgap-test"
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

func callTool(t *testing.T, root, name string, args map[string]any) string {
	t.Helper()
	adapter := core.NewFileAdapter(root)
	resp := Process(adapter, root, Request{
		JSONRPC: "2.0", ID: json.RawMessage("1"), Method: "tools/call",
		Params: map[string]any{"name": name, "arguments": args},
	})
	var parsed struct {
		Result *struct {
			Content []struct{ Text string } `json:"content"`
		} `json:"result"`
		Error *struct{ Message string } `json:"error"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil {
		t.Fatalf("invalid JSON-RPC response: %v\nraw: %s", err, resp)
	}
	if parsed.Error != nil {
		// A Go-level tool error is normally wrapped into a plain
		// "result" (see server.go's executeTool), so reaching an actual
		// JSON-RPC "error" means something more fundamental broke
		// (unknown tool, malformed params) — surfaced as text either
		// way, so callers can assert on it the same way.
		return "JSON-RPC error: " + parsed.Error.Message
	}
	if parsed.Result == nil || len(parsed.Result.Content) == 0 {
		t.Fatalf("empty result: %s", resp)
	}
	return parsed.Result.Content[0].Text
}

// TestChatCompletion_DryRun_NeverLeaksContextOrCallsProvider is the
// core air-gap contract for chat_completion, exercised through the
// exact JSON-RPC path MCP (stdio) and HTTP both use.
func TestChatCompletion_DryRun_NeverLeaksContextOrCallsProvider(t *testing.T) {
	srv, calls := countingOllama(t)
	defer srv.Close()
	root, project := writeAirgapProject(t, srv.URL, map[string]any{"dry_run": true})

	text := callTool(t, root, "chat_completion", map[string]any{"project": project, "message": "resume el contexto"})
	if strings.Contains(text, "TOP_SECRET_CONTEXT_MARKER_98213") {
		t.Fatalf("CONTEXT LEAK: the secret marker appeared in a dry_run response:\n%s", text)
	}
	if strings.Contains(text, "SECRET_MODEL_REPLY_MUST_NEVER_APPEAR") {
		t.Fatalf("the fake provider's reply leaked through despite dry_run: %s", text)
	}
	if !strings.Contains(text, "MOVA EGRESS AUDIT") {
		t.Fatalf("expected the air-gap message, got: %s", text)
	}
	if atomic.LoadInt32(calls) != 0 {
		t.Fatalf("provider was called %d time(s) during dry_run — must be 0", *calls)
	}
}

// TestGetFullContext_DryRun_NeverLeaksContext is THE regression test
// for the real gap found in review: get_full_context used to return
// the raw context unconditionally, bypassing dry_run entirely — a
// direct exfiltration path around chat_completion's own air-gap.
func TestGetFullContext_DryRun_NeverLeaksContext(t *testing.T) {
	root, project := writeAirgapProject(t, "http://127.0.0.1:0", map[string]any{"dry_run": true})

	text := callTool(t, root, "get_full_context", map[string]any{"project": project})
	if strings.Contains(text, "TOP_SECRET_CONTEXT_MARKER_98213") {
		t.Fatalf("CONTEXT LEAK: get_full_context returned the raw context despite dry_run:\n%s", text)
	}
	if !strings.Contains(text, "MOVA EGRESS AUDIT") {
		t.Fatalf("expected the air-gap message, got: %s", text)
	}
}

// TestGetFullContext_NoDryRun_ReturnsContextNormally is the negative
// control: without dry_run, get_full_context's existing behavior is
// unchanged — the context IS returned, exactly as before this feature.
func TestGetFullContext_NoDryRun_ReturnsContextNormally(t *testing.T) {
	root, project := writeAirgapProject(t, "http://127.0.0.1:0", nil)

	text := callTool(t, root, "get_full_context", map[string]any{"project": project})
	if !strings.Contains(text, "TOP_SECRET_CONTEXT_MARKER_98213") {
		t.Fatalf("expected the real context (no dry_run configured), got: %s", text)
	}
}

// writeMinimalDocx creates a valid-enough .docx (a zip with just
// word/document.xml) for documents.ReadDocumentLayer to parse — just
// enough structure for the one paragraph these tests need to find.
func writeMinimalDocx(t *testing.T, path, text string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	xml := `<?xml version="1.0"?><w:document xmlns:w="ns"><w:body><w:p><w:r><w:t>` + text + `</w:t></w:r></w:p></w:body></w:document>`
	if _, err := w.Write([]byte(xml)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestAirgapGatedTools_DryRun_NeverLeaksContent covers the 5 tools
// found missing in review: get_memory, get_memory_all, get_workflow,
// read_file, read_document_layer — each must return the SAME air-gap
// message chat_completion/get_full_context do, never the real content,
// while dry_run is active.
func TestAirgapGatedTools_DryRun_NeverLeaksContent(t *testing.T) {
	root, project := writeAirgapProject(t, "http://127.0.0.1:0", map[string]any{"dry_run": true})
	if err := os.WriteFile(filepath.Join(root, "projects", project, "memory.md"), []byte("TOP_SECRET_CONTEXT_MARKER_98213"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeMinimalDocx(t, filepath.Join(root, "repo", "secret.docx"), "TOP_SECRET_CONTEXT_MARKER_98213")

	cases := []struct {
		tool string
		args map[string]any
	}{
		{"get_memory", map[string]any{"project": project}},
		{"get_memory_all", map[string]any{"project": project}},
		{"get_workflow", map[string]any{"project": project}},
		{"read_file", map[string]any{"project": project, "filename": "secret.txt"}},
		{"read_document_layer", map[string]any{"project": project, "filename": "secret.docx"}},
	}
	for _, c := range cases {
		t.Run(c.tool, func(t *testing.T) {
			text := callTool(t, root, c.tool, c.args)
			if strings.Contains(text, "TOP_SECRET_CONTEXT_MARKER_98213") {
				t.Fatalf("CONTEXT LEAK in %q during dry_run:\n%s", c.tool, text)
			}
			if !strings.Contains(text, "MOVA EGRESS AUDIT") {
				t.Fatalf("%q: expected the air-gap message, got: %s", c.tool, text)
			}
		})
	}
}

// TestAirgapGatedTools_NoDryRun_ReturnsContentNormally is the negative
// control: without dry_run, all 5 tools behave exactly as before.
func TestAirgapGatedTools_NoDryRun_ReturnsContentNormally(t *testing.T) {
	root, project := writeAirgapProject(t, "http://127.0.0.1:0", nil)
	if err := os.WriteFile(filepath.Join(root, "projects", project, "memory.md"), []byte("TOP_SECRET_CONTEXT_MARKER_98213"), 0o644); err != nil {
		t.Fatal(err)
	}

	text := callTool(t, root, "get_memory", map[string]any{"project": project})
	if !strings.Contains(text, "TOP_SECRET_CONTEXT_MARKER_98213") {
		t.Fatalf("expected real memory content with no dry_run, got: %s", text)
	}
}

// TestSearchContext_IsNeverAirgapGated documents the deliberate scope
// boundary: search_context has no "project" argument and searches
// Mova's shared knowledge base, not a project's private context, so
// dry_run never touches it, even when a project happens to be
// dry_run-active.
func TestSearchContext_IsNeverAirgapGated(t *testing.T) {
	if airgapGatedTools["search_context"] {
		t.Fatal("search_context must stay out of airgapGatedTools — see its doc comment in server.go")
	}
}

// dry_run=false AND no llm_profile → Mova never calls a provider, and
// returns the governed context so the MCP HOST can run inference.
func TestChatCompletion_NoLLMProfile_DelegatesToHost(t *testing.T) {
	srv, calls := countingOllama(t)
	defer srv.Close()
	// No "llm_profile" key at all in the project — writeAirgapProject
	// never sets one; the project only relies on the globally active
	// provider (set up by writeAirgapProject for OTHER tests) — which
	// must NEVER be reached here.
	root, project := writeAirgapProject(t, srv.URL, nil)

	text := callTool(t, root, "chat_completion", map[string]any{"project": project, "message": "resume el contexto"})
	if !strings.Contains(text, "TOP_SECRET_CONTEXT_MARKER_98213") {
		t.Fatalf("expected the governed context to be returned for host-delegated inference, got: %s", text)
	}
	if strings.Contains(text, "SECRET_MODEL_REPLY_MUST_NEVER_APPEAR") {
		t.Fatalf("provider reply leaked despite no llm_profile — Mova must not call any provider in this mode: %s", text)
	}
	if atomic.LoadInt32(calls) != 0 {
		t.Fatalf("provider was called %d time(s) with no llm_profile configured — must be 0", *calls)
	}
}
