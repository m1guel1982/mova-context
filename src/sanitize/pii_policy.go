// pii_policy.go — loads config/policy.json via LoadPolicySet, supporting
// both legacy single-file format and the modular cascading policy engine.
package sanitize

import (
	"path/filepath"
)

// PIIShapeRules are the structural (word-shape) scoring knobs — see
// config/policy.json's own "_comment" field for what each one means.
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

// PIIPolicy maps the "pii_masking" object configuration.
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

// DefaultPIIPolicy is used only if config/policy.json (and its sub-policies)
// are missing, unreadable, or invalid — conservative values matching system defaults.
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

// PolicyPath resolves config/policy.json under root.
func PolicyPath(root string) string {
	return filepath.Join(root, "config", "policy.json")
}

// LoadPIIPolicy delegates resolution to LoadPolicySet, ensuring that any
// dynamic file listed in config/policy.json (like pii_permissive.json or custom JSONs)
// is correctly evaluated and loaded into the PII Masking stage.
func LoadPIIPolicy(root string) PIIPolicy {
	ps := LoadPolicySet(root)
	return ps.PII
}
