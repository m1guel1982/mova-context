package documents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteReadTextFiles(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"nota.txt":    "hola mundo",
		"README.md":   "# Título\n\nContenido de prueba.",
		"datos.json":  `{"nombre": "Ana", "edad": 30}`,
		"config.yml":  "clave: valor\nlista:\n  - uno\n  - dos",
		"config.yaml": "otra: cosa",
		"datos.xml":   `<root><item id="1">Texto</item></root>`,
	}
	for name, content := range cases {
		path := filepath.Join(dir, name)
		if err := WriteFile(path, content); err != nil {
			t.Fatalf("WriteFile(%s): %v", name, err)
		}
		got, err := ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", name, err)
		}
		if got != content {
			t.Errorf("%s: round-trip mismatch\nwant: %q\ngot:  %q", name, content, got)
		}
	}
}

func TestWriteFileRejectsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "malo.json")
	if err := WriteFile(path, "{esto no es json}"); err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestWriteFileRejectsInvalidXML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "malo.xml")
	if err := WriteFile(path, "<root><item></root>"); err == nil {
		t.Fatal("expected error for malformed XML, got nil")
	}
}

func TestPatchFileSurgicalReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notas.md")
	original := "# Notas\n\nEstado: pendiente\n\nOtros detalles sin tocar."
	if err := WriteFile(path, original); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := PatchFile(path, "Estado: pendiente", "Estado: completado"); err != nil {
		t.Fatalf("PatchFile: %v", err)
	}
	got, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	want := "# Notas\n\nEstado: completado\n\nOtros detalles sin tocar."
	if got != want {
		t.Errorf("patch mismatch\nwant: %q\ngot:  %q", want, got)
	}
}

func TestPatchFileRejectsAmbiguousMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "repetido.txt")
	WriteFile(path, "hola hola hola")
	if err := PatchFile(path, "hola", "chau"); err == nil {
		t.Fatal("expected error for ambiguous (non-unique) match, got nil")
	}
}

func TestPatchFileRejectsMissingMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archivo.txt")
	WriteFile(path, "contenido original")
	if err := PatchFile(path, "no existe", "reemplazo"); err == nil {
		t.Fatal("expected error for search text not found, got nil")
	}
}

func TestSupportedTextExt(t *testing.T) {
	for _, ext := range []string{".txt", ".md", ".json", ".yml", ".yaml", ".xml"} {
		if !SupportedTextExt(ext) {
			t.Errorf("expected %q to be supported", ext)
		}
	}
	if SupportedTextExt(".exe") {
		t.Error("expected .exe to be unsupported")
	}
}

func TestWriteReadSourceCodeFiles(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"app.js":     "console.log('hola');",
		"app.ts":     "const x: number = 1;",
		"script.py":  "print('hola')",
		"main.go":    "package main\n\nfunc main() {\n\tprintln(\"hola\")\n}\n",
		"Program.cs": "class Program { static void Main() {} }",
		"Main.java":  "class Main { public static void main(String[] a) {} }",
		"index.php":  "<?php echo 'hola'; ?>",
		"script.rb":  "puts 'hola'",
		"main.rs":    "fn main() {}",
		"main.c":     "int main() { return 0; }",
		"main.cpp":   "int main() { return 0; }",
		"header.h":   "#pragma once",
		"App.kt":     "fun main() {}",
		"App.swift":  "print(\"hola\")",
		"deploy.sh":  "#!/bin/bash\necho hola",
		"index.html": "<html><body>hola</body></html>",
		"style.css":  "body { color: red; }",
		"query.sql":  "SELECT * FROM tareas;",
	}
	for name, content := range cases {
		path := filepath.Join(dir, name)
		if err := WriteFile(path, content); err != nil {
			t.Fatalf("WriteFile(%s): %v", name, err)
		}
		got, err := ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", name, err)
		}
		if got != content {
			t.Errorf("%s: round-trip mismatch\nwant: %q\ngot:  %q", name, content, got)
		}
	}
}

func TestWriteFileValidatesGoSyntax(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "roto.go")
	if err := WriteFile(path, "package main\n\nfunc main( {\n"); err == nil {
		t.Fatal("expected error for invalid Go syntax, got nil")
	}
}

func TestWriteFileValidatesCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "malo.csv")
	// Ragged CSV: second row has an extra field vs the header.
	if err := WriteFile(path, "nombre,edad\nAna,30,extra\n"); err == nil {
		t.Fatal("expected error for malformed CSV, got nil")
	}
}

func TestWriteFileRejectsUnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "programa.exe")
	err := WriteFile(path, "contenido")
	if err == nil {
		t.Fatal("expected error for unsupported extension, got nil")
	}
	if !strings.Contains(err.Error(), "Unsupported file type") {
		t.Errorf("expected English 'Unsupported file type' message, got: %v", err)
	}
}

func TestPatchFileRejectsUnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "binario.exe")
	os.WriteFile(path, []byte("algo"), 0o644)
	err := PatchFile(path, "algo", "otro")
	if err == nil {
		t.Fatal("expected error for unsupported extension, got nil")
	}
	if !strings.Contains(err.Error(), "Unsupported file type") {
		t.Errorf("expected English 'Unsupported file type' message, got: %v", err)
	}
}
