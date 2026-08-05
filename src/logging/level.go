package logging

import "strings"

// Level is one of the four levels the spec requires: debug, info,
// warning, error — in increasing severity order.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarning
	LevelError
)

// parseLevel converts logging.json's "level" string to a Level,
// defaulting to LevelInfo for anything unrecognized (never fails and
// never blocks startup over a typo in config).
func parseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug
	case "warning", "warn":
		return LevelWarning
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelWarning:
		return "warning"
	case LevelError:
		return "error"
	default:
		return "info"
	}
}
