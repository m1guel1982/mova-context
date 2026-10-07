# FAQ — mova in 15 seconds

> **Per-task context specification, validated (dependency closure) and evidenced per execution.**

| In 15 seconds | |
|---|---|
| **What it is** | A local Go binary (CLI · `mova chat` · MCP · HTTP) that runs **before** the model call. |
| **What it does** | Applies a per-task `project.json`: checks that focus does not depend on excluded code, builds a deterministic context, applies per-block secrets/PII and budget, and writes `runs/<run_id>/` before releasing. |
| **What it is NOT** | A gateway, IDE, RAG, agent framework or compliance certification. It cannot see what an IDE/agent sends on its own. |
| **Try it** | `mova run 04-nebula-delivery recalcular-tarifas` → blocked by dependency closure, no API key, no model call. |

**Jump to:** [Essentials](#1-essentials) · [Privacy](#2-privacy-and-egress) · [Tokens & cost](#3-tokens-and-cost) · [Flow & architecture](#4-flow-and-architecture) · [context-trace & ranking](#5-context-trace-and-ranking) · [Chat & files](#6-chat-and-files) · [Install](#7-install-and-windows)

---

## 1. Essentials

**What is mova, exactly?** Two separable layers. **Layer 1 (always, zero dependencies):** a Markdown/JSON file convention (`workflow.md`, `agents/`, `skills/`, `prompts/`, `project.json`, `memory.md`) that any agent able to read files can follow. **Layer 2 (optional):** the Go engine that builds, audits, sanitizes and prices that context and, if you want, sends it to a local or cloud model. Layer 2 never hides or replaces Layer 1. It does not orchestrate reasoning: the LLM does.

**What does "context governance" concretely mean?** Deciding and leaving evidence of **what goes in, in what form, and under which controls**, before the context leaves your machine. It decides nothing during or after inference.

| What goes in | In what form | Under which controls |
|---|---|---|
| `focus` (files/symbols, not the whole repo) and declared agents/skills/prompts; never "everything on disk" | Sanitizer: duplicate paragraphs, repeated logs, optional comments/blank lines | PII Masking (optional), `max_tokens` (hard cap), spend circuit breaker |

**Does my data leave my machine?** Through Mova, only if `llm_profile` points to a cloud provider (`mova chat`/`chat_completion`) or an MCP host receives the context and sends it to its model. With `dry_run: true` Mova releases nothing; what it would have released is in `runs/<run_id>/context.txt`.

**Does it work with Claude Code?** Yes, as a local MCP server; Claude Code stays the model. Guide, permissions and limits: [`MCP_INTEGRATION.md`](MCP_INTEGRATION.md).

**What does it NOT guarantee?** It governs what goes **through Mova**. An agent's own tools (`Read`, `Bash`, @-mentions) stay outside, except reads that go through the `check_read`/`sanitize_tool_output` hooks in Claude Code. PII is heuristic, with no measured precision or recall.

**Why would I need it if I already have AGENTS.md, the agent and a gateway?** If the agent may read everything, you probably don't. Mova helps when you **restrict** context: AGENTS.md is an unverified instruction, the agent resolves dependencies by reading exactly what you restricted, and a gateway sees bytes, not the task or the symbols. Mova verifies the restriction does not drop code the task depends on, and records why each context was released or blocked.

---

## 2. Privacy and egress

**How does it protect PII?** Per block and by file type. In **data**: values of `field_keys` (name, address, RUT…) are pseudonymized and the same value is masked in other blocks; plus typed detectors (email, RUT, phone) and a shape/entropy score. In **code**: only typed detectors and secret-literal redaction, so syntax is not broken. Enabled with `budget.pii_masking.enabled`. It does not detect names in free text that never appear as a `field_keys` value.

**What does `dry_run` do?** With `egress_audit.dry_run: true` mova **hands over no context**: not to the provider, nor to an MCP/HTTP agent (it gets a notice with `tokens_sent: 0`). For a host agent (e.g. Claude Code) to read the **governed** context, use `dry_run: false`. Honest limit: mova guarantees *its* side; it cannot guarantee that a determined agent won't try to rebuild the context by reading other local files.

**Why does `pii-audit-log.json` show zeros under `security`?** That fine detail is filled only in *discovery* mode (remote repo, no `project.json`). In project mode, the real masking result is in `mova-budget-report.md`.

**What if I use a remote server (Oracle Cloud, AWS, my own)?** Nothing changes: `base_url` (in the model's `.json`, never in `project.json`) just points to another machine. Repo reading, Sanitizer, PII Masking, Budget Gate and Circuit Breaker **always run on the machine that runs the command**. The remote server only receives the final, already-sanitized payload and acts as a *stateless* inference coprocessor. Recommended: route it over a private network (Tailscale, WireGuard, the provider's virtual network), not an open public IP.

---

## 3. Tokens and cost

**How does it help with cost?** Three independent mechanisms, all **before** the call:

| Mechanism | What it does |
|---|---|
| Local estimate (`mova budget`, `context-trace`) | Counts tokens with `tiktoken-go` (`cl100k_base`, no network) and estimates USD per model from `config/prices.json`: `(tokens / unit) × input_price`. Writes `mova-budget-report.md`. |
| Budget Gate (`max_tokens`) | Hard content ceiling: if exceeded, it stops before spending and suggests `focus`, fewer agents/skills, etc. |
| Spend circuit breaker (`max_tokens_per_run` / `max_monthly_usd`) | A second ceiling that persists across runs (`mova-spend.json`). |

**What happens if I exceed the budget?** `on_exceed` in `project.json`: `warn` (notify) or `abort` (alias `block`, stop). It happens before sending, not after.

**How does MOVA Context handle token counting accuracy, especially across different tokenizers like Claude and GPT??** MOVA uses local token counting to estimate consumption before making a model call. It currently relies on an embedded BPE tokenizer via tiktoken-go with cl100k_base encoding, requiring zero network dependencies.

Because each provider and model may use a different tokenizer, the local count should be treated as an estimate rather than a universally exact measurement. MOVA addresses this difference through three mechanisms:

Local and deterministic estimation: Token counting happens locally prior to inference, allowing you to evaluate budgets and compare context variants without invoking a provider call. cl100k_base offers a reproducible baseline, which is particularly useful for models and workflows compatible with this encoding.

Comparison against actual usage: When a provider call returns usage metadata, MOVA logs the actual token count reported by the API alongside the local estimate. This variance is recorded in the token history (mova-token-history.json), making it easy to empirically evaluate how close the estimate was for a given workload and provider.

Counting on the final context: Context reduction and sanitization transformations are applied before token counting takes place. This ensures the budget is calculated on the exact context MOVA intends to pass to the model—after operations like deduplication, noise cleanup, and PII masking.

In summary: MOVA’s local count serves to estimate and control token budgets before execution, while the API-reported count measures actual usage afterward. The delta between the two depends on the specific tokenizer, model, and prompt content, so MOVA avoids assuming a fixed equivalence ratio across providers.


**What is the Cache Layout Guard?** It orders the prompt to exploit Anthropic/OpenAI/Gemini *prompt caching* (`cache_hint`, on by default): a **static prefix** first (agents + skills + base prompts + `workflow.md` rules, byte-identical across turns) and a **dynamic suffix** after (`focus`, `memory.md`, new turns). A layout hash verifies the prefix did not change and token deltas go to `mova-token-history.json`. On repetitive workloads it is the largest saving available.

---

## 4. Flow and architecture

**How does everything flow, end to end (`mova run` / `mova chat`)?**

| # | Step | Where it stops |
|---|---|---|
| 1 | Resolve project (`project.json`, agents/skills/prompts) | |
| 2 | Assemble context: agents + skills + prompt + memory + focus | |
| 3 | Sanitize (dedupe, logs, comments/blanks) | |
| 4 | PII Masking (optional) | |
| 5 | Budget Gate (`max_tokens`) | exceeded → stops, nothing leaves |
| 6 | Spend circuit breaker | `abort` → stops |
| 7 | Send to the model (`base_url`, local or remote) or, with `dry_run`, evidence only | |
| 8 | Feedback loop: real tokens → `mova-token-history.json` (cloud only) | |

Steps 1–6 **always run on your machine**. `mova run` builds the context, generates graphs/diagrams and does not call the model; `mova chat` also sends.

**Which architecture and why?** Pragmatic hexagonal (*Ports & Adapters*). The `core/` package depends only on the Go standard library and reasons over one port (`core.Adapter`). Storage adapters: `FileAdapter` (default) and `DBAdapter` (**PostgreSQL implemented; MongoDB is a stub**). CLI, Chat, MCP and HTTP are four thin entry points over the same functions (`http/server.go` wraps `mcp.Process()`; there is no second implementation). Reasons: portable core, same behaviour on all four doors, and extension without touching the engine (adapters, focus resolvers, providers, save writers; see [`SOURCE.md`](SOURCE.md)).

**Does it support concurrency?**
- **HTTP/MCP:** one goroutine per request, bounded by a semaphore (`MOVA_HTTP_MAX_CONCURRENCY`; default 4×CPU, min 8, max 64) plus read/write timeouts.
- **Multiagent:** a group's agents run in parallel through a worker pool (`MOVA_MAX_CONCURRENCY`; default CPU cores, capped at 8).
- **Shared state:** `mova-token-history.json`, `mova-spend.json` and `mova-context-cache.json` are serialized with a per-path mutex, so two simultaneous calls never lose an update.

**Can I customize the diagrams?** Everything drawn comes from `project.json`: only the agents, sources, privacy rules and metrics **actually configured and executed**. Each agent has its own context, so one group can mix a local flow (Ollama) and a cloud one.

---

## 5. context-trace and ranking

**What is `--task` and what is the "Top"?** `Top-ranked files` lists the files most relevant **to your task**, highest `score` first. It uses BM25 (the search-engine family) over file content, plus a small structural boost when an import/declaration or filename matches your text. It is **lexical, not semantic**: it compares words, not meaning.

**How do I read it?**
- One very high score with the rest flat and low = one strong match, the others weak. *Real example (FastAPI): `fastapi/dependencies/utils.py` 1019.04; the next 9, 3.00.*
- Many low, similar scores = a vague task: sharpen it and review by hand. Rule of thumb: >15 in a typical repo is a strong match.
- If the Top fills up with **translated docs**, filter languages: `--ignore "docs/!(en)/**"`. The engine already deduplicates translations (it keeps the best-scoring variant).
- **Unrelated tests** showing up is a known BM25 limit (one stray word like "error" is enough). Use specific tasks (`"fix OAuth2PasswordBearer token authentication"`, not `"fix bug"`).

**Does `--ignore` affect "Discovered"?** Yes, on purpose: ignored files are never read or tokenized. `--task` reads everything and then prioritizes. To isolate the effect of relevance, compare runs with and without `--ignore`.

**Why does `context-trace --repo` seem to hang?** At the end it asks `Generate a 'project.json'? [Y/n]` on stdin. In scripts/CI add `< /dev/null`.

---

## 6. Chat and files

**Does chat always create/edit/delete files when I just ask?** With cloud models (Claude, GPT, Gemini), reliably. With small local models (e.g. a 3B Qwen in LM Studio), not always: they sometimes answer "I can't create files" instead of calling `apply_file_changes`, even with the reinforced instruction `mova chat` adds. That is the model, not the engine: try a larger one, or ask explicitly "use `apply_file_changes` to create…".

**Is it the same on every door?** Same engine. The CLI asks for interactive confirmation (menu or y/n); MCP and HTTP, having no terminal, **return the proposal as text and require an explicit second call** (e.g. `delete_path` with `confirm:"true"`); they never apply on their own. Reference: [`FUNCTIONS.md`](FUNCTIONS.md), [`PROJECT_JSON.md`](PROJECT_JSON.md) (`apply`).

**Who writes `memory.md`?** `mova chat` (with `"memory": true`) and the `save_memory` tool. **`mova run` does not.**

---

## 7. Install and Windows

**How do I install and uninstall?** `make install` (needs Go ≥ 1.24) or `installers/<your OS>/…`. To remove the binary, `PATH` and `MOVA_PROJECT_ROOT`: [`uninstallers/`](../../../uninstallers/README.md) (tested on Linux; macOS/Windows not run — use `--dry-run`).

**Inherited fix (Windows paths):** an absolute `"repo"` with a drive letter (`E:\\…`) could fail to resolve `"focus": ["."]` and `"exclude"` when the letter's case differed. Fixed according to the previous FAQ; not re-verified in this review.

**Where is what was verified, and what was not?** [`docs/VERIFICATION.md`](../../VERIFICATION.md).
