// Package logging implements Mova Context's opt-in logging system
// (spec section 3). Configuration lives at config/log/logging.json
// (English keys, see config/log/README.en.md / README.es.md for the
// full parameter reference) and is DISABLED by default — Load returns a
// Config with Enabled=false whenever the file is missing or unreadable,
// so a fresh checkout never writes logs until someone opts in.
//
// Every entrypoint (CLI, chat, HTTP, MCP) shares one *Logger, obtained
// via logging.Open(root) once at startup — see cli/main.go, http/server.go,
// mcp/server.go. Call sites simply do logger.Info("jobs", "...") and pay
// almost nothing when logging is disabled (Open still returns a valid,
// safe-to-call no-op-when-disabled Logger, so no caller needs a nil check).
package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config maps config/log/logging.json exactly — plain, English field
// names per the spec, so any developer can read it without translation.
type Config struct {
	Enabled    bool            `json:"enabled"`
	Structured bool            `json:"structured"`
	Level      string          `json:"level"` // "debug" | "info" | "warning" | "error"
	Categories map[string]bool `json:"categories"`
	File       FileConfig      `json:"file"`
	Rotation   RotationConfig  `json:"rotation"`
	Retention  RetentionConfig `json:"retention"`
}

// FileConfig controls where the log file lives and whether it's created
// automatically the first time something is logged.
type FileConfig struct {
	Path       string `json:"path"` // relative to the Mova root, or absolute
	AutoCreate bool   `json:"auto_create"`
}

// RotationConfig controls how often the active log file is rotated into
// a dated backup. Mode: "daily" | "weekly" | "monthly" | "yearly" |
// "custom" (uses CustomDays as the rotation interval, in days).
type RotationConfig struct {
	Mode       string `json:"mode"`
	CustomDays int    `json:"custom_days"`
}

// RetentionConfig controls how long rotated log files are kept before
// automatic deletion. Policy: "daily" (1 day) | "weekly" (7) |
// "monthly" (30) | "yearly" (365) | "custom" (uses CustomDays).
type RetentionConfig struct {
	Policy     string `json:"policy"`
	CustomDays int    `json:"custom_days"`
}

// ConfigPath returns config/log/logging.json under root.
func ConfigPath(root string) string {
	return filepath.Join(root, "config", "log", "logging.json")
}

// defaultConfig is what an absent/unreadable/invalid logging.json is
// treated as: logging fully disabled, everything else at sane defaults
// so enabling "enabled" later without touching anything else just works.
func defaultConfig() Config {
	return Config{
		Enabled:    false,
		Structured: false,
		Level:      "info",
		Categories: map[string]bool{},
		File:       FileConfig{Path: "logs/mova.log", AutoCreate: true},
		Rotation:   RotationConfig{Mode: "daily"},
		Retention:  RetentionConfig{Policy: "daily", CustomDays: 30},
	}
}

// LoadConfig reads config/log/logging.json under root. Any error
// (missing file, invalid JSON) silently yields the disabled default —
// a broken/absent logging config must never stop Mova from running.
func LoadConfig(root string) Config {
	cfg := defaultConfig()
	data, err := os.ReadFile(ConfigPath(root))
	if err != nil {
		return cfg
	}
	var parsed Config
	if err := json.Unmarshal(data, &parsed); err != nil {
		return cfg
	}
	if parsed.Level == "" {
		parsed.Level = "info"
	}
	if parsed.Categories == nil {
		parsed.Categories = map[string]bool{}
	}
	if parsed.File.Path == "" {
		parsed.File.Path = "logs/mova.log"
	}
	if parsed.Rotation.Mode == "" {
		parsed.Rotation.Mode = "daily"
	}
	if parsed.Retention.Policy == "" {
		parsed.Retention.Policy = "daily"
	}
	return parsed
}

// categoryEnabled reports whether category should be logged. An empty
// Categories map (the field omitted entirely in logging.json) means
// "all categories enabled" — an explicit, non-empty map means only the
// categories present with true are logged (spec: "habilitar una, varias
// o todas" / "deshabilitar todas"). Setting every entry to false is how
// "disable all categories" is expressed without turning off "enabled".
func (c Config) categoryEnabled(category string) bool {
	if len(c.Categories) == 0 {
		return true
	}
	v, ok := c.Categories[category]
	if !ok {
		return false
	}
	return v
}
