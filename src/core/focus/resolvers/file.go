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

func isGlobPattern(target string) bool {
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

	cleanTarget := filepath.Clean(target)
	isRecursive := strings.Contains(target, "**")

	var blocks []focus.ContextBlock

	err := filepath.WalkDir(ctx.RepoPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(ctx.RepoPath, path)
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
		} else {
			matched, _ = filepath.Match(cleanTarget, relPath)
			if !matched {
				matched, _ = filepath.Match(cleanTarget, filepath.Base(path))
			}
		}

		if matched {
			content := readFile(path)
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
	if isSymbolNotation(target) {
		return ""
	}
	target = focus.StripExact(target)
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

// -----------------------------------------------------------------------------
// Directory Resolver
// -----------------------------------------------------------------------------

type DirectoryResolver struct{}

func NewDirectoryResolver() focus.Resolver { return &DirectoryResolver{} }

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

// -----------------------------------------------------------------------------
// Helpers del paquete
// -----------------------------------------------------------------------------

func stripSymbolNotation(target string) string {
	return strings.TrimSuffix(target, "()")
}

func isSymbolNotation(target string) bool {
	return strings.HasSuffix(target, "()")
}