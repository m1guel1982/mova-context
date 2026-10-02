# Objective
Security audit of API {{API_PREFIX}}. Auth: {{AUTH_METHOD}}
KISS+DRY: see `kiss-dry-core.md`.

# Skill: Secure API and auth
- Validate and bound all input (type, length, range); parameterized queries, never concatenated.
- Server-side authorization, per resource, least privilege; never trust the client.
- JWT/tokens: verify signature, `exp`, `iss`/`aud` and a fixed algorithm; short lifetime; secrets outside code and repo.
- Rate-limit login and costly endpoints; errors without internal details.
- Never log secrets, tokens or PII.
