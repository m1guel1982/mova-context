// policy_resolve.go — decides WHICH policy files a run loads, before
// sanitize.LoadPolicySetFor (mova.local/sanitize/policy_cascade.go)
// decides what's inside them. Two separate jobs, two separate
// packages: this one is pure path/precedence logic with no schema
// knowledge, and lives in `core` (not `sanitize`) specifically so any
// door — including core.BuildContext's own [debug] output for `mova
// chat` — can print resolved policy paths without importing sanitize.
//
// Precedence, highest first (see docs/i18n/{es,en}/PROJECT_JSON.md
// § policies and COMMANDS.md § context-trace):
//
//  1. CLI flags --policies_include / --policies_exclude
//  2. project.json's "policies"        (completely overrides #3)
//  3. config/policy.json's "policies"  (only when there is no project)
//
// A project.json that declares no "policies" key is opt-in OFF: no
// policy FILES are loaded for that run (see ResolvePolicyRequest's
// doc comment for the one safety caveat that rule carries).
package core

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"mova.local/documents"
)

// PolicyRequest is the resolved answer to "which policy files should
// this run load?" — handed to LoadPolicySetFor (policy_cascade.go).
type PolicyRequest struct {
	// Include/Exclude: see PolicySelector for the accepted forms.
	Include []string
	Exclude []string
	// Declared is false when no layer declared a selection at all. A
	// run with Declared == false loads NO policy files.
	Declared bool
	// Origin is human-readable provenance for the report's POLICY
	// section: "CLI", "project.json", "config/policy.json", or "none".
	Origin string
}

// ResolvePolicyRequest applies the precedence documented above.
//
// cliInclude/cliExclude come from --policies_include/--policies_exclude
// (already split on commas by the caller); proj is the loaded
// project.json, or nil in discovery mode (`context-trace --repo <url>`,
// where there is no project.json at all); orchestrator is
// config/policy.json's own "policies" value, or nil when absent.
//
// SAFETY NOTE — the opt-in rule ("no policies key in project.json =>
// no policies") is deliberately fail-OPEN, because it is what
// PROJECT_JSON.md documents. It only switches off the loading of policy
// FILES: LoadPolicySetFor still returns defaultPolicySet()'s
// conservative built-ins, so a project that declares nothing never
// silently loses private-key blocking or PII masking. Anything
// stricter than the built-ins has to be declared explicitly.
func ResolvePolicyRequest(proj *Project, orchestrator *PolicySelector, cliInclude, cliExclude []string) PolicyRequest {
	cliInclude, cliExclude = cleanList(cliInclude), cleanList(cliExclude)
	if len(cliInclude) > 0 || len(cliExclude) > 0 {
		return PolicyRequest{Include: cliInclude, Exclude: cliExclude, Declared: true, Origin: "CLI"}
	}

	if proj != nil && proj.Policies.Declared() {
		// A declared-but-empty "policies": [] is a real instruction —
		// "run with no policy files" — not a fallthrough to the
		// orchestrator. That's what "overrides completely" means.
		return PolicyRequest{
			Include:  cleanList(proj.Policies.Include),
			Exclude:  cleanList(proj.Policies.Exclude),
			Declared: true,
			Origin:   "project.json",
		}
	}

	// Discovery mode (no project.json) is the only case that falls
	// through to config/policy.json — a project that exists but stays
	// silent about policies has opted out, per the rule above.
	if proj == nil && orchestrator.Declared() {
		return PolicyRequest{
			Include:  cleanList(orchestrator.Include),
			Exclude:  cleanList(orchestrator.Exclude),
			Declared: true,
			Origin:   "config/policy.json",
		}
	}

	return PolicyRequest{Declared: false, Origin: "none"}
}

// ResolvedPolicyFiles turns a PolicyRequest's Include list into real,
// readable absolute paths, in the order given, with Exclude applied and
// duplicates dropped. Entries that resolve to nothing are skipped
// silently — a missing policy file has never been fatal in this engine
// (see loadCascade's doc comment).
func ResolvedPolicyFiles(root string, req PolicyRequest) []string {
	var out []string
	seen := map[string]bool{}
	for _, entry := range req.Include {
		if isExcluded(entry, req.Exclude) {
			continue
		}
		path, ok := resolvePolicyPath(root, entry)
		if !ok || seen[path] {
			continue
		}
		// Re-check after resolution: a bare name found recursively can
		// land on a file whose real base name is what Exclude names.
		if isExcluded(filepath.Base(path), req.Exclude) {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	return out
}

// resolvePolicyPath resolves ONE include entry to a real file:
//
//   - cross-platform absolute ("/etc/p.json", "C:\p.json",
//     "\\srv\share\p.json") → used as given, via the same
//     documents.IsAbsCrossPlatform/NormalizeAbsPath helpers write_file
//     and core.MemoryPath already use, so a Windows-style path is
//     recognized as absolute regardless of the OS Mova runs on (and
//     rejected with a clear error, rather than silently treated as a
//     relative name, when it can't apply to this OS).
//   - contains a separator → relative to the Mova root.
//   - bare file name → RECURSIVE search under config/policy/, first
//     match by base name wins (shallowest first, then alphabetical, so
//     the result never depends on filesystem walk order).
func resolvePolicyPath(root, entry string) (string, bool) {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return "", false
	}

	if documents.IsAbsCrossPlatform(entry) {
		normalized, err := documents.NormalizeAbsPath(entry)
		if err != nil {
			return "", false
		}
		if fileExists(normalized) {
			return normalized, true
		}
		return "", false
	}

	if strings.ContainsAny(entry, `/\`) {
		candidate := filepath.Join(root, filepath.FromSlash(entry))
		if fileExists(candidate) {
			return candidate, true
		}
		return "", false
	}

	return searchPolicyDir(filepath.Join(root, "config", "policy"), entry)
}

// searchPolicyDir walks config/policy/ recursively looking for a file
// whose base name equals name (case-insensitively). Shallower matches
// win over deeper ones; ties break alphabetically by full path — so
// two files with the same name in different sub-directories always
// resolve to the same one, run after run, on every OS.
func searchPolicyDir(dir, name string) (string, bool) {
	best, bestDepth := "", -1
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil //nolint:nilerr // an unreadable subtree is skipped, never fatal
		}
		if !strings.EqualFold(d.Name(), name) {
			return nil
		}
		depth := strings.Count(filepath.ToSlash(strings.TrimPrefix(path, dir)), "/")
		if bestDepth == -1 || depth < bestDepth || (depth == bestDepth && path < best) {
			best, bestDepth = path, depth
		}
		return nil
	})
	return best, best != ""
}

// isExcluded matches an Exclude entry against an include entry or a
// resolved path, comparing BASE FILE NAMES case-insensitively — so
// "pii_strict.json" in Exclude drops that file no matter which
// directory (default, custom, or absolute) it came from.
func isExcluded(entry string, exclude []string) bool {
	target := strings.ToLower(baseName(entry))
	for _, ex := range exclude {
		if strings.ToLower(baseName(ex)) == target {
			return true
		}
	}
	return false
}

// baseName is filepath.Base but separator-agnostic, so a Windows-style
// entry is handled correctly when Mova runs on Linux and vice versa.
func baseName(p string) string {
	p = strings.TrimSpace(p)
	if i := strings.LastIndexAny(p, `/\`); i >= 0 {
		return p[i+1:]
	}
	return p
}

// PolicyConfigPath is where config/policy.json lives — the orchestrator
// file that declares the lowest-precedence "policies" selection (see
// ResolvePolicyRequest). Same path sanitize.PolicyPath resolves; kept
// as its own tiny function here so core never has to import sanitize
// just to find this file.
func PolicyConfigPath(root string) string {
	return filepath.Join(root, "config", "policy.json")
}

// OrchestratorPolicySelector reads just the "policies" value out of
// config/policy.json, or nil when the file is missing/unreadable or
// declares no "policies" key. This is config/policy.json's own
// selection — the lowest-precedence layer in ResolvePolicyRequest.
func OrchestratorPolicySelector(root string) *PolicySelector {
	data, err := os.ReadFile(PolicyConfigPath(root))
	if err != nil {
		return nil
	}
	var probe struct {
		Policies *PolicySelector `json:"policies"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil
	}
	return probe.Policies
}

// fileExists reports whether path is an existing regular file (a
// directory named like a policy file is not a policy file).
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// PolicyDebugEntry is one resolved policy file — used only when
// project.json's "debug": true, so every door (CLI, chat, MCP, HTTP)
// can print the EXACT path each policy name resolved to, and why an
// excluded one was left out. See ResolvedPolicyDebug.
type PolicyDebugEntry struct {
	// Name is exactly what appeared in "include"/"exclude".
	Name string
	// Path is the resolved absolute path, or "" when Name didn't
	// resolve to any real file (still reported, so a typo is visible
	// instead of silently doing nothing).
	Path string
	// Included is false for anything left out — either because it was
	// in "exclude", or because it never resolved to a real file.
	Included bool
	// Reason is a stable code: "excluded", "excluded_not_found", or
	// "not_found" — "" for a normal include. Translated at render
	// time (see trace/console.go's renderPolicyDebug) so this package
	// stays free of i18n/presentation concerns.
	Reason string
}

// ResolvedPolicyDebug lists, in order, every "include" entry (resolved
// path + whether it was actually loaded or excluded) followed by any
// "exclude" entry that wasn't already covered by "include" — so a
// person reading debug output sees the exact file each name mapped
// to, not just the name they typed. Called only when proj.Debug is
// true (see trace/analyzer.go); resolving these paths is one extra,
// cheap filesystem walk, not something every run should pay for.
func ResolvedPolicyDebug(root string, req PolicyRequest) []PolicyDebugEntry {
	var out []PolicyDebugEntry
	seenExclude := map[string]bool{}

	for _, name := range req.Include {
		path, ok := resolvePolicyPath(root, name)
		entry := PolicyDebugEntry{Name: name, Path: path}
		switch {
		case isExcluded(name, req.Exclude):
			entry.Reason = "excluded"
			seenExclude[strings.ToLower(baseName(name))] = true
		case !ok:
			entry.Reason = "not_found"
		default:
			entry.Included = true
		}
		out = append(out, entry)
	}

	// Exclude entries that name a file not already listed in Include —
	// still resolved, so its path is visible too, not just its name.
	for _, name := range req.Exclude {
		if seenExclude[strings.ToLower(baseName(name))] {
			continue
		}
		path, ok := resolvePolicyPath(root, name)
		reason := "excluded"
		if !ok {
			reason = "excluded_not_found"
		}
		out = append(out, PolicyDebugEntry{Name: name, Path: path, Reason: reason})
	}
	return out
}

func cleanList(in []string) []string {
	var out []string
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// SplitPolicyList parses a --policies_include / --policies_exclude flag
// value: a comma-separated list that may mix bare names and full paths
// ("pii_strict.json,C:\custom\ventas.json"). Exported because the CLI
// (cli/trace_cmd.go) and the MCP/HTTP tool both parse the same shape.
func SplitPolicyList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return cleanList(strings.Split(raw, ","))
}
