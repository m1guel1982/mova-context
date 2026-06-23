# Acme Demo — Memoria del proyecto

Archivo de seguimiento. Cada sesión importante se registra aquí.
El LLM puede leer este archivo para entender el contexto acumulado antes de continuar trabajo.

---

## Cómo usar este archivo

Al iniciar una sesión con el LLM, pegar este archivo junto con el project.json y decir:
"Lee el project.json y memory.md, y continúa desde donde quedamos."

---

## 2024-01-15 — Auditoría inicial del módulo auth

**Qué se hizo:** Revisión de seguridad del módulo de autenticación.

**Hallazgos resueltos:**
- [x] Token sin expiración en refresh tokens → corregido, TTL 7 días
- [x] tenant_id no validado en endpoint GET /users → corregido

**Hallazgos pendientes:**
- [ ] Refresh token almacenado en localStorage en el portal → migrar a httpOnly cookie
- [ ] No hay rate limiting en POST /auth/login → agregar antes del siguiente release

**Decisiones tomadas:**
- Se decidió RS256 sobre HS256 por el modelo multi-tenant
- Rotación de claves cada 90 días, tarea agendada en el equipo

---

## 2024-01-22 — Módulo de pagos

**Qué se hizo:** Revisión de performance y queries.

**Hallazgos resueltos:**
- [x] N+1 en GET /payments → resuelta con eager loading del tenant

**Hallazgos pendientes:**
- [ ] Falta índice en payment_transactions(tenant_id, created_at) → crear en próxima migración
- [ ] process_payment no tiene idempotency key → diseñar antes de ir a producción

**Errores encontrados en sesión:**
- El LLM generó código usando HTTPException directamente → recordar en próxima sesión que Acme usa AcmeException

---

## 2024-02-01 — Pendientes globales

- [ ] Documentar el flujo de feature flags con Redis
- [ ] Crear módulo de auditoría de acciones por tenant
- [ ] Revisar permisos en Acme-api-perfiles (todavía no auditado)

---

## Contexto acumulado importante

- El sistema es multi-tenant estricto: un leak entre tenants es catástrofe
- Soft delete en: users, tenants, payments, subscriptions
- No usar DELETE físico en producción en ninguna de esas tablas
- AcmeException es el único mecanismo de error permitido hacia el cliente
