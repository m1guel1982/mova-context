# Objective

Alert messages that non-technical users can understand in under 30 seconds and know what to do.

KISS+DRY: see `kiss-dry-core.md`.

# Verification Checklist

* Must answer: what happened (specific event), where (branch/module/channel), magnitude compared to baseline, and what action to take
* Severity must be in the first line
* No raw statistical jargon (z-score, percentiles) in user-facing text — that belongs in dashboards, not notifications

# Anti-patterns

"Much above average of 0" (baseline at 0 is a bug, not an alert) · "requires immediate action" without specifying what action · identical message for different severities

# Template

```text
[SEVERITY] {problem_type} in {location}
{n} events in {time_window} vs. historical average of {baseline}.
Trend: {increasing|stable|decreasing}.
Suggested action: {action}.
```

# Output

One template per rule type, with a rendered example.
