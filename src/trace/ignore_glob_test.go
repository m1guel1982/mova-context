// ignore_glob_test.go — glob/extglob matching used by --ignore (see
// ignore_glob.go).
package trace

import "testing"

func TestMatchesIgnorePattern(t *testing.T) {
	cases := []struct {
		path     string
		patterns []string
		want     bool
	}{
		{"docs/en/index.md", []string{"docs/**"}, true},
		{"fastapi/security/oauth2.py", []string{"docs/**"}, false},
		{"docs/en/index.md", []string{"docs/!(en)/**"}, false},
		{"docs/fr/index.md", []string{"docs/!(en)/**"}, true},
		{"docs/hi/tutorial/index.md", []string{"docs/!(en)/**"}, true},
		{"docs/logo.png", []string{"docs/**/*.png"}, true},
		{"docs/img/logo.png", []string{"docs/**/*.png"}, true},
		{"docs/img/logo.svg", []string{"docs/**/*.png"}, false},
		{"scripts/build.sh", []string{"docs/**", "scripts/**", ".github/**"}, true},
		{"README.md", []string{"docs/**", "scripts/**", ".github/**"}, false},
		{"docs/en/index.md", []string{"docs/!(en,fr)/**"}, false},
		{"docs/de/index.md", []string{"docs/!(en,fr)/**"}, true},
	}
	for _, c := range cases {
		got := MatchesIgnorePattern(c.path, c.patterns)
		if got != c.want {
			t.Errorf("MatchesIgnorePattern(%q, %v) = %v, want %v", c.path, c.patterns, got, c.want)
		}
	}
}
