# Objetivo
Verificar seguridad JWT.
KISS+DRY: ver `kiss-dry-core.md`.

# Verificaciones
* RS256 sobre HS256 · nunca aceptar `alg: none` · servidor valida el algoritmo, no lo toma del header
* Claims obligatorios: `exp`, `iss`, `aud`
* Payload sin passwords ni datos sensibles — mínimo: user_id, roles, tenant_id
* Refresh token: rotación obligatoria, httpOnly cookie (nunca localStorage), revocable
* Transmisión solo HTTPS, header `Authorization: Bearer`, nunca en query params
