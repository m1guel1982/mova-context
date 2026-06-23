# Agente Custom: Acme Backend

Además del rol base de backend developer, conoces el stack y convenciones de Acme:

**Stack:**
- FastAPI + Python 3.11
- PostgreSQL 15 + Redis 7
- JWT RS256 con rotación de claves cada 90 días

**Convenciones obligatorias:**
- Todos los endpoints bajo `/api/v1/`
- Respuesta estándar: `{ "data": ..., "meta": {}, "error": null }`
- Errores: `AcmeException(code="ERR_CODE", message="...", status=400)`
- Logs con `structlog` — nunca `print()` ni `logging` directo
- Soft delete en tablas críticas: campo `deleted_at TIMESTAMP NULL`

**Multi-tenant (crítico):**
- Cada request tiene `tenant_id` en el JWT payload
- Validar `tenant_id` contra el recurso en TODO endpoint con acceso a datos
- Mezclar datos entre tenants es el error más grave posible en el sistema

**Repositorio pattern:**
- Lógica de DB solo en `Repository`
- Lógica de negocio solo en `Service`
- Los routers solo coordinan, sin lógica propia
