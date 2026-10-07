# What Mova generates

[README](README.md) · [Español](../es/ARTIFACTS.md)

## Per-execution evidence — `projects/<project>/runs/<run_id>/`

One directory per attempt to release context (`mova run`, `mova chat`, `get_full_context`, `chat_completion`, `run_agent`), plus one session run per MCP/HTTP process for reads and hooks.

| File | Write mode | Content |
|---|---|---|
| `context.txt` | once | The exact bytes Mova released, or would have released under `dry_run`. Empty when a gate blocked |
| `manifest.json` | once | `door`, `project`, `task`, `repo` (commit and dirty state, or a note when there is no git), `config` (sha256 of `project.json` and of each policy), `spec`, `selection`, `dependency_closure`, `governance`, `context` (sha256, bytes, estimated tokens, tokenizer), `agent` / `model` / `policy_author` with their source (`declared`, `observed`, `not_observable`), `decision`, `perimeter` |
| `events.jsonl` | append-only | `provider_call` (real tokens reported by the provider), `dry_run_block`, `tool_result`, `read`, `hook_check_read`, `hook_sanitize_output` |

Answers: **which context did Mova produce, under which rules, from which code, for which task?**

It does not record what the agent did outside Mova. The context carries no timestamps, so two runs with the same inputs share `context.sha256`. `runs/` is git-ignored.

## Other files

- `memory.md`: each entry carries `source=observed run=<id>` (Mova called the model), `source=host` (a host recorded it via `save_memory`; Mova did not see the model) or `source=user`.
- `mova-budget-report.md`, graph `*.png`, `context-report.*`: reports to read. They are **not** egress evidence.
