# Objetivo
Optimizar queries SQL en {{DATABASE}} para {{PROJECT}}
KISS+DRY: ver `kiss-dry-core.md`.

# Verificaciones
N+1 → JOIN/eager loading · `SELECT *` → columnas explícitas · filtro frecuente sin índice · sin paginación · query dentro de loop

# Formato por query problemática
```
Problema:
Query original: [SQL]
Query optimizada: [SQL]
Índice sugerido: (si aplica)
Mejora estimada:
```

# Output
Una entrada por query. EXPLAIN ANALYZE si el motor lo soporta.
