## Frequently Asked Questions (FAQ)

### 1. How does Mova Context handle token-count accuracy across different tokenizers, like Claude vs GPT?

Local token counting uses an embedded, network-free BPE tokenizer via `tiktoken-go` under the `cl100k_base` encoding (GPT-4's standard). Provider differences are resolved through three design pillars:

* **Minimal variance in practice:** although Anthropic and OpenAI use different vocabulary encodings, modern BPE algorithms split code, JSON schemas, and technical text with nearly identical characters-per-token ratios. In real workloads, the variance between `cl100k_base` and Claude's own tokenizer consistently stays below 3–5%, more than enough to validate pre-execution budget limits.
* **API feedback loop:** Mova doesn't rely solely on the local estimate. Every time a model call completes, it reads the real `prompt_tokens` count the provider's API returns (Anthropic, OpenAI, Gemini) and records the difference in `mova-token-history.json`. This calibrates future estimates against real data without bloating the binary with per-tokenizer dependencies.
* **Sanitize-before-count:** before counting tokens, the Token Firewall collapses duplicate whitespace, repeated log sequences, and standardizes PII replacements (`[PII_a1b2c3d4]`). Cleaning structural noise first keeps the token count stable and predictable regardless of the target model.

---

### 2. Can I customize what shows up in architecture diagrams, or is everything shown by default?

The diagramming engine is fully dynamic and configuration-driven:

* **Config-driven structure:** everything plotted on the diagram comes directly from the `project.json` configuration files defined in the project structure.
* **On-demand rendering:** the engine reads definitions at run time and cleanly renders only the agents, data sources, privacy rules, and metrics that were actually configured and executed for that specific pipeline.
* **Modularity:** each agent keeps its own context configuration, letting local flows (`llama3.2:3b` via Ollama) stay isolated from Cloud flows (`gemini-3-flash-preview`) within the same execution group.

---

### 3. How does the Cache Layout Guard work, and how does it keep dynamic conversations from breaking prompt caching?

The Cache Layout Guard is designed to maximize prompt-caching usage on providers like Anthropic, OpenAI, and Gemini by structuring the prompt into two clearly delimited sections from byte 0:

* **Static prefix (offset 0):** groups the permanent context pieces (`agents` + `skills` + base prompts + `workflow.md` rules). Mova guarantees this top block stays byte-for-byte identical across runs and chat turns. Since providers evaluate caching top-down (exact prefix matching), keeping this heavy block first avoids cache invalidation.
* **Dynamic suffix:** every changing piece of information — `focus` resolution (specific files), `memory.md` history, and new conversation turns — is appended exclusively *after* the static block.
* **Integrity check:** Mova generates a hash of the layout state to verify the static prefix hasn't drifted between calls, logging token deltas in `mova-token-history.json` without compromising speed or cache-hit ratio.

---

### 4. What kind of tool is Mova Context, exactly? A framework, an agent, a CLI?

None of those categories alone describes it well — it's more accurate to think of it as two separable layers:

* **Layer 1 — File convention (always available, zero dependencies):** `workflow.md` + `agents/` + `skills/` + `prompts/` + `project.json` + `memory.md`. This is, at its core, a **Markdown-based structured workflow/prompt format**. Any coding agent that already reads files (Claude Code, Cursor, or a plain copy-paste into a web chat) can follow it with nothing installed — in that sense it functions as a **workflow**, an **agent definition**, a **skill**, or a reusable **prompt**, depending on how it's used.
* **Layer 2 — Go engine (optional):** a binary that automates working with that same convention — assembles context, audits it, sanitizes it, estimates its cost, and optionally sends it to a model (local or Cloud) through CLI, Chat, MCP, or HTTP. This layer is what adds context governance, token budgeting, and inference control — but it never replaces or hides Layer 1.

Put differently: Mova Context is, first, a **lightweight convention**; and, second and optionally, a **context governance and optimization engine for LLMs** built on top of that convention. It is not an agent-orchestration framework (it doesn't orchestrate reasoning or decide what to do on the model's behalf — that's the LLM's job), nor a plain utility CLI (it maintains state, budget, and data protection across calls).

---

### 5. What software architecture and design pattern was used to build it, and why?

The Go engine follows **Hexagonal Architecture (Ports & Adapters)**, with a core (`core/`) that depends on nothing beyond Go's standard library:

* **The port:** the `core.Adapter` interface (`GetProject`, `GetKnowledge`, `Search`, `AppendMemory`, ...) — the entire engine reasons in terms of this contract, never of how it's implemented.
* **The adapters:** `core.FileAdapter` (file-based reading — the default case) and `adapters.DBAdapter` (Postgres/MongoDB) both implement the same port interchangeably. Adding a third storage backend means implementing the interface once, without touching a single line of the engine.
* **The same "entry ports" for all four doors:** CLI, Chat, MCP, and HTTP aren't four separate implementations of the business logic — they're four thin entry adapters that call exactly the same functions (`budget.BuildGatedContext`, `mcp.Process`, etc.). `http/server.go` is, literally, a thin wrapper over `mcp.Process()` — there's no second implementation of the protocol.

Why this pattern instead of, say, MVC or textbook Clean Architecture? Three concrete reasons:

1. **The domain (`core/`) has to be trivially portable.** By depending only on the standard library, the core never couples to a storage format, a model provider, or a transport — which is exactly what Ports & Adapters guarantees by design.
2. **Four entry points, one logic.** The "same behavior in CLI/Chat/MCP/HTTP" requirement is exactly the problem Ports & Adapters solves: entry ports (primary adapters) are swappable without duplicating business rules.
3. **Extensibility without breaking anything existing.** Every "Extension Point" documented in `SOURCE.md` (storage Adapters, Focus Resolvers, Model Providers, Save Writers) is, underneath, the same idea applied at different points: define the contract once, allow multiple interchangeable implementations behind it.

In practice this is a pragmatic variant of Hexagonal/Ports & Adapters — not a strict academic implementation with explicitly named "entities/use-cases/interface-adapters" layers like Clean Architecture, but the same central principle (the domain doesn't know about the outside world; the outside world adapts to the domain) applied with the minimum ceremony a single-module Go binary needs.

---

### 6. What does "context governance" mean in Mova Context, concretely?

Context governance is the set of rules and mechanisms that decide **what information enters the context sent to a model, in what form, and under what controls** — before that context ever leaves the user's machine:

* **What enters:** `focus` (specific files/directories/symbols, instead of the whole repository), agents/skills/prompts explicitly declared in `project.json` — never an implicit "everything on disk" inclusion.
* **In what form:** the Sanitizer (exact-paragraph deduplication, repeated-log collapsing, optional comment/blank-line removal) reduces structural noise before anything is counted or sent.
* **Under what controls:** PII Masking (optional, structural algorithm — word shape + Shannon entropy, see `config/policy.json`), the Budget Gate (`max_tokens` as a hard ceiling, see `budget.EnforceLimit`), and the spend Circuit Breaker (`max_tokens_per_run` / `max_monthly_usd`, see `budget/spend.go`).

All of this runs in Layer 2 (the engine), always on the machine that assembles the context — never on the remote server that eventually receives the already-sanitized result (see question 12 and `PROJECT_JSON.md § Distributed architecture`).

---

### 7. How does Mova Context protect personal information (PII) before it reaches an external model?

PII Masking (optional, `budget.pii_masking.enabled`) is **structural, not dictionary- or language-rule-based**: it analyzes a token's shape (proportion of digits, separators, symbols, length, sustained uppercase) and its Shannon entropy, combining both into a score (`min_score` in `config/policy.json`, 0.62 by default). A token that crosses that threshold is replaced with a deterministic label (`[PII_a1b2c3d4]`) before the context ever leaves the machine toward any model, local or Cloud.

This is a technical tool, not a substitute for a privacy policy or legal review — the full disclaimer lives in `COMMANDS.md § PII Masking`. Its concrete value is making visible and auditable, with an actual number (`N token(s) pseudonymized`), how much of what's being sent looks like personal data — something that today, in most LLM integrations, stays invisible inside an API call.

---

### 8. How does Mova Context help control costs and optimize the token budget?

The "Token Firewall" is the combination of three independent mechanisms, all running **before** the real model call:

* **Local estimation** (`mova budget`): counts tokens with an embedded BPE tokenizer (`tiktoken-go`, `cl100k_base`), no network call, and calculates the approximate USD cost against `config/prices.json` for every configured model — letting you compare before deciding which one to use.
* **Budget Gate** (`max_tokens`): a hard content ceiling — if the assembled context exceeds it, execution stops before spending anything, with a concrete suggestion (enable `focus`, trim agents/skills, etc.).
* **Spend Circuit Breaker** (`max_tokens_per_run` / `max_monthly_usd`): a second, independent ceiling that DOES persist across runs — it cuts off or warns when a project's accumulated monthly spend approaches a dollar limit.

On top of that, the **feedback loop** (`mova-token-history.json`) compares the local estimate against the real count each Cloud provider returns, calibrating estimate accuracy over time — and the **Cache Layout Guard** (see question 3) maximizes each provider's native prompt-caching, which is, in practice, the largest cost saving available for repetitive workloads.

---

### 9. What does the full end-to-end data flow look like when running `mova run` or `mova chat`?

1. **Project resolution** (`runtime.FindRoot` + `core.Adapter.GetProject`): `project.json` is located and its agent/skill/prompt references are loaded.
2. **Assembly** (`core.BuildContext` / Focus Resolvers): the raw context is assembled — agents + skills + prompt + memory + focus (if configured).
3. **Sanitization** (`sanitize.Apply`, optionally cached via `budget.SanitizeCached`): deduplication, log collapsing, optional comment/blank removal.
4. **PII Masking** (optional): structural replacement of personal-data-shaped tokens.
5. **Budget Gate**: checked against `max_tokens`; if exceeded, execution stops here — nothing is sent yet.
6. **Spend Circuit Breaker**: checked against `budget.spend` ceilings; if `on_exceed: "abort"` and the ceiling is exceeded, execution also stops here.
7. **Sending** (`models.Session.Send`/`SendStream`): only at this step does the final, already-sanitized payload leave the machine toward `base_url` (local or remote).
8. **Feedback loop**: if the provider is Cloud, the real token count it returns is logged in `mova-token-history.json` to calibrate future estimates.

Steps 1 through 6 always run on the machine executing the command — never on a remote server, even when step 7 points at one (see question 12).

---

### 10. How exactly is the "budget" (`mova budget`) calculated? What number does it give, and how reliable is it?

`mova budget <project> [task]` assembles the same context `mova run` would (steps 1-4 of question 9), counts its tokens with `tiktoken-go` under `cl100k_base`, and calculates the estimated USD cost for every model configured in `config/prices.json`, using `(tokens / unit_divisor) * input_price`. The result is written to `mova-budget-report.md` alongside a mandatory disclaimer: this is an **estimate**, not the exact amount each provider will charge — the real observed variance against different Cloud-provider tokenizers stays, in practice, below 3–5% (see question 1), enough for pre-execution budget decisions but not for exact billing.

---


### 11. What happens to my data if I use a centralized remote server (Oracle Cloud, AWS, my own server)?

None of this changes just because a remote endpoint is used — it's the same separation of responsibilities as always, only `base_url` (in the model's `.json`, never in `project.json`) points at another machine instead of `localhost`:

* Reading the repository, the Sanitizer, PII Masking, the Budget Gate, and the Circuit Breaker always run on the machine executing the command — the remote server never runs them, because it never receives the repository or `project.json`.
* The remote server only ever receives the final, already-sanitized payload, acts as a stateless inference coprocessor (it stores nothing about the project's content), and returns the model's reply.
* Routing that traffic over a private network (Tailscale, WireGuard, or the cloud provider's own virtual network) is strongly recommended over an unprotected public IP — see `DEPLOY.md § Network security` for the full detail.

See also `PROJECT_JSON.md § Distributed architecture (remote endpoints)` for the full table of what runs where.

---

### 12. How does Mova Context handle high concurrency (multiple users/calls at the same time)?

The engine is built to serve multiple simultaneous invocations from any of its four doors:

* **HTTP/MCP** (`http/server.go`): every request runs in its own goroutine, bounded by a configurable semaphore (`MOVA_HTTP_MAX_CONCURRENCY`, defaulting to 4× CPU cores) to avoid unbounded resource use under traffic bursts, plus read/write timeouts so a slow client can't hold a slot indefinitely.
* **Multi-agent runs** (`orchestrator.RunGroup`): agents in the same group run in parallel through a bounded worker pool (`MOVA_MAX_CONCURRENCY`, defaulting to available CPU cores, capped at 8), instead of one at a time.
* **Protected shared state:** `mova-token-history.json`, `mova-spend.json`, and `mova-context-cache.json` — the three files different concurrent invocations of the same project could read/write at once — are serialized with a per-path mutex (see `budget/filelock.go`), so two concurrent calls can never clobber each other and lose a spend or history update.

This is what makes the centralized deployment from question 12 viable: a single instance can serve a whole team without race conditions in its own internal state.
