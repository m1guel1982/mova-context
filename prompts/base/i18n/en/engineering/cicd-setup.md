# Configure CI/CD

Project: `{{PROJECT}}` · Stack: `{{STACK}}` · CI: `{{CI_PROVIDER}}` · Cloud: `{{CLOUD_PROVIDER}}`
Ockham: see `ockham-core.md`.

Output: pipeline lint → test → build → scan → deploy · secrets outside plaintext · health check + automatic rollback · dev/staging/prod environments · Dockerfile if applicable. Same artifact from staging must be promoted to production. Constraints: `{{CONSTRAINTS}}`.
