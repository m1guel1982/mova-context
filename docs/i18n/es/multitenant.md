# Multi-tenant — Extensión Opcional

> El core de Mova Context es single-tenant por diseño (KISS). Esta guía es para quien necesite soportar múltiples negocios/clientes sobre la misma base de datos.

---

## Cuándo lo necesitas

Solo si una misma instancia de PostgreSQL o MongoDB va a almacenar el conocimiento y los proyectos de **más de una empresa/cliente** que no deben verse entre sí.

Si cada cliente tiene su propia base de datos o su propio repo de archivos → **no necesitas esto**. Eso ya es multitenancy por separación física.

---

## El cambio mínimo

Agregar una sola columna a las tres tablas/colecciones existentes: `tenant_id`.

### PostgreSQL

```sql
ALTER TABLE knowledge ADD COLUMN tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE projects  ADD COLUMN tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE memory    ADD COLUMN tenant_id TEXT NOT NULL DEFAULT 'default';

-- Reemplazar los UNIQUE/índices existentes incluyendo tenant_id
ALTER TABLE knowledge DROP CONSTRAINT knowledge_kind_domain_lang_name_key;
ALTER TABLE knowledge ADD CONSTRAINT knowledge_tenant_unique
  UNIQUE (tenant_id, kind, domain, lang, name);

CREATE INDEX ON knowledge(tenant_id, kind, domain, lang) WHERE active;
CREATE INDEX ON memory(tenant_id, project, created_at DESC) WHERE NOT archived;

-- projects.name deja de ser único global → único por tenant
ALTER TABLE projects DROP CONSTRAINT projects_pkey;
ALTER TABLE projects ADD PRIMARY KEY (tenant_id, name);
```

### MongoDB

```javascript
// Agregar tenant_id a cada documento existente
db.knowledge.updateMany({}, { $set: { tenant_id: "default" } })
db.projects.updateMany({}, { $set: { tenant_id: "default" } })
db.memory.updateMany({}, { $set: { tenant_id: "default" } })

// Reemplazar índices únicos
db.knowledge.createIndex(
  { tenant_id: 1, kind: 1, domain: 1, lang: 1, name: 1 },
  { unique: true }
)
db.projects.createIndex({ tenant_id: 1, name: 1 }, { unique: true })
db.memory.createIndex({ tenant_id: 1, project: 1, created_at: -1 })
```

---

## El cambio en `project.json`

```json
{
  "project": "ley-21719",
  "tenant_id": "acme-corp",
  "adapter": "postgresql",
  "dsn": "postgres://..."
}
```

Si `tenant_id` no se declara, el adapter usa `"default"` — el comportamiento single-tenant no se rompe.

---

## El cambio en el adapter

Una sola regla, aplicada siempre que el adapter sea `postgresql` o `mongodb`:

```text
Toda lectura y escritura a knowledge / projects / memory
→ agregar WHERE tenant_id = [tenant_id del project.json]
```

El adapter de archivos (`file`) no necesita cambios: cada cliente ya tiene su propia carpeta `projects/[PROJECT]/` y su propio repo. La separación física ya es el aislamiento.

---

## Lo que NO cambia

* `workflow.md` — sigue igual, no sabe ni le importa si hay tenants
* `agents/`, `skills/`, `prompts/` — el conocimiento compartido entre tenants se queda en archivos (`adapter: file`), no en la base de datos
* La secuencia de ejecución del workflow — ningún paso nuevo

---

## Resumen

```text
1 columna nueva (tenant_id)
+ índices actualizados
+ 1 línea en project.json
+ 1 condición WHERE en el adapter
= soporte multitenant
```

Sin tablas nuevas. Sin lógica nueva en el workflow. Sin romper el modo single-tenant existente.
