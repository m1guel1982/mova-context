package documents

import (
	"fmt"
	"os"
	"strings"
)

// GenerateVectorGraphic writes svgCode to path after a minimal sanity check
// (must be a well-formed <svg>...</svg> root) — for engineering diagrams and
// architecture maps, per the original tool contract.
func GenerateVectorGraphic(path, svgCode string) error {
	trimmed := strings.TrimSpace(svgCode)
	lower := strings.ToLower(trimmed)
	if !strings.HasPrefix(lower, "<svg") && !strings.HasPrefix(lower, "<?xml") {
		return fmt.Errorf("generate_vector_graphic: el contenido no comienza con <svg> o <?xml>")
	}
	if !strings.Contains(lower, "</svg>") {
		return fmt.Errorf("generate_vector_graphic: falta la etiqueta de cierre </svg>")
	}
	return os.WriteFile(path, []byte(trimmed+"\n"), 0o644)
}

// svgWriter adapts GenerateVectorGraphic to IFileWriter for SaveService's
// WriterFactory (".svg" — see save_service.go).
type svgWriter struct{}

func (svgWriter) Write(path string, opts SaveOptions) error {
	return GenerateVectorGraphic(path, opts.Content)
}

func init() { RegisterWriter(".svg", svgWriter{}) }
