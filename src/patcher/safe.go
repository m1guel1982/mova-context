// safe.go — la aplicación SEGURA de bloques etiquetados, compartida por
// Chat, MCP y HTTP (ver mova.local/applyflow): contención en el repo,
// exclusiones, respaldo previo y reglas para no destruir archivos.
//
// Reglas (todas verificadas por tests):
//   - Ninguna escritura fuera del repo (rutas absolutas o ".." se rechazan).
//   - Nada de lo declarado en "exclude" se modifica (archivo o función).
//   - Un bloque de FUNCIÓN cuyo símbolo no se encuentra en un archivo que
//     ya existe se OMITE con motivo; nunca se sobrescribe el archivo con
//     un fragmento. Si el archivo no existe, se crea.
//   - Con Options.Backup, antes de sobrescribir un archivo existente se
//     copia al lado (mismo directorio) como <archivo>.mova-<sello>.bak.
package patcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mova.local/documents"
)

// Options configura ApplyBlocksOpts. Los campos vacíos desactivan la regla.
type Options struct {
	// Backup: antes de sobrescribir un archivo existente se copia AL LADO,
	// en el mismo directorio, como <archivo>.mova-<Stamp>.bak.
	Backup bool
	// Stamp: sello común de esta aplicación (p. ej. 20261003-153012); todos
	// los respaldos de una misma confirmación comparten sello. Vacío = ahora.
	Stamp string
	// Excluded: true + motivo si el destino (archivo, función) está excluido.
	Excluded func(path, symbol string) (bool, string)
}

// SkippedBlock: un bloque que NO se aplicó, y por qué.
type SkippedBlock struct{ Path, Symbol, Reason string }

// SafeTarget resuelve p dentro de root y rechaza todo lo que escape de él.
func SafeTarget(root, p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("ruta vacía")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	target := p
	if !filepath.IsAbs(target) {
		target = filepath.Join(rootAbs, filepath.FromSlash(strings.ReplaceAll(p, `\`, "/")))
	}
	target = filepath.Clean(target)
	rel, err := filepath.Rel(rootAbs, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("la ruta %q queda fuera del repositorio", p)
	}
	return target, nil
}

// BackupSuffix es la sigla del respaldo: <archivo>.mova-<sello>.bak. No
// termina en la extensión del código (.js, .go…), así que ningún bundler,
// test runner ni el propio escáner de Mova lo toma por código fuente.
func BackupName(target, stamp string) string { return target + ".mova-" + stamp + ".bak" }

// backupFile copia el archivo original al lado, en su mismo directorio.
// Si ya existe el respaldo de esta aplicación (varios bloques sobre el
// mismo archivo) conserva el ORIGINAL y no lo vuelve a escribir.
func backupFile(target, stamp string) (string, error) {
	data, err := os.ReadFile(target)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	dst := BackupName(target, stamp)
	if _, err := os.Stat(dst); err == nil {
		return dst, nil
	}
	return dst, os.WriteFile(dst, data, 0o644)
}

// ApplyBlocksOpts aplica los bloques uno a uno (cada uno es independiente:
// un fallo no detiene a los demás) y devuelve lo escrito, lo omitido y,
// si hubo omisiones/fallos, un error resumen.
func ApplyBlocksOpts(root string, blocks []documents.LabeledCodeBlock, opt Options) ([]AppliedFile, []SkippedBlock, error) {
	var applied []AppliedFile
	var skipped []SkippedBlock
	stamp := opt.Stamp
	if stamp == "" {
		stamp = time.Now().Format("20060102-150405")
	}
	skip := func(b documents.LabeledCodeBlock, why string) {
		skipped = append(skipped, SkippedBlock{Path: b.Path, Symbol: b.Symbol, Reason: why})
	}
	for _, b := range blocks {
		target, err := SafeTarget(root, b.Path)
		if err != nil {
			skip(b, err.Error())
			continue
		}
		if opt.Excluded != nil {
			if ex, why := opt.Excluded(b.Path, b.Symbol); ex {
				skip(b, why)
				continue
			}
		}
		_, statErr := os.Stat(target)
		exists := statErr == nil
		backup := ""
		if exists && opt.Backup {
			if backup, err = backupFile(target, stamp); err != nil {
				skip(b, "no se pudo respaldar el archivo original: "+err.Error())
				continue
			}
		}
		if b.Symbol == "" {
			if err := atomicWriteFile(target, b.Content); err != nil {
				skip(b, err.Error())
				continue
			}
			applied = append(applied, AppliedFile{Path: b.Path, WholeFile: true, Backup: backup})
			continue
		}
		ok, err := patchSymbol(target, b.Symbol, b.Content)
		switch {
		case err != nil:
			skip(b, err.Error())
		case ok:
			applied = append(applied, AppliedFile{Path: b.Path, Symbol: b.Symbol, Backup: backup})
		case !exists: // archivo nuevo: se crea con el bloque
			if err := atomicWriteFile(target, b.Content); err != nil {
				skip(b, err.Error())
				continue
			}
			applied = append(applied, AppliedFile{Path: b.Path, Symbol: b.Symbol, WholeFile: true})
		default:
			skip(b, fmt.Sprintf("la función %s() no se encontró con certeza en %s; el archivo NO se tocó (envía el archivo completo o corrige el nombre)", b.Symbol, b.Path))
		}
	}
	if len(skipped) == 0 {
		return applied, nil, nil
	}
	parts := make([]string, len(skipped))
	for i, s := range skipped {
		parts[i] = s.Path + ": " + s.Reason
	}
	return applied, skipped, fmt.Errorf("%d bloque(s) no aplicado(s): %s", len(skipped), strings.Join(parts, "; "))
}
