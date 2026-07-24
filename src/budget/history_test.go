package budget

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"mova.local/core"
)

func TestHistoryPath_DefaultsAlongsideProjectJSON(t *testing.T) {
	root := t.TempDir()
	path := HistoryPath(root, "my-project", &core.Project{})
	want := filepath.Join(root, "projects", "my-project", "mova-token-history.json")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestHistoryPath_UsesCustomPathFromProject(t *testing.T) {
	root := t.TempDir()
	proj := &core.Project{TokenHistoryPath: "custom/history.json"}
	path := HistoryPath(root, "my-project", proj)
	want := filepath.Join(root, "custom", "history.json")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestLoadHistory_MissingFileIsNotAnError(t *testing.T) {
	root := t.TempDir()
	history, err := LoadHistory(filepath.Join(root, "does-not-exist.json"))
	if err != nil {
		t.Fatalf("expected no error for a missing file, got: %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("expected empty history, got: %+v", history)
	}
}

func TestRecordUsage_AccumulatesPerProvider(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "mova-token-history.json")

	if err := RecordUsage(path, "anthropic", 1000, 1023); err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}
	if err := RecordUsage(path, "anthropic", 500, 520); err != nil {
		t.Fatalf("RecordUsage (second call): %v", err)
	}
	if err := RecordUsage(path, "openai", 200, 199); err != nil {
		t.Fatalf("RecordUsage (openai): %v", err)
	}

	history, err := LoadHistory(path)
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	anthropic := history["anthropic"]
	if anthropic.TotalLocalTokens != 1500 || anthropic.TotalAPITokens != 1543 {
		t.Fatalf("expected accumulated anthropic totals {1500,1543}, got %+v", anthropic)
	}
	openai := history["openai"]
	if openai.TotalLocalTokens != 200 || openai.TotalAPITokens != 199 {
		t.Fatalf("expected openai totals {200,199}, got %+v", openai)
	}
}

func TestRecordUsage_NeverStoresPromptsOrContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "mova-token-history.json")
	RecordUsage(path, "anthropic", 10, 11)

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading history file: %v", err)
	}
	// El archivo solo debe tener las dos claves numéricas — ni timestamps,
	// ni ids, ni nada que se parezca a contenido de proyecto.
	var parsed map[string]map[string]int
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unexpected file shape: %v (%s)", err, raw)
	}
	fields := parsed["anthropic"]
	if len(fields) != 2 {
		t.Fatalf("expected exactly 2 fields per provider, got %d: %v", len(fields), fields)
	}
}

func TestDeviationPercent_NoDataYet(t *testing.T) {
	history := TokenHistory{}
	_, ok := history.DeviationPercent("google")
	if ok {
		t.Fatal("expected ok=false for a provider with no recorded usage")
	}
}

func TestDeviationPercent_ComputesFromAccumulators(t *testing.T) {
	history := TokenHistory{
		"anthropic": ProviderAccumulator{TotalLocalTokens: 1000, TotalAPITokens: 1023},
	}
	percent, ok := history.DeviationPercent("anthropic")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if percent != 2.3 {
		t.Fatalf("expected (1023-1000)/1000*100 = 2.3, got %v", percent)
	}
}
