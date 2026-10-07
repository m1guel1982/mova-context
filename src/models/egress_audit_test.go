package models

import (
	"encoding/json"
	"mova.local/evidence"
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
	if !sess.LastReplyWasDryRun {
		t.Fatalf("LastReplyWasDryRun = false, want true")
	}
	if !strings.Contains(reply, "MOVA EGRESS AUDIT") {
		t.Fatalf("reply = %q, want the dry-run confirmation message", reply)
	}
	if atomic.LoadInt32(calls) != 0 {
		t.Fatalf("provider was called %d time(s) during a dry run — must be 0", *calls)
	}
	if len(sess.History) != 0 {
		t.Fatalf("expected no dangling turns in History after a dry run, got %d", len(sess.History))
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

// Every provider call and every dry-run block Mova observes is appended
// to the session run's events.jsonl (append-only), never to a merged file.
func TestSession_RecordsEventsInRun(t *testing.T) {
	srv, calls := countingOllama(t)
	defer srv.Close()
	sess := newTestSession(t, srv.URL)
	run, err := evidence.Start(filepath.Join(t.TempDir(), "runs"), evidence.Manifest{Door: "chat:provider"}, []byte(sess.System))
	if err != nil {
		t.Fatal(err)
	}
	sess.Run = run
	if _, err := sess.Send("hola"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	sess.EgressAuditDryRun = true
	if _, err := sess.Send("otra"); err != nil {
		t.Fatalf("Send dry: %v", err)
	}
	if got := atomic.LoadInt32(calls); got != 1 {
		t.Fatalf("provider calls = %d, want 1", got)
	}
	data, err := os.ReadFile(filepath.Join(run.Dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], `"kind":"provider_call"`) || !strings.Contains(lines[1], `"kind":"dry_run_block"`) {
		t.Fatalf("unexpected events:\n%s", data)
	}
	if !strings.Contains(lines[0], `"input_tokens_reported":42`) {
		t.Fatalf("real usage not recorded: %s", lines[0])
	}
}
