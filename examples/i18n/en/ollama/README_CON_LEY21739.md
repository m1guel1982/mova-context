# Ollama Example — Local LLM, Embeddings and Reranker

Mova Context works with local models without changing anything except `project.json`.

There are three distinct model roles. Each is optional and configured separately:

| Role | project.json field | Purpose |
|------|-------------------|---------|
| Generation | `llm_profile` | Reads context and generates the response |
| Embedding | `embedding` | Finds the most relevant agents/skills |
| Reranker | `reranker` | Reorders results for higher precision |

**What never changes:**
```text
workflow.md  ←  unchanged
agents/      ←  unchanged
skills/      ←  unchanged
prompts/     ←  unchanged
```

---

## Case 1 — LLM only (Llama 3.1)

Simplest setup. Mova generates context, you paste it into Ollama.

```json
{
  "project": "pruebas-locales",
  "lang": "es",
  "llm_profile": {
    "type": "local",
    "provider": "ollama",
    "model": "llama3.1",
    "base_url": "http://localhost:11434"
  }
}
```

```bash
ollama pull llama3.1
mova run pruebas-locales crear-proyecto > context.txt
ollama run llama3.1 "$(cat context.txt)"
```

**When to use:** small project, few agents/skills, full context fits the model well.

---

## Case 2 — LLM + Embedding (bge-m3)

The embedding model automatically decides which agents and skills to include in the context,
instead of manually listing them. Useful when you have many files and don't want
to choose by hand which are relevant for each task.

`bge-m3` is multilingual: works well with Spanish, English and mixed corpora.

```json
{
  "project": "omnicanal-demo",
  "lang": "es",
  "llm_profile": {
    "type": "local",
    "provider": "ollama",
    "model": "llama3.1",
    "base_url": "http://localhost:11434"
  },
  "embedding": {
    "provider": "ollama",
    "model": "bge-m3",
    "base_url": "http://localhost:11434"
  }
}
```

```bash
ollama pull llama3.1
ollama pull bge-m3

# Embedding improves mova search with semantic similarity:
mova search "sales policy for whatsapp"
# → finds by semantic similarity, not just exact keyword match

mova run omnicanal-demo ventas-whatsapp > context.txt
```

**When to use:**
- Many agents/skills and you want automatic selection by relevance
- Multilingual corpus (Spanish + English mixed)
- Semantic search over the knowledge base (`mova search`)
- Memory search: "what did we decide about the payments module?"

**Why bge-m3 specifically:**
- Works in Spanish, English and 100+ languages out of the box
- Dimensions: 1024 (adjustable to 512 or 256 without notable loss)
- Runs 100% locally, no data sent to external APIs

---

## Case 3 — LLM + Embedding + Reranker (full local stack)

After the embedding retrieves the most similar candidates,
the reranker (cross-encoder) evaluates them in pairs to maximize precision.

```json
{
  "project": "ley-21719",
  "lang": "es",
  "llm_profile": {
    "type": "local",
    "provider": "ollama",
    "model": "llama3.1",
    "base_url": "http://localhost:11434"
  },
  "embedding": {
    "provider": "ollama",
    "model": "bge-m3",
    "base_url": "http://localhost:11434"
  },
  "reranker": {
    "provider": "ollama",
    "model": "bge-reranker-v2-m3",
    "base_url": "http://localhost:11434",
    "min_score": 0.7
  }
}
```

```bash
ollama pull llama3.1
ollama pull bge-m3
ollama pull bge-reranker-v2-m3

mova run ley-21719 analizar-contrato > context.txt
```

**When to use:**
- Critical precision: legal, medical, compliance, security applications
- Large knowledge base where exact relevance matters
- Discard uncertain results with `min_score` (0.7 = cuts the uncertain)
- bge-reranker-v2-m3 is the natural pair for bge-m3 (same maker, same vector space)

**`min_score` guide:**
```text
0.0  → accept everything (no filter)
0.5  → filters clearly irrelevant
0.7  → recommended for general use
0.9  → high confidence only
```

---

## Case 4 — Claude/GPT + local embedding (hybrid)

Use a powerful cloud LLM with locally running embeddings.
Privacy: the documents you index never leave your machine.

```json
{
  "project": "acme-demo",
  "lang": "es",
  "llm_profile": {
    "type": "powerful",
    "provider": "claude",
    "model": "claude-sonnet-4-6"
  },
  "embedding": {
    "provider": "ollama",
    "model": "bge-m3",
    "base_url": "http://localhost:11434"
  }
}
```

**When to use:**
- You want Claude/GPT quality but your document embeddings must stay in your infrastructure
- Sensitive data: contracts, customer data, proprietary code

---

## Quick comparison

| Configuration | Search | Precision | Privacy | Resources |
|---------------|--------|-----------|---------|-----------|
| `llm_profile` only | keyword | baseline | depends on LLM | minimal |
| + `embedding` (bge-m3) | semantic | good | 100% local if Ollama | RAM: ~1GB |
| + `reranker` | semantic + reranking | high | 100% local | RAM: ~1.5GB |
| Claude + local embedding | semantic | high | embeddings local | RAM: ~1GB |

---

## Installing all models

```bash
# LLM
ollama pull llama3.1           # 4.7GB — generation

# Embeddings (pick one)
ollama pull bge-m3             # 1.2GB — multilingual, recommended
ollama pull nomic-embed-text   # 274MB — English, very lightweight

# Reranker (natural pair for bge-m3)
ollama pull bge-reranker-v2-m3 # 568MB — multilingual

# Verify
ollama list
```

---

## Conclusion

Only `project.json` changes. The workflow, agents, skills and prompts are exactly the same
regardless of whether you use Claude, Llama, embeddings or a reranker.

```text
Many agents available          →  add embedding
Critical precision             →  add reranker
Sensitive data in embeddings   →  local ollama + cloud LLM
Fully local, no internet       →  llm_profile + embedding + reranker all in Ollama
```
