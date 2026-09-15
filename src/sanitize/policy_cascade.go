// policy_cascade.go — the modular, cascading policy orchestrator:
// config/policy.json holds an ordered list of policy file names, each
// loaded from config/policy/ — so adding or changing a rule is always
// a JSON edit, never a Go recompile.
package sanitize

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// SecurityPolicy maps security settings.
type SecurityPolicy struct {
	BlockOnPrivateKey           bool `json:"block_on_private_key"`
	BlockOnAPIKey               bool `json:"block_on_api_key"`
	SanitizeOnGenericCredential bool `json:"sanitize_on_generic_credential"`
	SanitizeOnJWT               bool `json:"sanitize_on_jwt"`
}

// ReviewPolicy maps review and filtering settings.
type ReviewPolicy struct {
	DocumentationAction   string   `json:"documentation_action"`   // "ALLOW"
	GeneratedFilesAction  string   `json:"generated_files_action"` // "EXCLUDE"
	GeneratedFilePatterns []string `json:"generated_file_patterns"`
}

// CompliancePolicy maps compliance and token limits.
type CompliancePolicy struct {
	TokenBudget                 int      `json:"token_budget"`
	ExternalModelDenyCategories []string `json:"external_model_deny_categories"`
}

// piiPolicyFile maps PII masking settings and model permissions.
type piiPolicyFile struct {
	PIIMasking    PIIPolicy `json:"pii_masking"`
	ExternalModel string    `json:"external_model"` // "allow" | "sanitize" | "deny"
}

// orchestratorFile maps config/policy.json in cascade mode.
type orchestratorFile struct {
	Policies []string `json:"policies"`
	Version  string   `json:"version"`
}

// PolicySet is every governance dimension resolved dynamically from the cascade.
type PolicySet struct {
	Source           string // human-readable provenance
	Version          string
	Security         SecurityPolicy
	Review           ReviewPolicy
	Compliance       CompliancePolicy
	PII              PIIPolicy
	PIIExternalModel string
}

// defaultPolicySet provides safe fallback defaults.
func defaultPolicySet(source string) PolicySet {
	return PolicySet{
		Source:  source,
		Version: "1.0 (built-in defaults)",
		Security: SecurityPolicy{
			BlockOnPrivateKey:           true,
			BlockOnAPIKey:               false,
			SanitizeOnGenericCredential: true,
			SanitizeOnJWT:               true,
		},
		Review: ReviewPolicy{
			DocumentationAction:  "ALLOW",
			GeneratedFilesAction: "EXCLUDE",
			GeneratedFilePatterns: []string{
				"dist/", "build/", "vendor/", "node_modules/", ".git/", "*.min.js", "*.lock",
			},
		},
		Compliance: CompliancePolicy{
			TokenBudget:                 500000,
			ExternalModelDenyCategories: []string{"personal-data pattern"},
		},
		PII:              DefaultPIIPolicy(),
		PIIExternalModel: "sanitize",
	}
}

// LoadPolicySet reads config/policy.json under root and resolves the full PolicySet.
func LoadPolicySet(root string) PolicySet {
	path := PolicyPath(root)
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultPolicySet("config/policy.json not found — using built-in defaults")
	}

	var orch orchestratorFile
	if err := json.Unmarshal(data, &orch); err == nil && len(orch.Policies) > 0 {
		return loadCascade(root, orch)
	}

	var legacy policyFile
	if err := json.Unmarshal(data, &legacy); err == nil && legacy.PIIMasking.MinScore > 0 {
		ps := defaultPolicySet("config/policy.json (legacy single-file format — only pii_masking read)")
		ps.PII = legacy.PIIMasking
		return ps
	}

	return defaultPolicySet("config/policy.json present but not in a recognized format — using built-in defaults")
}

// loadCascade reads every file listed in orch.Policies regardless of its name.
// Decodes by inspecting schema structure, merging non-zero fields into PolicySet.
func loadCascade(root string, orch orchestratorFile) PolicySet {
	ps := defaultPolicySet("")
	dir := filepath.Join(root, "config", "policy")
	var loaded []string

	for _, name := range orch.Policies {
		filePath := filepath.Join(dir, name)
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		matched := false

		// 1. Probar estructura PII
		var pf piiPolicyFile
		if err := json.Unmarshal(data, &pf); err == nil && (pf.PIIMasking.MinScore > 0 || pf.ExternalModel != "") {
			if pf.PIIMasking.MinScore > 0 {
				ps.PII = pf.PIIMasking
			}
			if pf.ExternalModel != "" {
				ps.PIIExternalModel = pf.ExternalModel
			}
			matched = true
		}

		// 2. Probar estructura Security
		var sec SecurityPolicy
		if err := json.Unmarshal(data, &sec); err == nil && (sec.BlockOnPrivateKey || sec.BlockOnAPIKey || sec.SanitizeOnGenericCredential || sec.SanitizeOnJWT) {
			ps.Security = sec
			matched = true
		}

		// 3. Probar estructura Review
		var rev ReviewPolicy
		if err := json.Unmarshal(data, &rev); err == nil && (rev.DocumentationAction != "" || rev.GeneratedFilesAction != "" || len(rev.GeneratedFilePatterns) > 0) {
			ps.Review = rev
			matched = true
		}

		// 4. Probar estructura Compliance
		var comp CompliancePolicy
		if err := json.Unmarshal(data, &comp); err == nil && (comp.TokenBudget > 0 || len(comp.ExternalModelDenyCategories) > 0) {
			ps.Compliance = comp
			matched = true
		}

		if matched {
			loaded = append(loaded, name)
		}
	}

	version := orch.Version
	if version == "" {
		version = "1.0"
	}
	ps.Version = version
	ps.Source = "config/policy.json -> config/policy/{" + strings.Join(loaded, ", ") + "}"
	return ps
}
