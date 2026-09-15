// languages.go — registro de GRAMÁTICAS gotreesitter por nombre de
// lenguaje ("go", "python", "javascript", ...). Antes de la Extensión
// del AST (ver config.go) este archivo también contenía, hardcodeado,
// qué node-type de cada gramática correspondía a "func"/"class"/
// "namespace" — eso ahora vive en JSON (config/ast/keywords_*.json,
// con copia embebida en default_config/) y se resuelve dinámicamente
// en config.go. Este archivo sólo mapea un NOMBRE DE LENGUAJE a su
// función cargadora de gramática — agregar el SOPORTE DE UN LENGUAJE
// NUEVO (uno que gotreesitter/grammars todavía no trae compilado)
// sigue requiriendo tocar este archivo (es una gramática C incrustada
// distinta, no algo expresable en JSON); pero una vez que la gramática
// ya está aquí, agregar o ajustar SUS PALABRAS CLAVE filtrables
// (struct, enum, trait, otro alias, etc.) es puro JSON, sin tocar Go.
package astfilter

import (
	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// langSpec is one grammar's runtime handle plus its currently-loaded
// kind->node-type-rule table (populated by config.go's loadOne from
// whichever keywords_<language>.json won — embedded default or a
// root/config/ast/ override).
type langSpec struct {
	language func() *gts.Language
	kinds    map[string][]NodeMatcher
}

// nodeMatchersFor returns the node-type rules for kind (already
// normalized — see NormalizeKindName) under this language, or nil if
// this language's JSON never defined that kind (e.g. Python has no
// "const").
func (s langSpec) nodeMatchersFor(kind string) []NodeMatcher {
	return s.kinds[kind]
}

// grammarLoaders maps a language name (the "language" field inside
// keywords_<language>.json) to its gotreesitter grammar loader.
// Extending Mova to a brand-new tree-sitter grammar gotreesitter
// doesn't already embed means adding an entry here (a real Go/CGo-free
// grammar table has to come from somewhere); extending an
// ALREADY-LISTED language's filterable keywords never touches this
// file — see config.go.
var grammarLoaders = map[string]func() *gts.Language{
	"go":         grammars.GoLanguage,
	"python":     grammars.PythonLanguage,
	"javascript": grammars.JavascriptLanguage,
	"typescript": grammars.TypescriptLanguage,
	"tsx":        grammars.TsxLanguage,
	"java":       grammars.JavaLanguage,
	"csharp":     grammars.CSharpLanguage,
	"c":          grammars.CLanguage,
	"cpp":        grammars.CppLanguage,
	"php":        grammars.PhpLanguage,
	"ruby":       grammars.RubyLanguage,
	"rust":       grammars.RustLanguage,
}

// languageGrammar looks up name's grammar loader (case-sensitive,
// matching keywords_*.json's own "language" field exactly) and returns
// a fresh langSpec with that grammar wired in and an empty kinds map
// (config.go's loadOne fills kinds in right after calling this).
func languageGrammar(name string) (langSpec, bool) {
	loader, ok := grammarLoaders[name]
	if !ok {
		return langSpec{}, false
	}
	return langSpec{language: loader}, true
}
