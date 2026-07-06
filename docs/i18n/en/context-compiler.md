# Context Compiler — `mova compile`

> Docs: [English](context-compiler.md) · [Español](../es/context-compiler.md)

---

## What it does

Reduces the context sent to the LLM before sending it. It takes what `mova run` already loads (agents + skills + prompt + memory) and, optionally, exact fragments of code or project documents (`repo`), and produces a compact, non-human-formatted file: `contexto.txt` — along with an evidence report (`contexto.report.md` / `.json`, see below).

It doesn't replace `workflow.md` or `project.json`. It's an optional layer on top of them. A project without `contextCompiler` in its `project.json` works exactly as it always did — full backward compatibility.

**A note on editions:** the Compiler v2 (everything described on this page) is the commercial part of Mova Context. The Community (Open Source) edition can still run `mova compile`, but it falls back to the plain, undistilled context with an explicit warning — it never fails, never pretends to have compressed something it didn't. See [`../premium/en/editions.md`](../premium/en/editions.md).

---

## How it works

Two strategies, chosen with `contextCompiler.strategy`:

### `strategy: "semantic"` (default) — Phase 1 only

Applies to already-loaded Markdown files (agents, skills, prompts, memory):

- Removes greetings, thanks, and full courtesy phrases ("Hi, welcome...", "Thanks for your time...").
- Trims filler phrases ("it's worth noting that...", "it's important to mention that...") while leaving the substantive content that follows intact.
- Collapses repeated spaces and double blank lines.
- **Never** touches code blocks (` ``` `), `{{...}}` placeholders, or lines with critical-rule markers (`NEVER`, `ALWAYS`, `REQUIRED`, `FORBIDDEN`, `MUST`, `CRITICAL`...).

It's deterministic: it never calls an LLM to "summarize" — that would add latency and cost, the exact opposite of the goal.

### `strategy: "full"` — full Compiler v2 pipeline

Eight stages, each with a single responsibility, always in the same order:

```
Knowledge Parser → Semantic Cleanup → Normalization → Deduplication
  → Token Optimization → Priority Ranking → Budget Assembly → contexto.txt
```

| Stage | What it does | Never does |
|---|---|---|
| Knowledge Parser | Classifies content into units (Rule, Instruction, Memory, Example, Code, Table, Variable, Placeholder, Metadata) | Fragment code or tables |
| Semantic Cleanup | Phase 1 above, applied only to prose | Touch Code/Table/Metadata |
| Normalization | Bullets, smart quotes, repeated punctuation | Rewrite a sentence or change meaning |
| Deduplication | Removes **identical** repeated blocks | Merge blocks that are only "similar" |
| Token Optimization | Measures with a real BPE tokenizer (never characters/bytes) | Promise a fixed % upfront |
| Priority Ranking | Tags each block (Rule = critical, never trimmed) | Remove anything by itself |
| Budget Assembly | If `max_tokens` is set, assembles the best possible context within budget | Trim a critical rule, even if the budget is exceeded to keep them |

Full detail on every stage: [`compiler-v2-pipeline.md`](compiler-v2-pipeline.md).

### `focus` pruning (independent of strategy)

If `project.json` (or a `task`) defines `"focus"`, the compiler never sends whole `repo` files unrelated to the focus — for **any** strategy, including `"semantic"`. Each `focus` item is resolved against the **Focus Resolution Engine**, an extensible resolver engine (see [`focus-engine.md`](focus-engine.md) for its full architecture):

| Item type | Example | Extraction |
|---|---|---|
| File | `"file.js"` | Full content (explicitly requested) |
| Directory | `"src/services"` | Compact name listing, not the content |
| JSON node | `"config.json#database.host"` | Only that node, not the whole file |
| SQL definition | `"orders"` (table name) | Only `CREATE TABLE orders (...)`, not the whole `.sql` file |
| Code symbol | `"createOrder()"`, `"ClassName"` | Only the function/class/method, via brace or indent matching |
| Document section | `"Article 6"` | Only that section, up to the next same-level heading (Title/Chapter/Section/Article, or Markdown `#`/`##`) |
| Chronological event | keyword present in logs/history | Only the dated blocks that contain it |

If it can't find an exact structural match, the engine tries a cascade (code → document → legal → chronological) and, as a last resort, returns a bounded excerpt (±10 lines around the first textual mention) instead of the whole file — never empty, never the full file by default.

---

## When to use it

- Projects with large agents/skills/prompts repeated on every `mova run`.
- Tasks working on a specific file, function, table, or article within a large repo or document.
- `llm_profile.type: "local"` profiles, where every token counts more.
- `strategy: "full"` with `max_tokens`, when you need a hard budget guarantee (e.g. a small context window) and want reproducible evidence of what was trimmed and why.

Not necessary for small projects or when `focus` alone already reduces the payload enough without needing text distillation.

---

## Configuration (`project.json`)

```json
"contextCompiler": {
  "enabled": true,
  "mode": "manual",
  "strategy": "full",
  "max_tokens": 4000,
  "report_level": "full",
  "focus_exclude": ["tests", "coverage"]
},
"focus": ["file.js", "src/services", "functionName()"]
```

| Field | Values | Default | Effect |
|---|---|---|---|
| `enabled` | `true` / `false` | `false` | If the whole block is missing, the compiler never activates on its own. |
| `mode` | `"manual"` / `"automatic"` | `"manual"` | `manual`: only runs with `mova compile`. `automatic`: `mova run` and `get_full_context` (MCP) always use it. |
| `strategy` | `"semantic"` / `"full"` | `"semantic"` | `full` enables all 8 pipeline stages (see above). |
| `max_tokens` | integer | `0` (no limit) | Only with `strategy: "full"`. Budget **per knowledge block** (one agent, one skill, the prompt, or memory) — not yet a single budget for all of `contexto.txt`. Critical rules (`Rule`) are always included, even if they exceed the budget. |
| `report_level` | `"summary"` / `"full"` | `"summary"` | Only affects `contexto.report.md` — see [Transparency](#transparency-what-the-report-actually-shows) below. `contexto.report.json` always has every real number that exists, regardless of this setting. |
| `focus_exclude` | array of directory names | `[]` | Extra directories the Focus Resolution Engine's repo scan should skip, **added to** the fixed defaults (`.git`, `node_modules`, `vendor`, `dist`, `build`, `__pycache__`, `.venv`, `venv`, `.idea`, `.vscode`) — never replaces them. |

`focus` can be declared globally or inside a `task` — `task > global` priority, same as `variables` (workflow.md § FOCUS).

---

## Transparency: what the report actually shows

Every number in `contexto.report.json`/`.md` is measured, never estimated
— see [`compiler-roi-example.md`](../premium/en/compiler-roi-example.md)
for real runs. With `focus` configured, `report_level: "full"` adds a
second table with the Focus Resolution Engine's real scan evidence:

```text
| Files scanned in `repo`                      | 124 |
| Files included in contexto.txt               | 18  |
| Scanned but matched no `focus` target         | 104 |
| Excluded by directory `node_modules/`         | 40  |
| Excluded by directory `tests/`                | 12  |
```

Two honest rules behind this table, on purpose:

- **Only two exclusion reasons exist, because those are the only two the
  engine actually applies**: a directory it skipped, or a file it scanned
  that simply didn't match any requested `focus` target. There's no
  "duplicate files" or "unrelated docs" category — this compiler doesn't
  detect either of those, so it doesn't claim to.
- **`report_level` never hides data, only a view of it.** Leave it at
  `"summary"` (default) and `contexto.report.md` shows the same totals it
  always has; `contexto.report.json` shows this breakdown either way — a
  script that wants it doesn't need `"full"` at all, that setting is only
  for the human-readable file.

`contexto.report.md`'s labels follow the project's own `lang`
(`project.json`). If `lang` isn't `"es"` or `"en"` yet, the report falls
back to English and says so explicitly in the file — `contexto.report.json`
never depends on `lang` at all (its keys are English snake_case,
universal by design, e.g. `files_scanned`, not a translated phrase).

---

## Commands

```bash
mova compile [project] [task]     # forces the compiler regardless of "mode"
mova run     [project] [task]     # uses the compiler only if mode: "automatic"
```

`mova compile` always writes, with no extra steps:

```text
projects/[project]/contexto.txt
projects/[project]/contexto.report.md
projects/[project]/contexto.report.json
```

### Reports — evidence, not promises

Every compilation produces an objective report. A savings percentage is never claimed without being able to prove it: if the binary doesn't have Compiler v2 (Community edition), the report says exactly that instead of making up a number.

```json
{
  "edition": "premium",
  "project": "compiler-demo",
  "task": "revisar-ordenes",
  "tokenizer": "cl100k_base (aproximado)",
  "files_processed": 5,
  "tokens_before": 1814,
  "tokens_after": 1813,
  "blocks_total": 0,
  "blocks_removed": 0,
  "blocks_dropped": 0,
  "overflowed": false,
  "compile_time_ms": 3
}
```

`tokenizer` honestly flags when the encoding used is an approximation — Anthropic doesn't publish an offline Claude tokenizer, so `cl100k_base` (the closest public proxy) is used and documented as such instead of hidden.

---

## Editions and build profile

Compiler v2 is linked into the binary (or not) at build time (`-tags premium`), and its exact edition (`trial` / `premium` / `enterprise`) is set with `-ldflags`, never in a file editable at runtime. Full detail: [`building_the_executable.md`](building_the_executable.md) and [`../premium/en/editions.md`](../premium/en/editions.md).

If a Trial license expires or runs out of its execution quota, the binary **never breaks**: it automatically degrades to Community behavior (undistilled context) with an explicit console warning.

---

## Advantages

- Fewer tokens sent to the LLM → lower cost and latency, measured with a real tokenizer.
- Never sends whole code files when `focus` is defined — only the relevant fragment.
- 100% deterministic and local: no dependency on another LLM or the network (the tokenizer ships its vocabulary embedded in the binary).
- Zero impact on existing projects: without the `contextCompiler` block, nothing changes; with `strategy: "semantic"` (default), behavior is identical to before.
- With `strategy: "full"`, guaranteed per-block token budget, with reproducible evidence of what was trimmed.

## Limitations (documented, not hidden)

- Code symbol extraction is heuristic (brace/indent matching), not a real parser. Covers typical Go, JS/TS, Java, C#, PHP, and Python style well; very unusual symbols (macros, complex generics, minified code) may not resolve — in that case a bounded excerpt is returned, never empty or the full file.
- `max_tokens` budgets each knowledge block separately, not the whole `contexto.txt` — a global budget requires the orchestration layer to see the entire document at once (see roadmap).
- Deduplication operates at block/paragraph granularity, not line — two identical lines inside the same paragraph are not deduplicated against each other (avoids fragmenting a block for a marginal gain).
- The tokenizer used for non-OpenAI models (Claude, Gemini, Llama, Mistral, ...) is a public approximation (`cl100k_base`), not that provider's exact tokenizer — the report always states this.
- It doesn't interpret meaning — it's pattern-based text distillation, not real semantic understanding.

---

## Real use case

See [`projects/compiler-demo/`](../../../projects/compiler-demo/project.json), with `repo` pointing to a sample mini-project (`examples/compiler-demo-repo/`):

```json
{
  "project": "compiler-demo",
  "repo": "examples/compiler-demo-repo",
  "contextCompiler": { "enabled": true, "mode": "manual", "strategy": "semantic" },
  "tasks": {
    "revisar-ordenes": {
      "prompt": "review-project",
      "focus": ["CreateOrder()", "manual.md", "Artículo 6"]
    }
  }
}
```

```bash
mova compile compiler-demo revisar-ordenes
```

Result in `contexto.txt`: distilled agent/skill/prompt (no greetings or filler) + only the `CreateOrder` function, the full `manual.md` file (requested by name), and only "Artículo 6" from that manual — never the whole project. Alongside `contexto.txt`, `contexto.report.md` and `contexto.report.json` are generated with the compilation evidence.

---

## Best practices

- Start with `mode: "manual"` and review `contexto.txt` before switching to `automatic`.
- Use `focus` with exact names (`FunctionName()`, not descriptions) — resolution is literal.
- If a symbol isn't found, check the bounded excerpt that's still generated — it almost always means the name doesn't match exactly.
- Don't declare `focus` if you need the model to see the whole project — the absence of `focus` is the signal to work without restrictions (same as normal `mova run`).
- Before setting `max_tokens`, run once without a limit and check `contexto.report.json` to see how many tokens your project uses today — avoid arbitrary budgets.

---

## Validation

| Command | Expected result | Possible error | Fix |
|---|---|---|---|
| `mova compile [project]` | Creates `contexto.txt`, `contexto.report.md`, and `contexto.report.json`, confirms in console | `task not found` | Check `default_task` or pass the task explicitly |
| `mova run [project]` with `mode: "automatic"` | Output has the compact format (`PROJECT:... AGENT:...`), not the `## AGENTS` human headers | Still seeing the human format | `contextCompiler.enabled` is `false`, or `mode` isn't `"automatic"` |
| Focus on a nonexistent symbol | `FOCUS:` block with a bounded excerpt or `not found: <symbol>`, never empty | Block is empty | Shouldn't happen — report as a bug |
| Focus on a directory | `FOCUS:` block with `dir(N): file1, file2...` | File contents appear | Shouldn't happen — report as a bug |
| `mova compile` on Community edition | Generates all 3 files; report has `"edition": "community"` and a note explaining there's no distillation | Process fails | Shouldn't happen — the fallback is intentional, report as a bug |
