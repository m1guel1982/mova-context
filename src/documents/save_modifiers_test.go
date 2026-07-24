package documents

import "testing"

func TestDetectSaveModifiers(t *testing.T) {
	cases := []struct {
		text           string
		append         bool
		overwriteSet   bool
		overwriteValue bool
	}{
		{"Genera reporte.pdf", false, false, false},
		{"Agrega al final de notes.md", true, false, false},
		{"Append this to notes.md", true, false, false},
		{"Sobreescribe report.pdf", false, true, true},
		{"Overwrite report.pdf", false, true, true},
		{"Reemplaza report.pdf", false, true, true},
		{"No sobreescribas report.pdf", false, true, false},
		{"No lo sobrescribas", false, true, false},
		{"Don't overwrite report.pdf", false, true, false},
		{"Genera reporte.pdf sin sobreescribir", false, true, false},
	}
	for _, c := range cases {
		got := DetectSaveModifiers(c.text)
		if got.Append != c.append {
			t.Errorf("%q: Append = %v, want %v", c.text, got.Append, c.append)
		}
		if got.OverwriteSet != c.overwriteSet {
			t.Errorf("%q: OverwriteSet = %v, want %v", c.text, got.OverwriteSet, c.overwriteSet)
		}
		if got.OverwriteSet && got.OverwriteValue != c.overwriteValue {
			t.Errorf("%q: OverwriteValue = %v, want %v", c.text, got.OverwriteValue, c.overwriteValue)
		}
	}
}
