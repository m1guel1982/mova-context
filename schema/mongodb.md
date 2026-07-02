# Mova Context — MongoDB Schema

Three collections. Mirrors the file structure.

---

## Collection: knowledge

Mirrors `agents/`, `skills/`, `prompts/` folders.

```javascript
{
  _id: ObjectId,
  kind: "agent" | "skill" | "prompt" | "workflow",
  domain: "software" | "legal" | "callcenter" | "",  // "" = universal
  lang: "es" | "en" | "fr" | "",                      // "" = universal/legacy
  name: "backend-dev",
  content: "# Rol\nBackend developer senior...",
  is_custom: false,
  active: true,
  updated_at: ISODate
}
```

**Index:**
```javascript
db.knowledge.createIndex({ kind: 1, domain: 1, lang: 1, name: 1 }, { unique: true })
db.knowledge.createIndex({ "$**": "text" })  // full-text search
```

---

## Collection: projects

Mirrors `projects/` folder.

```javascript
{
  _id: "my-project",           // same as "name"
  name: "my-project",
  description: "Project description",
  repo: ".",
  lang: "en",
  adapter: "file" | "postgresql" | "mongodb",
  dsn: "mongodb://...",        // only for db adapters
  llm: "claude" | "gpt" | "gemini" | "ollama" | "openrouter",
  default_task: "my-task",
  variables: { company: "Acme" },
  agents: { domain: "software", use: ["backend-dev"] },
  skills: { domain: "software", use: ["api-security"] },
  tasks: {
    "my-task": {
      prompt: "review-project",
      variables: { module: "auth" }
    }
  },
  active: true,
  updated_at: ISODate
}
```

---

## Collection: memory

Mirrors `projects/[name]/memory.md`.

```javascript
{
  _id: ObjectId,
  project: "my-project",      // references projects.name
  content: "## 2026-06-01 — session\n**Done:**...",
  session_date: ISODate("2026-06-01"),
  archived: false,
  created_at: ISODate
}
```

**Index:**
```javascript
db.memory.createIndex({ project: 1, created_at: -1 })
db.memory.createIndex({ project: 1, archived: 1 })
```

---

## Why only three collections

Knowledge → Projects → Memory.

This mirrors exactly the folder structure (`agents/`, `skills/`, `prompts/` → `projects/` → `memory.md`).

No join tables. No intermediate collections. KISS.
