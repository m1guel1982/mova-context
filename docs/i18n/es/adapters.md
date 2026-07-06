# Adaptadores — Mova Context

Los adaptadores permiten que el mismo workflow funcione con diferentes sistemas de almacenamiento.

Solo cambia un campo en `project.json`. El resto permanece idéntico.

---

## Adaptador de Archivos (default)

```json
"adapter": "file"
```

El conocimiento vive en archivos Markdown dentro del repositorio.

```text
agents/software/i18n/es/backend-dev.md
skills/legal/i18n/es/ley-21719-obligaciones.md
prompts/callcenter/i18n/es/sales-response.md
```

Ventajas:
- Sin configuración adicional
- Versionable con Git
- Editable con cualquier editor
- Sin dependencias de infraestructura

---

## Adaptador PostgreSQL

```json
"adapter": "postgresql",
"dsn": "postgres://usuario:contraseña@host:5432/base_de_datos"
```

El conocimiento vive en una base de datos. La estructura espeja exactamente la estructura de carpetas.

### Esquema

```sql
-- knowledge espeja agents/, skills/, prompts/
CREATE TABLE knowledge (
    id        SERIAL PRIMARY KEY,
    kind      TEXT NOT NULL CHECK (kind IN ('agent','skill','prompt','workflow')),
    domain    TEXT NOT NULL DEFAULT '',
    lang      TEXT NOT NULL DEFAULT '',
    name      TEXT NOT NULL,
    content   TEXT NOT NULL DEFAULT '',
    is_custom BOOLEAN NOT NULL DEFAULT false,
    active    BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (kind, domain, lang, name)
);

-- projects espeja projects/
CREATE TABLE projects (
    name         TEXT PRIMARY KEY,
    description  TEXT,
    repo         TEXT,
    lang         TEXT NOT NULL DEFAULT '',
    adapter      TEXT NOT NULL DEFAULT 'file',
    dsn          TEXT,
    llm          TEXT,
    default_task TEXT,
    variables    JSONB NOT NULL DEFAULT '{}',
    agents       JSONB NOT NULL DEFAULT '{}',
    skills       JSONB NOT NULL DEFAULT '{}',
    tasks        JSONB NOT NULL DEFAULT '{}'
);

-- memory espeja projects/[name]/memory.md
CREATE TABLE memory (
    id           SERIAL PRIMARY KEY,
    project      TEXT NOT NULL REFERENCES projects(name),
    content      TEXT NOT NULL,
    session_date DATE NOT NULL DEFAULT CURRENT_DATE,
    archived     BOOLEAN NOT NULL DEFAULT false
);
```

El esquema completo está en `schema/postgresql.sql` — incluye además `compile_reports`, preparada para persistir la evidencia del Compiler v2 (ver [`context-compiler.md`](context-compiler.md)); hoy el adapter `file` ya persiste el equivalente como `contexto.report.json`, y `src/adapters/db_adapter.go` todavía no implementa el método correspondiente para el adapter `db`.

---

## Adaptador MongoDB

```json
"adapter": "mongodb",
"dsn": "mongodb://usuario:contraseña@host:27017/base_de_datos"
```

```javascript
// Colección: knowledge
{
  kind: "agent",        // "agent" | "skill" | "prompt" | "workflow"
  domain: "software",
  lang: "es",
  name: "backend-dev",
  content: "# Rol\nBackend developer...",
  is_custom: false,
  active: true
}

// Colección: projects
{
  name: "mi-proyecto",
  lang: "es",
  adapter: "mongodb",
  tasks: { ... }
}

// Colección: memory
{
  project: "mi-proyecto",
  content: "## 2026-06-01 — sesión\n...",
  session_date: ISODate("2026-06-01"),
  archived: false
}
```

El esquema completo está en `schema/mongodb.md` — incluye además `compileReports` (mismo propósito que `compile_reports` arriba).

---

## Comparación

| | Archivos | PostgreSQL | MongoDB |
|---|---|---|---|
| Configuración | ninguna | DSN | DSN |
| Versionable con Git | ✓ | ✗ | ✗ |
| Multi-tenant | ✗ | ✓ | ✓ |
| Búsqueda full-text | ✗ | ✓ | ✓ |
| Sin infraestructura | ✓ | ✗ | ✗ |
| Ideal para | proyectos individuales | empresas | empresas |

---

## Demostración de portabilidad

El mismo proyecto de Ley 21.719 con los tres adaptadores:

```json
// Proyecto con archivos
{ "project": "ley-21719", "adapter": "file" }

// El mismo proyecto con PostgreSQL
{ "project": "ley-21719", "adapter": "postgresql", "dsn": "postgres://..." }

// El mismo proyecto con MongoDB
{ "project": "ley-21719", "adapter": "mongodb", "dsn": "mongodb://..." }
```

El workflow.md, los agents, las skills y los prompts son exactamente los mismos.
Solo cambia el adaptador.
