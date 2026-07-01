# Multi-tenant — Optional Extension

> Mova Context's core is single-tenant by design (KISS). This guide is for anyone who needs to support multiple businesses/clients on the same database.

---

## When you need this

Only if a single PostgreSQL or MongoDB instance will store the knowledge and projects of **more than one company/client** that must not see each other's data.

If each client has their own database or their own file repo → **you don't need this**. That is already multitenancy through physical separation.

---

## The minimal change

Add a single column to the three existing tables/collections: `tenant_id`.

### PostgreSQL

```sql
ALTER TABLE knowledge ADD COLUMN tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE projects  ADD COLUMN tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE memory    ADD COLUMN tenant_id TEXT NOT NULL DEFAULT 'default';

-- Replace existing UNIQUE constraints/indexes to include tenant_id
ALTER TABLE knowledge DROP CONSTRAINT knowledge_kind_domain_lang_name_key;
ALTER TABLE knowledge ADD CONSTRAINT knowledge_tenant_unique
  UNIQUE (tenant_id, kind, domain, lang, name);

CREATE INDEX ON knowledge(tenant_id, kind, domain, lang) WHERE active;
CREATE INDEX ON memory(tenant_id, project, created_at DESC) WHERE NOT archived;

-- projects.name is no longer globally unique → unique per tenant
ALTER TABLE projects DROP CONSTRAINT projects_pkey;
ALTER TABLE projects ADD PRIMARY KEY (tenant_id, name);
```

### MongoDB

```javascript
// Add tenant_id to every existing document
db.knowledge.updateMany({}, { $set: { tenant_id: "default" } })
db.projects.updateMany({}, { $set: { tenant_id: "default" } })
db.memory.updateMany({}, { $set: { tenant_id: "default" } })

// Replace unique indexes
db.knowledge.createIndex(
  { tenant_id: 1, kind: 1, domain: 1, lang: 1, name: 1 },
  { unique: true }
)
db.projects.createIndex({ tenant_id: 1, name: 1 }, { unique: true })
db.memory.createIndex({ tenant_id: 1, project: 1, created_at: -1 })
```

---

## The change in `project.json`

```json
{
  "project": "privacy-law",
  "tenant_id": "acme-corp",
  "adapter": "postgresql",
  "dsn": "postgres://..."
}
```

If `tenant_id` is not declared, the adapter defaults to `"default"` — single-tenant behavior is not broken.

---

## The change in the adapter

One single rule, applied whenever the adapter is `postgresql` or `mongodb`:

```text
Every read and write to knowledge / projects / memory
→ add WHERE tenant_id = [tenant_id from project.json]
```

The file adapter (`file`) needs no changes: each client already has their own `projects/[PROJECT]/` folder and their own repo. Physical separation already provides isolation.

---

## What does NOT change

* `workflow.md` — stays the same, doesn't know or care if tenants exist
* `agents/`, `skills/`, `prompts/` — knowledge shared across tenants stays in files (`adapter: file`), not in the database
* The workflow execution sequence — no new steps

---

## Summary

```text
1 new column (tenant_id)
+ updated indexes
+ 1 line in project.json
+ 1 WHERE condition in the adapter
= multitenant support
```

No new tables. No new workflow logic. Single-tenant mode remains unbroken.
