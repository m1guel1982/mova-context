// pii_policy.go — loads config/policy.json, the ONLY place every
// threshold/weight/tag the PII Masking stage (pii.go) uses is defined.
// Same "no magic numbers, no hardcoded constants in Go" rule
// budget/prices.go already follows for config/prices.json: nothing in
// this file decides what counts as "PII-like" on its own — it only
// reads numbers a person configured, with a conservative built-in
// default for the (normal) case where config/policy.json is absent.
package sanitize

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// PIIShapeRules are the structural (word-shape) scoring knobs — see
// config/policy.json's own "_comment" field for what each one means.
// Every one of these is a plain number/threshold; none of them is a
// word, a language rule, or a dictionary entry (see wordShapeScore in
// pii.go for how they combine).
type PIIShapeRules struct {
	DigitRatioThreshold          float64 `json:"digit_ratio_threshold"`
	DigitRatioBonus              float64 `json:"digit_ratio_bonus"`
	SeparatorMinCount            int     `json:"separator_min_count"`
	SeparatorDigitRatioThreshold float64 `json:"separator_digit_ratio_threshold"`
	SeparatorBonus               float64 `json:"separator_bonus"`
	AtSymbolBonus                float64 `json:"at_symbol_bonus"`
	LongTokenBonusLen            int     `json:"long_token_bonus_len"`
	LongMixedTokenBonus          float64 `json:"long_mixed_token_bonus"`
	UpperRunRatioThreshold       float64 `json:"upper_run_ratio_threshold"`
	UpperRunBonus                float64 `json:"upper_run_bonus"`
}

// PIIPolicy maps config/policy.json's "pii_masking" object exactly.
type PIIPolicy struct {
	MinScore       float64       `json:"min_score"`
	ShapeWeight    float64       `json:"shape_weight"`
	EntropyWeight  float64       `json:"entropy_weight"`
	MinTokenLength int           `json:"min_token_length"`
	HashLength     int           `json:"hash_length"`
	TagFormat      string        `json:"tag_format"`
	ShapeRules     PIIShapeRules `json:"shape_rules"`
}

type policyFile struct {
	PIIMasking PIIPolicy `json:"pii_masking"`
}

// DefaultPIIPolicy is used only if config/policy.json is missing or
// unreadable — conservative values matching the ones this repository
// ships in config/policy.json, so behavior is identical whether the
// file is present or not (the file exists precisely so a person can
// change these without touching Go code, not because the code needs it
// to run).
func DefaultPIIPolicy() PIIPolicy {
	return PIIPolicy{
		MinScore:       0.62,
		ShapeWeight:    0.65,
		EntropyWeight:  0.35,
		MinTokenLength: 4,
		HashLength:     8,
		TagFormat:      "[PII_%s]",
		ShapeRules: PIIShapeRules{
			DigitRatioThreshold:          0.3,
			DigitRatioBonus:              0.3,
			SeparatorMinCount:            2,
			SeparatorDigitRatioThreshold: 0.2,
			SeparatorBonus:               0.25,
			AtSymbolBonus:                0.5,
			LongTokenBonusLen:            10,
			LongMixedTokenBonus:          0.15,
			UpperRunRatioThreshold:       0.6,
			UpperRunBonus:                0.15,
		},
	}
}

// PolicyPath resolves config/policy.json under root — same convention
// as budget.LoadPrices resolving config/prices.json.
func PolicyPath(root string) string {
	return filepath.Join(root, "config", "policy.json")
}

// LoadPIIPolicy reads config/policy.json. A missing file, invalid JSON,
// or a "pii_masking" object with MinScore==0 (i.e. the field was never
// set) all fall back to DefaultPIIPolicy() — this stage is off by
// default per-project anyway (core.PIIMaskingEnabled), so a missing
// policy file is never itself an error, only a signal to use the safe
// built-in numbers.
func LoadPIIPolicy(root string) PIIPolicy {
	data, err := os.ReadFile(PolicyPath(root))
	if err != nil {
		return DefaultPIIPolicy()
	}
	var f policyFile
	if err := json.Unmarshal(data, &f); err != nil {
		return DefaultPIIPolicy()
	}
	if f.PIIMasking.MinScore == 0 {
		return DefaultPIIPolicy()
	}
	return f.PIIMasking
}
