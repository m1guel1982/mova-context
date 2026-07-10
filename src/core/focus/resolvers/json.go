// json.go — JSON Resolver.
package resolvers

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"mova.local/core/focus"
)

type JSONResolver struct{}

func NewJSONResolver() *JSONResolver { return &JSONResolver{} }

// splitTarget separates "archivo.json#dot.path" into (file, path). ok=false
// si no hay "#" o el file no termina en .json — este resolver no aplica.
func (r *JSONResolver) splitTarget(target string) (file, path string, ok bool) {
	idx := strings.Index(target, "#")
	if idx == -1 {
		return "", "", false
	}
	file, path = target[:idx], target[idx+1:]
	if !strings.HasSuffix(strings.ToLower(file), ".json") || path == "" {
		return "", "", false
	}
	return file, path, true
}

func (r *JSONResolver) resolveFilePath(ctx focus.Context, file string) string {
	path := file
	if !filepath.IsAbs(path) {
		path = filepath.Join(ctx.RepoPath, file)
	}
	if readFile(path) != "" {
		return path
	}
	if !strings.ContainsAny(file, `/\`) {
		return findByName(ctx, ctx.RepoPath, file)
	}
	return ""
}

func (r *JSONResolver) Match(ctx focus.Context, target string) bool {
	file, _, ok := r.splitTarget(target)
	if !ok {
		return false
	}
	return r.resolveFilePath(ctx, file) != ""
}

func (r *JSONResolver) Resolve(ctx focus.Context, target string) ([]focus.ContextBlock, error) {
	file, dotPath, ok := r.splitTarget(target)
	if !ok {
		return nil, focus.ErrNotFound
	}
	fp := r.resolveFilePath(ctx, file)
	if fp == "" {
		return nil, focus.ErrNotFound
	}
	var root any
	if err := json.Unmarshal([]byte(readFile(fp)), &root); err != nil {
		return nil, focus.ErrNotFound
	}
	node, ok := navigateJSON(root, strings.Split(dotPath, "."))
	if !ok {
		return nil, focus.ErrNotFound
	}
	out, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		return nil, focus.ErrNotFound
	}
	return []focus.ContextBlock{{
		Source:  relOrBase(ctx.RepoPath, fp) + "#" + dotPath,
		Kind:    "json-node",
		Content: string(out),
	}}, nil
}

// navigateJSON walks a decoded JSON value (map[string]any / []any) following
// a dot-path. Determinista: no hay ambigüedad posible, cada segmento del
// path o existe como clave de mapa o no existe.
func navigateJSON(v any, path []string) (any, bool) {
	if len(path) == 0 {
		return v, true
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	next, ok := m[path[0]]
	if !ok {
		return nil, false
	}
	return navigateJSON(next, path[1:])
}
