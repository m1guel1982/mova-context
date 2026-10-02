# Mova's paths — `config/general/config.json`, multiplatform, factory defaults

Mova resolves 6 of its own directories — `agents`, `skills`, `prompts`, `projects`
 — plus an optional `output_dir`, through a single file: `config/general/config.json`,
at Mova's root (the same directory `workflow.md` lives in).

## The file as shipped

```json
{
  "agents": "agents",
  "skills": "skills",
  "prompts": "prompts",
  "projects": "projects"
}
```

These are exactly the values Mova **already used before this file existed** — it ships populated on
purpose, not empty, so opening it immediately shows what each field controls and what its factory
behavior is. Editing any of these 6 values doesn't break anything: they're both a reference and a
starting point for customizing.

`output_dir` is **not** in the factory file — see the dedicated section below.

## Resolution rule — identical to `project.json`'s `"repo"`

Every one of these fields resolves with the **same cross-platform rule** `project.json`'s `repo`
field already uses (see `PROJECT_JSON.md`), not a new convention:

| Path shape | Example | Result |
|---|---|---|
| Windows drive letter | `C:\agents`, `D:/mova/skills`, `C://agents` | Absolute, used as-is — works on any OS where that drive exists |
| UNC network path | `\\server\share\mova` | Absolute, used as-is |
| Unix absolute | `/var/mova/prompts`, `/skills` | Absolute, used as-is — **on Linux/macOS, `/skills` literally means `/skills` at the filesystem root**, not relative to Mova |
| Bare relative (no leading `/` or drive letter) | `agents`, `opt/projects`, `./custom` | Relative to Mova's root |

If a declared absolute path doesn't apply to the current OS (e.g. a Windows drive letter read on
Linux), Mova **never breaks the process**: it automatically falls back to that field's default, as
if nothing had been configured.

## Fallback: blank, missing, or the file doesn't exist

For each of the 6 fields, if the value is `""`, the key isn't present in the JSON, or
`config/general/config.json` doesn't exist at all, Mova uses the exact same default it already had
(the same table above, in its bare form). The file never needs to exist for Mova to work.

## The folder's name can be anything

Mova doesn't assume the agents folder is literally named "agents" — it only cares about what
`config.json` declares. For example:

```json
{ "agents": "custom-agent-catalog" }
```

makes Mova look for agents under `<root>/custom-agent-catalog`, with the exact same recursive
domain/language discovery it already has today — nothing else changes.

## `output_dir` — the special case NOT shipped in the factory file

Unlike the other 6 fields, `output_dir` has only one real consumer today: `mova context trace`'s
`--output` argument (and therefore `context-report.md`, `context-report.pdf`, and
`context-diagram.png`). That's why it was deliberately left out of the factory `config.json`: that
command has no single "current behavior" (it uses the analyzed project's own directory in local mode,
or the directory the command was run from in remote mode) — if `output_dir` shipped with a real
default, that everyday behavior would change for everyone from the very first use.

If you ever want to centralize those reports into one folder, add the key by hand:

```json
{ "output_dir": "centralized-reports" }
```

Mova recognizes it immediately (same resolution rule as the table above) and gives it **priority**
over `mova context trace`'s own default — no code changes needed. If you'd rather keep the current
behavior, simply don't add this key (or leave it as `""`).

An explicit `--output <path>` on the command line always wins regardless of what `config.json` says.

## `config/log/logging.json` — the same rule for the log file

`config/log/logging.json`'s `file.path` field (defaulting to `"logs/mova.log"`) uses the exact same
rule: a bare path like the factory one resolves relative to Mova's root; a recognized
Windows/UNC/Unix absolute path is used as-is; and if the value is blank, missing, or doesn't apply
to the current OS, Mova falls back to logging at `<root>/logs/mova.log`.

## Per-project hierarchy — project.json's own "paths"

Any project can declare its own "paths" block in its project.json, with the same 6 fields
config/general/config.json has (agents, skills, prompts, output_dir — **never**
projects, see why below). Resolution priority is:

1. **The current project's own project.json** — if it declares the field and it isn't blank, it
   always wins.
2. **config/general/config.json** — if the project declares nothing for that field.
3. **Mova's historical default path** — if neither of the above declares anything.

All 3 layers use the exact same resolution rule (the table above — a recognized Windows/UNC/Unix
absolute path is used as-is, a bare path resolves relative to Mova's root). A full, working
end-to-end example lives in `projects/02-pii-compliance-governance/project.json`:

```json
{
  "paths": {
    "agents": "projects/02-pii-compliance-governance/local-agents",
    "skills": "projects/02-pii-compliance-governance/local-skills",
    "prompts": "projects/02-pii-compliance-governance/local-prompts"
  }
}
```

That project ships its own copies of `ai-privacy-reviewer` (agent), `pii-context-reduction` (skill),
and `analizar-contexto-clientes-ia` (prompt) in those 3 folders — same content as the global catalog,
plus a comment at the top of each file confirming the LOADED version was the project's own, not the
global one. Run `mova run 02-pii-compliance-governance` (or
`mova context-trace --project 02-pii-compliance-governance`) and look for that comment in the
assembled context's AGENTS/SKILLS/PROMPT section (or in `context-report.md`) to see it in action.

**Why `projects` has no per-project equivalent:** a project declaring, from inside itself, where
"all projects" (including itself) live is circular — Mova needs to know where to look for
`project.json` BEFORE it can read any `project.json` at all. That field stays exclusive to
config/general/config.json.

## Hot reload across all 3 layers

Neither config/general/config.json NOR any project.json is ever cached — both are re-read from disk
on every path resolution / every GetProject call. This is what lets you edit either one while Mova
is running (e.g. the long-lived MCP/HTTP server) and see the change reflected immediately, on the
very next request, with no restart needed — not for the active language, not for global paths, and
not for per-project paths either.
