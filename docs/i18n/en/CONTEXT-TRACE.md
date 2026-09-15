# context-trace — how the context decision is made

`context-trace` answers, before spending a single real token: how much it'll cost, what private data could
slip through, and which rule made each decision. It's the checkpoint between "I have a code folder" and
"this is about to go to an LLM".

## Every file passes through one of these 5 states

| State | Means |
|---|---|
| `DISCOVERED` | Found during the scan. |
| `CANDIDATE` | In scope (`focus`/relevance). |
| `ALLOWED` | Nothing suspicious — passes as-is. |
| `SANITIZED` | Something was found (API key, email...) and masked before counting as sendable. |
| `BLOCKED` | Something critical (private key) — the **whole** file is left out. |
| `EXCLUDED` | Left out for non-security reasons (binary, build output, not relevant to the task). |

Nothing is ever sanitized silently: every decision is recorded with the exact rule that made it, in
`context-report.md` and `pii-audit-log.json`.

## Two modes

```bash
mova context-trace <project>                # project mode: uses project.json (focus/exclude/task)
mova context-trace --repo <url>              # discovery mode: scans everything, no project.json
```

# Local scan specifying task and pruning docstrings
mova context-trace --repo C:\testMovaContext\fastapi --task "solve_dependencies get_dependant in fastapi/dependencies/utils.py" --prune-docstrings

# Remote repository scan with PDF export and exclusion patterns
mova context-trace --repo https://github.com/fastapi/fastapi --export pdf --task "fix dependency injection" --ignore "docs/**, tests/**, *.lock, .github/**"


Project mode never counts tokens twice — it reuses `mova budget`. Discovery mode counts file by file (for
the "tokens by directory" breakdown).

Always outputs: `context-report.md`/`.pdf` + `context-diagram.png` (with `--diagram`) + `pii-audit-log.json`.
