# PostgreSQL Example — Mova Context with database adapter

Mova Context can read agents, skills and prompts from PostgreSQL.
The workflow, projects and CLI logic don't change.

## The only change: project.json

```json
{
  "project": "my-project",
  "lang": "en",
  "adapter": "db",
  "dsn": "postgres://user:password@localhost:5432/mova_db?sslmode=disable"
}
```

## Setup

### 1. Create the database

```bash
createdb mova_db
psql mova_db < schema/postgresql.sql
```

The schema is at `mova-context/schema/postgresql.sql`.

### 2. Populate with existing files

```bash
# Example: insert one agent
psql mova_db -c "
INSERT INTO knowledge (kind, domain, lang, name, content)
VALUES ('agent', 'base', 'en', 'backend-dev',
  \$\$(content of agents/base/i18n/en/engineering/backend-dev.md)\$\$);
"
```

### 3. Run

```bash
# With DSN in project.json
mova run my-project my-task > context.txt

# Or with environment variables (override)
MOVA_ADAPTER=db MOVA_DSN="postgres://..." mova run my-project my-task > context.txt
```

## Reading and writing memory

```bash
# Memory is also stored in the DB when adapter=db
mova memory my-project "```memory
## 2026-01-21 — session
**Done:** module X implemented
```"
```

## Reference schema

See `mova-context/schema/postgresql.sql` for the full table definitions.

Main tables:
- `knowledge` — agents, skills, prompts
- `projects` — project configuration
- `memory` — session history

## Conclusion

Only `adapter` and `dsn` change in `project.json`.
The workflow, agents, skills and prompts are exactly the same.
