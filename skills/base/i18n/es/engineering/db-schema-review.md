# Objetivo
Revisar diseño de schema en {{DATABASE}}
KISS+DRY: ver `kiss-dry-core.md`.

# Verificaciones
* Toda tabla con PK, created_at, updated_at
* FK con índice en tabla hija
* NOT NULL donde el negocio lo requiere
* Índice en columnas de búsqueda frecuente
* Migrations con rollback plan
* Sin datos sensibles sin cifrar (passwords, tokens, PII)

# Anti-patrones
Columna `data JSON` genérica sin justificación · FK sin índice · tabla pivote sin índice compuesto · migration sin rollback

# Output
Gaps, scripts de índices, plan de migración segura.
