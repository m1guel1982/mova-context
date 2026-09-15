# mova — Advanced

## Environment variables

| Variable | Use |
|---|---|
| `MOVA_PROJECT_ROOT` / `MOVA_PROJECT_PATH` | Forces the project root instead of searching upward for `workflow.md`. |
| `MOVA_ADAPTER` / `MOVA_DSN` | Switches the storage backend (`file` by default, or Postgres/MongoDB via DSN). |
| `MOVA_POLICY_AUTHOR` | Policy author for CI/CD when `project.json`/`config/policy.json` don't declare one — see Audit Matrix #3. |
| `MOVA_MAX_CONCURRENCY` / `MOVA_HTTP_MAX_CONCURRENCY` | Concurrent goroutine limit (CLI / HTTP server). |

## Installation

```bash
make install   # builds and copies the binary to $(go env GOPATH)/bin/mova
```

With that folder on `PATH`, `mova` runs from any directory.

## Multi-agent — agent groups

A `project.json` with `"is_group": true` and `"members": [...]` runs each member project as an agent, in
order. See `mova agents list` / `mova agents run <group>`.

## The governance policy cascade

`config/policy.json` lists files from `config/policy/` (`security.json`, `review.json`, `compliance.json`,
`pii.json`), loaded in order — adding/removing an entry never requires a Go recompile.

## `--diagram` — formats and destination

`--export svg,png,pdf` accepts one or more comma-separated formats. `--path` accepts a directory (auto-names
the file `<project>.<format>`) or, when a single format is requested, the full path of a file with that
extension (e.g. `--path ./diagrams/evidence.png`).
