# Ollama — Local LLM, Embeddings and Reranker

Mova Context works with local models without changing anything except `project.json`.

There are three independent model roles. Each one is optional and configured separately.

| Role       | project.json field | Purpose                                                |
| ---------- | ------------------ | ------------------------------------------------------ |
| Generation | `llm_profile`      | Reads the assembled context and generates the response |
| Embedding  | `embedding`        | Selects the most relevant agents, skills and prompts   |
| Reranker   | `reranker`         | Reorders retrieved results for maximum accuracy        |

The project structure never changes:

```text
workflow.md
agents/
skills/
prompts/
memory/
```

---

# Case 1 — Local LLM Only

The simplest configuration. Mova assembles the project context and the LLM generates the final response.

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

mova run pruebas-locales crear-proyecto > contexto.txt

ollama run llama3.1 "$(cat contexto.txt)"
```

Recommended when:

* Small or medium projects.
* The entire context comfortably fits within the model context window.
* No semantic retrieval is required.

---

# Case 2 — Local LLM + Embeddings

Embeddings automatically determine which agents, skills and prompts are relevant before assembling the final context.

```json
{
  "project": "pruebas-locales",
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

mova search "crear proyecto"

mova run pruebas-locales crear-proyecto > contexto.txt
```

Recommended when:

* The knowledge base contains many agents or skills.
* You want semantic search instead of keyword matching.
* You work with multilingual documentation.

---

# Case 3 — Local LLM + Embeddings + Reranker

The reranker evaluates the retrieved candidates and keeps only the most relevant ones.

```json
{
  "project": "pruebas-locales",
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

mova run pruebas-locales crear-proyecto > contexto.txt
```

Recommended when:

* Retrieval accuracy is critical.
* The knowledge base is large.
* Only highly relevant results should be included.

Recommended `min_score` values:

```text
0.0  → Accept everything
0.5  → Filter clearly irrelevant results
0.7  → Recommended for general use
0.9  → Only high-confidence results
```

---

# Quick Comparison

| Configuration | Retrieval            | Accuracy | Privacy            | Resources   |
| ------------- | -------------------- | -------- | ------------------ | ----------- |
| LLM only      | Keyword              | Standard | Depends on the LLM | Low         |
| + Embedding   | Semantic             | Good     | 100% local         | ~1 GB RAM   |
| + Reranker    | Semantic + reranking | High     | 100% local         | ~1.5 GB RAM |

---

# Install Required Models

```bash
# Generation
ollama pull llama3.1

# Embeddings
ollama pull bge-m3

# Reranker
ollama pull bge-reranker-v2-m3

# Verify installation
ollama list
```

---

# Conclusion

Only `project.json` changes.

Your workflow, agents, skills, prompts and memory remain exactly the same regardless of the models you use.

```text
Small project                     → Local LLM
Large knowledge base              → Add embeddings
Maximum retrieval precision       → Add reranker
Fully offline                     → Run everything locally with Ollama
```

---

# Appendix A — Using Ollama Inside Docker

If Ollama runs inside a Docker container, you can send the assembled context directly into the model without copying and pasting.

## Windows PowerShell

### 1. Generate the complete Mova Context

```powershell
.\mova-windows-amd64.exe run pruebas-locales crear-proyecto | Out-File -Encoding utf8 contexto_mova.txt
```

### 2. Copy the generated file into the Ollama container

```powershell
docker cp contexto_mova.txt mova_ollama:/tmp/contexto_mova.txt
```

### 3. Execute the model using the file inside the container

```powershell
docker exec -it mova_ollama sh -c "cat /tmp/contexto_mova.txt | ollama run llama3.1"
```

### 4. Remove the temporary local file

```powershell
Remove-Item contexto_mova.txt
```

This approach avoids large clipboard operations and keeps the complete context transfer entirely local between Mova Context and the Ollama container.
