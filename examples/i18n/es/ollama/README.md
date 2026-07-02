# Ollama — LLM Local, Embeddings y Reranker

Mova Context funciona con modelos locales sin cambiar nada, salvo `project.json`.

Existen tres roles independientes de modelos. Cada uno es opcional y se configura por separado.

| Rol        | Campo en `project.json` | Descripción                                                            |
| ---------- | ----------------------- | ---------------------------------------------------------------------- |
| Generación | `llm_profile`           | Lee el contexto ensamblado y genera la respuesta                       |
| Embedding  | `embedding`             | Selecciona automáticamente los agents, skills y prompts más relevantes |
| Reranker   | `reranker`              | Reordena los resultados recuperados para maximizar la precisión        |

La estructura del proyecto nunca cambia:

```text
workflow.md
agents/
skills/
prompts/
memory/
```

---

# Caso 1 — Solo LLM Local

La configuración más simple. Mova Context ensambla el contexto del proyecto y el LLM genera la respuesta final.

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

**Cuándo utilizarlo:**

* Proyectos pequeños o medianos.
* El contexto completo cabe dentro de la ventana de contexto del modelo.
* No se requiere búsqueda semántica.

---

# Caso 2 — LLM Local + Embeddings

El modelo de embeddings determina automáticamente qué agents, skills y prompts son relevantes antes de ensamblar el contexto final.

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

**Cuándo utilizarlo:**

* La base de conocimiento contiene muchos agents o skills.
* Se requiere búsqueda semántica en lugar de búsqueda por palabras clave.
* Se trabaja con documentación en múltiples idiomas.

---

# Caso 3 — LLM Local + Embeddings + Reranker

El reranker evalúa los candidatos recuperados por el embedding y conserva únicamente los resultados más relevantes.

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

**Cuándo utilizarlo:**

* La precisión de recuperación es crítica.
* La base de conocimiento es grande.
* Solo deben incluirse resultados altamente relevantes.

Valores recomendados para `min_score`:

```text
0.0  → Acepta todos los resultados
0.5  → Filtra resultados claramente irrelevantes
0.7  → Recomendado para uso general
0.9  → Solo resultados de alta confianza
```

---

# Comparación rápida

| Configuración | Recuperación          | Precisión | Privacidad      | Recursos    |
| ------------- | --------------------- | --------- | --------------- | ----------- |
| Solo LLM      | Palabras clave        | Normal    | Depende del LLM | Baja        |
| + Embedding   | Semántica             | Buena     | 100% local      | ~1 GB RAM   |
| + Reranker    | Semántica + reranking | Alta      | 100% local      | ~1.5 GB RAM |

---

# Instalación de los modelos

```bash
# LLM
ollama pull llama3.1

# Embeddings
ollama pull bge-m3

# Reranker
ollama pull bge-reranker-v2-m3

# Verificar instalación
ollama list
```

---

# Conclusión

Solo cambia `project.json`.

El workflow, los agents, los skills, los prompts y la memoria permanecen exactamente iguales independientemente de los modelos utilizados.

```text
Proyecto pequeño                 → Solo LLM local
Base de conocimiento grande      → Agregar embeddings
Máxima precisión                 → Agregar reranker
Todo sin conexión a Internet     → Ejecutar todo localmente con Ollama
```

---

# Anexo A — Uso de Ollama en Docker

Si Ollama se ejecuta dentro de un contenedor Docker, puedes enviar el contexto generado por Mova Context directamente al modelo sin necesidad de copiar y pegar.

## Windows PowerShell

### 1. Generar el contexto completo de Mova Context

```powershell
.\mova-windows-amd64.exe run pruebas-locales crear-proyecto | Out-File -Encoding utf8 contexto_mova.txt
```

### 2. Copiar el archivo generado al contenedor de Ollama

```powershell
docker cp contexto_mova.txt mova_ollama:/tmp/contexto_mova.txt
```

### 3. Ejecutar el modelo leyendo el archivo directamente desde el contenedor

```powershell
docker exec -it mova_ollama sh -c "cat /tmp/contexto_mova.txt | ollama run llama3.2:3b"
```

### 4. Eliminar el archivo temporal local

```powershell
Remove-Item contexto_mova.txt
```

Este procedimiento evita copiar grandes cantidades de texto al portapapeles y mantiene toda la transferencia del contexto de forma local entre Mova Context y el contenedor de Ollama.
