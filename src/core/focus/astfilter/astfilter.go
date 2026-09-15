// Package astfilter extends the Focus Resolution Engine (see
// mova.local/core/focus) with REAL AST-based symbol extraction, using
// github.com/odvcencio/gotreesitter — a PURE-GO tree-sitter-compatible
// runtime (no CGo, no C toolchain, cross-compiles like any other Go
// package) — for the "file::kind=name1,name2" focus/exclude syntax
// (see docs/i18n/{es,en}/context-trace.md § "Filtro por símbolo
// (AST)"). This is an ADDITION, not a replacement: the existing
// brace/indent-heuristic CodeSymbolResolver (see
// core/focus/resolvers/code_symbol.go, which resolves a bare symbol
// name with no file/kind prefix) is untouched and keeps working
// exactly as before — this package only powers the new, more precise
// "file::func=...", "file::class=...", "file::struct=...",
// "file::interface=...", "file::enum=...", "file::trait=...",
// "file::type=...", "file::const=...", "file::var=...",
// "file::comment=..." syntax (plus the "method"/"package"/"variable"/
// "property" aliases — see NormalizeKindName), in both "focus"
// (extract only the named symbols) and "exclude" (strip only the
// named symbols from an otherwise-included file).
//
// WHICH kinds exist for a given file's language, and which tree-sitter
// node type(s) each one maps to, is no longer hardcoded here — see
// config.go for the config/ast/keywords_<language>.json catalog
// (embedded defaults + optional project-level overrides) that this
// package now reads at Init(root) time. This file only implements the
// language-agnostic matching/extraction engine: given an already-
// resolved set of node-type rules for one kind, walk the tree and
// find/extract/strip matches — the exact same walk regardless of
// which language or which JSON file the rules came from.
//
// An earlier version of this package used smacker/go-tree-sitter
// (a CGo binding over tree-sitter's C core). It was replaced with
// this pure-Go engine specifically to keep `mova` buildable with
// CGO_ENABLED=0 and without a C compiler on the developer's machine
// — see docs/PROJECT.md § "AST filtering" for the size/build-tag
// tradeoff this package's languages.go makes.
package astfilter

import (
	"path/filepath"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
)

// kindAliases maps a project.json-facing alias onto the canonical kind
// name used inside every keywords_<language>.json file — so
// "file::method=..." and "file::func=..." resolve to the exact same
// node-type rules, without every language's JSON having to repeat
// "method" as a second copy of its "func" entry.
var kindAliases = map[string]string{
	"method":   "func",      // alias for readability in project.json
	"package":  "namespace", // alias for readability in project.json
	"variable": "var",       // Spanish-friendly / natural-language alias
	"property": "var",       // class/struct field, filtered the same way as a variable
}

// knownKinds is the full catalog of canonical kind names this package
// understands ACROSS every shipped language — used only to validate a
// "file::kind=..." spec's kind token before even knowing which file/
// language it targets (see resolvers/ast_symbol.go's
// parseAstSymbolTarget). A specific language's JSON may not define
// every one of these (e.g. Python has no "const" — see
// keywords_python.json's "notes"); when that happens Extract/Strip
// simply report ok=false, exactly like an unsupported language does.
var knownKinds = map[string]bool{
	"func": true, "class": true, "interface": true, "struct": true,
	"enum": true, "trait": true, "type": true, "namespace": true,
	"const": true, "var": true, "macro": true, "impl": true, "comment": true,
}

// NormalizeKindName lowercases/trims raw, resolves it through
// kindAliases if it's a known alias, and reports whether the result is
// a recognized canonical kind name at all (regardless of whether the
// TARGET FILE's specific language happens to define it — that check
// only happens once a file is actually being parsed, in findSymbols).
// The single source of truth for "is this a real kind keyword", used
// both internally (Extract/Strip/AllNames, via normalizeKind below)
// and externally by resolvers/ast_symbol.go's project.json spec parser
// — see this function's doc for why keeping this in one place matters:
// the alternative is two hardcoded lists silently drifting apart every
// time a new kind is added.
func NormalizeKindName(raw string) (string, bool) {
	k := strings.ToLower(strings.TrimSpace(raw))
	if alias, ok := kindAliases[k]; ok {
		k = alias
	}
	if !knownKinds[k] {
		return "", false
	}
	return k, true
}

// normalizeKind is NormalizeKindName without the ok flag, for internal
// callers that already know k is well-formed (Extract/Strip/AllNames
// pass through whatever findSymbols/AllNames already validated against
// the language's own kinds map — an unrecognized name here simply
// yields no node-type rules, i.e. ok=false downstream, same end
// result as rejecting it up front).
func normalizeKind(k string) string {
	norm, ok := NormalizeKindName(k)
	if !ok {
		return strings.ToLower(strings.TrimSpace(k))
	}
	return norm
}

// Supported reports whether path's extension has a registered
// tree-sitter-compatible grammar — callers use this to decide whether
// to even attempt AST parsing vs. falling back to the older heuristic
// resolver. Accepts a bare extension ("py"/".py") or a full path
// ("archivo1.py") equally, same as Extract/Strip.
func Supported(path string) bool {
	_, ok := languageFor(filepath.Ext(withDot(path)))
	return ok
}

// withDot makes filepath.Ext behave for a bare extension like "py" (no
// path separators, no existing dot) by treating it as a fake filename
// "x.py" first — filepath.Ext("py") would otherwise return "" (no dot
// found), losing a legitimate "just the extension" caller.
func withDot(s string) string {
	if s == "" || strings.Contains(s, ".") || strings.ContainsAny(s, `/\`) {
		return s
	}
	return "x." + s
}

// symbolMatch is one located declaration: its name, and the exact
// byte range [Start,End) of the WHOLE declaration node.
type symbolMatch struct {
	Name  string
	Start uint32
	End   uint32
}

// matchesNode reports whether n satisfies matcher: its own Type() must
// equal matcher.Type, and — when matcher.RequiresChildType is set — at
// least one of n's DIRECT children must have that exact Type(). See
// this file's — and config.go's — header for why this exists: several
// grammars reuse one node shape for more than one keyword (Go's
// type_spec for struct/interface, JS/TS's lexical_declaration for
// const/let, Ruby's assignment for constant/variable), and the
// keyword's own token (or an unambiguous child) is what disambiguates.
func matchesNode(n *gts.Node, lang *gts.Language, matcher NodeMatcher) bool {
	if n.Type(lang) != matcher.Type {
		return false
	}
	if matcher.RequiresChildType == "" {
		return true
	}
	for i := 0; i < n.ChildCount(); i++ {
		if c := n.Child(i); c != nil && c.Type(lang) == matcher.RequiresChildType {
			return true
		}
	}
	return false
}

// nodeMatchesAny reports whether n satisfies any one of matchers.
func nodeMatchesAny(n *gts.Node, lang *gts.Language, matchers []NodeMatcher) bool {
	for _, m := range matchers {
		if matchesNode(n, lang, m) {
			return true
		}
	}
	return false
}

// findSymbols parses content with ext's grammar and returns every
// declaration of the given kind whose name is in names (case-sensitive
// — symbol names are, matching every supported language's own rules).
// A name may be given with or without a trailing "()" (the project.json
// syntax allows "insertar_usuario()" for readability; this is trimmed
// before matching).
func findSymbols(content []byte, ext string, kind string, names []string) ([]symbolMatch, bool) {
	spec, ok := languageFor(ext)
	if !ok {
		return nil, false
	}
	wanted := map[string]bool{}
	for _, n := range names {
		wanted[strings.TrimSuffix(strings.TrimSpace(n), "()")] = true
	}
	if len(wanted) == 0 {
		return nil, false
	}

	matchers := spec.nodeMatchersFor(normalizeKind(kind))
	if len(matchers) == 0 {
		return nil, false
	}

	lang := spec.language()
	parser := gts.NewParser(lang)
	tree, err := parser.Parse(content)
	if err != nil || tree == nil {
		return nil, false
	}

	var matches []symbolMatch
	var walk func(n *gts.Node)
	walk = func(n *gts.Node) {
		if n == nil {
			return
		}
		if nodeMatchesAny(n, lang, matchers) {
			if name := declName(n, lang, content); name != "" && wanted[name] {
				matches = append(matches, symbolMatch{Name: name, Start: n.StartByte(), End: n.EndByte()})
			}
		}
		for i := 0; i < n.ChildCount(); i++ {
			walk(n.Child(i))
		}
	}
	walk(tree.RootNode())
	return matches, len(matches) > 0
}

// declName reads a declaration node's identifier. Most grammars expose
// it directly via the "name" field (tried first). C/C++ function
// definitions instead nest it inside a "declarator" field (itself
// possibly wrapped in a pointer_declarator for `int *foo()`), so that
// field is tried next. Anything else falls back to a depth-first scan
// for the first *_identifier-shaped node, deliberately not descending
// into parameter/argument lists so a parameter's name is never
// mistaken for the declaration's own name (this is also what recovers
// Go's package_clause, which has neither a "name" nor a "declarator"
// field, only a bare package_identifier child).
func declName(n *gts.Node, lang *gts.Language, content []byte) string {
	if id := n.ChildByFieldName("name", lang); id != nil {
		return id.Text(content)
	}
	if decl := n.ChildByFieldName("declarator", lang); decl != nil {
		if name := firstIdentifier(decl, lang, content); name != "" {
			return name
		}
	}
	return firstIdentifier(n, lang, content)
}

func firstIdentifier(n *gts.Node, lang *gts.Language, content []byte) string {
	if n == nil {
		return ""
	}
	t := n.Type(lang)
	// "constant" is Ruby's own node type for an uppercase constant
	// reference/assignment target (PI, Usuario, ...) — it does NOT end
	// in "identifier" like every other supported grammar's leaf does,
	// so it needs an explicit exception here (verified empirically:
	// `PI = 3.14` parses as assignment{constant, "=", float}).
	if strings.HasSuffix(t, "identifier") || t == "constant" {
		return n.Text(content)
	}
	if strings.Contains(t, "parameter") || strings.Contains(t, "argument") {
		return ""
	}
	for i := 0; i < n.ChildCount(); i++ {
		if name := firstIdentifier(n.Child(i), lang, content); name != "" {
			return name
		}
	}
	return ""
}

// ParseTree parses content with path's grammar and returns the tree
// plus the language handle callers need for Node.Type/ChildByFieldName
// (see trace/ast_relevance.go, which walks the tree directly for
// signals — comment density, docstring detection — that Extract/Strip
// don't need to expose). ok is false when the language isn't
// supported or parsing failed.
func ParseTree(content []byte, path string) (*gts.Tree, *gts.Language, bool) {
	spec, ok := languageFor(filepath.Ext(path))
	if !ok {
		return nil, nil, false
	}
	lang := spec.language()
	parser := gts.NewParser(lang)
	tree, err := parser.Parse(content)
	if err != nil || tree == nil {
		return nil, nil, false
	}
	return tree, lang, true
}

// AllNames returns every declaration name of the given kind in an
// already-parsed tree — unlike findSymbols/Extract/Strip, this is not
// filtered against a wanted-names set: it lists everything, for
// callers (trace/ast_relevance.go's role/boost classifier) that need
// to compare a file's OWN declared names against a task's words
// rather than extract one already-known symbol. path is only used to
// look the language spec back up by extension (ParseTree already used
// it for the same lookup) — tree/lang must already match path's
// grammar.
func AllNames(tree *gts.Tree, lang *gts.Language, content []byte, path string, kind string) []string {
	spec, ok := languageFor(filepath.Ext(path))
	if !ok {
		return nil
	}
	matchers := spec.nodeMatchersFor(normalizeKind(kind))
	if len(matchers) == 0 {
		return nil
	}
	var names []string
	var walk func(n *gts.Node)
	walk = func(n *gts.Node) {
		if n == nil {
			return
		}
		if nodeMatchesAny(n, lang, matchers) {
			if name := declName(n, lang, content); name != "" {
				names = append(names, name)
			}
		}
		for i := 0; i < n.ChildCount(); i++ {
			walk(n.Child(i))
		}
	}
	walk(tree.RootNode())
	return names
}

// PruneDocs strips every comment and docstring-shaped statement from
// content — the --prune-docstrings feature (see trace/analyzer.go):
// collapsing documentation out of a file's content right before it's
// packaged into the final context, when the token budget is tight.
// Uses the exact same "comment node, or a bare string as the first
// statement of a block/module" detection as trace/ast_relevance.go's
// docCommentBytes (kept in sync deliberately — see that function's
// doc comment for why a bare first-statement string counts and an
// ordinary string expression doesn't). ok is false when path's
// language isn't supported (caller then packages the file unpruned,
// exactly as before this flag existed).
func PruneDocs(content []byte, path string) (string, bool) {
	tree, lang, ok := ParseTree(content, path)
	if !ok {
		return "", false
	}
	type byteRange struct{ start, end uint32 }
	var ranges []byteRange
	var walk func(n *gts.Node)
	walk = func(n *gts.Node) {
		if n == nil {
			return
		}
		t := n.Type(lang)
		if t == "comment" {
			ranges = append(ranges, byteRange{n.StartByte(), n.EndByte()})
		}
		if (t == "block" || t == "module" || t == "program") && n.ChildCount() > 0 {
			if first := n.Child(0); first != nil && first.Type(lang) == "string" {
				ranges = append(ranges, byteRange{first.StartByte(), first.EndByte()})
			}
		}
		for i := 0; i < n.ChildCount(); i++ {
			walk(n.Child(i))
		}
	}
	walk(tree.RootNode())
	if len(ranges) == 0 {
		return string(content), true
	}
	var b strings.Builder
	prev := uint32(0)
	for _, r := range ranges {
		if r.start < prev {
			continue
		}
		b.Write(content[prev:r.start])
		prev = r.end
	}
	b.Write(content[prev:])
	return b.String(), true
}

// Extract returns ONLY the requested symbols' source, concatenated in
// file order and separated by a blank line — the "focus" use case
// ("file::func=a(),b()" keeps just a and b, dropping the rest of the
// file). ok is false when the language isn't supported or none of the
// requested names were found (the caller falls back to whatever it
// would have done without AST filtering — see resolvers/ast_symbol.go).
func Extract(content []byte, path string, kind string, names []string) (string, bool) {
	matches, ok := findSymbols(content, filepath.Ext(path), kind, names)
	if !ok {
		return "", false
	}
	var out []string
	for _, m := range matches {
		out = append(out, string(content[m.Start:m.End]))
	}
	return strings.Join(out, "\n\n"), true
}

// Strip returns content with the requested symbols' source REMOVED —
// the "exclude" use case ("file::func=insertar_producto()" keeps the
// rest of an otherwise-included file, dropping just that function).
// ok is false when the language isn't supported or nothing matched
// (the caller then uses the original, unmodified content).
func Strip(content []byte, path string, kind string, names []string) (string, bool) {
	matches, ok := findSymbols(content, filepath.Ext(path), kind, names)
	if !ok {
		return "", false
	}
	var b strings.Builder
	prev := uint32(0)
	for _, m := range matches {
		if m.Start < prev {
			continue // overlapping match (e.g. a nested method also named in "class"), already covered
		}
		b.Write(content[prev:m.Start])
		prev = m.End
	}
	b.Write(content[prev:])
	return b.String(), true
}
