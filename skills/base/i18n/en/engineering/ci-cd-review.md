# Objective

Review CI/CD pipeline. Provider: {{CI_PROVIDER}}
KISS+DRY: see `kiss-dry-core.md`.

# Checks

* Pipeline must be reproducible: same commit = same artifact
* Full test suite must run before merge
* Secrets must be stored in a secret manager, never in plain text
* Artifacts must be immutable; do not rebuild at deployment time
* Post-deploy health checks with automatic rollback
* Progressive environments: dev → staging → prod

# Anti-patterns

Skipping tests in "urgent" branches · secrets exposed in logs · environment-specific builds · no post-deploy validation

# Output

Identified gaps, refactored CI/CD pipeline with correct stages.
