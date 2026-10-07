package sanitize

import (
	"strings"
	"testing"
)

func governOpts(pii bool) GovernOptions {
	ps := defaultPolicySet("test")
	ps.Security.BlockOnAPIKey = false
	return GovernOptions{Policy: ps, PIIEnabled: pii}
}

// Code stays syntactically intact: only the secret literal is replaced.
func TestGovernBlock_CodeKeepsSyntax(t *testing.T) {
	in := "const aws=\"AKIAIOSFODNN7EXAMPLE\";\nconst password = \"hunter2hunter2hunter2\";\nfunction f(){ return 1 }\n"
	out, rep := GovernBlock("src/app.js", in, governOpts(true))
	if strings.Contains(out, "AKIAIOSFODNN7EXAMPLE") || strings.Contains(out, "hunter2hunter2") {
		t.Fatalf("secret leaked:\n%s", out)
	}
	if !strings.Contains(out, `const aws="[REDACTED_SECRET]";`) || !strings.Contains(out, `const password = "[REDACTED_SECRET]";`) || !strings.Contains(out, "function f(){ return 1 }") {
		t.Fatalf("code syntax not preserved:\n%s", out)
	}
	if rep.ShapePII != 0 {
		t.Fatalf("shape PII scorer must not run on code, masked %d", rep.ShapePII)
	}
}

// Section markers survive; names/addresses are covered via field_keys and
// propagated to other blocks (e.g. a PDF text layer).
func TestGovernFocus_FieldKeysAndMarkers(t *testing.T) {
	focus := "\n\n---\n## FOCUS\nFOCUS:clientes.json\n{\"nombre\": \"Andrea Fuentes Rojas\", \"direccion\": \"Av. Los Alerces 1234, Ñuñoa\"}\nFOCUS:ficha.pdf\nCliente: Andrea Fuentes Rojas vive en Av. Los Alerces 1234, Ñuñoa. RUT 12.345.678-5\n"
	out, _, reps := GovernFocus(focus, "", governOpts(true))
	if !strings.Contains(out, "## FOCUS") || !strings.Contains(out, "FOCUS:clientes.json") || !strings.Contains(out, "FOCUS:ficha.pdf") {
		t.Fatalf("markers changed:\n%s", out)
	}
	for _, leak := range []string{"Andrea Fuentes Rojas", "Los Alerces 1234", "12.345.678-5"} {
		if strings.Contains(out, leak) {
			t.Fatalf("leaked %q:\n%s", leak, out)
		}
	}
	if len(reps) != 2 || reps[1].KnownValues == 0 {
		t.Fatalf("expected known-value propagation into the PDF block: %+v", reps)
	}
}

func TestGovernBlock_BlocksPrivateKey(t *testing.T) {
	out, rep := GovernBlock("id_rsa", "-----BEGIN RSA PRIVATE KEY-----\nMIIabc\n-----END RSA PRIVATE KEY-----\n", governOpts(false))
	if !rep.Blocked || strings.Contains(out, "MIIabc") || rep.Rule != "security.block_on_private_key" {
		t.Fatalf("expected block, got %+v %q", rep, out)
	}
}
