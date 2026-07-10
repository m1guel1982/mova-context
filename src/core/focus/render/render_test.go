package render

import "testing"

func TestDedupParagraphs_RemovesExactRepeat(t *testing.T) {
	seen := map[string]bool{}
	first := "Este es un párrafo de ejemplo.\n\nOtro párrafo distinto."
	out1, removed1 := dedupParagraphs(first, seen)
	if removed1 != 0 {
		t.Fatalf("primera pasada no debería quitar nada, quitó %d", removed1)
	}
	if out1 != first {
		t.Fatalf("primera pasada no debería cambiar el texto: got %q", out1)
	}

	// Segunda vez con el MISMO primer párrafo (espacios distintos, mismo
	// texto normalizado) — debe detectarse y quitarse.
	second := "Este es un párrafo    de ejemplo.\n\nUn párrafo nuevo, no visto antes."
	out2, removed2 := dedupParagraphs(second, seen)
	if removed2 != 1 {
		t.Fatalf("esperaba 1 párrafo duplicado quitado, quitó %d (out=%q)", removed2, out2)
	}
	if out2 == second {
		t.Fatalf("el párrafo duplicado debería haberse quitado del resultado")
	}
}

func TestDedupParagraphs_NeverTouchesDistinctText(t *testing.T) {
	seen := map[string]bool{}
	a := "Párrafo A."
	b := "Párrafo B, completamente distinto."
	if _, removed := dedupParagraphs(a, seen); removed != 0 {
		t.Fatalf("no debería quitar nada en la primera pasada")
	}
	if _, removed := dedupParagraphs(b, seen); removed != 0 {
		t.Fatalf("párrafos distintos nunca deben contarse como duplicados, quitó %d", removed)
	}
}
