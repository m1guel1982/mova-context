package documents

import "testing"

func TestDetectSaveIntent(t *testing.T) {
	cases := []struct {
		text    string
		dirs    []string
		files   []string
	}{
		{"Genera carpeta/reporte.pdf", nil, []string{"carpeta/reporte.pdf"}},
		{"Genera c:/reportes/salida.pdf", nil, []string{"c:/reportes/salida.pdf"}},
		{"Crea c:/proyecto/docs/manual.md", nil, []string{"c:/proyecto/docs/manual.md"}},
		{"Crea el directorio c:/temp/test y genera reporte.pdf", []string{"c:/temp/test"}, []string{"reporte.pdf"}},
		{"hola, como estas?", nil, nil},
		{"Create the file report.docx please", nil, []string{"report.docx"}},
		{"Hazme un resumen en resumen.md", nil, []string{"resumen.md"}},
		{"Elabora el informe.docx", nil, []string{"informe.docx"}},
		{"Escribe notas.txt", nil, []string{"notas.txt"}},
		{"Prepara la carpeta salidas", []string{"salidas"}, nil},
		{"Build report.xlsx", nil, []string{"report.xlsx"}},
		{"Draft summary.md", nil, []string{"summary.md"}},
	}
	for _, c := range cases {
		got := DetectSaveIntent(c.text)
		if !equalStrSlices(got.Directories, c.dirs) {
			t.Errorf("%q: dirs = %v, want %v", c.text, got.Directories, c.dirs)
		}
		if !equalStrSlices(got.Files, c.files) {
			t.Errorf("%q: files = %v, want %v", c.text, got.Files, c.files)
		}
	}
}

func equalStrSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
