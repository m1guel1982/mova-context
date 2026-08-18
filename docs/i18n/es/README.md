# Mova Context

> **El motor universal de contexto para IA: audita, protege y visualiza lo que realmente le llega a tu LLM — local o cloud, en un comando.**

Docs: **[Español](README.md)** · **[English](../en/README.md)**

---

```text
Fuentes  ──▶  Firewall de Privacidad (PII)  ──▶  Agentes  ──▶  Diagrama auditable
(datos)         (Sanitizer + PII Masking)      (la lógica propia)      (SVG / PNG / PDF)
```

- ✅ **Se sabe exactamente qué datos salen de la máquina** — antes de que salgan, no después de una fuga.
- ✅ **Un solo comando genera un diagrama real de la arquitectura de contexto** — auto-documentación viva, no un dibujo que alguien hizo una vez y quedó desactualizado.
- ✅ **Funciona igual desde CLI, Chat, MCP o HTTP** — mismo motor, mismo resultado, cero fricción de integración.

---

## Índice

1. [El diagrama, en acción](#1-el-diagrama-en-acción) — el artefacto que resume todo el proyecto
2. [Por qué existe Mova Context](#2-por-qué-existe-mova-context) — los dos problemas que resuelve
3. [Instalación y prueba rápida (2 minutos)](#3-instalación-y-prueba-rápida-2-minutos)
4. [Las cuatro puertas de entrada](#4-las-cuatro-puertas-de-entrada) — CLI, Chat, MCP, HTTP
5. [Cómo funciona](#5-cómo-funciona) — el mapa completo en un diagrama
6. [La convención](#6-la-convención) — las 6 piezas de las que depende todo
7. [Token Firewall y Tokenomics](#7-token-firewall-y-tokenomics) — la capa de protección y control de costo
8. [Job Engine, Cron y Multiagente](#8-job-engine-cron-y-multiagente)
9. [Interfaz visual — `mova ui`](#9-interfaz-visual--mova-ui)
10. [¿Necesito el CLI?](#10-necesito-el-cli) — tabla de decisión
11. [Seguir profundizando](#11-seguir-profundizando)

---
 
## 1. El diagrama, en acción — Cloud y Local en el mismo grupo multiagente

Este es el resultado real de correr `mova run <proyecto> --diagram` contra este repositorio — un caso de auditoría de datos de clientes (Ley 21.719, Chile) con tres agentes trabajando en paralelo (`data-analyst`, `purpose-analyst` y `ai-privacy-reviewer`).

El diagrama muestra cómo Mova Context ejecuta modelos locales y en la nube dentro del mismo flujo de trabajo multiagente:

![Ejemplo de diagrama multiagente Cloud y Local](../assets/example-cloud-local.png)

- **Ejecución híbrida por agente:** Mientras `data-analyst` y `purpose-analyst` corren sobre Ollama local (`llama3.2:3b`), el agente `ai-privacy-reviewer` procesa sobre Google Cloud (`gemini-3-flash-preview`).
- **Protección de PII selectiva:** El Token Firewall aplica `PII Masking: true` exclusivamente al agente que envía datos fuera de la máquina (`ai-privacy-reviewer`), seudonimizando la información antes de la transmisión (`78/1694 token(s) pseudonymized`), mientras los agentes locales mantienen `PII Masking: false`.
- **Desglose de costos en `FINAL SUMMARY`:**
  - **Agentes locales:** Muestran `(local — no cost)`, confirmando procesamiento sin consumo ni facturación de API.
  - **Agente Cloud:** Calcula el costo exacto del envío (`ai-privacy-reviewer: 7503 tok, cheapest: google/gemini $0.0094`).

```bash
# Cloud
mova run ejemplo-ley21719-pii-context --diagram --export svg,png,pdf --path ./diagramas

# Local — mismo comando, mismo proyecto, solo cambia project.json
```

El proyecto de ejemplo incluye ambas variantes listas para probar: `projects/ejemplo-ley21719-pii-context/ai-privacy-reviewer/project.json` (local, activo por defecto) y `project_cloud.json` (la alternativa Cloud) — para alternar entre ambas alcanza con renombrar el archivo activo a `project_local.json` y renombrar `project_cloud.json` a `project.json`, sin tocar ningún otro archivo ni ninguna línea de código.

**Leído en tres partes, en cualquiera de los dos diagramas:**

- **`SOURCES` (qué entra):** los archivos reales que arman el contexto — un JSON de clientes, un PDF, un DOCX, un log técnico. Cada fuente es un archivo real del proyecto, nunca un dato inventado.
- **`TOKEN FIREWALL` (qué se limpia):** el Sanitizer elimina ruido repetido antes de contar un solo token; el PII Masking, cuando está activo, reemplaza datos con forma de información personal por pseudónimos determinísticos (`[PII_a1b2c3d4]`) antes de que salgan de la máquina.
- **`AGENTS` & `METRICS` (qué se entrega):** cada agente aparece con su modelo real, si es local o cloud, y si tiene protección de PII activada — y al final, el resumen: tokens antes, tokens después, porcentaje de reducción, y costo real (solo si el modelo factura; un modelo local nunca muestra una cifra de costo).

Un comando. Sin código adicional. El resultado es un archivo que se puede adjuntar a un ticket de auditoría, mostrar en una reunión de cumplimiento, o versionar en el propio repositorio como prueba de qué hace el sistema, actualizada en cada corrida.

---

## 2. Por qué existe Mova Context

Cualquiera que integra un modelo de lenguaje en un sistema real termina, tarde o temprano, con dos preguntas sin una respuesta simple:

**a) ¿Qué datos sensibles se están mandando a un modelo externo (o incluso a uno local)?**
Nombres, correos, números de documento, historiales — muchas veces terminan dentro de un prompt sin que nadie los haya revisado uno por uno. No hay una forma sencilla de saber, antes de enviar, qué información con forma de dato personal viaja en ese contexto.

**b) ¿Cómo se documenta y se le muestra a otra persona cómo fluye el contexto en un sistema de IA?**
Un diagrama de arquitectura hecho a mano queda desactualizado la semana siguiente. Explicarlo en una reunión a partir de archivos de configuración es lento y poco confiable.

Mova Context resuelve ambos problemas con el mismo mecanismo: un pipeline de contexto que pasa, siempre, por una capa de sanitización y protección de datos antes de llegar al modelo, y que puede convertirse en una imagen auditable con un solo comando, sin importar qué tan grande sea el proyecto.

Esto no reemplaza una política de privacidad ni una revisión legal — es una herramienta técnica que hace visible y auditable lo que hoy, en la mayoría de los sistemas de IA, queda oculto dentro de una llamada a una API.

---

## 3. Instalación y prueba rápida (2 minutos)

### Opción A — Instalador automático (recomendado)

1. Ir a la carpeta `installers/` del repositorio.
2. Ejecutar el instalador correspondiente al sistema operativo:

| Sistema operativo | Instalador |
|---|---|
| Windows | `install.bat` |
| macOS | `install.command` |
| Linux | `install.sh` |

El instalador compila el ejecutable, lo instala en el sistema y configura el `PATH` automáticamente. Al finalizar, desde cualquier terminal:

```bash
mova run pruebas-locales --diagram --export png
```

También se puede abrir la interfaz interactiva con `mova ui`, o simplemente `mova` para ver los comandos disponibles.

### Opción B — Makefile (para quienes ya tienen Go)

```bash
make install
mova run pruebas-locales
```

O solo compilar localmente sin instalar:

```bash
make build
./dist/mova run pruebas-locales
```

### Opción C — Compilación manual

```bash
go build -o mova ./src/cli
./mova run pruebas-locales
```

### Trabajar con proyectos en cualquier carpeta

No hace falta que el proyecto viva dentro del repositorio de Mova. El campo `"repo"` de `project.json` acepta una ruta absoluta a cualquier ubicación (otra unidad en Windows, otro punto de montaje en Linux/macOS) — los instaladores de doble clic ya dejan todo configurado para que esto funcione sin pasos adicionales. Ver [COMMANDS.md § Trabajar entre distintas unidades/ubicaciones](COMMANDS.md#trabajar-entre-distintas-unidadesubicaciones-windowslinuxmacos).

---

## 4. Las cuatro puertas de entrada

La misma lógica de auditoría y protección de contexto se aprovecha exactamente igual sin importar cómo se llegue a ella — un único motor detrás de cuatro formas distintas de usarlo.

### MCP Server — para Cursor, VS Code, Claude Desktop

Un paso de configuración. A partir de ahí, cada contexto que el editor arma para el modelo pasa primero por el firewall de privacidad de Mova, en tiempo real, mientras se programa — sin cambiar el flujo de trabajo habitual. Ver [COMMANDS.md § Servidor MCP](COMMANDS.md#13-servidor-mcp--mova-mcp-start).

### API HTTP — el middleware ultraligero en Go

Se coloca por delante de las llamadas a cualquier LLM. Sin dependencias pesadas, arranca en milisegundos. Es la pieza que le falta a un equipo que ya tiene un pipeline de IA en producción y necesita auditarlo sin reescribirlo — una llamada `POST` es suficiente para sanitizar y generar el diagrama de una petición.

```bash
curl -X POST http://localhost:3000/diagram -d '{"project": "mi-proyecto", "export": "svg"}'
```

### CLI — el copiloto de terminal

Un comando genera el diagrama de arquitectura del sistema de contexto. Se integra directo en CI/CD: cada commit puede dejar, automáticamente, la foto actualizada de qué datos fluyen hacia dónde — auto-documentación que nunca queda vieja porque se regenera sola.

```bash
mova run mi-proyecto --diagram --export png --path ./docs/diagramas
```

### Chat — la consola de diagnóstico

Antes de llevar un prompt a producción, se prueba acá: se ve el contexto ensamblado tal cual llegaría al modelo, se verifica si el firewall de privacidad detectó algo, se ajustan parámetros al instante — sin salir de la terminal.

```bash
mova chat mi-proyecto
> /diagram
```

Los cuatro canales comparten el mismo motor de ensamblado, el mismo firewall de privacidad y el mismo generador de diagramas — lo que cambia es solo la puerta de entrada. Ver [COMMANDS.md § Diagramas visuales](COMMANDS.md#21-diagramas-visuales--mova-run-proyecto---diagram) para un ejemplo de una línea por cada canal.

---

## 5. Cómo funciona

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
el repositorio            Chat, MCP, HTTP)
      │                     │
      └──────────┬──────────┘
                 ▼
      Token Firewall (Sanitizer,
      PII Masking, Cache Guard)
                 │
                 ▼
        Contexto auditado
                 │
                 ▼
 Claude · GPT · Gemini · Ollama
      o cualquier otro LLM
```

---

## 6. La convención

**Mova Context es, en su base, una convención de archivos — no una herramienta obligatoria.** Todo lo que se necesita cabe en esta estructura, y funciona sin instalar nada:

```text
workflow.md                       ← especificación: cómo se construye el contexto

agents/[dominio]/                 ← quién razona (rol, experiencia)
skills/[dominio]/                 ← qué sabe (conocimiento técnico o de negocio)
prompts/[dominio]/                ← qué debe hacer (la tarea)

projects/[proyecto]/
├── project.json                  ← qué agents, skills y prompts usar
└── memory.md                     ← historial de sesiones del proyecto
```

Con un agente que pueda leer el repositorio (Claude Code, Cursor, Gemini CLI, Claude Desktop...) alcanza con pedirle:

```text
Lee workflow.md, resuelve el proyecto [nombre], ejecuta la task [task] y construye el contexto.
```

El agente sigue `workflow.md`, resuelve el `project.json`, carga `agents`/`skills`/`prompts`, inyecta variables, suma la `memory.md` y arma el contexto final. Si se trabaja desde un chat web que no puede tocar el repositorio (ChatGPT, Claude.ai, Gemini), **`mova run`** genera exactamente ese mismo contexto, listo para copiar y pegar.

**Si el binario `mova` desapareciera mañana, el proyecto seguiría funcionando igual** — porque el conocimiento vive en el repositorio, nunca en la herramienta. El CLI solo automatiza tareas: ensamblar contexto, auditar y proteger datos, generar diagramas, administrar memoria, o exponer todo por HTTP/MCP.

Esto nació de escribir el mismo prompt mil veces, de explicarle un proyecto a un modelo el lunes y volver a explicárselo el martes en otro chat, de cambiar de proveedor y sentir que el trabajo de una semana se esfumaba. El conocimiento operativo de un proyecto —convenciones, reglas de negocio, decisiones ya tomadas— no debería quedar atrapado dentro de una conversación:

```text
ANTES                               MOVA CONTEXT

Contexto dentro del chat      →      Contexto dentro del repositorio
Cambiar de modelo              →      Cambiar una línea en project.json
  significa empezar de nuevo
Cada persona explica distinto  →      Una única fuente de verdad
Las decisiones se pierden      →      memory.md conserva el historial
Nadie sabe qué datos salieron  →      El diagrama lo muestra, siempre
```

---

## 7. Token Firewall y Tokenomics

Además de la auditoría y protección de datos, Mova Context también controla — de forma determinística y sin IA — cuánto contexto se envía y cuánto cuesta.

### El Token Firewall

Tres etapas automáticas que corren delante de cada ejecución (`mova run`, `mova chat`, jobs, MCP, HTTP, el mismo TUI), sin ningún comando adicional:

| Etapa | Qué hace | Resultado real, medido (ejemplo incluido) |
|---|---|---|
| **Sanitizer** | Colapsa líneas de log repetidas, corridas de líneas en blanco y encabezados de archivo duplicados — antes de contar nada | 2.737 → 1.764 tokens: **35,6% menos tokens, misma información** |
| **PII Masking** (opcional) | Reemplaza tokens con forma estructural de dato personal por pseudónimos determinísticos, antes de contar o enviar nada | Ver la sección 1 — el propio diagrama muestra cuántos tokens se protegieron |
| **Cache Layout Guard** | Reordena el prompt para que sus primeros tokens sean un prefijo estable, byte a byte — lo que activa el prompt caching real de Claude/GPT/Gemini | ~1.050 de 1.167 tokens del prefijo estático estimados reutilizables en un acierto de caché |
| **Circuit Breaker** | Límites por corrida y mensuales en USD, verificados **antes** de enviar nada | Una corrida que superaría su límite nunca llega al modelo |

Cada etapa está activada por defecto (excepto PII Masking, que requiere activación explícita por proyecto — ver [PROJECT_JSON.md § Budget](PROJECT_JSON.md#budget-y-el-token-firewall)), se puede desactivar de forma independiente, y cada número de la tabla es real, medido corriendo el ejemplo incluido en este repositorio. Ver [COMMANDS.md § Token Firewall](COMMANDS.md#20-token-firewall) para la mecánica completa.

### El gate: `budget.max_tokens`

```json
{ "budget": { "max_tokens": 8000 } }
```

Si el contexto ensamblado supera ese límite, Mova detiene la ejecución antes de que salga un solo token hacia el modelo — desde `mova chat`, la tool MCP `chat_completion`, o HTTP, siempre igual. El control de costo se vuelve una regla de arquitectura, no un hábito que alguien puede olvidar.

### El reporte: `mova budget`, cero llamadas a ningún proveedor

`mova budget mi-proyecto` calcula, 100% en la propia máquina, cuántos tokens usaría el contexto real y cuánto costaría en OpenAI, Anthropic y Google, según `config/prices.json`. Es una estimación calculada con [tiktoken-go](https://github.com/tiktoken-go/tokenizer), cruzada contra precios manuales — no reemplaza la factura real, es una brújula para decidir qué optimizar.

### El aprendizaje: cada llamada real afina la estimación

Cada vez que se hace una llamada real a un proveedor, Mova compara la estimación local contra el conteo real que devuelve la API, y guarda esa desviación en `mova-token-history.json` — nunca el contenido, solo dos números por proveedor. Con el tiempo, la estimación se calibra específicamente para cada proyecto: su mezcla de idioma, su código, sus documentos. Ver [COMMANDS.md § mova budget](COMMANDS.md#15-tokenomics--mova-budget) para la mecánica completa con ejemplos reales.

---

## 8. Job Engine, Cron y Multiagente

### Programar trabajo: el Job Engine

Un proyecto puede declarar **jobs** en su `project.json` — ejecuciones programadas y desatendidas que arman contexto, auditan datos, guardan reportes, actualizan memoria y generan diagramas, todo según un horario cron:

```json
{
  "jobs": [
    {
      "schedule": "0 2 * * *",
      "tasks": ["auditar-checkout", "auditar-cookies"],
      "save": "reports/auditoria_{date}.pdf",
      "budget": { "focus": true },
      "memory": "Auditoría de checkout y cookies realizada"
    }
  ]
}
```

Se ejecuta bajo demanda con `mova jobs run <proyecto>`, o se inicia el daemon que revisa cada proyecto una vez por minuto con `mova jobs start`. Ver [PROJECT_JSON.md § Jobs](PROJECT_JSON.md#jobs) para cada campo.

`schedule` usa la sintaxis cron estándar de 5 campos (minuto, hora, día del mes, mes, día de la semana) — por ejemplo, `"0 2 * * *"` corre todos los días a las 2:00 AM, y `"*/15 * * * *"` corre cada 15 minutos.

### Multiagente: varios agentes bajo un grupo

Un directorio bajo `projects/` puede contener varios agentes independientes, cada uno un proyecto normal, orquestados por un `config.json` padre:

```text
projects/
    ventas_online/
        config.json          ← el orquestador
        vendedor/project.json
        atencionCliente/project.json
        soporte/project.json
```

```bash
mova agents run ventas_online          # todos los agentes, en secuencia
mova agents run ventas_online vendedor # un solo agente
mova run ventas_online --diagram       # el diagrama de todo el grupo
```

Cada agente mantiene su propia memoria, budget, focus, tareas, jobs, y — como se ve en la sección 1 — su propio estado de protección de datos. Ver [PROJECT_JSON.md § Multiagente](PROJECT_JSON.md#multiagente-grupos-de-agentes).

---

## 9. Interfaz visual — `mova ui`

Todo lo de arriba también se puede usar desde una interfaz de terminal simple y liviana:

```bash
mova ui
```

Un solo comando, navegado con las flechas y Enter: chat, `project.json`, `workflow.md`, configuración de modelos, memoria, jobs, multiagentes, búsqueda con navegación a archivo/línea exacta, y diagramas — todo desde el mismo lugar, sin comandos nuevos por función. La interfaz no reemplaza nada: llama exactamente a los mismos componentes que ya usan `mova chat`, `mova jobs run` y `mova run --diagram`. Ver [COMMANDS.md § Interfaz visual](COMMANDS.md#19-interfaz-visual--mova-ui).

---

## 10. ¿Necesito el CLI?

| Situación | ¿CLI? |
|---|---|
| Ya se usa Claude Code, Cursor u otro agente que lee el repositorio | **No.** El agente sigue `workflow.md` directo. |
| Se quiere pegar el contexto en un chat web (Claude.ai, ChatGPT, Gemini) | **Sí.** `mova run` lo entrega listo para copiar. |
| Se necesita auditar qué datos personales viajan hacia el modelo | **Sí.** `mova run --diagram` lo muestra en una imagen. |
| Se quiere llamar la API de un modelo desde un script | **Sí.** Más rápido que hacer que el modelo lea todos los archivos. |
| Se quiere correr un modelo local (Ollama) | **Sí.** `mova run ... \| ollama run modelo` en una línea. |
| Se quiere exponer el contexto por HTTP o como servidor MCP | **Sí.** `mova http` o `mova mcp start`. |

Con o sin CLI, la fuente de verdad nunca cambia: `workflow.md`, `agents/`, `skills/`, `prompts/`, `project.json`, `memory.md`. Sin CLI se pierde comodidad; con CLI se gana velocidad, auditoría automática y protección de datos por defecto — nunca al revés.

---

## 11. Seguir profundizando

| Se busca... | Documento |
|---|---|
| Ver todos los comandos (memoria, Focus, MCP, HTTP, diagramas, tokenomics) | [COMMANDS.md](COMMANDS.md) |
| Leer la especificación completa que siguen los modelos | [workflow.md](../../../workflow.md) |
| Entender el código fuente (Resolvers, Adapters, cómo extenderlo) | [SOURCE.md](../SOURCE.md) *(en inglés)* |

---

> **El conocimiento operativo pertenece al proyecto. El razonamiento pertenece al modelo.**
>
> Mova Context es la convención formada por `workflow.md`, `agents/`, `skills/`, `prompts/`, `project.json` y `memory.md`. El CLI solo automatiza el trabajo con esa convención — nunca la reemplaza.
