// policy_cascade.go — the modular, cascading policy orchestrator:
// config/policy.json holds an ordered list of policy file names, each
// loaded from config/policy/ — so adding or changing a rule is always
// a JSON edit, never a Go recompile (same discipline pii_policy.go
// already established for the single pii_masking object; this simply
// extends it to the other governance dimensions context-trace needs:
// security, review, and compliance). LoadPIIPolicy (pii_policy.go)
// keeps working exactly as before for every OTHER caller in this
// codebase (the Context Governance's PII Masking stage) — this file only
// adds a second, richer way to read policy for context-trace, backed
// by the very same config/policy.json path.
package sanitize

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"mova.local/core"
)

// SecurityPolicy maps config/policy/security.json.
type SecurityPolicy struct {
	BlockOnPrivateKey           bool `json:"block_on_private_key"`
	BlockOnAPIKey               bool `json:"block_on_api_key"`
	SanitizeOnGenericCredential bool `json:"sanitize_on_generic_credential"`
	SanitizeOnJWT               bool `json:"sanitize_on_jwt"`
}

// ReviewPolicy maps config/policy/review.json.
type ReviewPolicy struct {
	DocumentationAction   string   `json:"documentation_action"`   // "ALLOW"
	GeneratedFilesAction  string   `json:"generated_files_action"` // "EXCLUDE"
	GeneratedFilePatterns []string `json:"generated_file_patterns"`
}

// CompliancePolicy maps config/policy/compliance.json.
type CompliancePolicy struct {
	TokenBudget                 int      `json:"token_budget"`
	ExternalModelDenyCategories []string `json:"external_model_deny_categories"`
}

// piiPolicyFile maps config/policy/pii.json — the same "pii_masking"
// object LoadPIIPolicy already knows how to read, plus one field
// specific to context-trace's governance decision.
type piiPolicyFile struct {
	PIIMasking    PIIPolicy `json:"pii_masking"`
	ExternalModel string    `json:"external_model"` // "allow" | "sanitize" | "deny"
}

// orchestratorFile maps config/policy.json in cascade mode. Policies
// accepts both the bare-array and the {include,exclude} object form —
// see core.PolicySelector.
type orchestratorFile struct {
	Policies *core.PolicySelector `json:"policies"`
	Version  string               `json:"version"`
}

// PolicySet is every governance dimension context-trace evaluates a
// file against, resolved from the cascade (or safe defaults) exactly
// once per run.
type PolicySet struct {
	Source           string // human-readable provenance, shown in the "POLICY" report section
	Version          string
	Security         SecurityPolicy
	Review           ReviewPolicy
	Compliance       CompliancePolicy
	PII              PIIPolicy
	PIIExternalModel string
}

// defaultPolicySet is the conservative, production-ready built-in
// configuration used whenever config/policy.json (or one of the files
// it references) is missing, unreadable, or incomplete — behavior is
// never "no policy", only "the documented safe default".
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

// LoadPolicySet reads config/policy.json under root and resolves the
// full PolicySet using ONLY that file's own selection — the behavior
// every caller had before per-project/CLI policy selection existed.
// New callers should prefer LoadPolicySetFor, which honors the full
// precedence chain (CLI > project.json > config/policy.json).
func LoadPolicySet(root string) PolicySet {
	sel := core.OrchestratorPolicySelector(root)
	// sel is nil whenever config/policy.json is missing/unreadable or
	// declares no "policies" key (see OrchestratorPolicySelector's own
	// doc comment) — every deployed copy of Mova ships one, but a
	// project root that doesn't (a fresh `mova init`, a test fixture,
	// an embedded/sandboxed root) must fall through to
	// defaultPolicySet via LoadPolicySetFor exactly like a missing
	// config/policy/*.json sub-file already does, not panic. Only
	// sel.Include/sel.Exclude (plain field reads) needed the nil
	// guard — sel.Declared() is already nil-safe (see
	// PolicySelector.Declared in core/types.go).
	var include, exclude []string
	if sel != nil {
		include, exclude = sel.Include, sel.Exclude
	}
	return LoadPolicySetFor(root, core.PolicyRequest{
		Include:  include,
		Exclude:  exclude,
		Declared: sel.Declared(),
		Origin:   "config/policy.json",
	})
}

// LoadPIIPolicyForProject resolves the PII Masking policy honoring the
// FULL policy precedence (project.json's own "policies" overrides
// config/policy.json's orchestrator selection, which falls back to
// built-in defaults — see core.ResolvePolicyRequest) — the exact same
// precedence context_trace (trace/analyzer.go, trace/governance.go)
// and `mova chat`'s own [debug] policy report (core.
// writePolicyDebugLines) already honor.
//
// This exists because pii_policy.go's LoadPIIPolicy(root) — kept
// unchanged for backward compatibility — only ever reads
// config/policy.json's orchestrator-level selection; it has no project
// argument, so it silently IGNORES a project.json that declares its
// own "policies" (e.g. a project opting into a stricter
// "pii_strict.json" or a custom one). Every Context Governance PII
// Masking call site that has a *core.Project in scope (budget.
// applyPIIMasking, used by BuildGatedContext AND estimate.go's
// preview) should call THIS function instead, so "which PII policy
// applies" answers the same question everywhere Mova asks it — not
// just for context-trace's own report but for the pipeline that
// actually masks Focus/Memory before anything leaves. proj may be nil
// (discovery mode) and behaves exactly like LoadPIIPolicy's
// orchestrator/defaults fallback in that case.
func LoadPIIPolicyForProject(root string, proj *core.Project) PIIPolicy {
	req := core.ResolvePolicyRequest(proj, core.OrchestratorPolicySelector(root), nil, nil)
	return LoadPolicySetFor(root, req).PII
}

// OrchestratorSelector reads just the "policies" value out of
// config/policy.json — a thin alias for core.OrchestratorPolicySelector
// kept here so existing callers importing "sanitize" don't need to
// change. See core/policy_resolve.go for why the real implementation
// lives in `core`: core.BuildContext (used by `mova chat`) needs the
// exact same resolution for its own [debug] output, and `core` cannot
// import `sanitize` (sanitize already imports core).
func OrchestratorSelector(root string) *core.PolicySelector {
	return core.OrchestratorPolicySelector(root)
}

// orchestratorVersion reads config/policy.json's "version", or "" when
// unavailable — kept separate so a run driven by CLI/project.json
// policies still reports the repository's policy version.
func orchestratorVersion(root string) string {
	data, err := os.ReadFile(PolicyPath(root))
	if err != nil {
		return ""
	}
	var orch orchestratorFile
	if err := json.Unmarshal(data, &orch); err != nil {
		return ""
	}
	return orch.Version
}

// LoadPolicySetFor resolves the full PolicySet for an already-decided
// core.PolicyRequest (see core.ResolvePolicyRequest for precedence).
//
// Three outcomes, all of which return a usable PolicySet — this engine
// never returns "no policy at all", only "the documented safe default"
// (see defaultPolicySet):
//   - req.Declared == false → built-in defaults only, no files read.
//   - req.Declared == true  → every resolved file is merged, in order.
//   - legacy config/policy.json ({"pii_masking": {...}} with no
//     "policies" key) → read for the PII dimension, as before.
func LoadPolicySetFor(root string, req core.PolicyRequest) PolicySet {
	if !req.Declared {
		if legacy, ok := loadLegacyPolicyFile(root); ok {
			return legacy
		}
		return defaultPolicySet("no policies declared — built-in defaults only")
	}
	return loadCascade(root, req)
}

// loadLegacyPolicyFile reads the pre-cascade single-file format (a bare
// {"pii_masking": {...}} object), which this repository shipped before
// the orchestrator existed — still honored so an old checkout keeps its
// PII settings without editing anything.
func loadLegacyPolicyFile(root string) (PolicySet, bool) {
	data, err := os.ReadFile(PolicyPath(root))
	if err != nil {
		return PolicySet{}, false
	}
	var legacy policyFile
	if err := json.Unmarshal(data, &legacy); err != nil || legacy.PIIMasking.MinScore <= 0 {
		return PolicySet{}, false
	}
	ps := defaultPolicySet("config/policy.json (legacy single-file format — only pii_masking read)")
	ps.PII = legacy.PIIMasking
	return ps, true
}

// loadCascade reads every file the request resolves to (see
// core.ResolvedPolicyFiles for path resolution, recursive search and
// exclusion), merging each into a PolicySet that starts from safe
// defaults — a policy file that's missing or invalid JSON is skipped,
// never fatal (see analyzer.go's DISCOVERY ONLY mode, which must
// always be able to run even with an incomplete config/ tree).
//
// Which dimension a file feeds is decided by its CONTENT, not its
// name, so a custom file like "pii_strict_ventas.json" or
// "security-prod.json" merges into the right dimension without having
// to be named exactly "pii.json"/"security.json".
func loadCascade(root string, req core.PolicyRequest) PolicySet {
	ps := defaultPolicySet("")
	var loaded []string

	for _, path := range core.ResolvedPolicyFiles(root, req) {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if mergePolicyFile(&ps, data) {
			loaded = append(loaded, filepath.Base(path))
		}
	}

	version := orchestratorVersion(root)
	if version == "" {
		version = "1.0"
	}
	ps.Version = version
	if len(loaded) == 0 {
		ps.Source = req.Origin + " — no policy file resolved, built-in defaults only"
	} else {
		ps.Source = req.Origin + " -> {" + strings.Join(loaded, ", ") + "}"
	}
	return ps
}

// mergePolicyFile detects which governance dimension a policy file
// describes by looking at which known top-level keys it actually
// declares, then merges it. Returns false for JSON that parses but
// matches no known dimension (accepted for forward compatibility, but
// not counted as loaded). A single file MAY declare more than one
// dimension; each is merged independently.
func mergePolicyFile(ps *PolicySet, data []byte) bool {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return false
	}
	has := func(keys ...string) bool {
		for _, k := range keys {
			if _, ok := probe[k]; ok {
				return true
			}
		}
		return false
	}

	merged := false
	if has("block_on_private_key", "block_on_api_key", "sanitize_on_generic_credential", "sanitize_on_jwt") {
		if json.Unmarshal(data, &ps.Security) == nil {
			merged = true
		}
	}
	if has("documentation_action", "generated_files_action", "generated_file_patterns") {
		if json.Unmarshal(data, &ps.Review) == nil {
			merged = true
		}
	}
	if has("token_budget", "external_model_deny_categories") {
		if json.Unmarshal(data, &ps.Compliance) == nil {
			merged = true
		}
	}
	if has("pii_masking", "external_model") {
		var pf piiPolicyFile
		if json.Unmarshal(data, &pf) == nil {
			if pf.PIIMasking.MinScore > 0 {
				ps.PII = pf.PIIMasking
			}
			if pf.ExternalModel != "" {
				ps.PIIExternalModel = pf.ExternalModel
			}
			merged = true
		}
	}
	return merged
}
