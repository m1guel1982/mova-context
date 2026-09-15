// relevance_test.go — sanity checks for the BM25 task-relevance
// scorer (see relevance.go): a file whose content and name actually
// relate to the task must outrank an unrelated file.
package trace

import "testing"

func TestRankByTask_RelevantFileScoresHigher(t *testing.T) {
	files := map[string]string{
		"auth/login.go":     "func login(user string) error {\n  // handles user login and password check\n  return nil\n}",
		"docs/CHANGELOG.md": "v1.0.0 - initial release\nv1.0.1 - bugfixes\nv1.0.2 - more bugfixes\nv1.0.3 - docs\nv1.0.4 - typos\nv1.0.5 - polish\nv1.0.6 - release notes only, nothing else here",
	}
	scores, cutoff := RankByTask(files, "fix login authentication bug", 0.5)
	if scores == nil {
		t.Fatal("expected non-nil scores for a non-empty task")
	}
	if scores["auth/login.go"].Total <= scores["docs/CHANGELOG.md"].Total {
		t.Fatalf("expected login.go to outrank CHANGELOG.md: %+v vs %+v", scores["auth/login.go"], scores["docs/CHANGELOG.md"])
	}
	if cutoff <= 0 && scores["docs/CHANGELOG.md"].Total > 0 {
		t.Fatalf("expected a meaningful cutoff, got %f", cutoff)
	}
}

func TestSelectBelowCutoff_HandlesTiesWithoutSelectingNothing(t *testing.T) {
	files := map[string]string{
		"auth/login.go": "func login(user string) error {\n  // handles user login and password check\n  return nil\n}",
		"docs/a.md":     "unrelated release notes only, nothing else here at all",
		"docs/b.md":     "unrelated release notes only, nothing else here at all",
		"docs/c.md":     "unrelated release notes only, nothing else here at all",
		"docs/d.md":     "unrelated release notes only, nothing else here at all",
		"docs/e.md":     "unrelated release notes only, nothing else here at all",
	}
	scores, _ := RankByTask(files, "fix login authentication bug", 0.35)
	excluded := SelectBelowCutoff(scores, 0.35)
	if len(excluded) == 0 {
		t.Fatal("expected SOME files excluded even though several tie at the same low score")
	}
	if excluded["auth/login.go"] {
		t.Fatal("the one clearly relevant file must never be excluded")
	}
}

func TestRankByTask_EmptyTaskDisablesNarrowing(t *testing.T) {
	files := map[string]string{"a.go": "package a"}
	scores, cutoff := RankByTask(files, "", 0.5)
	if scores != nil || cutoff != 0 {
		t.Fatalf("expected no-op for an empty task, got scores=%v cutoff=%f", scores, cutoff)
	}
}
