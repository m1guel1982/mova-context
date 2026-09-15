# 03 — Tokens, Budget, and Context Trace

**What it is:** `focus`, `exclude` (with AST), `task`, and budget/circuit breaker working together

on a log with repeated lines and a file containing a legacy function that must be excluded

**without** splitting the file in two.

## Run (1 command)

```bash
mova run 03-tokenomics-context-trace --diagram --export png --path ./evidencia.png
```

## What you will see

| Step             | Result                                                                                                                        |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| Context selected | `checkout.js` + `server.log` (`focus`)                                                                                        |
| AST exclude      | `exclude: ["checkout.js::func=procesarPagoLegacy"]` — the file is analyzed in full; only the body of that function is removed |
| Sanitizer        | 48 identical lines from `server.log` collapsed into `[×48 identical repetitions omitted]`                                     |
| Budget           | `max_tokens_per_run: 5000` + `max_monthly_usd: 5.0`, `on_exceed: warn` (circuit breaker)                                      |
| Evidence         | `evidencia.png` (per-file reduction pipeline) + `context-report.md`                                                           |

`evidencia-ejemplo.png` in this folder is a previously generated sample. To see the AST `exclude` in action directly in the assembled context:

```bash
mova run 03-tokenomics-context-trace | grep -A3 procesarPagoLegacy
```

## `mova trace` Scenario Against a Remote Repository (Same Engine, One Command)

```bash
mova trace --repo https://github.com/<user>/<repo> --task "review module X" --ignore "docs/**,*.lock"
```

```text
Remote repository

        ↓

task + --ignore   (AST and relevance are automatically applied when ranking files)

        ↓

Context Trace

        ↓

CANDIDATE / ALLOWED / WOULD SANITIZE / BLOCKED / EXCLUDED

        ↓

tokens + cost

        ↓

context-report.md + context-diagram.png + pii-audit-log.json
```

**Honest note:** against a remote repository (without `project.json`), the CLI currently only accepts `--task` and `--ignore` — not explicit `--focus`/`--exclude` options (relevance ranking and AST are applied automatically). The complete `focus + exclude (AST) + task` combination shown in the table above is exclusive to `project.json` mode, which is what this example already demonstrates working.

`mova trace` is a short alias for `mova context-trace` (same command, same engine).

`project/project.json` in this folder is a read-only copy — the actual file lives at `projects/03-tokenomics-context-trace/project.json`.
