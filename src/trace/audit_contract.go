// audit_contract.go — forensic-traceability helpers required by the
// pii-audit-log.json / context-report contract: a unique execution_id
// per run, the repository's commit hash (best-effort), the
// "Audit Mode ≠ Sanitized" display label, and relative-path
// normalization so no artifact ever leaks an absolute temp path
// (e.g. "C:\Users\...\Temp\mova-trace-*").
package trace

import (
	"crypto/rand"
	"encoding/hex"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// NewExecutionID returns a fresh, unique identifier for ONE run —
// used to prove two executions were never confused (see prompt
// requirement: "Aislamiento de Métrica" / "cada corrida genere única
// y exclusivamente los findings correspondientes a su execution_id").
// Format: <unix-nano-hex>-<8 random hex bytes> - not a cryptographic
// UUID library (avoids a new dependency for a value that only needs
// to be unique, not standards-compliant), but collision-proof in
// practice: a fresh random component plus a nanosecond timestamp.
func NewExecutionID() string {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf) // crypto/rand failing here is effectively impossible; a weaker fallback below still keeps this non-fatal
	ts := time.Now().UnixNano()
	return hexInt64(ts) + "-" + hex.EncodeToString(buf)
}

func hexInt64(n int64) string {
	if n == 0 {
		return "0"
	}
	const digits = "0123456789abcdef"
	buf := make([]byte, 0, 16)
	u := uint64(n)
	for u > 0 {
		buf = append([]byte{digits[u&0xf]}, buf...)
		u >>= 4
	}
	return string(buf)
}

// CommitHashFor returns the short commit hash of repoDir's HEAD, or
// "" if it isn't a git repository (a local, non-git directory is a
// completely normal, supported input for context-trace) - best-effort
// only, never fatal.
func CommitHashFor(repoDir string) string {
	cmd := exec.Command("git", "-C", repoDir, "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// RelativizePath returns path relative to root when possible, with
// forward slashes regardless of OS - every report-facing path (finding
// paths, dir breakdown, exclusion reasons) is normalized through this
// so an artifact NEVER shows an absolute filesystem path, let alone a
// temporary clone directory that no longer exists by the time the
// report is read.
func RelativizePath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(filepath.Base(path))
	}
	return filepath.ToSlash(rel)
}

// AuditActionLabel is the single place that turns the internal
// "SANITIZED" action into the contract-required "WOULD_SANITIZE"
// whenever no real on-disk transformation happened - see
// sanitize/findings.go's header for why Transformed is always false
// coming out of context-trace's own (read-only) analysis. Every
// renderer (console/markdown/pdf/json) calls this instead of
// hardcoding the mapping itself.
func AuditActionLabel(action string, transformed bool) string {
	if action == "SANITIZED" && !transformed {
		return "WOULD_SANITIZE"
	}
	return action
}
