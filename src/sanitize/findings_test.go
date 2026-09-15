// findings_test.go — validates the DETECT vs SANITIZE vs BLOCK
// distinction (prompt requirement: no opaque "PII Masking: ON", every
// state change must be explainable) and the state-machine transition
// rules in state.go.
package sanitize

import "testing"

func TestEvaluateFile_AllowedWhenNothingDetected(t *testing.T) {
	ps := defaultPolicySet("test")
	f, out, state := EvaluateFile("docs/readme.md", "Just plain documentation text.", 10, ps)
	if f.Action != "ALLOWED" || state != StateAllowed {
		t.Fatalf("want ALLOWED, got action=%s state=%s", f.Action, state)
	}
	if out != "Just plain documentation text." {
		t.Fatalf("ALLOWED must never modify content")
	}
	if len(f.PatternsDetected) != 0 {
		t.Fatalf("expected no patterns detected")
	}
}

func TestEvaluateFile_SanitizedOnGenericCredential(t *testing.T) {
	ps := defaultPolicySet("test")
	content := `api_key = "abcdefghijklmnopqrstuvwxyz123456"`
	f, out, state := EvaluateFile("config/example.yaml", content, 20, ps)
	if f.Action != "SANITIZED" || state != StateSanitized {
		t.Fatalf("want SANITIZED, got action=%s state=%s", f.Action, state)
	}
	if out == content {
		t.Fatalf("SANITIZED must change the content")
	}
	if len(f.PatternsDetected) == 0 {
		t.Fatalf("expected at least one pattern recorded")
	}
}

func TestEvaluateFile_BlockedOnPrivateKey(t *testing.T) {
	ps := defaultPolicySet("test")
	content := "-----BEGIN RSA PRIVATE KEY-----\nMIIBogIBAAKCAQ==\n-----END RSA PRIVATE KEY-----"
	f, out, state := EvaluateFile("secrets/id_rsa", content, 30, ps)
	if f.Action != "BLOCKED" || state != StateBlocked {
		t.Fatalf("want BLOCKED, got action=%s state=%s", f.Action, state)
	}
	if out != "" {
		t.Fatalf("BLOCKED must never let content through")
	}
	if f.FinalTokens != 0 {
		t.Fatalf("BLOCKED must contribute zero tokens to the final context")
	}
	if f.RuleID == "" {
		t.Fatalf("BLOCKED must always cite the deciding rule")
	}
}

func TestEvaluateFile_BlockedOnPIIExternalModelDeny(t *testing.T) {
	ps := defaultPolicySet("test")
	ps.PIIExternalModel = "deny"
	ps.PII.MinScore = 0.01 // force a match for the test's simple token
	content := "user@example-domain-value.internal 04-1234-5678"
	f, _, state := EvaluateFile("tests/fixtures/users.json", content, 40, ps)
	if state != StateBlocked {
		t.Fatalf("want BLOCKED under pii.external_model=deny, got %s (patterns=%v)", state, f.PatternsDetected)
	}
}

func TestDetectSecrets_JWTShape(t *testing.T) {
	jwt := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PYE"
	hits := DetectSecrets("token: " + jwt)
	found := false
	for _, h := range hits {
		if h.Type == SecretJWT {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a JWT-like hit, got %v", hits)
	}
}

func TestIsAssetPath_MultiDotFilename(t *testing.T) {
	// Regression test for a real bug found while testing against
	// fastapi/fastapi: a multi-dot filename like
	// "https01.drawio.svg" was NOT recognized as an asset because
	// extLower returned ".drawio.svg" instead of ".svg" - see
	// findings_classify.go's extLower fix.
	cases := []string{
		"docs/img/https01.drawio.svg",
		"a.b.c.png",
		"style.min.css",
		"bundle.min.js",
	}
	for _, p := range cases {
		if !IsAssetPath(p) {
			t.Errorf("expected %q to be recognized as an asset path", p)
		}
	}
	if IsAssetPath("auth/login.go") {
		t.Errorf("a regular source file must never be treated as an asset")
	}
}

func TestStateMachine_TransitionsAreOneDirectional(t *testing.T) {
	if !IsValidTransition(StateDiscovered, StateCandidate) {
		t.Fatalf("DISCOVERED -> CANDIDATE must be valid")
	}
	if !IsValidTransition(StateCandidate, StateBlocked) {
		t.Fatalf("CANDIDATE -> BLOCKED must be valid")
	}
	if IsValidTransition(StateBlocked, StateCandidate) {
		t.Fatalf("BLOCKED must be terminal (no transition back to CANDIDATE)")
	}
	if IsValidTransition(StateDiscovered, StateAllowed) {
		t.Fatalf("DISCOVERED must go through CANDIDATE first")
	}
}

func TestStateCounts_ReductionPercent(t *testing.T) {
	c := StateCounts{DiscoveredTokens: 1000, AllowedTokens: 80, SanitizedTokens: 20}
	if got := c.FinalSendableTokens(); got != 100 {
		t.Fatalf("want 100 final sendable tokens, got %d", got)
	}
	if got := c.ReductionPercent(); got < 89.9 || got > 90.1 {
		t.Fatalf("want ~90%% reduction, got %.2f", got)
	}
}
