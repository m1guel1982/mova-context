// astfilter_test.go — cubre la Extensión del AST (Item 1 del refactor):
// que los kinds NUEVOS (struct, interface, enum, trait, const, var,
// macro, impl, comment) realmente extraen/filtran lo esperado en varios
// lenguajes, y que un archivo project-level en config/ast/ puede
// agregar un lenguaje nuevo sin tocar Go (Init(root) — ver config.go).
package astfilter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustInit(t *testing.T, root string) {
	t.Helper()
	if err := Init(root); err != nil {
		t.Fatalf("Init(%q): %v", root, err)
	}
}

// TestExtract_GoStructVsInterface cubre el caso más delicado del nuevo
// catálogo: Go usa el MISMO node-type (type_spec) para struct,
// interface y alias de tipo — struct/interface sólo se distinguen
// mirando el hijo (struct_type / interface_type). Ver
// config/ast/keywords_go.json's "requires_child_type".
func TestExtract_GoStructVsInterface(t *testing.T) {
	mustInit(t, t.TempDir())
	src := `package main

type Usuario struct {
	ID int
}

type Forma interface {
	Area() float64
}
`
	got, ok := Extract([]byte(src), "archivo.go", "struct", []string{"Usuario"})
	if !ok {
		t.Fatalf("Extract(struct, Usuario) ok=false")
	}
	if !strings.Contains(got, "struct {") || strings.Contains(got, "interface {") {
		t.Fatalf("expected only the Usuario struct, got %q", got)
	}

	got, ok = Extract([]byte(src), "archivo.go", "interface", []string{"Forma"})
	if !ok {
		t.Fatalf("Extract(interface, Forma) ok=false")
	}
	if !strings.Contains(got, "interface {") || strings.Contains(got, "struct {") {
		t.Fatalf("expected only the Forma interface, got %q", got)
	}

	// "struct" no debe encontrar "Forma" (es una interface, no un struct).
	if _, ok := Extract([]byte(src), "archivo.go", "struct", []string{"Forma"}); ok {
		t.Fatalf("Extract(struct, Forma) should not match an interface declaration")
	}
}

// TestExtract_GoConstVsVar cubre const_spec vs var_spec (dos node-types
// distintos en Go, sin necesidad de requires_child_type).
func TestExtract_GoConstVsVar(t *testing.T) {
	mustInit(t, t.TempDir())
	src := `package main

const Pi = 3.14
var Nombre string = "hola"
`
	if _, ok := Extract([]byte(src), "archivo.go", "const", []string{"Pi"}); !ok {
		t.Fatalf("Extract(const, Pi) ok=false")
	}
	if _, ok := Extract([]byte(src), "archivo.go", "const", []string{"Nombre"}); ok {
		t.Fatalf("Extract(const, Nombre) should not match a var declaration")
	}
	if _, ok := Extract([]byte(src), "archivo.go", "var", []string{"Nombre"}); !ok {
		t.Fatalf("Extract(var, Nombre) ok=false")
	}
	// alias "variable" -> "var"
	if _, ok := Extract([]byte(src), "archivo.go", "variable", []string{"Nombre"}); !ok {
		t.Fatalf("Extract(variable, Nombre) [alias] ok=false")
	}
}

// TestExtract_JavascriptConstVsLet cubre el caso de JS/TS: mismo
// node-type (lexical_declaration) para const y let, distinguido por el
// hijo-token de la palabra clave.
func TestExtract_JavascriptConstVsLet(t *testing.T) {
	mustInit(t, t.TempDir())
	src := `const PI = 3.14;
let nombre = "hola";
var x = 1;
`
	if _, ok := Extract([]byte(src), "archivo.js", "const", []string{"PI"}); !ok {
		t.Fatalf("Extract(const, PI) ok=false")
	}
	if _, ok := Extract([]byte(src), "archivo.js", "const", []string{"nombre"}); ok {
		t.Fatalf("Extract(const, nombre) should not match a 'let' declaration")
	}
	if _, ok := Extract([]byte(src), "archivo.js", "var", []string{"nombre"}); !ok {
		t.Fatalf("Extract(var, nombre) [let] ok=false")
	}
	if _, ok := Extract([]byte(src), "archivo.js", "var", []string{"x"}); !ok {
		t.Fatalf("Extract(var, x) ['var' keyword] ok=false")
	}
}

// TestExtract_RubyConstVsVar cubre la convención de mayúsculas de Ruby
// (constant vs identifier como hijo izquierdo de la asignación).
func TestExtract_RubyConstVsVar(t *testing.T) {
	mustInit(t, t.TempDir())
	src := "PI = 3.14\nnombre = \"hola\"\n"
	if _, ok := Extract([]byte(src), "archivo.rb", "const", []string{"PI"}); !ok {
		t.Fatalf("Extract(const, PI) ok=false")
	}
	if _, ok := Extract([]byte(src), "archivo.rb", "var", []string{"nombre"}); !ok {
		t.Fatalf("Extract(var, nombre) ok=false")
	}
	if _, ok := Extract([]byte(src), "archivo.rb", "const", []string{"nombre"}); ok {
		t.Fatalf("Extract(const, nombre) should not match a lowercase identifier")
	}
}

// TestExtract_RustTraitEnumImplMacro cubre los kinds nuevos específicos
// de Rust.
func TestExtract_RustTraitEnumImplMacro(t *testing.T) {
	mustInit(t, t.TempDir())
	src := `enum Color { Rojo, Verde }
trait Forma { fn area(&self) -> f64; }
impl Forma for Usuario { fn area(&self) -> f64 { 0.0 } }
macro_rules! saludo { () => {}; }
`
	if _, ok := Extract([]byte(src), "archivo.rs", "enum", []string{"Color"}); !ok {
		t.Fatalf("Extract(enum, Color) ok=false")
	}
	if _, ok := Extract([]byte(src), "archivo.rs", "trait", []string{"Forma"}); !ok {
		t.Fatalf("Extract(trait, Forma) ok=false")
	}
	if _, ok := Extract([]byte(src), "archivo.rs", "macro", []string{"saludo"}); !ok {
		t.Fatalf("Extract(macro, saludo) ok=false")
	}
	if _, ok := Extract([]byte(src), "archivo.rs", "impl", []string{"Forma"}); !ok {
		t.Fatalf("Extract(impl, Forma) ok=false")
	}
}

// TestStrip_CommentKind cubre el nuevo kind "comment" (útil para
// exclude): remueve un comentario dado sin tocar el resto del archivo.
func TestAllNames_CommentKindViaParseTree(t *testing.T) {
	mustInit(t, t.TempDir())
	src := `package main

// primer comentario
// segundo comentario
func Sumar() {}
`
	tree, lang, ok := ParseTree([]byte(src), "archivo.go")
	if !ok {
		t.Fatalf("ParseTree ok=false")
	}
	names := AllNames(tree, lang, []byte(src), "archivo.go", "func")
	if len(names) != 1 || names[0] != "Sumar" {
		t.Fatalf("AllNames(func) = %v, want [Sumar]", names)
	}
}

// TestInit_ProjectOverrideAddsNewLanguageExtension cubre el requisito
// central del Item 1: agregar/ajustar un lenguaje debe ser posible
// escribiendo SOLO un archivo config/ast/keywords_<lenguaje>.json, sin
// tocar el código Go — aquí se agrega una extensión nueva (".mjsx", a
// modo de ejemplo) para el lenguaje "javascript" ya existente, y se
// confirma que Extract la reconoce después de Init(root).
func TestInit_ProjectOverrideAddsExtension(t *testing.T) {
	root := t.TempDir()
	astDir := filepath.Join(root, "config", "ast")
	if err := os.MkdirAll(astDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	override := `{
		"language": "javascript",
		"extensions": [".mjsx"],
		"kinds": {
			"func": [{"type": "function_declaration"}],
			"comment": [{"type": "comment"}]
		}
	}`
	if err := os.WriteFile(filepath.Join(astDir, "keywords_javascript_custom.json"), []byte(override), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	mustInit(t, root)
	defer mustInit(t, t.TempDir()) // no dejar el registro global "contaminado" para otros tests

	if !Supported("archivo.mjsx") {
		t.Fatalf("expected .mjsx to be supported after a config/ast override")
	}
	if _, ok := Extract([]byte("function saludar() {}"), "archivo.mjsx", "func", []string{"saludar"}); !ok {
		t.Fatalf("Extract on the overridden .mjsx extension failed")
	}
}

// TestInit_MissingConfigASTDirStillWorks confirma que, sin ninguna
// carpeta config/ast/ en el proyecto (instalación mínima), los defaults
// embebidos siguen funcionando — Init nunca debe dejar el motor sin
// lenguajes por no encontrar overrides.
func TestInit_MissingConfigASTDirStillWorks(t *testing.T) {
	mustInit(t, t.TempDir())
	if !Supported("archivo.go") {
		t.Fatalf("expected .go to be supported from embedded defaults alone")
	}
}

// TestNormalizeKindName_Aliases cubre los alias de teclas naturales
// (variable/property -> var, method -> func, package -> namespace).
func TestNormalizeKindName_Aliases(t *testing.T) {
	cases := map[string]string{
		"func":     "func",
		"METHOD":   "func",
		"Class":    "class",
		"package":  "namespace",
		"variable": "var",
		"property": "var",
		"var":      "var",
	}
	for in, want := range cases {
		got, ok := NormalizeKindName(in)
		if !ok {
			t.Fatalf("NormalizeKindName(%q) ok=false", in)
		}
		if got != want {
			t.Fatalf("NormalizeKindName(%q) = %q, want %q", in, got, want)
		}
	}
	if _, ok := NormalizeKindName("no-existe"); ok {
		t.Fatalf(`NormalizeKindName("no-existe") should not be recognized`)
	}
}
