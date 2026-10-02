# Objetivo
Revisar diseño de schema en {{DATABASE}}
KISS+DRY: ver `kiss-dry-core.md`.

# Skill: Revisión de base de datos
- N+1 y consultas en bucles: agrupar (JOIN/IN/batch).
- Índices solo para columnas reales de filtro/join/orden; confirmar con EXPLAIN antes de crear.
- Sin `SELECT *` en rutas calientes; paginar.
- Integridad en la BD (NOT NULL/UNIQUE/FK), tipos ajustados, migraciones reversibles y compatibles hacia atrás.
- Optimizar solo con evidencia (plan de ejecución o medición).
