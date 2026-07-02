# Context Compiler — `mova compile`

> Documentation: [Español](../es/context-compiler.md) · [English](context-compiler.md)

---

## What it does

Reduces the context sent to the LLM before it's sent. It takes what `mova run` already loads (agents + skills + prompt + memory) and, optionally, exact fragments of the project's code or documents (`repo`), producing a compact, machine-oriented file: `contexto.txt`.

It does not replace `workflow.md` or `project.json`. It's an optional layer on top of them. A project without `contextCompiler` in its `project.json` behaves exactly as before — full backward compatibility.

---

## How it works

Two independent phases, each toggled separately:

### Phase 1 — Semantic Telegram (`strategy: "semantic"`)

Applies to the Markdown files already loaded (agents, skills, prompts, memory):

- Removes full greeting/courtesy sentences ("Hi there, welcome...", "Thanks for your time...").
- Trims filler lead-ins ("it's worth mentioning that...", "please note that...") while keeping the substantive content that follows.
- Collapses repeated spaces and double blank lines.
- **Never** touches code fences (` ``` `), `{{...}}` placeholders, or lines carrying a critical-rule marker (`NEVER`, `ALWAYS`, `MUST`, `REQUIRED`, `CRITICAL`, `NUNCA`, `SIEMPRE`...).

It's deterministic — it never calls another LLM to "summarize", which would add latency and cost, defeating the whole point.

### Phase 2 — Surgical pruning (`focus`)

If `project.json` (or a `task`) defines `"focus"`, the compiler never sends whole `repo` files unrelated to the focus. Each focus item resolves as:

| Item type | Example | Extraction |
|---|---|---|
| File | `"file.js"` | Full content (it was explicitly requested) |
| Directory | `"src/services"` | Compact name listing, not the contents |
| Code symbol | `"createOrder()"`, `"ClassName"` | Just the function/class/method, via brace-counting or indentation |
| Document node | `"Article 6"` | Just that section, up to the next same-level heading (Title/Chapter/Section/Article/Clause or `#`/`##`) |
| Chronological event | keyword present in logs/histories | Just the dated blocks containing it |

If no exact structural match is found, it returns a bounded excerpt (±10 lines) instead of the whole file, and says so.

---

## When to use it

- Projects with large agents/skills/prompts repeated on every `mova run`.
- Tasks that work on one specific file, function, or article inside a large repo or document.
- `llm_profile.type: "local"` profiles, where every token counts more.

Not needed for small projects, or when `focus` alone already narrows things down enough without needing Phase 1.

---

## Configuration (`project.json`)

```json
"contextCompiler": {
  "enabled": true,
  "mode": "manual",
  "strategy": "semantic"
},
"focus": ["file.js", "src/services", "functionName()"]
```

| Field | Values | Default | Effect |
|---|---|---|---|
| `enabled` | `true` / `false` | `false` | With the block absent, the compiler never activates on its own. |
| `mode` | `"manual"` / `"automatic"` | `"manual"` | `manual`: only runs via `mova compile`. `automatic`: `mova run` and MCP's `get_full_context` always use it. |
| `strategy` | `"semantic"` / `"full"` | `"semantic"` | `full`: no Phase 1 (only applies Phase 2 if `focus` is set). |

`focus` can be declared globally or inside a `task` — priority `task > global`, same as `variables` (workflow.md § FOCUS).

---

## Commands

```bash
mova compile [project] [task]     # forces the compiler regardless of "mode"
mova run     [project] [task]     # uses the compiler only if mode: "automatic"
```

`mova compile` always writes `projects/[project]/contexto.txt`. It's the inspection/debugging path mentioned in workflow.md — works whether or not `mode: "automatic"` is set.

---

## Advantages

- Fewer tokens sent to the LLM → lower cost and latency.
- Never sends whole code files when `focus` is defined — only the relevant fragment.
- 100% deterministic and local: no dependency on another LLM or the network.
- Zero impact on existing projects: without the `contextCompiler` block, nothing changes.

## Limitations

- Code-symbol extraction is heuristic (brace-counting / indentation), not a real parser. It covers typical Go, JS/TS, Java, C#, PHP and Python style well; unusual symbols (macros, complex generics, minified code) may not resolve exactly — in that case a bounded excerpt is returned, never an empty result or the whole file.
- Phase 1's token reduction is modest (instructions are usually already concise). The large reduction comes from Phase 2, by avoiding sending whole files or documents when only a part is needed.
- It doesn't interpret meaning — it's pattern-based text distillation, not real semantic understanding.

---

## Real use case

See [`projects/compiler-demo/`](../../../projects/compiler-demo/project.json), with `repo` pointing at a small example project:

```json
{
  "project": "compiler-demo",
  "repo": "/tmp/demo-repo",
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

Result in `contexto.txt`: distilled agents/skills/prompt (no greetings or filler) + only the `CreateOrder` function, the full `manual.md` file (requested by name), and only `Artículo 6` of that manual — never the whole project.

---

## Best practices

- Start with `mode: "manual"` and review `contexto.txt` before switching to `automatic`.
- Use exact names in `focus` (`FunctionName()`, not descriptions) — resolution is literal.
- If a symbol isn't found, check the bounded excerpt that's produced anyway — it almost always means the name doesn't match exactly.
- Don't declare `focus` when the model needs to see the whole project — the absence of `focus` is the signal to work unrestricted (same as plain `mova run`).

---

## Validation

| Command | Expected result | Possible error | Fix |
|---|---|---|---|
| `mova compile [project]` | Creates `projects/[project]/contexto.txt`, confirms on stdout | `task not found` | Check `default_task` or pass the task explicitly |
| `mova run [project]` with `mode: "automatic"` | Compact output format (`PROJECT:... AGENT:...`), not the human `## AGENTS` headers | Still seeing the human format | `contextCompiler.enabled` is `false`, or `mode` isn't `"automatic"` |
| Focus on a nonexistent symbol | `FOCUS:` block with `not found: <symbol>` or a bounded excerpt, never empty | Block is empty | Shouldn't happen — report as a bug |
| Focus on a directory | `FOCUS:` block with `dir(N): file1, file2...` | File contents appear | Shouldn't happen — report as a bug |
