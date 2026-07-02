# OPEN_SOURCE_STRATEGY.md

> Documento privado. Estrategia de open source y monetización de Mova Context.

---

## Principio

> Lo que es abierto atrae. Lo que es valioso se paga.

El proyecto open source es el mejor argumento de venta para las versiones comerciales.
Un proyecto open source mediocre no atrae a nadie.

---

## Qué publicar gratis (siempre)

### Core del sistema
- `workflow.md` — el corazón del proyecto
- `project.json` — la fuente de verdad y su esquema
- Convención i18n completa
- Estructura de agents/skills/prompts

### Agentes y skills base
- Todos los agents base (yagni-core, backend-dev, architect, etc.)
- Todas las skills base (kiss-dry-core, ockham-core, etc.)
- Los dos ejemplos oficiales completos

### CLI
- Binarios precompilados para Linux, macOS, Windows
- Código fuente de la CLI (Go)

### MCP
- Servidor MCP básico
- Integración con Claude Desktop

### Adaptadores
- Adapter filesystem (default)
- Adapter PostgreSQL
- Adapter MongoDB

### Documentación
- Documentación completa en español e inglés
- Guía de adopción
- FAQ

### Esquemas
- Esquema PostgreSQL completo
- Esquema MongoDB completo

---

## Qué reservar para versiones comerciales

### Conectores empresariales
- Salesforce
- SAP
- Oracle
- Dynamics 365
- ServiceNow
- Zendesk
- HubSpot
- Jira Enterprise
- Confluence
- SharePoint

### Sincronización avanzada
- Sync bidireccional con bases de datos existentes
- Import/export masivo de conocimiento
- Versionado automático con rollback
- Diff de contexto entre versiones

### UI de administración
- Panel web para gestionar proyectos sin CLI
- Editor visual de agents/skills/prompts
- Vista de historial de memory.md
- Dashboard de uso por proyecto y modelo

### Monitoreo y métricas
- Dashboard de tokens consumidos por proyecto
- Métricas de calidad de respuestas del LLM
- Alertas de anomalías en el contexto
- Trazabilidad completa de decisiones del LLM

### Asistentes especializados preentrenados
- Asistente de cumplimiento legal (por país)
- Asistente de análisis de contratos
- Asistente de cobranza
- Asistente de RRHH
- Asistente de soporte técnico
- Asistente de ventas por industria

### Multi-tenant Enterprise
- Gestión de múltiples organizaciones desde una sola instancia
- Roles y permisos por proyecto
- SSO (SAML, OIDC)
- Auditoría de acceso

### Herramientas de calidad
- Evaluador automático de calidad de agents y skills
- Benchmark de prompts contra múltiples modelos
- Detector de inconsistencias en el conocimiento
- Sugerencias automáticas de mejora

---

## Modelo de monetización

### Open Source (gratuito)
- Todo lo del apartado anterior
- Atrae desarrolladores, arquitectos, equipos técnicos
- Genera comunidad y contribuciones

### Mova Context Pro (individual / equipo pequeño)
**USD 29/mes por usuario**
- UI de administración web
- Sincronización con GitHub/GitLab
- Métricas básicas de uso
- Soporte por email

### Mova Context Business (equipos medianos)
**USD 199/mes hasta 10 usuarios**
- Todo lo de Pro
- Conectores empresariales básicos (Jira, Confluence, Notion)
- Dashboard de métricas avanzadas
- Soporte prioritario

### Mova Context Enterprise (organizaciones grandes)
**Precio por negociación**
- Todo lo de Business
- Todos los conectores empresariales
- Multi-tenant
- SSO
- Auditoría completa
- SLA garantizado
- Asistentes especializados por industria
- Implementación y onboarding incluidos

---

## Servicios profesionales

### Implementación
- Migración del conocimiento existente a Mova Context
- Diseño de la arquitectura de agents/skills/prompts
- Integración con sistemas existentes

Precio referencial: USD 5.000 – 50.000 por proyecto

### Asistentes especializados a medida
- Diseño y desarrollo de un asistente específico para la industria del cliente
- Agentes, skills y prompts para el caso de uso

Precio referencial: USD 10.000 – 80.000 por asistente

### Capacitación
- Workshops para equipos de desarrollo
- Capacitación para equipos jurídicos/compliance
- Certificación de arquitectos Mova Context

Precio referencial: USD 2.000 – 8.000 por taller

### Soporte de cumplimiento legal
- Mantenimiento del conocimiento legal cuando cambia la ley
- Suscripción de actualización: USD 500 – 2.000/mes por jurisdicción

---

## Estrategia de atracción

### Lo que atrae a desarrolladores individuales
- CLI simple y bien documentada
- Ejemplos reales y concretos
- Sin dependencias complicadas
- Funciona con cualquier LLM

### Lo que atrae a equipos técnicos
- Versionamiento del contexto con Git
- Portabilidad entre modelos y proveedores
- Reducción del tiempo de onboarding de nuevos desarrolladores
- Trazabilidad de decisiones

### Lo que atrae a empresas
- Los ejemplos de industria (legal, callcenter, etc.)
- El argumento de "el backend nunca cambia"
- El cumplimiento legal en todos los canales simultáneamente
- La independencia de proveedor de IA

### Lo que convierte empresas en clientes pagos
- El dolor de actualizar el conocimiento en N sistemas cuando algo cambia
- La necesidad de auditoría y trazabilidad
- La necesidad de UI sin CLI
- El soporte y SLA garantizado

---

## Funcionalidades que convierten sin presionar

Estas funcionalidades hacen que las empresas quieran pagar sin que nadie las presione:

1. **Actualización de la ley en un solo lugar** → El argumento de la Ley 21.719. Cuando la ley cambia, solo cambia Mova Context. Sin esto, cada sistema se actualiza por separado.

2. **Trazabilidad** → Qué versión del conocimiento tomó qué decisión, cuándo y para qué cliente. Crítico para auditorías de cumplimiento.

3. **Métricas** → Cuántos tokens consume cada proyecto. Qué tan bien responde el LLM con el contexto actual. Detectar cuando el conocimiento está desactualizado.

4. **UI de administración** → El equipo jurídico puede actualizar el conocimiento sin saber Git.

5. **Conectores** → Sincronizar el conocimiento desde Jira, Confluence o SharePoint sin trabajo manual.

---

## Lo que NO debe hacerse

- No cobrar por características que son fundamentales para el open source
- No crear un "free tier" artificial con límites absurdos
- No abandonar el proyecto open source para priorizar el comercial
- No romper la compatibilidad entre versiones
- No añadir complejidad al core para beneficiar solo a la versión comercial

---

## Métricas de éxito del open source

- Stars en GitHub: > 1.000 en 6 meses
- Contribuciones externas: > 10 contribuidores en 12 meses
- Forks: > 200 en 12 meses
- Menciones en Reddit/HN: > 5 en 6 meses
- Empresas que usan el proyecto: > 20 en 12 meses

Estas métricas son el argumento más fuerte para inversión y para ventas Enterprise.
