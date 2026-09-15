// policy_author.go — resolves the audit question "who authorized this
// policy?" (see README § Audit Matrix, #3). Hierarchy, in order — the
// first non-empty source wins:
//  1. "author" in projects/<project>/project.json
//  2. "author" in config/policy.json (project fallback, CI/CD)
//  3. environment variable MOVA_POLICY_AUTHOR
//  4. "system:default" — never left empty.
package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DefaultPolicyAuthor is the final value when no source declares an
// author — an audit report never leaves this field empty.
const DefaultPolicyAuthor = "system:default"

// ResolvePolicyAuthor implements the hierarchy documented above.
// project may be "" (remote analysis, no project.json) — step 1 is
// then skipped and resolution continues with config/policy.json → env.
func ResolvePolicyAuthor(root, project string) string {
	if project != "" {
		if a := readAuthorField(filepath.Join(root, "projects", project, "project.json")); a != "" {
			return a
		}
	}
	if a := readAuthorField(filepath.Join(root, "config", "policy.json")); a != "" {
		return a
	}
	if a := strings.TrimSpace(os.Getenv("MOVA_POLICY_AUTHOR")); a != "" {
		return a
	}
	return DefaultPolicyAuthor
}

// TargetModelFor resolves the audit question "which model received
// it?" (see README § Audit Matrix, #11) from the llm_profile declared
// in project.json — "n/a" when the project declares none (e.g. a
// remote analysis with no project.json).
func TargetModelFor(root, project string) string {
	if project == "" {
		return "n/a"
	}
	data, err := os.ReadFile(filepath.Join(root, "projects", project, "project.json"))
	if err != nil {
		return "n/a"
	}
	var probe struct {
		LLMProfile *LLMProfile `json:"llm_profile"`
		LLM        string      `json:"llm"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return "n/a"
	}
	if probe.LLMProfile != nil {
		provider := strings.TrimSpace(probe.LLMProfile.Provider)
		model := strings.TrimSpace(probe.LLMProfile.Config)
		switch {
		case provider != "" && model != "":
			return provider + "/" + model
		case model != "":
			return model
		case provider != "":
			return provider
		}
	}
	if probe.LLM != "" {
		return probe.LLM
	}
	return "n/a"
}

// readAuthorField reads only the "author" key from a JSON file on
// disk, without requiring the rest of its schema — this way it works
// for both project.json (struct Project) and config/policy.json
// (which doesn't declare "author" in its Go struct, only in the file).
func readAuthorField(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var probe struct {
		Author string `json:"author"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return ""
	}
	return strings.TrimSpace(probe.Author)
}
