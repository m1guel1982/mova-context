# Objetivo
Auditar seguridad de API {{API_PREFIX}}. Auth: {{AUTH_METHOD}}
KISS+DRY: ver `kiss-dry-core.md`.

# Skill: API y auth seguras
- Validar y acotar toda entrada (tipo, largo, rango); consultas parametrizadas, nunca concatenadas.
- Autorización en el servidor, por recurso y con mínimo privilegio; nunca confiar en el cliente.
- JWT/tokens: verificar firma, `exp`, `iss`/`aud` y algoritmo fijo; vida corta; secretos fuera del código y del repo.
- Rate limit en login y endpoints costosos; errores sin detalles internos.
- No registrar secretos, tokens ni PII en logs.
