# Mova Context

> **El conocimiento operativo pertenece al proyecto. El razonamiento pertenece al modelo.**

Docs: **[Español](README.md)** · **[English](README.en.md)**

---

## Índice

1. [La convención](#1-la-convención) — las 6 piezas de las que depende todo
2. [Por qué existe Mova Context](#2-por-qué-existe-mova-context) — la historia corta
3. [Cómo funciona](#3-cómo-funciona) — el mapa completo en un diagrama
4. [¿Necesito el CLI?](#4-necesito-el-cli) — tabla de decisión
5. [Tokenomics](#5-tokenomics--el-plato-fuerte) — por qué cada token cuenta, y cómo Mova lo controla
6. [Lo que trae el CLI](#6-lo-que-trae-el-cli) — resumen de las novedades
7. [Probarlo en 2 minutos](#7-probarlo-en-2-minutos)
8. [Seguir profundizando](#8-seguir-profundizando)

---

## 1. La convención

**Mova Context es una convención de archivos, no una herramienta.** Todo lo que necesitás cabe en esta estructura, y funciona sin instalar absolutamente nada:

```text
workflow.md                       ← especificación: cómo se construye el contexto

agents/[dominio]/                 ← quién razona (rol, experiencia)
skills/[dominio]/                 ← qué sabe (conocimiento técnico o de negocio)
prompts/[dominio]/                ← qué debe hacer (la tarea)

projects/[proyecto]/
├── project.json                  ← qué agents, skills y prompts usar
└── memory.md                     ← historial de sesiones del proyecto
```

Con un agente que pueda leer tu repositorio (Claude Code, Cursor, Gemini CLI, Claude Desktop...) alcanza con pedirle:

```text
Lee workflow.md, resuelve el proyecto [nombre], ejecuta la task [task] y construye el contexto.
```

El agente sigue `workflow.md`, resuelve el `project.json`, carga `agents`/`skills`/`prompts`, inyecta variables, suma la `memory.md` y arma el contexto final. Si trabajás desde un chat web que no puede tocar tu repo (ChatGPT, Claude.ai, Gemini), **`mova run`** genera exactamente ese mismo contexto, listo para copiar y pegar.

**Si el binario `mova` desapareciera mañana, el proyecto seguiría funcionando igual** — porque el conocimiento vive en el repositorio, nunca en la herramienta. El CLI solo automatiza tareas: ensamblar contexto, administrar memoria, o exponerlo por HTTP/MCP.

---

## 2. Por qué existe Mova Context

Esto nació de escribir el mismo prompt mil veces. De explicarle el proyecto a un modelo el lunes, y volver a explicárselo el martes porque abrí otro chat. De cambiar de GPT a Claude y sentir que el trabajo de una semana se esfumaba. En algún punto me aburrí y necesité orden.

El conocimiento operativo de un proyecto —convenciones, reglas de negocio, decisiones ya tomadas, memoria del trabajo hecho— termina **atrapado dentro del chat**. Y con el tiempo aparecen siempre los mismos síntomas:

```text
ANTES                               MOVA CONTEXT

Contexto dentro del chat      →      Contexto dentro del repositorio
Cambiar de modelo              →      Cambiar una línea en project.json
  significa empezar de nuevo
Cada dev explica distinto      →      Una única fuente de verdad
Las decisiones se pierden      →      memory.md conserva el historial
El conocimiento depende        →      El conocimiento pertenece al proyecto,
  del proveedor                        no al proveedor
```

No es magia ni una promesa exagerada: es simplemente mover ese conocimiento de la conversación al repositorio, para que cualquier modelo —Claude, GPT, Gemini, Ollama— lo lea sin que tengas que volver a explicárselo.

---

## 3. Cómo funciona

```text
                     Mova Context

          agents/  skills/  prompts/
          project.json  memory.md
                 │
                 ▼
           workflow.md
        (la especificación)
                 │
      ┌──────────┴──────────┐
      ▼                     ▼
Un agente que lee       mova run (CLI,
tu repositorio           opcional)
      │                     │
      └──────────┬──────────┘
                 ▼
        Contexto ensamblado
                 │
                 ▼
 Claude · GPT · Gemini · Ollama
      o cualquier otro LLM
```

---

## 4. ¿Necesito el CLI?

| Situación | ¿CLI? |
|---|---|
| Ya usás Claude Code, Cursor u otro agente que lee el repo | **No.** El agente sigue `workflow.md` directo. |
| Querés pegar el contexto en un chat web (Claude.ai, ChatGPT, Gemini) | **Sí.** `mova run` te lo da listo para copiar. |
| Querés llamar la API de un modelo desde un script | **Sí.** Más rápido que hacer que el modelo lea todos los archivos. |
| Querés correr un modelo local (Ollama) | **Sí.** `mova run ... \| ollama run modelo` en una línea. |
| Querés guardar memoria de sesión sin tocar `memory.md` a mano | **Sí.** `mova memory` lo hace por vos. |
| Querés exponer el contexto por HTTP o como servidor MCP | **Sí.** `mova http` o `mova mcp start`. |

Con o sin CLI, la fuente de verdad nunca cambia: `workflow.md`, `agents/`, `skills/`, `prompts/`, `project.json`, `memory.md`. Sin CLI perdés comodidad; con CLI ganás velocidad y automatización — nunca al revés.

---

## 5. Tokenomics — el plato fuerte

Cada token que le mandás a un modelo cuesta algo: plata si es Cloud, o precisión si es local y el contexto no entra en su ventana. Mova Context no te promete magia — te da **control real, antes de gastar nada**.

### El gate: `budget.max_tokens` corta la ejecución, no solo avisa

```json
// project.json
{ "budget": { "max_tokens": 8000 } }
```

Si el contexto ensamblado se pasa de ese límite, **Mova detiene la ejecución antes de que salga un solo token hacia el modelo** — desde `mova chat`, la tool MCP `chat_completion`, o HTTP, siempre igual:

```text
ERROR
Current context (14,250 tokens) exceeds the configured limit (8,000).
Suggestion: Use --focus to reduce the included files.
```

Esto convierte el control de costo en una regla de arquitectura, no en un hábito que alguien puede olvidar.

### El reporte: `mova budget`, cero llamadas a ningún proveedor

`mova budget mi-proyecto` calcula, **100% en tu máquina**, cuántos tokens usaría el contexto real (agents + skills + prompt + focus + memory) y cuánto costaría en OpenAI, Anthropic y Google, según los precios que vos mismo cargás en `config/prices.json`. El reporte desglosa el gasto pieza por pieza para que sepas exactamente qué recortar primero:

```text
mova budget mi-proyecto mi-task --focus
→ mova-budget-report.md
```

**Que quede claro:** esto es una estimación calculada con [tiktoken-go](https://github.com/tiktoken-go/tokenizer) (el tokenizador de OpenAI), cruzada contra precios manuales. No reemplaza la factura real — es una brújula para decidir qué optimizar, y el propio reporte lo advierte tres veces.

### El aprendizaje: cada llamada real afina la estimación

Cuando `mova chat` o `chat_completion` llaman a un proveedor Cloud de verdad, ese proveedor devuelve cuántos tokens contó él. Mova guarda esos dos números —estimado local vs. real— en `mova-token-history.json`, nunca el contenido ni los prompts:

```json
{ "anthropic": { "total_local_tokens": 120000, "total_api_tokens": 122760 } }
```

Con el tiempo, `mova budget` te muestra qué tan preciso es tu estimador para **ese proyecto específico** — una calibración propia, no un benchmark genérico.

### La limpieza automática: deduplicación en todo el contexto

Si un párrafo se copió y pegó en dos agents, o se repite entre un skill y un prompt, Mova lo detecta (texto idéntico, nunca código/SQL/JSON) y lo deja una sola vez:

```text
[Dedup] Removed 3 duplicated paragraphs (~450 tokens saved).
```

### Resumido en una frase

En la nube, exceso de contexto = factura más alta. En local, exceso de contexto = el modelo trunca o degrada en silencio. Es el mismo problema con dos costos distintos — y Mova aplica el mismo mecanismo de control en ambos casos: medir antes de gastar, cortar si se pasa, y aprender de cada llamada real.

Más detalle y ejemplos completos en [COMMANDS.md § mova budget](COMMANDS.md#mova-budget--estimación-local-de-tokens-y-costo).

---

## 6. Lo que trae el CLI

- **`/save`** — un único comando para crear o editar cualquier archivo o carpeta desde el chat. `/save "informe.docx"` guarda la última respuesta ahí; el formato (`.md`, `.docx`, `.pdf`, `.xlsx`, `.svg`, código, ~20 extensiones más) se elige solo por la extensión. Mismo comportamiento por MCP (`save`) y HTTP (`POST /save`).
- **Lenguaje natural en el chat** — escribí *"genera el informe en docs/salida.pdf"* y listo, sin comandos. Ver [COMMANDS.md § lenguaje natural](COMMANDS.md#crear-archivos-hablando-lenguaje-natural-en-el-chat).
- **Documentos de oficina y medios** — `.docx`, `.xlsx`, `.pdf` reales (sin dependencias externas, solo librería estándar de Go), SVG nativo, e imágenes vía un modelo de difusión local.
- **`mova chat`** — hablá con Ollama, LM Studio, vLLM, OpenAI, Anthropic o Google desde la terminal, con el mismo contexto de siempre inyectado como system prompt.
- **`mova budget`** — ver sección 5.

---

## 7. Probarlo en 2 minutos

```bash
go build -o mova ./src/cli
mova run pruebas-locales
```

Hay un proyecto de ejemplo completo en `projects/pruebas-locales/` — inspeccioná su `project.json` o corré el comando de arriba para ver el contexto ensamblado.

---

## 8. Seguir profundizando

| Quiero... | Documento |
|---|---|
| Ver todos los comandos (memoria, Focus, MCP, HTTP, tokenomics) | [COMMANDS.md](COMMANDS.md) |
| Leer la especificación completa que siguen los modelos | [workflow.md](../../../workflow.md) |
| Entender el código fuente (Resolvers, Adapters, cómo extenderlo) | [SOURCE.md](../SOURCE.md) *(English)* |

---

> **El conocimiento operativo pertenece al proyecto. El razonamiento pertenece al modelo.**
>
> Mova Context es la convención formada por `workflow.md`, `agents/`, `skills/`, `prompts/`, `project.json` y `memory.md`. El CLI solo automatiza el trabajo con esa convención — nunca la reemplaza.
