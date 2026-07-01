# Objective

Optimize SQL queries in {{DATABASE}} for {{PROJECT}}
KISS+DRY: see `kiss-dry-core.md`.

# Verification Checklist

N+1 → JOIN / eager loading · `SELECT *` → explicit columns · frequently used filter without index · missing pagination · query inside loop

# Format per problematic query

```txt id="sqlopt1"
Problem:
Original Query: [SQL]
Optimized Query: [SQL]
Suggested Index: (if applicable)
Estimated Improvement:
EXPLAIN ANALYZE:
```

# Output

One entry per query.
