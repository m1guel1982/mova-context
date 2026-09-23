# Controls that stop the process — `debug`, `policies`, `on_exceed`, `dry_run`

Four `project.json` fields. All four work **the same in CLI, `mova chat`, MCP, and HTTP** — one implementation, four doors.

## `debug: true` — see the exact path of every policy

```json
"debug": true
```

This makes `mova context-trace` add one line per included/excluded policy, with its **real absolute path**:

```
[debug] included: security.json -> /real/path/config/policy/security.json
[debug] excluded: pii_permissive.json -> /real/path/config/policy/pii_permissive.json (excluded by config)
```

Without `debug`, those lines don't appear — everything else in the report is identical. `debug` never changes a decision, only explains it.

## `policies` — which rules load

```json
"policies": { "include": ["security.json", "pii_strict.json"], "exclude": ["pii_permissive.json"] }
```

See `PROJECT_JSON.md § policies` for path resolution and the full precedence order (CLI > `project.json` > `config/policy.json`).

## `budget.on_exceed` — what happens when you go over budget

| Value | Effect |
|---|---|
| `"warn"` (default) | Warns in the console, **continues**, still calls the LLM. |
| `"abort"` / `"block"` | **Stops before calling the LLM.** 0 tokens leave. Same stop on all 4 doors. |

## `egress_audit.dry_run` — test everything without calling the LLM

```json
"egress_audit": { "dry_run": true, "output_file": "egress_sanitized.md" }
```

With `dry_run: true`: context gets assembled, sanitized, audited — and **the provider is never called, on ANY tool/command that exposes context** (`chat_completion`, `get_full_context`, `get_memory`, `get_memory_all`, `get_workflow`, `read_file`, `read_document_layer`, and `mova run` alike — see `PROJECT_JSON.md § Real air-gap`). Returns a success reply saying it was a dry run, plus an **explicit anti-bypass directive** aimed at the model/agent receiving it (see the "Honest limits" note below — that directive is a hardened instruction, not a technical guarantee). Use this to test the whole governance pipeline **without a running model** — see `MCP_HTTP_TESTING.md` to test this way, free, no GPU needed.

The directive and the audit block's header are 100% customizable and multi-language — they live in
`config/lang/{es,en}.json` (`reports.egress_airgap_message` and `reports.egress_airgap_directive`)
and hot-reload with no restart: edit the file, the change applies within ~1 second. Deleting the
`egress_airgap_directive` key (or blanking it out) leaves the block working exactly the same — only
the extra directive disappears, the process never breaks and the raw key name never leaks into the
message.

### Honest limits: `dry_run` is what Mova controls, not what the host does afterward

Mova guarantees its own side of the contract: while `dry_run: true` is active, the real context
**never leaves Mova's own process** — not to the provider, not in the tool result — and every attempt
is logged to disk. What Mova **cannot guarantee** is what the **host model/agent** (Cursor, Claude
Code, Grok, etc.) does with the block it receives: a determined host can still try reading other local
files (`context-report.md`, `project.json`, memory, etc.) to "reconstruct" the context on its own
instead of stopping — this already happened in real testing. The anti-bypass directive lowers the
odds of that (it explicitly tells the model not to) but cannot technically prevent it: Mova has no
control over the host's own process once it decides to call other tools on its own. Treat `dry_run` as
"Mova will never hand you the real context", not as "it's impossible for the host to get it any other
way".

## No `llm_profile` — Mova never calls a model on its own

If `project.json` declares no `llm_profile`, Mova won't attempt to reach any local/cloud provider. Over MCP/HTTP, `chat_completion` returns the already-governed context so the **host LLM** (Cursor, Claude Code, etc.) can answer. In `mova chat`, an explicit notice states which global model is used instead.

## Verified on all 4 doors (not just documented)

| Door | `debug` | `on_exceed: block` | `dry_run: true` |
|---|---|---|---|
| CLI (`mova context-trace` / `mova chat` / `mova run`) | ✅ | ✅ stops | ✅ stops |
| MCP (stdio) | ✅ | ✅ stops | ✅ stops |
| HTTP (`POST /mcp`) | ✅ | ✅ stops | ✅ stops |

Real example, run against `examples/02-pii-compliance-governance` (ships with all 4 fields on):

```bash
mova context-trace 02-pii-compliance-governance   # shows [debug] with real paths
```
