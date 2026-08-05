// rotate.go — rotation ("when does the active log file roll over into a
// dated backup") and retention ("how long do rotated files survive
// before automatic deletion") policy math. Kept pure/testable: no
// file I/O here — logger.go calls these and does the actual
// rename/remove.
package logging

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// rotationIntervalDays converts a RotationConfig into a day count.
func rotationIntervalDays(r RotationConfig) int {
	switch strings.ToLower(r.Mode) {
	case "weekly":
		return 7
	case "monthly":
		return 30
	case "yearly":
		return 365
	case "custom":
		if r.CustomDays > 0 {
			return r.CustomDays
		}
		return 1
	default: // "daily"
		return 1
	}
}

// retentionDays converts a RetentionConfig into a day count — entries
// older than this are deleted by cleanupOldLogs.
func retentionDays(r RetentionConfig) int {
	switch strings.ToLower(r.Policy) {
	case "weekly":
		return 7
	case "monthly":
		return 30
	case "yearly":
		return 365
	case "custom":
		if r.CustomDays > 0 {
			return r.CustomDays
		}
		return 30
	default: // "daily"
		return 1
	}
}

// shouldRotate reports whether the active log file (created/rotated at
// lastRotation) is due to roll over, given now and the configured
// rotation interval.
func shouldRotate(lastRotation, now time.Time, cfg RotationConfig) bool {
	days := rotationIntervalDays(cfg)
	return now.Sub(lastRotation) >= time.Duration(days)*24*time.Hour
}

// rotatedName returns the dated backup name for path, rotated at t —
// e.g. "logs/mova.log" → "logs/mova-2026-07-30.log".
func rotatedName(path string, t time.Time) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	return base + "-" + t.Format("2006-01-02") + ext
}

// cleanupOldLogs removes rotated backups (siblings of path matching its
// "<base>-YYYY-MM-DD<ext>" pattern) older than the configured retention
// policy. Errors are ignored — a failed cleanup pass must never crash
// whatever operation triggered logging.
func cleanupOldLogs(path string, cfg RetentionConfig, now time.Time) {
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	base := filepath.Base(strings.TrimSuffix(path, ext))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := now.AddDate(0, 0, -retentionDays(cfg))

	var candidates []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, base+"-") || !strings.HasSuffix(name, ext) {
			continue
		}
		candidates = append(candidates, name)
	}
	sort.Strings(candidates)

	for _, name := range candidates {
		dateStr := strings.TrimSuffix(strings.TrimPrefix(name, base+"-"), ext)
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil || t.Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}
