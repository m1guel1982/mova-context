package dedup

import "testing"

func TestParagraphs_RemovesExactDuplicate(t *testing.T) {
	seen := map[string]bool{}
	first, removed1, chars1 := Paragraphs("Hello world.\n\nSecond paragraph.", seen)
	if removed1 != 0 || chars1 != 0 {
		t.Fatalf("expected 0 removed/0 chars on first pass, got %d/%d", removed1, chars1)
	}
	if first != "Hello world.\n\nSecond paragraph." {
		t.Fatalf("unexpected first pass output: %q", first)
	}

	second, removed2, chars2 := Paragraphs("Hello world.\n\nA new paragraph.", seen)
	if removed2 != 1 {
		t.Fatalf("expected 1 removed on second pass (Hello world. repeats), got %d", removed2)
	}
	if chars2 != len("Hello world.") {
		t.Fatalf("expected removedChars to equal the removed paragraph's length, got %d", chars2)
	}
	if second != "A new paragraph." {
		t.Fatalf("unexpected second pass output: %q", second)
	}
}

func TestParagraphs_IgnoresWhitespaceDifferences(t *testing.T) {
	seen := map[string]bool{}
	Paragraphs("Hello   world.\nWith  a line break.", seen)
	_, removed, _ := Paragraphs("Hello world. With a line break.", seen)
	if removed != 1 {
		t.Fatalf("expected whitespace-normalized match to count as duplicate, got removed=%d", removed)
	}
}

func TestParagraphs_KeepsBlankParagraphsAndDistinctText(t *testing.T) {
	seen := map[string]bool{}
	out, removed, chars := Paragraphs("One.\n\n\nTwo.\n\nThree.", seen)
	if removed != 0 || chars != 0 {
		t.Fatalf("expected 0 removed for all-distinct paragraphs, got %d/%d", removed, chars)
	}
	if out == "" {
		t.Fatal("expected non-empty output")
	}
}

func TestParagraphs_SharedSeenAcrossMultipleCalls(t *testing.T) {
	seen := map[string]bool{}
	Paragraphs("Shared warning text.", seen)
	Paragraphs("Something else entirely.", seen)
	_, removed, _ := Paragraphs("Shared warning text.", seen)
	if removed != 1 {
		t.Fatalf("expected a third call to still detect the duplicate from the first, got removed=%d", removed)
	}
}
