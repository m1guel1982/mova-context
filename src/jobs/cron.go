// cron.go — minimal 5-field cron matcher ("min hour dom month dow"), the
// same syntax section 3 of the spec documents ("0 2 * * *"). No external
// dependency: the grammar this engine actually needs (*, exact numbers,
// comma lists, ranges "a-b", steps "*/n") is small and self-contained,
// and pulling in a third-party cron library would be the only new
// runtime dependency in the whole binary for one parser. Extending the
// grammar later (e.g. "@daily") only means adding a case to
// expandField/normalizeAlias — the rest of the engine never changes.
package jobs

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CronSpec is a parsed "schedule" field, one set of allowed values per
// column. An empty schedule ("" ) never matches — jobs without a
// schedule must be triggered explicitly (see RunJob).
type CronSpec struct {
	Minutes map[int]bool
	Hours   map[int]bool
	Doms    map[int]bool // day of month, 1-31
	Months  map[int]bool // 1-12
	Dows    map[int]bool // 0-6, 0=Sunday
}

var fieldBounds = [5][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 6}}

// ParseSchedule parses a 5-field cron expression like "0 2 * * *".
func ParseSchedule(expr string) (*CronSpec, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return nil, fmt.Errorf("invalid schedule %q: expected 5 fields (min hour dom month dow), got %d", expr, len(fields))
	}
	sets := make([]map[int]bool, 5)
	for i, f := range fields {
		set, err := expandField(f, fieldBounds[i][0], fieldBounds[i][1])
		if err != nil {
			return nil, fmt.Errorf("invalid schedule %q: field %d (%q): %w", expr, i+1, f, err)
		}
		sets[i] = set
	}
	return &CronSpec{
		Minutes: sets[0], Hours: sets[1], Doms: sets[2], Months: sets[3], Dows: sets[4],
	}, nil
}

// Matches reports whether t falls on this schedule, at minute precision.
// Dom/Dow follow standard cron OR semantics when both are restricted
// (i.e. neither is "*"): the run fires when EITHER matches.
func (c *CronSpec) Matches(t time.Time) bool {
	if !c.Minutes[t.Minute()] || !c.Hours[t.Hour()] || !c.Months[int(t.Month())] {
		return false
	}
	domAll := len(c.Doms) == 31
	dowAll := len(c.Dows) == 7
	domOK := c.Doms[t.Day()]
	dowOK := c.Dows[int(t.Weekday())]
	switch {
	case domAll && dowAll:
		return true
	case domAll:
		return dowOK
	case dowAll:
		return domOK
	default:
		return domOK || dowOK
	}
}

func expandField(f string, min, max int) (map[int]bool, error) {
	set := map[int]bool{}
	for _, part := range strings.Split(f, ",") {
		if err := expandPart(part, min, max, set); err != nil {
			return nil, err
		}
	}
	return set, nil
}

func expandPart(part string, min, max int, set map[int]bool) error {
	step := 1
	base := part
	if idx := strings.Index(part, "/"); idx != -1 {
		base = part[:idx]
		s, err := strconv.Atoi(part[idx+1:])
		if err != nil || s <= 0 {
			return fmt.Errorf("invalid step %q", part[idx+1:])
		}
		step = s
	}

	lo, hi := min, max
	switch {
	case base == "*":
		// lo/hi already the full range
	case strings.Contains(base, "-"):
		bounds := strings.SplitN(base, "-", 2)
		a, err1 := strconv.Atoi(bounds[0])
		b, err2 := strconv.Atoi(bounds[1])
		if err1 != nil || err2 != nil || a > b {
			return fmt.Errorf("invalid range %q", base)
		}
		lo, hi = a, b
	default:
		v, err := strconv.Atoi(base)
		if err != nil {
			return fmt.Errorf("invalid value %q", base)
		}
		lo, hi = v, v
	}
	if lo < min || hi > max {
		return fmt.Errorf("value out of range [%d-%d]: %q", min, max, base)
	}
	for v := lo; v <= hi; v += step {
		set[v] = true
	}
	return nil
}
