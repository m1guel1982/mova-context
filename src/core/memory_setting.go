// memory_setting.go — el campo "memory" de project.json y la ruta de
// memory.md. Reglas:
//
//	ausente / false      → NO se registra memoria automática
//	true                 → memory.md junto a project.json
//	"<ruta>"             → esa ruta (archivo, o carpeta → <carpeta>/memory.md)
//
// La ruta es multiplataforma, con las mismas reglas que "memory_path" y
// "egress_audit.output_file": C:\, D:\, E:\ (Windows), /mnt, /home, /Volumes
// (Linux/macOS), \\servidor\recurso (red, solo en Windows), "~/..." y rutas
// relativas (a la carpeta del proyecto). Precedencia de ubicación:
// "memory" (ruta) > "memory_path" > projects/<proyecto>/memory.md.
package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mova.local/documents"
	"mova.local/mpaths"
)

// MemorySetting interpreta el campo "memory": enabled (¿registrar?) y la
// ruta explícita ("" = la de por defecto). Un valor ilegible = apagado.
func (p *Project) MemorySetting() (enabled bool, path string) {
	raw := bytes.TrimSpace(p.Memory)
	if len(raw) == 0 || string(raw) == "null" {
		return false, ""
	}
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		return b, ""
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return false, ""
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "false", "no", "off", "0":
		return false, ""
	case "true", "yes", "si", "sí", "on", "1":
		return true, ""
	}
	return true, strings.TrimSpace(s)
}

// MemoryEnabled: ¿el proyecto pidió registro automático de memoria?
func MemoryEnabled(p *Project) bool {
	if p == nil {
		return false
	}
	on, _ := p.MemorySetting()
	return on
}

func defaultMemoryPath(root, project string) string {
	return filepath.Join(mpaths.ProjectsDir(root), project, "memory.md")
}

// resolveMemoryValue convierte el valor configurado en una ruta de archivo.
func resolveMemoryValue(root, project, v string) (string, error) {
	isDir := strings.HasSuffix(v, "/") || strings.HasSuffix(v, `\`)
	switch {
	case v == "~" || strings.HasPrefix(v, "~/") || strings.HasPrefix(v, `~\`):
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("memory: no se pudo resolver %q: %w", v, err)
		}
		v = filepath.Join(home, strings.TrimLeft(strings.TrimPrefix(v, "~"), `/\`))
	case documents.IsAbsCrossPlatform(v):
		n, err := documents.NormalizeAbsPath(v)
		if err != nil {
			return "", fmt.Errorf("memory: %w", err) // p.ej. "C:\..." en un servidor Linux: error claro, no un archivo fantasma
		}
		v = n
	default: // relativa → carpeta del proyecto (donde está project.json)
		v = filepath.Join(mpaths.ProjectsDir(root), project, filepath.FromSlash(strings.ReplaceAll(v, `\`, "/")))
	}
	if st, err := os.Stat(v); isDir || (err == nil && st.IsDir()) || (err != nil && filepath.Ext(v) == "") {
		v = filepath.Join(v, "memory.md")
	}
	return v, nil
}

// ResolveMemoryTarget devuelve la ruta real de memory.md del proyecto y,
// si el valor configurado no se puede usar en este sistema, un error.
func ResolveMemoryTarget(root, project string) (string, error) {
	def := defaultMemoryPath(root, project)
	data, err := os.ReadFile(ProjectJSONPath(root, project))
	if err != nil {
		return def, nil
	}
	var p Project
	if json.Unmarshal(data, &p) != nil {
		return def, nil
	}
	if _, v := p.MemorySetting(); v != "" {
		return resolveMemoryValue(root, project, v)
	}
	if p.MemoryPath == "" {
		return def, nil
	}
	if documents.IsAbsCrossPlatform(p.MemoryPath) { // "memory_path" (campo histórico): comportamiento sin cambios
		if n, err := documents.NormalizeAbsPath(p.MemoryPath); err == nil {
			return n, nil
		}
		return p.MemoryPath, nil
	}
	return filepath.Join(root, p.MemoryPath), nil
}

// MemoryPath: como ResolveMemoryTarget, sin error (cae al valor por
// defecto). Lo usan lecturas y herramientas que no pueden fallar.
func MemoryPath(root, project string) string {
	p, err := ResolveMemoryTarget(root, project)
	if err != nil {
		return defaultMemoryPath(root, project)
	}
	return p
}

// memoryArchiveDir: carpeta de archivos mensuales, junto a memory.md.
func memoryArchiveDir(memPath string) string {
	return filepath.Join(filepath.Dir(memPath), "memory-archive")
}
