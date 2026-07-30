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

**Mova Context es una convención de archivos, no una herramienta.** Todo lo que se necesita cabe en esta estructura, y funciona sin instalar absolutamente nada:

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

El agente sigue `workflow.md`, resuelve el `project.json`, carga `agents`/`skills`/`prompts`, inyecta variables, suma la `memory.md` y arma el contexto final. Si se trabaja desde un chat web que no puede tocar tu repo (ChatGPT, Claude.ai, Gemini), **`mova run`** genera exactamente ese mismo contexto, listo para copiar y pegar.

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
| Ya se usa Claude Code, Cursor u otro agente que lee el repo | **No.** El agente sigue `workflow.md` directo. |
| ¿Quieres pegar el contexto en un chat web (Claude.ai, ChatGPT, Gemini)? | **Sí.** `mova run` lo entrega listo para copiar. |
| ¿Quieres llamar la API de un modelo desde un script? | **Sí.** Más rápido que hacer que el modelo lea todos los archivos. |
| ¿Quieres correr un modelo local (Ollama)? | **Sí.** `mova run ... \| ollama run modelo` en una línea. |
| Se quiere guardar memoria de sesión sin tocar `memory.md` a mano | **Sí.** `mova memory` lo hace automáticamente. |
| ¿Quieres exponer el contexto por HTTP o como servidor MCP? | **Sí.** `mova http` o `mova mcp start`. |

Con o sin CLI, la fuente de verdad nunca cambia: `workflow.md`, `agents/`, `skills/`, `prompts/`, `project.json`, `memory.md`. Sin CLI se pierde comodidad; con CLI se gana velocidad y automatización — nunca al revés.

---

## 5. Tokenomics  

### La analogía: la balanza del aeropuerto

Antes de viajar, pesás la valija en tu casa con una balanza de baño. Te da una idea aproximada, pero no es la balanza oficial. En el aeropuerto, la valija se pesa de verdad — y ahí aparece la diferencia entre lo que calculaste y lo que realmente pesa. Si supieras de antemano cuánto se equivoca tu balanza casera (por ejemplo, "siempre marca 3% menos de lo real"), podrías ajustar el cálculo la próxima vez y dejar de llevarte sorpresas en el mostrador.

**Mova Tokenomics hace exactamente eso, pero con tokens en vez de kilos:**

| En la analogía | En Mova |
|---|---|
| Balanza de baño en casa | Estimación local con `tiktoken-go`, antes de mandar nada a ningún proveedor |
| Balanza oficial del aeropuerto | Conteo real de tokens que devuelve el proveedor (Anthropic, OpenAI, Google) cuando la API se llama de verdad |
| Límite de equipaje de la aerolínea | `budget.max_tokens` en tu `project.json` |
| La libreta donde anotás "en casa dio X, en el aeropuerto dio Y" | El archivo `mova-token-history.json` que vive en tu proyecto |

Y como esa libreta es tuya —no un promedio genérico de miles de valijas ajenas— la calibración que aprende Mova es específica de **tu proyecto**: su mezcla de idioma, su código, sus documentos.

**Por qué esto importa para el bolsillo (y la cordura):** cada token que le mandás a un modelo Cloud cuesta dinero; si el modelo es local, un contexto que no entra en la ventana se trunca o degrada en silencio, sin avisar. Mova ataca los dos problemas con el mismo mecanismo: **medir antes de gastar, cortar si se pasa, y aprender de cada llamada real.**

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

`mova budget mi-proyecto` calcula, **100% en tu máquina**, cuántos tokens usaría el contexto real (agents + skills + prompt + focus + memory) y cuánto costaría en OpenAI, Anthropic y Google, según los precios que se cargan en `config/prices.json`. El reporte desglosa el gasto pieza por pieza para que sepas exactamente qué recortar primero:

```text
mova budget mi-proyecto mi-task --focus
→ mova-budget-report.md
```

**Que quede claro:** esto es una estimación calculada con [tiktoken-go](https://github.com/tiktoken-go/tokenizer) (el tokenizador de OpenAI), cruzada contra precios manuales. No reemplaza la factura real — es una brújula para decidir qué optimizar, y el propio reporte lo advierte tres veces.

#### El reporte, explicado en simple

Cuando corrés el comando de arriba, obtenés un archivo `mova-budget-report.md`. En criollo, dice tres cosas:

| Sección del reporte | Qué te dice |
|---|---|
| **Token & Cost Breakdown** | Cuánto pesa, en tokens y en dólares, cada pedazo de tu contexto (agents, skills, prompt, focus, memory) — para saber qué recortar primero. |
| **Budget Limit** | Tu límite configurado vs. lo que realmente estás usando, en tokens y en porcentaje. |
| **Historical Token Accuracy** | Qué tan bien le achunta tu estimador local a la realidad, medido con tus propias llamadas pasadas a cada proveedor. |

Ejemplo real (recortado de un caso concreto): un proyecto con límite de 5.000 tokens usa 1.207 tokens (24.1% del límite) y costaría entre USD 0.0015 y USD 0.0060 según el proveedor — menos de 3 pesos chilenos. Con eso ya sabés, **antes de gastar un centavo**, si te conviene mandarlo a OpenAI, Anthropic o Google, y cuánto margen te queda antes de chocar con el límite.

### El aprendizaje: cada llamada real afina la estimación

Esta es, probablemente, la función más importante de Tokenomics — y la que más vale la pena entender a fondo.

**Qué se guarda, y dónde.** Cada vez que `mova chat` o la tool MCP `chat_completion` hacen una llamada real a un proveedor Cloud (Anthropic, OpenAI o Google), la respuesta de la API viene acompañada de un campo de uso (`usage`) que indica cuántos tokens contó el proveedor de verdad — no una estimación: el número exacto por el que se te factura. Mova toma ese número real, junto con la estimación local que había calculado con `tiktoken-go` *antes* de mandar el pedido, y los suma a dos contadores acumulados guardados en un archivo local de tu proyecto: `mova-token-history.json`. **Nunca se guarda el contenido, ni los prompts, ni las respuestas** — solo dos números por proveedor:

```json
{ "anthropic": { "total_local_tokens": 120000, "total_api_tokens": 122760 } }
```

**Cómo se calcula la desviación.** Con esos dos acumulados, la fórmula es simple y transparente — nada de caja negra:

```text
desviación % = (total_api_tokens − total_local_tokens) / total_local_tokens × 100
```

Con los números de arriba: `(122.760 − 120.000) / 120.000 × 100 = +2.3%`. Es decir: para *este proyecto en particular*, el estimador local de Mova subestima el gasto real en Anthropic por un 2.3%, en promedio.

**Por qué se acumula en vez de guardar un historial llamada por llamada.** Cada llamada real que hacés suma a esos dos totales, así que la desviación mostrada no depende del último request (que puede traer ruido: un prompt muy corto, un caso raro), sino que es un promedio ponderado por todo tu uso real acumulado. Cuantas más llamadas reales hagas, más se afina el número — y más confiable se vuelve — para ese proyecto puntual, con su propia mezcla de idioma, código y densidad de contexto.

**Ejemplo con Google (tomado del propio reporte de este repo):**

```json
{ "google": { "total_local_tokens": 1205, "total_api_tokens": 1201 } }
```

`(1.201 − 1.205) / 1.205 × 100 = −0.33%` → así es exactamente como `mova budget` llega al **"−0.3%"** que se ve en la sección *Historical Token Accuracy* del reporte. No es una cifra inventada ni un promedio bajado de internet: es matemática directa sobre tus propias llamadas.

**Cómo evoluciona el archivo con el tiempo.** A medida que se acumulan más llamadas reales, la desviación deja de dar saltos bruscos y se estabiliza en un número confiable:

| Después de... | `total_local_tokens` | `total_api_tokens` | Desviación acumulada |
|---|---|---|---|
| 1ª llamada real | 1.000 | 1.030 | +3.0% |
| 5ª llamadas reales | 5.400 | 5.505 | +1.9% |
| 20ª llamadas reales | 21.800 | 22.190 | +1.8% |

**Qué hace Mova con ese número.** `mova budget` usa esa desviación acumulada para mostrarte, junto a cada estimación nueva, qué tan lejos podría estar de la realidad — y con el tiempo te deja saber, para ese proyecto puntual, si tu estimador tiende a quedarse corto o largo con cada proveedor, para que ajustes `budget.max_tokens` con margen real en vez de a ciegas.

**Qué pasa si nunca llamaste a un proveedor real.** Si `total_local_tokens` es 0 (nunca hiciste una llamada real con `mova chat` a ese proveedor), el reporte muestra `No historical data` — Mova no inventa una desviación sin datos reales que la respalden.

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
