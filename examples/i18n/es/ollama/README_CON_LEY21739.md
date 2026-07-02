# Ejemplo Ollama — LLM local, Embeddings y Reranker

Mova Context funciona con modelos locales sin cambiar nada salvo `project.json`.

Hay tres roles de modelo distintos. Cada uno es opcional y se configura por separado:

| Rol | Campo en project.json | Para qué sirve |
|-----|----------------------|----------------|
| Generación | `llm_profile` | Lee el contexto y genera la respuesta |
| Embedding | `embedding` | Encuentra los agents/skills más relevantes |
| Reranker | `reranker` | Reordena resultados para mayor precisión |

**Lo que nunca cambia:**
```text
workflow.md  ←  igual
agents/      ←  igual
skills/      ←  igual
prompts/     ←  igual
```

---

## Caso 1 — Solo LLM local (Llama 3.1)

El más simple. Mova genera el contexto y tú lo pegas en Ollama.

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

**Cuándo usarlo:** proyecto pequeño, pocos agents/skills, contexto completo cabe bien en el modelo.

---

## Caso 2 — LLM + Embedding (bge-m3)

El embedding decide automáticamente qué agents y skills incluir en el contexto,
en lugar de cargar todo manualmente. Útil cuando tienes muchos archivos y no quieres
elegir a mano cuáles son relevantes para cada tarea.

`bge-m3` es multilingual: funciona bien con español, inglés y mezclas.

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

# El embedding mejora mova search:
mova search "política de ventas por whatsapp"
# → encuentra por similitud semántica, no solo por keyword exacta

# El contexto se genera igual — el embedding trabaja en la selección previa
mova run omnicanal-demo ventas-whatsapp > contexto.txt
```

**Cuándo usarlo:**
- Muchos agents/skills disponibles y quieres selección automática por relevancia
- Corpus multilingüe (español + inglés mezclados)
- Búsquedas semánticas sobre la base de conocimiento (`mova search`)
- Búsqueda en memoria: "¿qué decidimos sobre el módulo de pagos?"

**Por qué bge-m3 en particular:**
- Funciona en español, inglés y +100 idiomas sin configuración extra
- Dimensiones: 1024 (ajustable a 512 o 256 sin pérdida notable)
- Corre 100% local, sin enviar datos a ninguna API externa

---

## Caso 3 — LLM + Embedding + Reranker (stack completo local)

Después de que el embedding recupera los candidatos más similares,
el reranker (cross-encoder) los evalúa en pares para maximizar la precisión.

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

mova run ley-21719 analizar-contrato > contexto.txt
```

**Cuándo usarlo:**
- Precisión crítica: aplicaciones legales, médicas, compliance, seguridad
- Base de conocimiento grande donde la relevancia exacta importa mucho
- Quieres descartar resultados dudosos con `min_score` (0.7 = corta lo incierto)
- El bge-reranker-v2-m3 es el par natural de bge-m3 (mismo fabricante, mismos vectores)

**`min_score`:**
```text
0.0  → acepta todo (sin filtro)
0.5  → filtra claramente irrelevante
0.7  → recomendado para uso general
0.9  → solo resultados de alta confianza
```

---

## Caso 4 — Claude/GPT + Embedding local (híbrido)

Puedes usar un LLM potente en la nube con embeddings corriendo localmente.
Privacidad: los documentos que indexas nunca salen de tu máquina.

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

**Cuándo usarlo:**
- Quieres la calidad de Claude/GPT pero los embeddings de tus documentos no deben salir de tu infraestructura
- Datos sensibles: contratos, datos de clientes, código propietario

---

## Comparación rápida

| Configuración | Búsqueda | Precisión | Privacidad | Recursos |
|---------------|----------|-----------|------------|----------|
| Solo `llm_profile` | keyword | normal | depende del LLM | mínimos |
| + `embedding` (bge-m3) | semántica | buena | 100% local si Ollama | RAM: ~1GB |
| + `reranker` | semántica + reranking | alta | 100% local | RAM: ~1.5GB |
| Claude + embedding local | semántica | alta | embeddings locales | RAM: ~1GB |

---

## Instalación de todos los modelos

```bash
# LLM
ollama pull llama3.1           # 4.7GB — generación

# Embeddings (elegir uno)
ollama pull bge-m3             # 1.2GB — multilingual, recomendado
ollama pull nomic-embed-text   # 274MB — inglés, muy liviano

# Reranker (par natural de bge-m3)
ollama pull bge-reranker-v2-m3 # 568MB — multilingual

# Verificar que están disponibles
ollama list
```

---

## Conclusión

Solo cambia `project.json`. El workflow, agents, skills y prompts son exactamente los mismos
sin importar si usas Claude, Llama, embeddings o reranker.

```text
Más agentes disponibles       →  agregar embedding
Precisión crítica             →  agregar reranker
Datos sensibles en embeddings →  ollama local + LLM en nube
Todo local, sin internet      →  llm_profile + embedding + reranker todos en Ollama
```
