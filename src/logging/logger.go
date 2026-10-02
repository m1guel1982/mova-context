package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"mova.local/documents"
)

// Logger is the one object every door (CLI, chat, HTTP, MCP) opens via
// Open(root) and shares — never re-reads logging.json per call, never
// opens a second file handle for the same log. Safe for concurrent use.
type Logger struct {
	mu       sync.Mutex
	cfg      Config
	minLevel Level
	path     string // absolute path to the active log file
	rotateAt time.Time
	file     *os.File
}

// Open loads config/log/logging.json under root and prepares a Logger.
// Always returns a non-nil, safe-to-call Logger — when logging is
// disabled (the default), every Debug/Info/Warning/Error call is a
// cheap no-op, so callers never need a nil check or an "if enabled"
// guard of their own.
//
// file.path resolution uses the SAME cross-platform rule project.json's
// "repo" field already has (documents.IsAbsCrossPlatform +
// NormalizeAbsPath) instead of the plain filepath.IsAbs this function
// used before: filepath.IsAbs is specific to whatever OS Mova happens
// to run on right now, so a logging.json shared across a mixed
// Windows/Linux/macOS team — or just moved from one machine to another
// — could silently misjoin an absolute path from a different OS style
// (e.g. a Windows drive letter read on Linux was neither treated as
// absolute nor rejected, just wrongly joined under root as a bogus
// nested path). The sample value ("logs/mova.log", a bare relative
// path with no leading marker) is unaffected either way — it already
// resolves under root exactly as documented.
func Open(root string) *Logger {
	cfg := LoadConfig(root)
	path := cfg.File.Path
	if documents.IsAbsCrossPlatform(path) {
		if normalized, err := documents.NormalizeAbsPath(path); err == nil {
			path = normalized
		} else {
			// Absolute-looking but for a different OS than the one
			// Mova runs on now (e.g. a Windows drive letter read on
			// Linux/macOS) — fall back to the DEFAULT relative path
			// (never to a raw join of the unusable value, which would
			// itself produce a broken "<root>/C:\..." path), same
			// "never break the process" contract every other path
			// fallback in this codebase already follows.
			path = filepath.Join(root, "logs", "mova.log")
		}
	} else {
		path = filepath.Join(root, filepath.FromSlash(path))
	}
	return &Logger{
		cfg:      cfg,
		minLevel: parseLevel(cfg.Level),
		path:     path,
		rotateAt: time.Now(),
	}
}

// Enabled reports whether this Logger will actually write anything —
// useful for callers that want to skip building an expensive log
// message entirely when logging is off.
func (l *Logger) Enabled() bool { return l != nil && l.cfg.Enabled }

func (l *Logger) Debug(category, format string, args ...any) {
	l.write(LevelDebug, category, format, args...)
}
func (l *Logger) Info(category, format string, args ...any) {
	l.write(LevelInfo, category, format, args...)
}
func (l *Logger) Warning(category, format string, args ...any) {
	l.write(LevelWarning, category, format, args...)
}
func (l *Logger) Error(category, format string, args ...any) {
	l.write(LevelError, category, format, args...)
}

// Close flushes and closes the underlying file handle, if open. Safe to
// call on a disabled Logger (no-op).
func (l *Logger) Close() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
}

func (l *Logger) write(level Level, category, format string, args ...any) {
	if l == nil || !l.cfg.Enabled {
		return
	}
	if level < l.minLevel || !l.cfg.categoryEnabled(category) {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if err := l.ensureFile(now); err != nil {
		return // logging must never crash the caller's real operation
	}

	line := l.formatLine(now, level, category, fmt.Sprintf(format, args...))
	_, _ = l.file.WriteString(line)
}

// ensureFile opens the log file (creating parent directories if
// AutoCreate is set), rotating first if the configured interval has
// elapsed, and runs a retention cleanup pass right after rotating.
func (l *Logger) ensureFile(now time.Time) error {
	if l.file != nil && shouldRotate(l.rotateAt, now, l.cfg.Rotation) {
		_ = l.file.Close()
		_ = os.Rename(l.path, rotatedName(l.path, l.rotateAt))
		l.file = nil
		cleanupOldLogs(l.path, l.cfg.Retention, now)
	}
	if l.file != nil {
		return nil
	}
	if l.cfg.File.AutoCreate {
		if err := os.MkdirAll(filepath.Dir(l.path), 0755); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	l.file = f
	l.rotateAt = now
	return nil
}

// formatLine renders one log entry, plain or structured (JSON) per
// cfg.Structured.
func (l *Logger) formatLine(t time.Time, level Level, category, message string) string {
	if l.cfg.Structured {
		entry := map[string]any{
			"time": t.Format(time.RFC3339), "level": level.String(),
			"category": category, "message": message,
		}
		data, _ := json.Marshal(entry)
		return string(data) + "\n"
	}
	return fmt.Sprintf("%s [%s] [%s] %s\n", t.Format("2006-01-02 15:04:05"), level.String(), category, message)
}
