// Package documents implements the "Documentos Avanzados y Formatos de
// Oficina" and "Generación de Medios" capability groups: reading PDF/DOCX/
// XLSX text layers, generating PDF/DOCX/XLSX files, SVG vector graphics, and
// routing image prompts to a local diffusion server. It deliberately lives
// outside core/ (same reasoning as adapters/): anything that needs a
// third-party driver or reaches out to a local network service is kept out
// of the zero-dependency engine on purpose.
package documents

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolvePath resolves filename relative to the project's repo directory —
// the same "generación de código y archivos de proyecto: siempre en la ruta
// indicada por repo" rule workflow.md already defines for source code.
// Creates the repo directory if it doesn't exist yet.
func ResolvePath(root, repo, filename string) (string, error) {
	var repoDir string
	switch {
	case repo == "" || repo == ".":
		repoDir = root
	case filepath.IsAbs(repo):
		repoDir = repo
	default:
		repoDir = filepath.Join(root, repo)
	}
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		return "", fmt.Errorf("no se pudo crear el directorio de trabajo %q: %w", repoDir, err)
	}
	full := filepath.Join(repoDir, filename)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}
	return full, nil
}
