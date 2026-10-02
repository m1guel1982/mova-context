// graph.go — Analyze extrae de UN archivo los hechos que el generador de
// grafos (paquete graph) necesita, reutilizando el mismo Tree-Sitter y los
// mismos config/ast/keywords_*.json que Extract/Strip: declaraciones por
// kind (func/class/var/...), llamadas (call_expression y equivalentes),
// referencias a identificadores e imports/require. No escribe ni reescribe
// nada y no usa LLM: es una sola pasada determinista por archivo.
package astfilter

import (
	"regexp"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
)

// Decl es una declaración con nombre que puede ser nodo del grafo.
type Decl struct {
	Name, Kind string
	Start, End uint32
}

// Call es una llamada hecha dentro de la declaración Caller ("" = nivel
// de archivo). Qualifier es el receptor textual ("Admin", "this.admin",
// "" si no hay; "?" si es una cadena no resoluble como `a().b()`).
type Call struct{ Caller, Qualifier, Name string }

// Ref es un identificador suelto usado dentro de Caller (posible lectura
// de una variable/constante del mismo archivo o importada).
type Ref struct{ Caller, Name string }

// Import liga un nombre local a un módulo: `const X = require('m')`
// (Local X, Member ""), `const {a: b} = require('m')` (Local b, Member
// a), `import * as X from 'm'` (Local X, Member "*"), etc.
type Import struct{ Local, Member, Module string }

// Facts reúne todo lo anterior para un archivo.
type Facts struct {
	Decls   []Decl
	Calls   []Call
	Refs    []Ref
	Imports []Import
}

// graphKinds: orden fijo en el que se prueba qué kind es un nodo.
var graphKinds = []string{"func", "class", "interface", "struct", "enum", "trait", "type", "namespace", "const", "var", "impl", "macro"}

// classLike: kinds que contienen miembros; func/class-like pueden ser
// "caller" aunque estén anidados, var/const solo a nivel superior.
var classLike = map[string]bool{"class": true, "interface": true, "struct": true, "enum": true, "trait": true, "impl": true, "namespace": true}

var callTypes = map[string]bool{
	"call_expression": true, "call": true, "method_invocation": true, "invocation_expression": true,
	"function_call_expression": true, "member_call_expression": true, "scoped_call_expression": true,
	"new_expression": true, "object_creation_expression": true,
}

var identRe = regexp.MustCompile(`^[A-Za-z_$][\w$]*$`)

// Analyze parsea content con la gramática de path. ok=false si el
// lenguaje no está soportado o el parseo falló.
func Analyze(content []byte, path string) (*Facts, bool) {
	tree, lang, ok := ParseTree(content, path)
	if !ok {
		return nil, false
	}
	spec, _ := languageFor(extOf(path))
	km := map[string][]NodeMatcher{}
	for _, k := range graphKinds {
		km[k] = spec.nodeMatchersFor(k)
	}
	f := &Facts{}
	type frame struct{ name, kind string }
	var stack []frame
	seenRef := map[Ref]bool{}
	text := func(n *gts.Node) string {
		if n == nil {
			return ""
		}
		return n.Text(content)
	}

	var walk func(n *gts.Node)
	walk = func(n *gts.Node) {
		if n == nil {
			return
		}
		t := n.Type(lang)
		pushed := false
		for _, k := range graphKinds {
			if !nodeMatchesAny(n, lang, km[k]) {
				continue
			}
			name := declName(n, lang, content)
			if name == "" || name == "require" {
				break
			}
			inFunc := false
			for _, fr := range stack {
				if fr.kind == "func" {
					inFunc = true
				}
			}
			canPush := classLike[k] || (k == "func" && !inFunc) || (len(stack) == 0 && (k == "var" || k == "const" || k == "type"))
			if canPush {
				f.Decls = append(f.Decls, Decl{Name: name, Kind: k, Start: n.StartByte(), End: n.EndByte()})
				stack = append(stack, frame{name, k})
				pushed = true
			}
			break
		}
		caller := ""
		if len(stack) > 0 {
			caller = stack[len(stack)-1].name
		}

		switch {
		case callTypes[t]:
			if q, nm := calleeOf(n, lang, text); nm != "" && nm != "require" {
				f.Calls = append(f.Calls, Call{Caller: caller, Qualifier: q, Name: nm})
			}
		case t == "variable_declarator":
			f.Imports = append(f.Imports, requireImports(n, lang, text)...)
		case t == "import_statement" || t == "import_from_statement" || t == "import_spec":
			f.Imports = append(f.Imports, importStatement(n, lang, content)...)
		case t == "identifier" && caller != "":
			if nm := text(n); nm != caller {
				if r := (Ref{Caller: caller, Name: nm}); !seenRef[r] {
					seenRef[r] = true
					f.Refs = append(f.Refs, r)
				}
			}
		}
		for i := 0; i < n.ChildCount(); i++ {
			walk(n.Child(i))
		}
		if pushed {
			stack = stack[:len(stack)-1]
		}
	}
	walk(tree.RootNode())
	return f, true
}

func extOf(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[i:]
	}
	return ""
}

// calleeOf devuelve (qualifier, name) de un nodo de llamada.
func calleeOf(n *gts.Node, lang *gts.Language, text func(*gts.Node) string) (string, string) {
	switch n.Type(lang) {
	case "new_expression", "object_creation_expression":
		c := n.ChildByFieldName("constructor", lang)
		if c == nil {
			c = n.ChildByFieldName("type", lang)
		}
		return splitCallee(text(c))
	}
	if obj, nm := n.ChildByFieldName("object", lang), n.ChildByFieldName("name", lang); obj != nil && nm != nil {
		return cleanQualifier(text(obj)), text(nm)
	}
	if m := n.ChildByFieldName("method", lang); m != nil { // Ruby
		return cleanQualifier(text(n.ChildByFieldName("receiver", lang))), text(m)
	}
	if f := n.ChildByFieldName("function", lang); f != nil {
		return splitCallee(text(f))
	}
	return splitCallee(text(n.ChildByFieldName("name", lang)))
}

func cleanQualifier(q string) string {
	q = strings.TrimSpace(q)
	if strings.ContainsAny(q, "()[]{} \n") {
		return "?"
	}
	return q
}

// splitCallee parte "a.b.c" / "pkg::f" / "o->m" en (qualifier, name).
func splitCallee(s string) (string, string) {
	s = strings.TrimSpace(s)
	cut, sepLen := -1, 0
	for _, sep := range []string{"?.", "::", "->", "."} {
		if i := strings.LastIndex(s, sep); i > cut {
			cut, sepLen = i, len(sep)
		}
	}
	if cut < 0 {
		if identRe.MatchString(s) {
			return "", s
		}
		return "", ""
	}
	name := s[cut+sepLen:]
	if !identRe.MatchString(name) {
		return "", ""
	}
	return cleanQualifier(s[:cut]), name
}

func unquote(s string) string { return strings.Trim(strings.TrimSpace(s), "'\"`") }

// requireArg devuelve el módulo si n es `require('m')` (o `require('m').x`).
func requireArg(n *gts.Node, lang *gts.Language, text func(*gts.Node) string) (module, member string, ok bool) {
	if n == nil {
		return "", "", false
	}
	switch n.Type(lang) {
	case "call_expression":
		if strings.TrimSpace(text(n.ChildByFieldName("function", lang))) != "require" {
			return "", "", false
		}
		args := n.ChildByFieldName("arguments", lang)
		for i := 0; args != nil && i < args.ChildCount(); i++ {
			if c := args.Child(i); c != nil && strings.Contains(c.Type(lang), "string") {
				return unquote(text(c)), "", true
			}
		}
	case "member_expression":
		if m, _, ok := requireArg(n.ChildByFieldName("object", lang), lang, text); ok {
			return m, text(n.ChildByFieldName("property", lang)), true
		}
	}
	return "", "", false
}

func requireImports(n *gts.Node, lang *gts.Language, text func(*gts.Node) string) []Import {
	module, member, ok := requireArg(n.ChildByFieldName("value", lang), lang, text)
	if !ok {
		return nil
	}
	name := n.ChildByFieldName("name", lang)
	if name == nil {
		return nil
	}
	switch name.Type(lang) {
	case "identifier":
		return []Import{{Local: text(name), Member: member, Module: module}}
	case "object_pattern":
		var out []Import
		for i := 0; i < name.ChildCount(); i++ {
			c := name.Child(i)
			switch c.Type(lang) {
			case "shorthand_property_identifier_pattern":
				out = append(out, Import{Local: text(c), Member: text(c), Module: module})
			case "pair_pattern":
				if v := text(c.ChildByFieldName("value", lang)); identRe.MatchString(v) {
					out = append(out, Import{Local: v, Member: text(c.ChildByFieldName("key", lang)), Module: module})
				}
			}
		}
		return out
	}
	return nil
}

// importStatement cubre `import ... from 'm'` (JS/TS), Python y Go.
func importStatement(n *gts.Node, lang *gts.Language, content []byte) []Import {
	text := func(x *gts.Node) string {
		if x == nil {
			return ""
		}
		return x.Text(content)
	}
	switch n.Type(lang) {
	case "import_spec": // Go
		mod := unquote(text(n.ChildByFieldName("path", lang)))
		local := text(n.ChildByFieldName("name", lang))
		if local == "" {
			local = mod[strings.LastIndex(mod, "/")+1:]
		}
		return []Import{{Local: local, Member: "*", Module: mod}}
	case "import_from_statement": // Python
		modNode := n.ChildByFieldName("module_name", lang)
		mod := text(modNode)
		var out []Import
		for i := 0; i < n.ChildCount(); i++ {
			c := n.Child(i)
			if modNode != nil && c.StartByte() == modNode.StartByte() {
				continue
			}
			switch c.Type(lang) {
			case "dotted_name":
				out = append(out, Import{Local: text(c), Member: text(c), Module: mod})
			case "aliased_import":
				out = append(out, Import{Local: text(c.ChildByFieldName("alias", lang)), Member: text(c.ChildByFieldName("name", lang)), Module: mod})
			}
		}
		return out
	}
	// import_statement: JS/TS (campo source) o Python (import a.b as c)
	if src := n.ChildByFieldName("source", lang); src != nil {
		mod := unquote(text(src))
		var out []Import
		for i := 0; i < n.ChildCount(); i++ {
			clause := n.Child(i)
			if clause.Type(lang) != "import_clause" {
				continue
			}
			for j := 0; j < clause.ChildCount(); j++ {
				c := clause.Child(j)
				switch c.Type(lang) {
				case "identifier":
					out = append(out, Import{Local: text(c), Member: "default", Module: mod})
				case "namespace_import":
					for k := 0; k < c.ChildCount(); k++ {
						if id := c.Child(k); id.Type(lang) == "identifier" {
							out = append(out, Import{Local: text(id), Member: "*", Module: mod})
						}
					}
				case "named_imports":
					for k := 0; k < c.ChildCount(); k++ {
						sp := c.Child(k)
						if sp.Type(lang) != "import_specifier" {
							continue
						}
						nm := text(sp.ChildByFieldName("name", lang))
						loc := text(sp.ChildByFieldName("alias", lang))
						if loc == "" {
							loc = nm
						}
						out = append(out, Import{Local: loc, Member: nm, Module: mod})
					}
				}
			}
		}
		return out
	}
	var out []Import
	for i := 0; i < n.ChildCount(); i++ {
		c := n.Child(i)
		switch c.Type(lang) {
		case "dotted_name":
			mod := text(c)
			out = append(out, Import{Local: strings.Split(mod, ".")[0], Member: "*", Module: mod})
		case "aliased_import":
			out = append(out, Import{Local: text(c.ChildByFieldName("alias", lang)), Member: "*", Module: text(c.ChildByFieldName("name", lang))})
		}
	}
	return out
}
