package models

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const ctxV1 = "# Mova Context — p / analizar\nGenerated: 2026-10-01 10:00 | Repo: r\n\n---\n## AGENTS\n\n<!-- agent: data-mapper-dev -->\nAGENTE v1\n\n---\n## PROMPT\n\n<!-- prompt: analizar -->\nPROMPT analizar\n\n---\n## FOCUS\nFOCUS: a.js::func=f\nCODIGO f v1\nFOCUS: b.js\nCODIGO b\n\n---\n## INSTRUCTION\nResponde.\n"

func TestEgressNoDuplicatesOnRepeat(t *testing.T) {
	f := filepath.Join(t.TempDir(), "sub", "egress_sanitized.md")
	for i := 0; i < 5; i++ { // 5 turnos de chat con el mismo contexto
		// "Generated:" cambia en cada turno y NO debe contar como cambio
		ctx := strings.Replace(ctxV1, "10:00", "10:0"+string(rune('0'+i)), 1)
		if err := WriteEgressAuditLog(f, ctx); err != nil {
			t.Fatal(err)
		}
	}
	data, _ := os.ReadFile(f)
	for _, needle := range []string{"AGENTE v1", "PROMPT analizar", "CODIGO f v1", "CODIGO b", "Responde."} {
		if n := strings.Count(string(data), needle); n != 1 {
			t.Fatalf("%q appears %d times, want exactly 1\n%s", needle, n, data)
		}
	}
	stat1, _ := os.Stat(f)
	_ = WriteEgressAuditLog(f, ctxV1)
	stat2, _ := os.Stat(f)
	if !stat1.ModTime().Equal(stat2.ModTime()) {
		t.Fatalf("an identical context must not rewrite the file")
	}
}

func TestEgressChangedBlockReplacedNewBlockAppended(t *testing.T) {
	f := filepath.Join(t.TempDir(), "e.md")
	_ = WriteEgressAuditLog(f, ctxV1)
	v2 := strings.Replace(ctxV1, "CODIGO f v1", "CODIGO f v2 cambiado", 1)
	v2 = strings.Replace(v2, "FOCUS: b.js\nCODIGO b\n", "FOCUS: b.js\nCODIGO b\nFOCUS: c.js\nCODIGO c nuevo\n", 1)
	if err := WriteEgressAuditLog(f, v2); err != nil {
		t.Fatal(err)
	}
	s, _ := os.ReadFile(f)
	got := string(s)
	if strings.Contains(got, "CODIGO f v1") || strings.Count(got, "CODIGO f v2 cambiado") != 1 {
		t.Fatalf("changed block must be replaced, not duplicated:\n%s", got)
	}
	if strings.Count(got, "CODIGO c nuevo") != 1 || strings.Count(got, "AGENTE v1") != 1 {
		t.Fatalf("new block appended once, untouched ones kept once:\n%s", got)
	}
	// el orden relativo se conserva: el bloque nuevo va al final
	if strings.Index(got, "CODIGO c nuevo") < strings.Index(got, "CODIGO b") {
		t.Fatalf("new block should be appended after existing ones")
	}
}

func TestEgressMigratesLegacyAppendFormat(t *testing.T) {
	f := filepath.Join(t.TempDir(), "legacy.md")
	one := "## Egress audit — execution abc\n\n### Sanitized context sent to the LLM\n\n```text\n"
	legacy := one + ctxV1 + "\n```\n\n" + one + ctxV1 + "\n```\n\n" + one + ctxV1 + "\n```\n"
	_ = os.WriteFile(f, []byte(legacy), 0o644)
	if err := WriteEgressAuditLog(f, ctxV1); err != nil {
		t.Fatal(err)
	}
	s, _ := os.ReadFile(f)
	if n := strings.Count(string(s), "AGENTE v1"); n != 1 {
		t.Fatalf("legacy file with 3 copies must collapse to 1, got %d", n)
	}
	// reescribir de nuevo: estable
	_ = WriteEgressAuditLog(f, ctxV1)
	s2, _ := os.ReadFile(f)
	if string(s) != string(s2) {
		t.Fatalf("second write must be a no-op")
	}
}

func TestEgressRoundTripKeepsCodeFences(t *testing.T) {
	f := filepath.Join(t.TempDir(), "fence.md")
	ctx := "# Mova Context — p / t\n\n---\n## FOCUS\nFOCUS: x.md\nantes\n```js\nconst a = 1\n```\ndespués\n"
	_ = WriteEgressAuditLog(f, ctx)
	_ = WriteEgressAuditLog(f, ctx)
	s, _ := os.ReadFile(f)
	if strings.Count(string(s), "const a = 1") != 1 || strings.Count(string(s), "después") != 1 {
		t.Fatalf("content with ``` fences must round-trip once:\n%s", s)
	}
}
