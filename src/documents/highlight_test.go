package documents

import "testing"

func TestAutoTagCodeFences_TagsUntaggedGoBlock(t *testing.T) {
	in := "Here:\n```\npackage main\n\nfunc main() {\n\tfmt.Println(\"hi\")\n}\n```\ndone"
	out := AutoTagCodeFences(in)
	if !containsSubstr(out, "```go\n") {
		t.Errorf("expected auto-tagged go fence, got:\n%s", out)
	}
}

func TestAutoTagCodeFences_LeavesTaggedBlockAlone(t *testing.T) {
	in := "```python\nprint('hi')\n```"
	out := AutoTagCodeFences(in)
	if out != in {
		t.Errorf("expected unchanged, got:\n%s", out)
	}
}

func TestAutoTagCodeFences_NoFencesIsNoop(t *testing.T) {
	in := "just plain text, no code here"
	if AutoTagCodeFences(in) != in {
		t.Error("expected no-op on text without fences")
	}
}

func TestDetectLanguage_SQL(t *testing.T) {
	if got := DetectLanguage("SELECT * FROM users WHERE id = 1"); got != "sql" {
		t.Errorf("expected sql, got %q", got)
	}
}

func TestDetectLanguage_Empty(t *testing.T) {
	if got := DetectLanguage("   "); got != "" {
		t.Errorf("expected empty for blank input, got %q", got)
	}
}

func containsSubstr(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
