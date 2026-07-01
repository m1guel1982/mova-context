# Objective

Security audit of API {{API_PREFIX}}. Auth: {{AUTH_METHOD}}
KISS+DRY: see `kiss-dry-core.md`.

# Verification Checklist

* Authentication required for all endpoints not explicitly public
* Authorization enforced at resource level, not only route level (anti-IDOR)
* Rate limiting per user and IP on public endpoints
* CORS configured with explicit origins, never `*` in production
* Security headers enabled:

  * Strict-Transport-Security
  * X-Content-Type-Options
  * X-Frame-Options
* Input validation: type, length, format
* Errors must not expose stack traces or internal paths

# Anti-patterns

Role-only authorization without ownership checks (IDOR) · different error messages for "user not found" vs "wrong password" · exposing dependency versions in headers

# Output

Findings per endpoint with severity and fix recommendation.
