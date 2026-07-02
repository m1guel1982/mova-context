# Core + Extensions

> Documentation: [Español](../es/core-extensions.md) · [English](core-extensions.md)

---

## What it is

Mova Context exists in two editions sharing a single core:

```text
Core (Open Source, GitHub)          ← workflow.md, project.json, agents,
                                        skills, prompts, memory, MCP, CLI,
                                        adapters, i18n, Context Compiler

Extensions (commercial binaries)    ← install on top of the Core
                                        without modifying or replacing it
```

There are no two architectures. There's one Core that works on its own, and optional Extensions that extend it.

---

## How it works

The CLI (`mova`) is always the same binary for both editions. On startup:

```text
1. Load Core
2. Read project.json
3. Initialize workflow.md
4. Detect license (if any)
5. Detect installed modules
6. Register only the additional functionality
7. Continue normal execution
```

With no license and no modules installed, the CLI works exactly like the Open Source edition. It never errors on their absence — that's the default state of every Mova Context project.

The dependency direction is always:

```text
Extensions → Core     (correct)
Core → Extensions     (never)
```

The Core defines contracts (the `Adapter` interface in `cli/adapter.go`, the `contextCompiler` block in `project.json`, the Context Compiler's extension points). Extensions implement or extend them — never modify them.

---

## What stays Open Source

Everything needed to run a project end-to-end: `workflow.md`, `project.json`, base agents/skills/prompts, the file adapter, PostgreSQL/MongoDB adapters, the basic MCP server, the complete Context Compiler (Phase 1 and Phase 2), and all bilingual documentation. See [`OPEN_SOURCE_STRATEGY.md`](../../../OPEN_SOURCE_STRATEGY.md) for the full breakdown.

## What the commercial edition adds

Enterprise connectors (Salesforce, SAP, ServiceNow...), an admin UI, advanced sync, usage metrics, industry-specific pretrained assistants, and multi-tenant support. None of these replace a Core piece — they register as additional modules during CLI startup.

---

## Upgrading an Open Source install to the commercial edition

```text
1. Install the commercial modules (official binaries) + the license
2. Restart the CLI
3. The additional functionality becomes available automatically
```

Never reinstall the project. Never modify `workflow.md`, `project.json`, agents, skills, prompts, memory, or the i18n structure. The CLI executable doesn't change.

---

## Keeping future compatibility

- Any project created with the Core must open unchanged when Extensions are installed.
- Any project using only Core functionality must keep working if Extensions are uninstalled.
- `project.json` remains the single source of configuration; `workflow.md`, the single entry point.
- Every new Core feature (like the Context Compiler) is designed to work standalone, without depending on any Extension — that's the case here: `contextCompiler` is 100% Core.

---

## Validation

| Command | Expected result |
|---|---|
| `mova run [project]` with no Extension installed | Works exactly like the Open Source edition |
| `mova list` | Lists projects regardless of Extensions being installed |
| Remove installed Extensions, run any command again | CLI keeps working, no errors from their absence |
