# {{REVIEW_TYPE}} review — `{{PROJECT}}`
Query: {{QUERY}}
Ockham: see `ockham-core.md`.

Scope by type: **security** (OWASP, authn/authz, secrets, sensitive data, validation, CVE dependencies) · **architecture** (responsibilities, coupling, maintainability) · **quality** (complexity, duplication, errors, tests) · **performance** (N+1, queries/indexes, locks, CPU/memory/I/O) · **full** = all four in that order.

One line per finding: `[Severity] file:line — problem → impact → minimal fix`. If the fix is code, the full corrected function, keeping the existing architecture and style.
Deliver: findings by severity (Critical | High | Medium | Low), then Quick Wins.
