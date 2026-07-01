# Objetivo
Auditar seguridad de API {{API_PREFIX}}. Auth: {{AUTH_METHOD}}
KISS+DRY: ver `kiss-dry-core.md`.

# Verificaciones
* Autenticación en todo endpoint no explícitamente público
* Autorización a nivel de recurso, no solo de ruta (anti-IDOR)
* Rate limiting por usuario e IP en endpoints públicos
* CORS con origins explícitos, nunca `*` en producción
* Headers: `Strict-Transport-Security`, `X-Content-Type-Options`, `X-Frame-Options`
* Inputs validados: tipo, longitud, formato
* Errores sin stack trace ni rutas internas

# Anti-patrones
Autorización solo por rol sin ownership (IDOR) · mensaje distinto para usuario inexistente vs password incorrecta · versión de dependencia en headers

# Output
Hallazgos por endpoint con severidad y fix.
