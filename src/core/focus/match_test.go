package focus

import "testing"

func TestLikeContains_AccentAndCaseInsensitive(t *testing.T) {
	cases := []struct{ haystack, needle string; want bool }{
		{"## Artículo 3 — Pluralización", "articulo 3", true},
		{"## Artículo 3 — Pluralización", "ARTÍCULO 3", true},
		{"func LoadTranslations(locale string)", "loadtranslations", true},
		{"func LoadTranslations(locale string)", "loadtranslation", true}, // substring real, sigue siendo LIKE válido
		{"func LoadTranslations(locale string)", "loadtranslator", false}, // no es substring: nunca se inventa una relación
		{"nada que ver", "articulo 3", false},
	}
	for _, c := range cases {
		if got := LikeContains(c.haystack, c.needle); got != c.want {
			t.Errorf("LikeContains(%q, %q) = %v, want %v", c.haystack, c.needle, got, c.want)
		}
	}
}

func TestLikeContainsAllWords(t *testing.T) {
	if !LikeContainsAllWords("las traducciones se cargan desde locales/", "carga traducciones") {
		t.Error("esperaba match: todas las palabras están presentes, en cualquier orden")
	}
	if LikeContainsAllWords("nada relacionado aquí", "carga traducciones") {
		t.Error("no esperaba match: ninguna palabra está presente")
	}
}
