// patch_apply_test.go — confirmation detection and labeled code block
// parsing for Mova Chat's Auto-Apply flow (see patch_apply.go).
package documents

import "testing"

func TestDetectApplyConfirmation(t *testing.T) {
	cases := map[string]bool{
		"sí":                     true,
		"si":                     true,
		"  yes  ":                true,
		"s":                      true,
		"y":                      true,
		"#1":                     true,
		"#1, #3":                 true,
		"si claro que sí, hazlo": false, // not a bare confirmation
		"":                       false,
		"no":                     false,
	}
	for input, want := range cases {
		got, _ := DetectApplyConfirmation(input)
		if got != want {
			t.Errorf("DetectApplyConfirmation(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestDetectApplyConfirmation_ExtractsIDs(t *testing.T) {
	_, ids := DetectApplyConfirmation("#1, #3")
	if len(ids) != 2 || ids[0] != "1" || ids[1] != "3" {
		t.Fatalf("expected [1 3], got %v", ids)
	}
}

func TestExtractLabeledCodeBlocks(t *testing.T) {
	text := "Here is the fix:\n\n```javascript:server.js\nconsole.log('hi')\n```\n\nAnd:\n\n```go:auth/login.go::Login()\nfunc Login() {}\n```\n\nUntagged:\n\n```\nplain code\n```\n"
	blocks := ExtractLabeledCodeBlocks(text)
	if len(blocks) != 2 {
		t.Fatalf("expected 2 labeled blocks (untagged one skipped), got %d: %+v", len(blocks), blocks)
	}
	if blocks[0].Path != "server.js" || blocks[0].Symbol != "" {
		t.Errorf("block 0 = %+v", blocks[0])
	}
	if blocks[1].Path != "auth/login.go" || blocks[1].Symbol != "Login" {
		t.Errorf("block 1 = %+v", blocks[1])
	}
}

func TestLooksLikeDirectApplyProposal(t *testing.T) {
	if !LooksLikeDirectApplyProposal("Aplicación Directa: aquí está el cambio\n```go:x.go\npackage x\n```") {
		t.Error("expected marker phrase to be recognized")
	}
	if !LooksLikeDirectApplyProposal("```go:x.go\npackage x\n```") {
		t.Error("expected a labeled block alone to be recognized")
	}
	if LooksLikeDirectApplyProposal("just a normal answer, no code") {
		t.Error("expected an ordinary answer to NOT look like a proposal")
	}
}
