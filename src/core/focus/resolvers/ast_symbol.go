// ast_symbol.go — AST Symbol Resolver: "archivo.py::func=a(),b()" /
// "archivo.js::class=X,Y" / "archivo.go::namespace=NS" in "focus"
// (see docs/i18n/{es,en}/context-trace.md § "Filtro por símbolo
// (AST)"). This is an ADDITION alongside CodeSymbolResolver (the
// older bare-name, no-file, brace/indent-heuristic resolver just
// below this one in DefaultResolvers — see render.go), never a
// replacement: a bare "insertar_usuario()" with no "file::kind="
// prefix still goes through CodeSymbolResolver exactly as before.
// Powered by mova.local/core/focus/astfilter (real tree-sitter AST,
// not text heuristics) — see that package for which languages are
// supported and why a name that doesn't parse under a supported
// grammar falls through to the rest of the resolver cascade instead
// of erroring.
package resolvers

import (
	"strings"

	"mova.local/core/focus"
	"mova.local/core/focus/astfilter"
)

type AstSymbolResolver struct{}

func NewAstSymbolResolver() focus.Resolver { return &AstSymbolResolver{} }

// astSymbolSpec is one parsed "file::kind=name1,name2" target.
type astSymbolSpec struct {
	file  string
	kind  string
	names []string
}

// parseAstSymbolTarget recognizes "file::kind=names" — kind is any
// name astfilter.NormalizeKindName recognizes: func/method, class,
// interface, struct, enum, trait, type, namespace/package, const,
// var/variable/property, macro, impl, comment (case-insensitive). ok
// is false for anything else — including a plain "file::something"
// with an unrecognized kind, which safely falls through to the next
// resolver rather than being silently misinterpreted. The actual list
// of recognized kinds lives in astfilter (NormalizeKindName) — not
// duplicated here — precisely so adding a new kind to
// config/ast/keywords_*.json (see that package's config.go) doesn't
// also require touching this parser.
func parseAstSymbolTarget(target string) (astSymbolSpec, bool) {
	file, rest, found := strings.Cut(target, "::")
	if !found || file == "" || rest == "" {
		return astSymbolSpec{}, false
	}
	kindRaw, namesRaw, found := strings.Cut(rest, "=")
	if !found || namesRaw == "" {
		return astSymbolSpec{}, false
	}
	kind, ok := astfilter.NormalizeKindName(kindRaw)
	if !ok {
		return astSymbolSpec{}, false
	}
	var names []string
	for _, n := range strings.Split(namesRaw, ",") {
		if n = strings.TrimSpace(n); n != "" {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		return astSymbolSpec{}, false
	}
	return astSymbolSpec{file: file, kind: kind, names: names}, true
}

func (r *AstSymbolResolver) Match(ctx focus.Context, target string) bool {
	_, ok := parseAstSymbolTarget(target)
	return ok
}

func (r *AstSymbolResolver) Resolve(ctx focus.Context, target string) ([]focus.ContextBlock, error) {
	spec, ok := parseAstSymbolTarget(target)
	if !ok {
		return nil, focus.ErrNotFound
	}
	if !astfilter.Supported(spec.file) {
		// Language not covered by astfilter yet — not an error, just
		// "this resolver can't help", so the cascade can still try
		// CodeSymbolResolver/FallbackResolver.
		return nil, focus.ErrNotFound
	}
	m := newExcludeMatcher(ctx.RepoPath, ctx.Exclude)
	// Absolute host path (cross-platform: Windows drive letter, UNC
	// share, or a real Unix absolute path) — tried BEFORE repo-relative
	// resolution, same order FileResolver.candidatePath already uses
	// (see file.go). Bug fixed here: before this, an AST-filtered
	// target whose FILE part was an absolute path (e.g.
	// "C:\legacy\settings.py::var=ADMIN_EMAIL", or a UNC network
	// share) always fell straight to repoRelativePath, which silently
	// discards absolute-looking prefixes and re-joins under RepoPath
	// — meaning it could only ever resolve to a nonexistent path
	// under the repo, never the real external file, and the target
	// was reported as "not found" even when the file plainly existed.
	var path string
	if abs, ok := resolveAbsoluteFile(spec.file); ok {
		if m.excludesPath(abs) {
			return nil, focus.ErrNotFound
		}
		path = abs
	} else {
		path = repoRelativePath(ctx.RepoPath, spec.file)
		if path == "" || m.excludesPath(path) {
			return nil, focus.ErrNotFound
		}
	}
	content := readFile(path)
	if content == "" {
		return nil, focus.ErrNotFound
	}
	extracted, found := astfilter.Extract([]byte(content), spec.file, spec.kind, spec.names)
	if !found {
		return nil, focus.ErrNotFound
	}
	return []focus.ContextBlock{{
		Source:  relOrBase(ctx.RepoPath, path),
		Kind:    "ast-symbol",
		Content: extracted,
	}}, nil
}

// ApplyAstSymbolExcludes strips any "path::kind=names" entries in
// exclude that target the SAME file as source (b.Source from an
// already-resolved focus.ContextBlock) — the exclude-side counterpart
// to AstSymbolResolver: a file already included via focus stays
// included, but the named function/class/etc. is removed from it,
// instead of the older all-or-nothing "the whole file is excluded".
// Non-matching or unparseable exclude entries are ignored here (they
// are handled elsewhere, by excludeMatcher, exactly as before) — this
// function only ever REMOVES content, never adds or blocks a file.
func ApplyAstSymbolExcludes(exclude []string, source, content string) string {
	for _, e := range exclude {
		spec, ok := parseAstSymbolTarget(e)
		if !ok {
			continue
		}
		if !sameFileTarget(spec.file, source) {
			continue
		}
		if stripped, found := astfilter.Strip([]byte(content), spec.file, spec.kind, spec.names); found {
			content = stripped
		}
	}
	return content
}

// sameFileTarget compares an exclude spec's file part against a
// resolved block's Source loosely (exact match, or matching by base
// name) — Source is already repo-relative (see relOrBase), while the
// person may have written the exclude spec with a different-looking
// but equivalent relative prefix ("./archivo1.py" vs "archivo1.py").
func sameFileTarget(specFile, source string) bool {
	specFile = strings.TrimPrefix(strings.ReplaceAll(specFile, `\`, "/"), "./")
	src := strings.TrimPrefix(strings.ReplaceAll(source, `\`, "/"), "./")
	return specFile == src || strings.HasSuffix(src, "/"+specFile)
}
