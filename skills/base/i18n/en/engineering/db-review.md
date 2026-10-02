# Objective

Review schema design in {{DATABASE}}
KISS+DRY: see `kiss-dry-core.md`.

# Skill: Database review
- N+1 and queries in loops: batch them (JOIN/IN/batch).
- Indexes only for real filter/join/sort columns; confirm with EXPLAIN before creating.
- No `SELECT *` on hot paths; paginate.
- Integrity in the DB (NOT NULL/UNIQUE/FK), tight types, reversible, backward-compatible migrations.
- Optimize only with evidence (execution plan or measurement).
