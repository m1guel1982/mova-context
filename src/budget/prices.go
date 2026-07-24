// prices.go — lee config/prices.json del disco. Mismo patrón de recarga
// en caliente que models/config.go (ConfigCache): se cachea junto con el
// mtime del archivo, y se relee solo si cambió. Ningún precio queda
// "compilado" en el binario — cambiar config/prices.json y volver a
// correr `mova budget` toma los valores nuevos de inmediato, sin
// recompilar ni reiniciar nada.
package budget

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// PriceEntry son los precios de un modelo, en USD por unidad (ver Unit).
type PriceEntry struct {
	Input  float64 `json:"input"`
	Output float64 `json:"output"`
}

// ProviderPrices agrupa los modelos de un proveedor (openai, anthropic,
// google, ...). Agregar un proveedor o modelo nuevo es solo JSON — nunca
// requiere tocar código (ver EstimateCost).
type ProviderPrices struct {
	Models map[string]PriceEntry `json:"models"`
}

// PricesConfig mapea config/prices.json exactamente.
type PricesConfig struct {
	ReportPath      string                    `json:"report_path"`
	Currency        string                    `json:"currency"`
	ExchangeRateCLP float64                   `json:"exchange_rate_clp"`
	Unit            string                    `json:"unit"` // "per_1k_tokens" (default) | "per_1m_tokens"
	Providers       map[string]ProviderPrices `json:"providers"`
}

// PricesPath — config/prices.json dentro de la raíz del proyecto Mova.
func PricesPath(root string) string {
	return filepath.Join(root, "config", "prices.json")
}

// UnitDivisor devuelve el divisor de tokens según Unit ("per_1k_tokens" —
// default si Unit está vacío — o "per_1m_tokens").
func (p *PricesConfig) UnitDivisor() float64 {
	if p.Unit == "per_1m_tokens" {
		return 1_000_000
	}
	return 1_000
}

// ── caché con recarga en caliente (mismo patrón que models.ConfigCache) ────

type cachedPrices struct {
	cfg     PricesConfig
	modTime int64
}

var (
	priceMu    sync.RWMutex
	priceCache map[string]cachedPrices // key: ruta absoluta de prices.json
)

// LoadPrices lee config/prices.json, releyendo el archivo solo si cambió
// desde la última lectura de este proceso. Error claro (y sin caché) si el
// archivo no existe o el JSON es inválido — nunca un default silencioso.
func LoadPrices(root string) (*PricesConfig, error) {
	path := PricesPath(root)
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("no existe %s — creá el archivo con tus precios (ver docs/i18n/es/COMMANDS.md#mova-budget): %w", path, err)
	}
	mtime := info.ModTime().UnixNano()

	priceMu.RLock()
	cached, ok := priceCache[path]
	priceMu.RUnlock()
	if ok && cached.modTime == mtime {
		cfg := cached.cfg
		return &cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg PricesConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%s: JSON inválido: %w", path, err)
	}
	if len(cfg.Providers) == 0 {
		return nil, fmt.Errorf("%s: no define ningún \"providers\" — no hay nada que estimar", path)
	}

	priceMu.Lock()
	if priceCache == nil {
		priceCache = map[string]cachedPrices{}
	}
	priceCache[path] = cachedPrices{cfg: cfg, modTime: mtime}
	priceMu.Unlock()

	return &cfg, nil
}

// ReportPath resuelve dónde escribir mova-budget-report.md: cfg.ReportPath
// relativo a root si no es absoluto (mismo criterio que el resto de la
// configuración global en config/), con "./mova-budget-report.md" como
// default si el campo viene vacío.
func ReportPath(root string, cfg *PricesConfig) string {
	p := cfg.ReportPath
	if p == "" {
		p = "./mova-budget-report.md"
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(root, p)
}

// SortedProviderNames — nombres de proveedor en orden determinista (para
// que el reporte y los tests nunca dependan del orden de un map de Go).
func (p *PricesConfig) SortedProviderNames() []string {
	names := make([]string, 0, len(p.Providers))
	for name := range p.Providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
