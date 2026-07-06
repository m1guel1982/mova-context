# Adapters — Mova Context

Adapters allow the same workflow to work with different storage systems.

Only one field changes in `project.json`. Everything else is identical.

---

## File adapter (default)

```json
"adapter": "file"
```

Knowledge lives in Markdown files inside the repository.

```text
agents/software/i18n/en/backend-dev.md
skills/legal/i18n/en/gdpr-obligations.md
prompts/callcenter/i18n/en/sales-response.md
```

Advantages:
- No extra configuration
- Git-versionable
- Editable with any editor
- No infrastructure dependencies

---

## PostgreSQL adapter

```json
"adapter": "postgresql",
"dsn": "postgres://user:password@host:5432/database"
```

Knowledge lives in a database. The structure mirrors the folder structure exactly.

### Schema

```sql
-- knowledge mirrors agents/, skills/, prompts/
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

-- projects mirrors projects/
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

-- memory mirrors projects/[name]/memory.md
CREATE TABLE memory (
    id           SERIAL PRIMARY KEY,
    project      TEXT NOT NULL REFERENCES projects(name),
    content      TEXT NOT NULL,
    session_date DATE NOT NULL DEFAULT CURRENT_DATE,
    archived     BOOLEAN NOT NULL DEFAULT false
);
```

Full schema at `schema/postgresql.sql` — it also includes `compile_reports`, prepared to persist Compiler v2 evidence (see [`context-compiler.md`](context-compiler.md)); today the `file` adapter already persists the equivalent as `contexto.report.json`, and `src/adapters/db_adapter.go` doesn't yet implement the matching method for the `db` adapter.

---

## MongoDB adapter

```json
"adapter": "mongodb",
"dsn": "mongodb://user:password@host:27017/database"
```

```javascript
// Collection: knowledge
{
  kind: "agent",        // "agent" | "skill" | "prompt" | "workflow"
  domain: "software",
  lang: "en",
  name: "backend-dev",
  content: "# Role\nSenior backend developer...",
  is_custom: false,
  active: true
}

// Collection: projects
{
  name: "my-project",
  lang: "en",
  adapter: "mongodb",
  tasks: { ... }
}

// Collection: memory
{
  project: "my-project",
  content: "## 2026-06-01 — session\n...",
  session_date: ISODate("2026-06-01"),
  archived: false
}
```

Full schema at `schema/mongodb.md` — it also includes `compileReports` (same purpose as `compile_reports` above).

---

## Comparison

| | Files | PostgreSQL | MongoDB |
|---|---|---|---|
| Configuration | none | DSN | DSN |
| Git-versionable | ✓ | ✗ | ✗ |
| Multi-tenant | ✗ | ✓ | ✓ |
| Full-text search | ✗ | ✓ | ✓ |
| No infrastructure | ✓ | ✗ | ✗ |
| Best for | individual projects | enterprises | enterprises |

---

## Portability demonstration

The same project with all three adapters:

```json
// With files
{ "project": "privacy-law", "adapter": "file" }

// With PostgreSQL
{ "project": "privacy-law", "adapter": "postgresql", "dsn": "postgres://..." }

// With MongoDB
{ "project": "privacy-law", "adapter": "mongodb", "dsn": "mongodb://..." }
```

workflow.md, agents, skills, and prompts are exactly the same.
Only the adapter changes.
