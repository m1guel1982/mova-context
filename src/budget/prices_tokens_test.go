package budget

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const samplePrices = `{
	"report_path": "./mova-budget-report.md",
	"currency": "USD",
	"exchange_rate_clp": 950,
	"unit": "per_1k_tokens",
	"providers": {
		"openai": {"models": {"gpt-5": {"input": 0.005, "output": 0.015}}},
		"anthropic": {"models": {"claude": {"input": 0.003, "output": 0.015}}}
	}
}`

func writePrices(t *testing.T, root, content string) {
	t.Helper()
	path := PricesPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadPrices_MissingFileReturnsClearError(t *testing.T) {
	root := t.TempDir()
	if _, err := LoadPrices(root); err == nil {
		t.Fatal("expected an error when config/prices.json doesn't exist, got nil")
	}
}

func TestLoadPrices_InvalidJSONReturnsError(t *testing.T) {
	root := t.TempDir()
	writePrices(t, root, "{not valid json")
	if _, err := LoadPrices(root); err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
}

func TestLoadPrices_NoProvidersReturnsError(t *testing.T) {
	root := t.TempDir()
	writePrices(t, root, `{"currency":"USD","providers":{}}`)
	if _, err := LoadPrices(root); err == nil {
		t.Fatal("expected an error when \"providers\" is empty, got nil")
	}
}

func TestLoadPrices_HotReload(t *testing.T) {
	root := t.TempDir()
	writePrices(t, root, samplePrices)

	cfg, err := LoadPrices(root)
	if err != nil {
		t.Fatalf("LoadPrices: %v", err)
	}
	if cfg.Providers["openai"].Models["gpt-5"].Input != 0.005 {
		t.Fatalf("unexpected initial price: %+v", cfg.Providers["openai"])
	}

	// Simulate the user editing prices.json by hand — no recompile, no
	// restart: the very next LoadPrices call must see the new value.
	time.Sleep(10 * time.Millisecond) // ensure a distinct mtime
	updated := `{
		"report_path": "./mova-budget-report.md",
		"currency": "USD",
		"exchange_rate_clp": 950,
		"unit": "per_1k_tokens",
		"providers": {"openai": {"models": {"gpt-5": {"input": 0.999, "output": 0.015}}}}
	}`
	writePrices(t, root, updated)

	cfg2, err := LoadPrices(root)
	if err != nil {
		t.Fatalf("LoadPrices (reload): %v", err)
	}
	if cfg2.Providers["openai"].Models["gpt-5"].Input != 0.999 {
		t.Fatalf("expected hot-reloaded price 0.999, got %v — prices.json was cached instead of reread",
			cfg2.Providers["openai"].Models["gpt-5"].Input)
	}
}

func TestCountTokens_NonEmptyText(t *testing.T) {
	n, encoding, err := CountTokens("Hello, world! This is a test.", "")
	if err != nil {
		t.Fatalf("CountTokens: %v", err)
	}
	if n <= 0 {
		t.Fatalf("expected a positive token count, got %d", n)
	}
	if encoding == "" {
		t.Fatal("expected a non-empty encoding label")
	}
}

func TestCountTokens_ResolvesOpenAIModel(t *testing.T) {
	_, encoding, err := CountTokens("some text", "gpt-4")
	if err != nil {
		t.Fatalf("CountTokens: %v", err)
	}
	if encoding != "cl100k_base" {
		t.Errorf("expected gpt-4 to resolve to cl100k_base, got %q", encoding)
	}
}

func TestCountTokens_FallsBackForUnknownModel(t *testing.T) {
	// Claude/Gemini aren't in tiktoken-go's model list — must still work,
	// falling back to the universal cl100k_base approximation.
	n, encoding, err := CountTokens("some text", "claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("CountTokens: %v", err)
	}
	if n <= 0 || encoding == "" {
		t.Fatalf("expected a valid fallback count/encoding, got n=%d encoding=%q", n, encoding)
	}
}

func TestEstimateCost_ProportionalToTokens(t *testing.T) {
	root := t.TempDir()
	writePrices(t, root, samplePrices)
	prices, err := LoadPrices(root)
	if err != nil {
		t.Fatalf("LoadPrices: %v", err)
	}

	costs := EstimateCost(1000, prices)
	if len(costs) != 2 {
		t.Fatalf("expected 2 provider/model rows, got %d", len(costs))
	}
	for _, c := range costs {
		if c.Provider == "openai" && c.Model == "gpt-5" && c.USD != 0.005 {
			t.Errorf("expected $0.005 for 1000 tokens at $0.005/1k, got %v", c.USD)
		}
	}

	doubled := EstimateCost(2000, prices)
	for i := range doubled {
		if doubled[i].USD != costs[i].USD*2 {
			t.Errorf("cost must scale linearly with tokens: %v vs %v", doubled[i].USD, costs[i].USD)
		}
	}
}

func TestEstimateCost_ZeroTokensIsZeroCost(t *testing.T) {
	root := t.TempDir()
	writePrices(t, root, samplePrices)
	prices, _ := LoadPrices(root)
	for _, c := range EstimateCost(0, prices) {
		if c.USD != 0 {
			t.Errorf("expected $0 for 0 tokens, got %v", c.USD)
		}
	}
}
