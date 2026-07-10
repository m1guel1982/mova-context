// Package runtime resuelve la raíz del proyecto Mova Context (el
// directorio que contiene workflow.md) y hace auto-detección del proyecto
// cuando hay uno solo bajo projects/. Extraído de cli/main.go — es lógica
// reutilizable por cualquier entrypoint (CLI, HTTP, MCP, una futura GUI),
// ninguno de los cuales debería reimplementarla.
package runtime

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindRoot resuelve la raíz del proyecto. Se usa desde cualquier
// entrypoint (CLI, HTTP, MCP, futura GUI) — nunca reimplementar esta
// búsqueda en otro lado.
//
// Orden de resolución:
//
//  1. MOVA_PROJECT_PATH — override directo, sin búsqueda. Pensado para
//     cuando quien invoca (p.ej. la config de un cliente MCP) ya conoce
//     la ruta exacta y quiere evitarse la subida de directorios.
//  2. Búsqueda de workflow.md subiendo desde cada uno de estos puntos de
//     partida, en orden, hasta encontrarlo:
//     a) MOVA_PROJECT_ROOT, si está definida
//     b) el directorio de trabajo actual (uso normal del CLI)
//     c) el directorio del binario (mova mcp start lanzado por un
//     cliente MCP casi siempre arranca con un cwd que no tiene
//     relación con el proyecto — Claude Desktop y Cursor son el
//     caso típico — así que el directorio del ejecutable es el
//     último recurso razonable antes de rendirse)
//
// Ningún comportamiento existente cambia: si ya funcionabas ejecutando
// mova desde dentro del árbol del proyecto, (b) sigue resolviendo
// exactamente igual que antes.
func FindRoot() (string, error) {
	if p := os.Getenv("MOVA_PROJECT_PATH"); p != "" {
		return filepath.Clean(p), nil
	}

	var starts []string
	if envRoot := os.Getenv("MOVA_PROJECT_ROOT"); envRoot != "" {
		starts = append(starts, filepath.Clean(envRoot))
	}
	if cwd, err := os.Getwd(); err == nil {
		starts = append(starts, filepath.Clean(cwd))
	}
	if exe, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(filepath.Clean(exe)))
	}

	for _, start := range starts {
		if dir, ok := searchUpward(start); ok {
			return dir, nil
		}
	}

	return "", fmt.Errorf(
		"workflow.md not found.\n" +
			"Search locations:\n" +
			"  • MOVA_PROJECT_ROOT (or MOVA_PROJECT_PATH for a direct path)\n" +
			"  • current working directory\n" +
			"  • the mova binary's directory\n\n" +
			"Run mova from inside the project, or set MOVA_PROJECT_ROOT / MOVA_PROJECT_PATH " +
			"(useful when configuring mova as an MCP server in Claude Desktop / Cursor).",
	)
}

// searchUpward sube desde start hasta encontrar workflow.md.
func searchUpward(start string) (string, bool) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "workflow.md")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// AutoDetect devuelve el nombre del proyecto si hay exactamente uno bajo
// projects/, o "" si hay cero o más de uno (ambiguo — el usuario debe
// especificarlo explícitamente).
func AutoDetect(root string) string {
	entries, _ := os.ReadDir(filepath.Join(root, "projects"))
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 1 {
		return names[0]
	}
	return ""
}
