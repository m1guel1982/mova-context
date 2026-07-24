package documents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateWordContractRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "contrato.docx")
	md := "# Contrato\n\nEste es un **párrafo** de prueba.\n\n## Cláusula 1\nTexto normal."

	if err := GenerateWordContract(path, md); err != nil {
		t.Fatalf("GenerateWordContract: %v", err)
	}
	text, err := ReadDocumentLayer(path)
	if err != nil {
		t.Fatalf("ReadDocumentLayer: %v", err)
	}
	for _, want := range []string{"Contrato", "párrafo", "Cláusula 1", "Texto normal"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected extracted text to contain %q, got:\n%s", want, text)
		}
	}
}

func TestGenerateExcelReportRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reporte.xlsx")
	sheets := SheetsData{
		"Gastos": [][]Cell{
			{{Type: "string", Value: "Item"}, {Type: "string", Value: "Monto"}, {Type: "boolean", Value: true}},
			{{Type: "string", Value: "Café"}, {Type: "number", Value: 4.5}, {Type: "boolean", Value: false}},
		},
	}
	if err := GenerateExcelReport(path, sheets); err != nil {
		t.Fatalf("GenerateExcelReport: %v", err)
	}
	text, err := ReadDocumentLayer(path)
	if err != nil {
		t.Fatalf("ReadDocumentLayer: %v", err)
	}
	for _, want := range []string{"Item", "Monto", "Café", "4.5"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected extracted text to contain %q, got:\n%s", want, text)
		}
	}
}

func TestGenerateExcelReportRejectsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vacio.xlsx")
	if err := GenerateExcelReport(path, SheetsData{}); err == nil {
		t.Fatal("expected error for empty sheets_data, got nil")
	}
}

func TestGeneratePDFDocumentRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "documento.pdf")
	html := `<h1>Ley 21.719</h1><p>El consentimiento debe ser <b>informado</b> y expreso.</p>`

	if err := GeneratePDFDocument(path, html); err != nil {
		t.Fatalf("GeneratePDFDocument: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		t.Fatalf("expected a non-empty PDF file, err=%v", err)
	}
	text, err := ReadDocumentLayer(path)
	if err != nil {
		t.Fatalf("ReadDocumentLayer: %v", err)
	}
	for _, want := range []string{"Ley 21", "consentimiento", "informado"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected extracted PDF text to contain %q, got:\n%s", want, text)
		}
	}
}

func TestGenerateVectorGraphic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "diagrama.svg")
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100"><circle cx="50" cy="50" r="40"/></svg>`

	if err := GenerateVectorGraphic(path, svg); err != nil {
		t.Fatalf("GenerateVectorGraphic: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading generated svg: %v", err)
	}
	if !strings.Contains(string(data), "<circle") {
		t.Errorf("expected svg content to be preserved, got: %s", data)
	}
}

func TestGenerateVectorGraphicRejectsInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalido.svg")
	if err := GenerateVectorGraphic(path, "no es svg"); err == nil {
		t.Fatal("expected error for non-svg content, got nil")
	}
}

func TestReadDocumentLayerUnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archivo.txt")
	os.WriteFile(path, []byte("hola"), 0o644)
	if _, err := ReadDocumentLayer(path); err == nil {
		t.Fatal("expected error for unsupported extension, got nil")
	}
}

func TestResolvePathCreatesRepoDir(t *testing.T) {
	root := t.TempDir()
	path, err := ResolvePath(root, "salida/subcarpeta", "archivo.txt")
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join("salida", "subcarpeta", "archivo.txt")) {
		t.Errorf("unexpected resolved path: %s", path)
	}
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Errorf("expected repo directory to be created: %v", err)
	}
}

func TestAspectToSize(t *testing.T) {
	cases := map[string][2]int{
		"square":   {768, 768},
		"1:1":      {768, 768},
		"":         {768, 768},
		"portrait": {576, 1024},
		"9:16":     {576, 1024},
		"wide":     {1024, 576},
		"16:9":     {1024, 576},
	}
	for in, want := range cases {
		w, h := aspectToSize(in)
		if w != want[0] || h != want[1] {
			t.Errorf("aspectToSize(%q) = (%d,%d), want (%d,%d)", in, w, h, want[0], want[1])
		}
	}
}
