package sanitize

import "testing"

func TestMaskPII_Deterministic(t *testing.T) {
	policy := DefaultPIIPolicy()
	text := "RUT del cliente: 18.234.567-9 contacto juan.perez@correo.cl"

	out1, stats1 := MaskPII(text, policy)
	out2, stats2 := MaskPII(text, policy)

	if out1 != out2 {
		t.Fatalf("MaskPII is not deterministic: %q vs %q", out1, out2)
	}
	if stats1 != stats2 {
		t.Fatalf("PIIStats differ across identical runs: %+v vs %+v", stats1, stats2)
	}
	if stats1.TokensMasked == 0 {
		t.Fatalf("expected at least one masked token for RUT/email-shaped input, got 0 (out=%q)", out1)
	}
}

func TestMaskPII_SameInputSamePseudonym(t *testing.T) {
	policy := DefaultPIIPolicy()
	// The same RUT-shaped value repeated twice must produce the exact
	// same [PII_xxxxxxxx] tag both times — that's the whole point of
	// pseudonymization (a reader can tell "same person mentioned twice"
	// without ever seeing the original value).
	text := "18.234.567-9 ... 18.234.567-9"
	out, _ := MaskPII(text, policy)

	first := extractFirstTag(t, out)
	if first == "" {
		t.Fatalf("expected a [PII_...] tag in output, got %q", out)
	}
	occurrences := countOccurrences(out, first)
	if occurrences != 2 {
		t.Fatalf("expected the same pseudonym twice, found %d occurrences in %q", occurrences, out)
	}
}

func TestMaskPII_EmptyInput(t *testing.T) {
	out, stats := MaskPII("", DefaultPIIPolicy())
	if out != "" || stats != (PIIStats{}) {
		t.Fatalf("empty input must be a no-op, got out=%q stats=%+v", out, stats)
	}
}

func TestMaskPII_ShortWordsUntouched(t *testing.T) {
	// Ordinary short prose words (below MinTokenLength, no digits, no
	// separators) must never be masked — this is what keeps the
	// algorithm from destroying normal sentences.
	policy := DefaultPIIPolicy()
	text := "el uso es interno y limitado"
	out, stats := MaskPII(text, policy)
	if out != text {
		t.Fatalf("expected plain short-word prose untouched, got %q", out)
	}
	if stats.TokensMasked != 0 {
		t.Fatalf("expected 0 masked tokens for plain prose, got %d", stats.TokensMasked)
	}
}

func TestMaskPII_Idempotent(t *testing.T) {
	// Running MaskPII twice in a row must not re-mask an already-masked
	// tag into a second layer of pseudonymization.
	policy := DefaultPIIPolicy()
	text := "18.234.567-9"
	once, _ := MaskPII(text, policy)
	twice, _ := MaskPII(once, policy)
	if once != twice {
		t.Fatalf("MaskPII is not idempotent: once=%q twice=%q", once, twice)
	}
}

func extractFirstTag(t *testing.T, s string) string {
	t.Helper()
	start := indexOf(s, "[PII_")
	if start < 0 {
		return ""
	}
	end := indexOf(s[start:], "]")
	if end < 0 {
		return ""
	}
	return s[start : start+end+1]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func countOccurrences(s, sub string) int {
	count := 0
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			count++
			i += len(sub) - 1
		}
	}
	return count
}
