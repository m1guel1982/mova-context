// tokencount.go — envoltura fina sobre github.com/tiktoken-go/tokenizer.
// Todo el cálculo ocurre en memoria, en esta máquina: la librería es un
// tokenizador embebido (vocabularios compilados en el binario), nunca hace
// una llamada de red. Ver docs/i18n/es/COMMANDS.md#mova-budget para el
// texto de aviso obligatorio sobre la naturaleza estimada del resultado.
package budget

import (
	"strings"

	"github.com/tiktoken-go/tokenizer"
)

// CountTokens cuenta los tokens de text usando tiktoken-go. Si modelHint
// coincide con un modelo OpenAI conocido (p. ej. "gpt-4o", "gpt-5") usa su
// encoding real; en cualquier otro caso (Claude, Gemini, Ollama, o
// cualquier nombre no reconocido) usa cl100k_base como aproximación
// universal razonable — la MISMA estimación de tokens se aplica luego a
// todos los proveedores en EstimateCost, precisamente porque no existe un
// tokenizador local para Claude o Gemini: es una aproximación declarada,
// nunca un valor exacto por proveedor.
func CountTokens(text, modelHint string) (count int, encoding string, err error) {
	codec, label := resolveCodec(modelHint)
	n, err := codec.Count(text)
	if err != nil {
		return 0, label, err
	}
	return n, label, nil
}

func resolveCodec(modelHint string) (tokenizer.Codec, string) {
	if modelHint != "" {
		if codec, err := tokenizer.ForModel(tokenizer.Model(strings.ToLower(modelHint))); err == nil {
			return codec, string(codec.GetName())
		}
	}
	// Fallback universal: cl100k_base cubre GPT-3.5/GPT-4 y es la
	// aproximación más común citada para estimar tokens de otros
	// proveedores cuando no hay tokenizador local propio disponible.
	codec, _ := tokenizer.Get(tokenizer.Cl100kBase)
	return codec, string(codec.GetName())
}
