package resolvers

import (
	"os"
	"path/filepath"
	"strings"

	"mova.local/core/focus"
)

// -----------------------------------------------------------------------------
// Glob Resolver
// -----------------------------------------------------------------------------

type GlobResolver struct{}

func NewGlobResolver() focus.Resolver { return &GlobResolver{} }

// isGlobPattern reports whether target carries glob syntax ("*", "?", "[")
// or is "." — shorthand for "todo, recursivamente, desde la raíz del
// repo". "." no tiene metacaracteres propios pero se resuelve igual que
// "**/*", así que pertenece al Glob Resolver en vez de caer a
// File/Directory (que de otro modo lo tratarían como listado de un
// directorio literal llamado ".").
func isGlobPattern(target string) bool {
	if target == "." {
		return true
	}
	return strings.ContainsAny(target, "*?[")
}

func (r *GlobResolver) Match(ctx focus.Context, target string) bool {
	return isGlobPattern(target)
}

func (r *GlobResolver) Resolve(ctx focus.Context, target string) ([]focus.ContextBlock, error) {
	if !isGlobPattern(target) {
		return nil, focus.ErrNotFound
	}

	target = focus.StripExact(target)

	// "." == "recorré todo el repo" == "**/*".
	if target == "." {
		target = "**/*"
	}

	// Glob con raíz absoluta del host (multiplataforma) — p. ej.
	// "/mnt/**/*.java" o "C:\repos\**\*.go": si la porción antes del
	// primer metacaracter existe de verdad como directorio absoluto,
	// se recorre DESDE AHÍ en vez de desde la raíz del repo. Si no
	// existe (o target no tiene forma de ruta absoluta), cae al
	// comportamiento de siempre — ningún project.json existente cambia.
	walkRoot := ctx.RepoPath
	if looksAbsoluteHostPath(target) {
		norm := normalizeHostPath(target)
		if root, pattern, ok := splitAbsoluteGlobRoot(norm); ok {
			if info, err := os.Stat(root); err == nil && info.IsDir() {
				walkRoot = root
				target = pattern
			}
		}
	}

	cleanTarget := filepath.Clean(target)
	isRecursive := strings.Contains(target, "**")
	m := newExcludeMatcher(ctx.RepoPath, ctx.Exclude)

	var blocks []focus.ContextBlock

	err := filepath.WalkDir(walkRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		// Misma exclusión por defecto que walkFiles/DirectoryResolver
		// (.git, node_modules, vendor, ...), MÁS la clave "exclude" de
		// project.json — ver el comentario en DirectoryResolver.Resolve
		// para el motivo exacto. Un "." o "**/*" nunca debe volcar el
		// contenido de esas carpetas, ni de nada marcado explícitamente
		// como excluido, al contexto.
		if d.IsDir() {
			if path != walkRoot && skipDirOrExcluded(ctx, m, d.Name()) {
				ctx.RecordExcluded(path, d.Name(), 0)
				return filepath.SkipDir
			}
			return nil
		}
		if m.excludesPath(path) {
			ctx.RecordExcluded(path, d.Name(), 1)
			return nil
		}

		relPath, err := filepath.Rel(walkRoot, path)
		if err != nil {
			return nil
		}

		matched := false
		if cleanTarget == "**/*" || cleanTarget == "**" || target == "**/*" {
			matched = true
		} else if isRecursive {
			basePattern := strings.ReplaceAll(target, "**/", "")
			basePattern = strings.ReplaceAll(basePattern, "**", "*")
			matched, _ = filepath.Match(basePattern, filepath.Base(path))
			if !matched {
				// También matchear contra la ruta relativa completa,
				// para que un patrón como "src/**/*.go" siga
				// funcionando aunque el archivo esté anidado más
				// profundo que un solo nivel bajo "src".
				matched, _ = filepath.Match(basePattern, relPath)
			}
		} else {
			matched, _ = filepath.Match(cleanTarget, relPath)
			if !matched {
				matched, _ = filepath.Match(cleanTarget, filepath.Base(path))
			}
		}

		if matched {
			content := readFileText(path)
			if content != "" {
				blocks = append(blocks, focus.ContextBlock{
					Source:  relOrBase(ctx.RepoPath, path),
					Kind:    "file",
					Content: content,
				})
			}
		}

		return nil
	})

	if err != nil || len(blocks) == 0 {
		return nil, focus.ErrNotFound
	}

	return blocks, nil
}

// -----------------------------------------------------------------------------
// File Resolver
// -----------------------------------------------------------------------------

type FileResolver struct{}

func NewFileResolver() focus.Resolver { return &FileResolver{} }

func (r *FileResolver) candidatePath(ctx focus.Context, target string) string {
	if isSymbolNotation(target) || isGlobPattern(target) {
		return ""
	}
	target = focus.StripExact(target)
	m := newExcludeMatcher(ctx.RepoPath, ctx.Exclude)
	// Ruta absoluta del host (multiplataforma) — ver fsutil.go. Se
	// intenta ANTES de la resolución relativa al repo: una letra de
	// unidad Windows o UNC es inequívoca; un "/algo" estilo Unix solo
	// gana aquí si existe de verdad como archivo absoluto, si no cae al
	// comportamiento de siempre dos líneas más abajo.
	if abs, ok := resolveAbsoluteFile(target); ok {
		if m.excludesPath(abs) {
			return ""
		}
		return abs
	}
	path := repoRelativePath(ctx.RepoPath, target)
	if path != "" && !m.excludesPath(path) && !isDir(path) && readFile(path) != "" {
		return path
	}
	if !strings.ContainsAny(target, `/\`) {
		if found := findByName(ctx, ctx.RepoPath, target); found != "" && !isDir(found) {
			return found // findByName ya filtra por exclude vía walkFiles
		}
	}
	return ""
}

func (r *FileResolver) Match(ctx focus.Context, target string) bool {
	return r.candidatePath(ctx, target) != ""
}

func (r *FileResolver) Resolve(ctx focus.Context, target string) ([]focus.ContextBlock, error) {
	path := r.candidatePath(ctx, target)
	if path == "" {
		return nil, focus.ErrNotFound
	}
	return []focus.ContextBlock{{
		Source:  relOrBase(ctx.RepoPath, path),
		Kind:    "file",
		Content: readFileText(path),
	}}, nil
}

// -----------------------------------------------------------------------------
// Directory Resolver
// -----------------------------------------------------------------------------

// DirectoryResolver resuelve una referencia a un directorio — "src",
// "/src", "./src" — como el contenido RECURSIVO COMPLETO de ese
// directorio: cada archivo debajo, como su propio ContextBlock. Esto
// espeja lo que GlobResolver hace para "**/*", pero acotado a un
// subárbol puntual, de forma que "focus": ["src"] se comporta igual
// que "focus": ["src/**/*"].
type DirectoryResolver struct{}

func NewDirectoryResolver() focus.Resolver { return &DirectoryResolver{} }

func (r *DirectoryResolver) resolvePath(ctx focus.Context, target string) string {
	if isSymbolNotation(target) || isGlobPattern(target) {
		return ""
	}
	target = focus.StripExact(target)
	m := newExcludeMatcher(ctx.RepoPath, ctx.Exclude)
	// Directorio absoluto del host (multiplataforma) — ver
	// resolveAbsoluteFile's comentario arriba, misma regla, para
	// directorios en vez de archivos ("/mnt", "C:\repos").
	if abs, ok := resolveAbsoluteDir(target); ok {
		if m.excludesPath(abs) {
			return "" // el propio directorio target está en "exclude"
		}
		return abs
	}
	path := repoRelativePath(ctx.RepoPath, target)
	if path != "" && isDir(path) {
		if m.excludesPath(path) {
			return ""
		}
		return path
	}
	return ""
}

func (r *DirectoryResolver) Match(ctx focus.Context, target string) bool {
	return r.resolvePath(ctx, target) != ""
}

func (r *DirectoryResolver) Resolve(ctx focus.Context, target string) ([]focus.ContextBlock, error) {
	dirPath := r.resolvePath(ctx, target)
	if dirPath == "" {
		return nil, focus.ErrNotFound
	}
	m := newExcludeMatcher(ctx.RepoPath, ctx.Exclude)

	var blocks []focus.ContextBlock

	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		// SEGURIDAD/consistencia: un directorio de `focus` respeta la
		// MISMA exclusión por defecto que walkFiles usa para todos los
		// demás resolvers (.git, node_modules, vendor, .env-adjacent,
		// etc. — ver ctx.SkipDir/core/focus/stats.go), MÁS lo que la
		// clave "exclude" de project.json agregue (nombres, rutas
		// completas o globs — ver exclude.go) — sin esto, un
		// "focus": ["."] o ["src"] podía volcar credenciales de
		// .git/config o secretos anidados en node_modules/, o
		// cualquier ruta que el usuario haya marcado explícitamente
		// como sensible, directo al contexto que se le manda al
		// modelo.
		name := d.Name()
		if d.IsDir() {
			if path != dirPath && skipDirOrExcluded(ctx, m, name) {
				ctx.RecordExcluded(path, name, 0)
				return filepath.SkipDir
			}
			return nil
		}
		if m.excludesPath(path) {
			ctx.RecordExcluded(path, name, 1)
			return nil
		}
		content := readFileText(path)
		if content != "" {
			blocks = append(blocks, focus.ContextBlock{
				Source:  relOrBase(ctx.RepoPath, path),
				Kind:    "file",
				Content: content,
			})
		}
		return nil
	})

	if err != nil || len(blocks) == 0 {
		return nil, focus.ErrNotFound
	}

	return blocks, nil
}

// -----------------------------------------------------------------------------
// Agregación multi-target
// -----------------------------------------------------------------------------

// ResolveAll expande una lista completa de focus/memory — p. ej. el
// "focus": ["server.js", "backend-test.py"] de project.json, o una
// mezcla de archivos, globs y directorios — a TODOS los ContextBlock
// que matcheen, en vez de detenerse apenas el primer target resuelve.
// resolvers se prueba en orden para cada target (típicamente Glob,
// luego File, luego Directory); el primer resolver que haga Match para
// un target dado se queda con ese target.
//
// Los blocks se deduplican por Source: si el mismo archivo termina
// siendo alcanzado dos veces (p. ej. vía una ruta explícita Y un
// glob/directorio que también lo cubre), se incluye una sola vez.
func ResolveAll(ctx focus.Context, resolvers []focus.Resolver, targets []string) ([]focus.ContextBlock, error) {
	seen := make(map[string]bool)
	var all []focus.ContextBlock

	for _, target := range targets {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}

		for _, resolver := range resolvers {
			if !resolver.Match(ctx, target) {
				continue
			}
			blocks, err := resolver.Resolve(ctx, target)
			if err != nil {
				continue
			}
			for _, b := range blocks {
				if seen[b.Source] {
					continue
				}
				seen[b.Source] = true
				all = append(all, b)
			}
			break // ya encontramos el resolver dueño de este target; no dejamos que otro lo reclame de nuevo
		}
	}

	if len(all) == 0 {
		return nil, focus.ErrNotFound
	}
	return all, nil
}

// -----------------------------------------------------------------------------
// Helpers del paquete
// -----------------------------------------------------------------------------

func stripSymbolNotation(target string) string {
	return strings.TrimSuffix(target, "()")
}

func isSymbolNotation(target string) bool {
	return strings.HasSuffix(target, "()")
}

// repoRelativePath une target a la raíz del repo, tratando un "/"
// inicial como "relativo a la raíz del repo" y NO como "ruta absoluta
// del filesystem del host": patrones de project.json como "/src"
// describen una ruta DENTRO del proyecto, nunca una ruta absoluta del
// host. Un target vacío o "." resuelve a la raíz del repo.
//
// SEGURIDAD — contención de path traversal: un target con "../" (p.
// ej. "../../etc/passwd") NUNCA debe poder escapar repoPath a través
// de esta función — esa es responsabilidad EXCLUSIVA del soporte
// explícito de rutas absolutas del host (resolveAbsoluteFile/Dir en
// este mismo archivo), que exige que el path exista de verdad y está
// documentado como tal, nunca un efecto colateral accidental de un
// "/" o "../" relativo mal manejado. Si el join con target termina
// FUERA de repoPath, se devuelve "" — el llamador lo trata igual que
// "no encontrado", nunca lee el archivo resuelto.
func repoRelativePath(repoPath, target string) string {
	target = strings.TrimPrefix(target, "/")
	target = strings.TrimPrefix(target, `\`)
	if target == "" || target == "." {
		return repoPath
	}
	if filepath.IsAbs(target) {
		// Seguía viéndose absoluto tras sacar un separador inicial
		// (p. ej. letra de unidad en Windows) — lo tratamos igual como
		// relativo al repo en vez de confiar en él como ruta del host.
		target = strings.TrimPrefix(target, string(filepath.Separator))
	}
	joined := filepath.Join(repoPath, target)
	if rel, err := filepath.Rel(repoPath, joined); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	return joined
}
