// file.go — File Resolver y Directory Resolver.
package resolvers

import (
	"path/filepath"
	"strings"

	"mova.local/core/focus"
)

// FileResolver resuelve un target que apunta a un archivo concreto — por
// ruta exacta o, si no existe esa ruta, por nombre de archivo en cualquier
// parte del repo. Cuando el usuario nombra un archivo explícitamente, se
// entrega completo: no es "contenido no relacionado" que el compilador
// decidió incluir, fue pedido por nombre.
type FileResolver struct{}

func NewFileResolver() *FileResolver { return &FileResolver{} }

func (r *FileResolver) candidatePath(ctx focus.Context, target string) string {
	if isSymbolNotation(target) {
		return ""
	}
	target = focus.StripExact(target) // "=" no aplica a rutas: ya son exactas por nombre
	path := target
	if !filepath.IsAbs(path) {
		path = filepath.Join(ctx.RepoPath, target)
	}
	if !isDir(path) && readFile(path) != "" {
		return path
	}
	if !strings.ContainsAny(target, `/\`) {
		if found := findByName(ctx, ctx.RepoPath, target); found != "" && !isDir(found) {
			return found
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
		Content: readFile(path),
	}}, nil
}

// DirectoryResolver resuelve un target que apunta a un directorio. Nunca
// vuelca el contenido de los archivos — solo un índice compacto y ordenado.
type DirectoryResolver struct{}

func NewDirectoryResolver() *DirectoryResolver { return &DirectoryResolver{} }

func (r *DirectoryResolver) resolvePath(ctx focus.Context, target string) string {
	if isSymbolNotation(target) {
		return ""
	}
	target = focus.StripExact(target)
	path := target
	if !filepath.IsAbs(path) {
		path = filepath.Join(ctx.RepoPath, target)
	}
	if isDir(path) {
		return path
	}
	return ""
}

func (r *DirectoryResolver) Match(ctx focus.Context, target string) bool {
	return r.resolvePath(ctx, target) != ""
}

func (r *DirectoryResolver) Resolve(ctx focus.Context, target string) ([]focus.ContextBlock, error) {
	path := r.resolvePath(ctx, target)
	if path == "" {
		return nil, focus.ErrNotFound
	}
	return []focus.ContextBlock{{
		Source:  relOrBase(ctx.RepoPath, path),
		Kind:    "dir-index",
		Content: listDir(path),
	}}, nil
}

// isSymbolNotation reconoce la notación "()" que workflow.md usa para pedir
// explícitamente un símbolo (función/método) en vez de un archivo.
func isSymbolNotation(target string) bool {
	return strings.HasSuffix(target, "()")
}

// stripSymbolNotation quita el sufijo "()" si está presente.
func stripSymbolNotation(target string) string {
	return strings.TrimSuffix(target, "()")
}
