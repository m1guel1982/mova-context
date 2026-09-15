// egress_audit.go — resolves project.json's optional "egress_audit"
// block (see EgressAuditConfig in types.go) into a ready-to-use
// (dryRun bool, outputFile string) pair. The actual writing/dry-run
// logic lives in mova.local/models (Session.Send/SendStream) — this
// file only answers "what does this project's config say", the same
// split as ResolvePolicyAuthor/TargetModelFor in policy_author.go.
package core

import (
	"path/filepath"
	"strings"

	"mova.local/documents"
)

// DefaultEgressAuditFileName is used when "output_file" names a
// directory (no file name component) instead of a concrete file —
// e.g. "output_file": ".mova/" or ".mova" resolves to
// ".mova/egress_sanitized.md", not a file literally named ".mova".
const DefaultEgressAuditFileName = "egress_sanitized.md"

// ResolveEgressAudit resolves proj.EgressAudit for one project into
// (dryRun, outputFile). outputFile is "" when egress_audit is absent
// or its output_file is empty — audit logging is then fully disabled,
// independent of dryRun (a project can dry-run without logging, and
// vice versa).
//
// output_file resolution — ALWAYS relative to the directory that
// contains THIS project's project.json (projects/<project>/), never
// the process's working directory:
//   - absolute (Unix "/...", Windows "C:\...\..."/"D:\..."/"E:\...",
//     UNC "\\server\share\...") → used exactly as given, via the same
//     documents.IsAbsCrossPlatform/NormalizeAbsPath helpers
//     write_file/create_directory already use (see
//     documents/pathresolve.go) — recognizes every OS's absolute-path
//     style regardless of which OS Mova itself runs on, using only
//     os/path/filepath from the standard library under the hood.
//   - relative → filepath.Join(root, "projects", project, output_file).
//   - names only a directory (ends in a path separator, e.g.
//     ".mova/") → DefaultEgressAuditFileName is appended.
func ResolveEgressAudit(root, project string, proj *Project) (dryRun bool, outputFile string) {
	if proj == nil || proj.EgressAudit == nil {
		return false, ""
	}
	dryRun = proj.EgressAudit.DryRun

	raw := strings.TrimSpace(proj.EgressAudit.OutputFile)
	if raw == "" {
		return dryRun, ""
	}

	var resolved string
	if documents.IsAbsCrossPlatform(raw) {
		if normalized, err := documents.NormalizeAbsPath(raw); err == nil {
			resolved = normalized
		} else {
			resolved = raw
		}
	} else {
		resolved = filepath.Join(root, "projects", project, raw)
	}

	if isDirectoryLikeTarget(raw) {
		resolved = filepath.Join(resolved, DefaultEgressAuditFileName)
	}
	return dryRun, resolved
}

// isDirectoryLikeTarget reports whether the configured output_file
// explicitly names a directory rather than a concrete file — only a
// trailing path separator ("logs/", "logs\") counts. A bare name with
// no separator (".mova", "audit") is intentionally NOT treated as a
// directory: filepath.Ext can't reliably tell a dotfile-style name
// like ".mova" apart from an extension-less filename the person
// actually meant, so guessing there would be more surprising than
// useful. Document this rule in PROJECT_JSON.md § egress_audit.
func isDirectoryLikeTarget(raw string) bool {
	return strings.HasSuffix(raw, "/") || strings.HasSuffix(raw, "\\")
}
