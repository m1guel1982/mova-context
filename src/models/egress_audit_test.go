package models

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// countingOllama is like fakeOllama but exposes how many times /api/chat
// was actually hit — the one fact these tests need to prove: a dry run
// (or a failed audit write) must reach the provider ZERO times.
func countingOllama(t *testing.T) (*httptest.Server, *int32) {
	t.Helper()
	var calls int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"message":           map[string]string{"role": "assistant", "content": "hola desde " + req["model"].(string)},
			"done":              true,
			"prompt_eval_count": 42,
			"eval_count":        7,
		})
	})
	return httptest.NewServer(mux), &calls
}

func newTestSession(t *testing.T, srvURL string) *Session {
	t.Helper()
	root := setupProject(t, srvURL)
	if err := SetActiveProvider(root, "ollama"); err != nil {
		t.Fatal(err)
	}
	sess, err := NewSession(root)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if err := sess.SetModel("llama3.1"); err != nil {
		t.Fatalf("SetModel: %v", err)
	}
	sess.SetSystem("contexto ya sanitizado de prueba")
	return sess
}

// TestWriteEgressAuditLog_CreatesDirsAndAppends covers: creates missing
// directories, never overwrites a previous run, and each execution is
// clearly delimited with its own execution_id.
func TestWriteEgressAuditLog_CreatesDirsAndAppends(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".mova", "egress_sanitized.log")

	if err := writeEgressAuditLog(target, "primer contexto sanitizado"); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := writeEgressAuditLog(target, "segundo contexto sanitizado"); err != nil {
		t.Fatalf("second write: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading %s: %v", target, err)
	}
	content := string(data)

	if !strings.Contains(content, "primer contexto sanitizado") || !strings.Contains(content, "segundo contexto sanitizado") {
		t.Fatalf("expected BOTH executions present (append, not overwrite); got:\n%s", content)
	}
	if strings.Count(content, "## Egress audit — execution") != 2 {
		t.Fatalf("expected 2 clearly delimited execution blocks, got:\n%s", content)
	}
	if !strings.Contains(content, "execution_id:") || !strings.Contains(content, "timestamp:") {
		t.Fatalf("expected execution_id and timestamp in every block, got:\n%s", content)
	}
}

// TestSession_DryRun_NeverCallsProvider is the core contract of
// section 4: dry_run=true completes sanitization/audit-logging, never
// calls the LLM provider, and returns a successful reply saying so.
func TestSession_DryRun_NeverCallsProvider(t *testing.T) {
	srv, calls := countingOllama(t)
	defer srv.Close()

	sess := newTestSession(t, srv.URL)
	sess.EgressAuditDryRun = true

	reply, err := sess.Send("hola")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if reply != DryRunReply {
		t.Fatalf("reply = %q, want the dry-run confirmation message", reply)
	}
	if atomic.LoadInt32(calls) != 0 {
		t.Fatalf("provider was called %d time(s) during a dry run — must be 0", *calls)
	}
	if len(sess.History) != 0 {
		t.Fatalf("expected no dangling turns in History after a dry run, got %d", len(sess.History))
	}
}

// TestSession_DryRun_AlsoWritesAuditLog: dry_run and output_file can be
// combined — the sanitized context still gets logged even though no
// inference happens.
func TestSession_DryRun_AlsoWritesAuditLog(t *testing.T) {
	srv, calls := countingOllama(t)
	defer srv.Close()

	sess := newTestSession(t, srv.URL)
	sess.EgressAuditDryRun = true
	sess.EgressAuditOutputFile = filepath.Join(t.TempDir(), "egress_sanitized.md")

	if _, err := sess.Send("hola"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	data, err := os.ReadFile(sess.EgressAuditOutputFile)
	if err != nil {
		t.Fatalf("expected audit log to exist: %v", err)
	}
	if !strings.Contains(string(data), "contexto ya sanitizado de prueba") {
		t.Fatalf("expected the sanitized context (Session.System) in the log, got:\n%s", data)
	}
	if atomic.LoadInt32(calls) != 0 {
		t.Fatalf("provider was called during a dry run with logging enabled")
	}
}

// TestSession_NormalRun_LogsThenCallsProvider: with dry_run=false
// (default), egress_audit only observes — the provider call still
// happens exactly as before.
func TestSession_NormalRun_LogsThenCallsProvider(t *testing.T) {
	srv, calls := countingOllama(t)
	defer srv.Close()

	sess := newTestSession(t, srv.URL)
	sess.EgressAuditOutputFile = filepath.Join(t.TempDir(), "egress_sanitized.md")

	reply, err := sess.Send("hola")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.Contains(reply, "hola desde") {
		t.Fatalf("expected a real provider reply, got %q", reply)
	}
	if atomic.LoadInt32(calls) != 1 {
		t.Fatalf("expected exactly 1 provider call, got %d", *calls)
	}
	data, _ := os.ReadFile(sess.EgressAuditOutputFile)
	if !strings.Contains(string(data), "contexto ya sanitizado de prueba") {
		t.Fatalf("expected the sanitized context logged before the provider call, got:\n%s", data)
	}
}

// TestSession_EgressAuditWriteFailure_BlocksProviderCall covers
// section 3's hard requirement: if the configured file can't be
// created/written, Send must return an error and NEVER reach the LLM.
func TestSession_EgressAuditWriteFailure_BlocksProviderCall(t *testing.T) {
	srv, calls := countingOllama(t)
	defer srv.Close()

	sess := newTestSession(t, srv.URL)
	// A regular FILE where a directory is expected makes MkdirAll fail
	// deterministically on every OS, without needing real permission
	// tricks (which don't behave consistently across CI environments).
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	sess.EgressAuditOutputFile = filepath.Join(blocker, "egress_sanitized.md")

	_, err := sess.Send("hola")
	if err == nil {
		t.Fatalf("expected an error when the audit log can't be written")
	}
	if atomic.LoadInt32(calls) != 0 {
		t.Fatalf("provider was called %d time(s) despite the audit write failing — must be 0", *calls)
	}
	if len(sess.History) != 0 {
		t.Fatalf("expected the failed turn to be rolled back from History, got %d entries", len(sess.History))
	}
}

// TestSession_EgressAuditDisabledByDefault is the compatibility
// requirement of section 5: with no egress_audit configured at all
// (zero-value Session fields), behavior is 100% unchanged.
func TestSession_EgressAuditDisabledByDefault(t *testing.T) {
	srv, calls := countingOllama(t)
	defer srv.Close()

	sess := newTestSession(t, srv.URL)
	// sess.EgressAuditDryRun / EgressAuditOutputFile left at zero value.

	reply, err := sess.Send("hola")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.Contains(reply, "hola desde") {
		t.Fatalf("expected the normal provider reply, got %q", reply)
	}
	if atomic.LoadInt32(calls) != 1 {
		t.Fatalf("expected exactly 1 provider call, got %d", *calls)
	}
}
