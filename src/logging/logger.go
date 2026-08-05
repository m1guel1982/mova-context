package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
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
func Open(root string) *Logger {
	cfg := LoadConfig(root)
	path := cfg.File.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
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
