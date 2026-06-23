# Skill Custom: Convenciones Acme

Al revisar o generar código para Acme respetar siempre:

**Naming:**
- Python: `snake_case` en todo (funciones, variables, columnas DB)
- TypeScript: `camelCase` variables/funciones, `PascalCase` componentes/clases
- Tablas DB: plural snake_case (`payment_transactions`, `tenant_profiles`)
- FK: `{tabla_singular}_id` (`user_id`, `tenant_id`)

**Base de datos:**
- Nunca `DELETE` físico en producción en tablas de negocio
- Soft delete con `deleted_at TIMESTAMP NULL`
- Índices obligatorios en: todas las FK, columnas de búsqueda frecuente, `created_at`

**Errores:**
- Usar `AcmeException`, nunca `HTTPException` directamente
- Códigos internos con prefijo: `ERR_AUTH_*`, `ERR_TENANT_*`, `ERR_NOT_FOUND`
- Stack traces nunca al cliente, solo a los logs internos

**Feature flags:**
- Tabla `feature_flags` en DB con cache en Redis (TTL: 60s)
- Nunca hardcodear comportamiento que pueda cambiar sin deploy
