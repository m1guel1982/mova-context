# Objective

Verify JWT security.
KISS+DRY: see `kiss-dry-core.md`.

# Verification Checklist

* RS256 over HS256 · never accept `alg: none` · server must enforce algorithm, not trust header value
* Mandatory claims: `exp`, `iss`, `aud`
* Payload must not contain passwords or sensitive data — minimum: `user_id`, `roles`, `tenant_id`
* Refresh token: mandatory rotation, stored in httpOnly cookie (never localStorage), must be revocable
* Transmission only over HTTPS, via `Authorization: Bearer` header, never in query parameters
