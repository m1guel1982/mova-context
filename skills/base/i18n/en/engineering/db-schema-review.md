# Objective

Review schema design in {{DATABASE}}
KISS+DRY: see `kiss-dry-core.md`.

# Verification Checklist

* Every table must include PK, created_at, updated_at
* Foreign keys must have indexes on the child table
* NOT NULL where required by business rules
* Indexes on frequently searched columns
* Migrations must include rollback plan
* No unencrypted sensitive data (passwords, tokens, PII)

# Anti-patterns

Generic `data JSON` column without justification · FK without index · pivot table without composite index · migration without rollback

# Output

Gaps, index scripts, and safe migration plan.
