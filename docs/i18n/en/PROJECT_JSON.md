# PROJECT.JSON — field reference

Every project lives in `projects/<name>/project.json`. This is the single source of configuration for that project — model prices are the only thing that lives elsewhere (`config/prices.json`; see [COMMANDS.md §15](COMMANDS.md#15-tokenomics--mova-budget)).

## Minimal example

```json
{
  "project": "my-project",
  "repo": "examples/my-project-repo",
  "lang": "en",
  "adapter": "file",
  "default_task": "review",

  "agents": { "domain": "base", "use": ["backend-dev"], "custom": [] },
  "skills": { "domain": "base", "use": ["lazy-minimalism"], "custom": [] },

  "tasks": {
    "review": { "prompt": "review-project" }
  },

  "llm_profile": { "provider": "ollama", "config": "llama3.2.3b" }
}
```

## All fields

| Field | Type | What it's for |
|---|---|---|
| `project` | string | the project's name — matches the folder under `projects/` |
| `description` | string | free text, shown in `mova projects` |
| `repo` | string | the project's single repository. One repository per project, one path — see **More than one directory?** below if you're tempted to add a second |
| `lang` | string | `"es"`, `"en"`, ... — which prompt/agent/skill language variant to load |
| `adapter` | string | `"file"` (default) or `"db"` |
| `dsn` | string | connection string, only when `adapter: "db"` |
| `llm` | string | legacy: `"claude"` \| `"gpt"` \| `"ollama"` — still works |
| `llm_profile` | object | `{ "provider", "config" }` — the modern way to pick a model (see [COMMANDS.md §6](COMMANDS.md#6-models-and-providers)) |
| `default_task` | string | task used when none is given on the command line |
| `variables` | object | `{name: value}` injected into prompts/agents/skills |
| `agents` / `skills` | object | `{ "domain", "use": [...], "custom": [...] }` |
| `tasks` | object | named tasks — each may override `prompt`/`agents`/`skills`/`variables`/`focus`/`budget` |
| `archive` | object | memory management config (see [COMMANDS.md §4](COMMANDS.md#4-memory)) |
| `focus` | array | files/dirs/symbols to work on instead of the whole repo — see **Working on part of `repo`** below |
| `budget` | object | `{ "max_tokens": N }` — the context ceiling (see [COMMANDS.md §15](COMMANDS.md#15-tokenomics--mova-budget)) |
| `workflow_path` | string | where `workflow.md` lives for this project — see **Workflow** below |
| `budget_path` | string | where `mova-budget-report.md` is written — see **Budget report** below |
| `token_history_path` | string | where `mova-token-history.json` is written — see **Token history** below |
| `tools` | object | `{ "enabled": true/false }` — lets chat call file/document tools mid-conversation |

`repo`, `workflow_path`, `budget_path`, and `token_history_path` are all single strings — on purpose. One value per field keeps a project.json easy to read and easy to reason about; there is no array form for any of them.

## Working on part of `repo`, instead of a second repository

If part of your work only touches one folder inside `repo` — one service in a monorepo, one module, one directory — that's what `focus` is for, not a second `repo` entry (there isn't one):

```json
{
  "project": "my-monorepo-project",
  "repo": "examples/my-monorepo",
  "tasks": {
    "review-api": { "prompt": "review-api", "focus": ["services/api/"] },
    "review-web": { "prompt": "review-ui", "focus": ["services/web/"] }
  }
}
```

Each task scopes itself to its own directory with `focus` — see [COMMANDS.md's Focus section](COMMANDS.md#3-focus--scoping-context-to-part-of-the-repo) for the full glob/directory syntax (`"**/*"`, `"."`, `"src/"`, ...). This keeps `project.json` simple (one `repo`, one path) while still letting each task work on just its own slice of the repository.

If your work genuinely spans two UNRELATED repositories — different owners, different release cycles, never touched by the same task — the simpler and more honest setup is two separate projects (`projects/service-a/project.json`, `projects/service-b/project.json`), each with its own `repo`, `workflow_path`, `budget_path`, and `token_history_path`. See **More than one project** below.

## Workflow example

```json
{
  "project": "my-project",
  "repo": "examples/my-project-repo",
  "workflow_path": "workflow.md"
}
```

Saying "lee workflow.md", "ejecuta workflow.md", "workflow.md my-project", etc. resolves this project, validates its Budget, and only then loads the file — see [COMMANDS.md §9](COMMANDS.md#workflowmd--budget-gated-execution). Without `workflow_path`, Mova looks for a plain `workflow.md` at the Mova root, same as before this field existed.

## Budget example

```json
{
  "budget": { "max_tokens": 8000 },
  "budget_path": "mova-budget-report.md"
}
```

Without `budget_path`, the report is written to `projects/<project>/mova-budget-report.md`.

## Token history example

```json
{
  "token_history_path": "mova-token-history.json"
}
```

Without it, the file is written to `projects/<project>/mova-token-history.json`.

## Tasks example

```json
{
  "tasks": {
    "review": { "prompt": "review-project" },
    "audit": {
      "prompt": "audit-consent-flow",
      "agents": ["security-architect"],
      "variables": { "module": "checkout" },
      "focus": ["checkout.html"],
      "budget": { "max_tokens": 4000 }
    }
  }
}
```

## Agents / Skills / Prompts example

```json
{
  "agents": { "domain": "base", "use": ["backend-dev", "qa-engineer"], "custom": [] },
  "skills": { "domain": "base", "use": ["lazy-minimalism"], "custom": ["my-custom-skill"] }
}
```

Prompts are referenced by name from each task's `"prompt"` field (see the **Tasks** example above) — there's no separate top-level `prompts` field; a prompt is chosen per task.

## More than one project — agents/skills/prompts shared, nothing duplicated

There is no such thing as one `project.json` holding several projects — each project is its own folder with its own file:

```text
projects/
├── project-a/
│   └── project.json
└── project-b/
    └── project.json
```

**`projects/project-a/project.json`**
```json
{
  "project": "project-a",
  "repo": "examples/project-a-repo",
  "lang": "en",
  "agents": { "domain": "base", "use": ["backend-dev"], "custom": [] },
  "skills": { "domain": "base", "use": ["lazy-minimalism"], "custom": [] },
  "tasks": { "review": { "prompt": "review-project" } },
  "budget_path": "mova-budget-report.md",
  "token_history_path": "mova-token-history.json"
}
```

**`projects/project-b/project.json`**
```json
{
  "project": "project-b",
  "repo": "examples/project-b-repo",
  "lang": "en",
  "agents": { "domain": "base", "use": ["security-architect"], "custom": [] },
  "skills": { "domain": "base", "use": ["lazy-minimalism"], "custom": ["my-custom-skill"] },
  "tasks": { "audit": { "prompt": "audit-consent-flow" } },
  "budget_path": "mova-budget-report.md",
  "token_history_path": "mova-token-history.json"
}
```

Agents/skills/prompts are global resources (they live once, under `agents/`, `skills/`, `prompts/` at the Mova root); each project only *references* them by name in `use`/`prompt`. If `project-a` and `project-b` both use `"lazy-minimalism"`, it's the exact same skill file for both — nothing is copied per project. Each project keeps its own `budget_path`/`token_history_path`/`workflow_path`, so their Budget reports and token history never mix.
