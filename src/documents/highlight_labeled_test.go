package documents

import (
	"strings"
	"testing"
)

// Regresión: las cercas con destino (```js:ruta::f()) no deben tomarse por «sin etiqueta».
func TestAutoTagCodeFencesLeavesLabeledBlocksIntact(t *testing.T) {
	in := "Propuesta.\n\n```javascript:src/plan.js::normalizarPedido()\nfunction a() {\n  return 1;\n}\n```\n\ntexto\n\n```javascript:src/cobros.js\nconst b = 2;\n```\n"
	if got := AutoTagCodeFences(in); got != in {
		t.Fatalf("labeled fences must pass through untouched, got:\n%s", got)
	}
	untagged := "```\nfunction x() {\n  return 1;\n}\n```"
	if got := AutoTagCodeFences(untagged); !strings.HasPrefix(got, "```javascript") {
		t.Fatalf("untagged fences are still auto-tagged, got %q", got)
	}
}
