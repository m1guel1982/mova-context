# FAQ — mova en 15 segundos

> **Especificación de contexto por tarea, validada (cierre de dependencias) y con evidencia inmutable por ejecución.**

| En 15 segundos | |
|---|---|
| **Qué es** | Un binario Go local (CLI · `mova chat` · MCP · HTTP) que corre **antes** de la llamada al modelo. |
| **Qué hace** | Aplica un `project.json` por tarea: valida que el focus no dependa de código excluido, arma un contexto determinista, aplica secretos/PII por bloque y presupuesto, y escribe `runs/<run_id>/` antes de liberar. |
| **Qué NO es** | Un gateway, IDE, RAG, framework de agentes ni certificación de cumplimiento. No ve lo que un IDE/agente envía por su cuenta. |
| **Pruébalo** | `mova run 04-nebula-delivery recalcular-tarifas` → bloqueado por cierre de dependencias, sin API key y sin llamar a un modelo. |

**Atajos:** [Esencial](#1-lo-esencial) · [Privacidad](#2-privacidad-y-egreso) · [Tokens y costo](#3-tokens-y-costo) · [Flujo y arquitectura](#4-flujo-y-arquitectura) · [context-trace y ranking](#5-context-trace-y-ranking) · [Chat y archivos](#6-chat-y-archivos) · [Instalación](#7-instalación-y-windows)

---

## 1. Lo esencial

**¿Qué es mova exactamente?** Dos capas separables. **Capa 1 (siempre, cero dependencias):** una convención de archivos Markdown/JSON (`workflow.md`, `agents/`, `skills/`, `prompts/`, `project.json`, `memory.md`) que cualquier agente que lea archivos puede seguir. **Capa 2 (opcional):** el motor Go que arma, audita, sanitiza y cotiza ese contexto y, si quieres, lo envía a un modelo local o cloud. La Capa 2 nunca oculta ni reemplaza la Capa 1. No orquesta razonamiento: eso lo hace el LLM.

**¿Qué significa «gobernanza de contexto», concretamente?** Decidir y dejar evidencia de **qué entra, en qué forma y con qué controles**, antes de que el contexto salga de tu máquina. No decide nada durante ni después de la inferencia.

| Qué entra | En qué forma | Con qué controles |
|---|---|---|
| `focus` (archivos/símbolos, no todo el repo) y agents/skills/prompts declarados; nunca «todo lo que haya en disco» | Sanitizer: párrafos duplicados, logs repetidos, comentarios/líneas en blanco (opcional) | PII Masking (opcional), `max_tokens` (techo duro), circuit breaker de gasto |

**¿Mis datos salen de mi máquina?** Por Mova, solo si `llm_profile` apunta a un proveedor cloud (`mova chat`/`chat_completion`) o si un host MCP recibe el contexto y lo envía a su modelo. Con `dry_run: true`, Mova no libera nada; lo que habría liberado queda en `runs/<run_id>/context.txt`.

**¿Funciona con Claude Code?** Sí, como servidor MCP local; Claude Code sigue siendo el modelo. Guía, permisos y límites: [`MCP_INTEGRATION.md`](MCP_INTEGRATION.md).

**¿Qué NO garantiza?** Gobierna lo que pasa **por Mova**. Las tools propias de un agente (`Read`, `Bash`, @-menciones) quedan fuera, salvo las lecturas que pasen por los hooks `check_read`/`sanitize_tool_output` en Claude Code. El PII es heurístico, sin precisión ni recall medidos.

**¿Para qué lo necesito si ya tengo AGENTS.md, el agente y un gateway?** Si el agente puede leerlo todo, probablemente no lo necesitas. Mova aporta cuando **restringes** el contexto: AGENTS.md es una instrucción que nadie verifica, el agente resuelve dependencias leyendo justamente lo que restringiste, y el gateway ve bytes, no la tarea ni los símbolos. Mova verifica que la restricción no deje fuera código del que depende la tarea, y registra por qué liberó o bloqueó cada contexto.

---

## 2. Privacidad y egreso

**¿Cómo protege la PII?** Por bloque y según el tipo de archivo. En **datos**: los valores de `field_keys` (nombre, dirección, RUT…) se seudonimizan y el mismo valor se enmascara en los demás bloques; además hay detectores tipados (email, RUT, teléfono) y un puntaje de forma/entropía. En **código**: solo detectores tipados y redacción del literal de secretos, para no romper la sintaxis. Se activa con `budget.pii_masking.enabled`. No detecta nombres en texto libre que no aparezcan como valor de `field_keys`.

**¿Qué hace `dry_run`?** Con `egress_audit.dry_run: true` mova **no entrega** contexto: ni al proveedor ni a un agente MCP/HTTP (recibe un aviso con `tokens_sent: 0`). Para que un agente anfitrión (p. ej. Claude Code) lea el contexto **gobernado**, usa `dry_run: false`. Límite honesto: mova garantiza *su* lado; no puede garantizar que un agente decidido no intente reconstruir el contexto leyendo otros archivos locales.

**¿Por qué `pii-audit-log.json` trae ceros en `security`?** Ese detalle fino solo se llena en modo *discovery* (repo remoto sin `project.json`). En modo proyecto, el resultado real del enmascarado está en `mova-budget-report.md`.

**¿Y si uso un servidor remoto (Oracle Cloud, AWS, propio)?** Nada cambia: `base_url` (en el `.json` del modelo, nunca en `project.json`) apunta a otra máquina. Lectura del repo, Sanitizer, PII Masking, Budget Gate y Circuit Breaker corren **siempre en la máquina que ejecuta el comando**. El servidor remoto solo recibe el payload final ya sanitizado y actúa como coprocesador *stateless*. Recomendación: enrutarlo por red privada (Tailscale, WireGuard, red virtual del proveedor), no por IP pública abierta.

---

## 3. Tokens y costo

**¿Cómo ayuda con el costo?** Tres mecanismos independientes, todos **antes** de la llamada:

| Mecanismo | Qué hace |
|---|---|
| Estimación local (`mova budget`, `context-trace`) | Cuenta tokens con `tiktoken-go` (`cl100k_base`, sin red) y estima USD por modelo con `config/prices.json`: `(tokens / unidad) × precio_de_entrada`. Escribe `mova-budget-report.md`. |
| Budget Gate (`max_tokens`) | Techo duro de contenido: si lo supera, se detiene antes de gastar y sugiere `focus`, menos agents/skills, etc. |
| Circuit breaker de gasto (`max_tokens_per_run` / `max_monthly_usd`) | Segundo techo, persistente entre corridas (`mova-spend.json`). |

**¿Qué pasa si excedo el presupuesto?** `on_exceed` en `project.json`: `warn` (avisa) o `abort` (alias `block`, corta). Ocurre antes del envío, no después.

**¿Cómo maneja Mova Context la precisión en el conteo de tokens, especialmente entre diferentes tokenizadores como Claude y GPT?** Mova utiliza un conteo local de tokens para estimar el consumo antes de ejecutar una llamada al modelo. Actualmente utiliza un tokenizador BPE embebido, sin dependencias de red, mediante tiktoken-go con la codificación cl100k_base.

Como cada proveedor y modelo puede utilizar un tokenizador diferente, el conteo local debe considerarse una estimación, no una medición universalmente exacta. Mova aborda esta diferencia mediante tres mecanismos:

Estimación local y determinista: El conteo se realiza localmente antes de la inferencia, lo que permite evaluar presupuestos y comparar variantes de contexto sin realizar una llamada al proveedor. cl100k_base proporciona una referencia reproducible, especialmente útil para modelos y flujos compatibles con esta codificación.
Comparación con el consumo real: Cuando se realiza una llamada a un proveedor que devuelve información de uso, Mova puede registrar el conteo real informado por la API y compararlo con la estimación local. Esta diferencia queda disponible en el historial de tokens (mova-token-history.json), permitiendo evaluar empíricamente qué tan cercana fue la estimación para una determinada carga de trabajo y proveedor.
Conteo sobre el contexto resultante: Las transformaciones de sanitización y reducción de contexto se aplican antes del conteo. De esta forma, el presupuesto se calcula sobre el contexto que Mova tiene previsto entregar al modelo, después de operaciones como eliminación de duplicados, limpieza de ruido y reemplazo de PII.

En resumen: el conteo local de Mova sirve para estimar y controlar el presupuesto antes de la ejecución; el conteo informado posteriormente por la API sirve para medir el consumo real. La diferencia entre ambos depende del tokenizador, modelo y contenido concreto, por lo que Mova no asume un porcentaje fijo de equivalencia entre proveedores.

**¿Qué es el Cache Layout Guard?** Ordena el prompt para aprovechar el *prompt caching* de Anthropic/OpenAI/Gemini (`cache_hint`, activo por defecto): **prefijo estático** primero (agents + skills + prompts base + reglas de `workflow.md`, idéntico byte a byte entre turnos) y **sufijo dinámico** después (`focus`, `memory.md`, turnos nuevos). Un hash del layout verifica que el prefijo no cambió y los deltas quedan en `mova-token-history.json`. En cargas repetitivas es el mayor ahorro disponible.

---

## 4. Flujo y arquitectura

**¿Cómo fluye todo, de punta a punta (`mova run` / `mova chat`)?**

| # | Paso | Dónde se corta |
|---|---|---|
| 1 | Resolver proyecto (`project.json`, agents/skills/prompts) | |
| 2 | Ensamblar contexto: agents + skills + prompt + memoria + focus | |
| 3 | Sanitizar (dedupe, logs, comentarios/blancos) | |
| 4 | PII Masking (opcional) | |
| 5 | Budget Gate (`max_tokens`) | excede → se detiene, nada sale |
| 6 | Circuit breaker de gasto | `abort` → se detiene |
| 7 | Envío al modelo (`base_url` local o remoto) o, con `dry_run`, solo evidencia | |
| 8 | Feedback loop: tokens reales → `mova-token-history.json` (solo cloud) | |

Los pasos 1–6 ocurren **siempre en tu máquina**. `mova run` arma el contexto, genera grafos/diagramas y no llama al modelo; `mova chat` además envía.

**¿Qué arquitectura usa y por qué?** Hexagonal pragmática (*Ports & Adapters*). El núcleo `core/` depende solo de la librería estándar y razona sobre un puerto (`core.Adapter`). Adaptadores de almacenamiento: `FileAdapter` (por defecto) y `DBAdapter` (**PostgreSQL implementado; MongoDB es un stub**). CLI, Chat, MCP y HTTP son cuatro entradas delgadas sobre las mismas funciones (`http/server.go` envuelve `mcp.Process()`; no hay una segunda implementación). Razones: núcleo portable, mismo comportamiento en las cuatro puertas, y extensión sin tocar el motor (adaptadores, focus resolvers, proveedores, save writers; ver [`SOURCE.md`](SOURCE.md)).

**¿Soporta concurrencia?**
- **HTTP/MCP:** una goroutine por request, acotada por un semáforo (`MOVA_HTTP_MAX_CONCURRENCY`; por defecto 4×CPU, mínimo 8, máximo 64) y timeouts de lectura/escritura.
- **Multiagente:** los agentes de un grupo corren en paralelo con un worker pool (`MOVA_MAX_CONCURRENCY`; por defecto los núcleos, tope 8).
- **Estado compartido:** `mova-token-history.json`, `mova-spend.json` y `mova-context-cache.json` se serializan con un mutex por ruta, así dos llamadas simultáneas no pierden actualizaciones.

**¿Puedo personalizar los diagramas?** Todo lo que se dibuja sale de `project.json`: solo los agentes, fuentes, reglas de privacidad y métricas **realmente configurados y ejecutados**. Cada agente tiene su propio contexto, así que un mismo grupo puede mezclar un flujo local (Ollama) y uno cloud.

---

## 5. context-trace y ranking

**¿Para qué sirve `--task` y qué es el «Top»?** `Top-ranked files` lista los archivos más relevantes **para tu tarea**, de mayor a menor `score`. Usa BM25 (la familia de los buscadores) sobre el contenido, más un pequeño impulso estructural si un import/declaración o nombre de archivo coincide con tu texto. Es **léxico, no semántico**: compara palabras, no significado.

**¿Cómo lo interpreto?**
- Un solo score muy alto y el resto parejo y bajo = una coincidencia fuerte y las demás débiles. *Ejemplo real (FastAPI): `fastapi/dependencies/utils.py` 1019,04; los 9 siguientes, 3,00.*
- Muchos scores bajos y parecidos = tarea vaga: afínala y revisa a mano. Regla práctica: >15 en un repo típico es coincidencia fuerte.
- Si el Top se llena de **documentación traducida**, filtra idiomas: `--ignore "docs/!(en)/**"`. El motor ya deduplica traducciones (queda la variante mejor puntuada).
- Si aparecen **tests poco relacionados**, es un límite conocido de BM25 (una palabra suelta como «error» basta). Usa tareas específicas (`"fix OAuth2PasswordBearer token authentication"`, no `"fix bug"`).

**¿`--ignore` afecta «Discovered»?** Sí, a propósito: lo ignorado nunca se lee ni se tokeniza. `--task` sí lee todo y luego prioriza. Para aislar el efecto de la relevancia, compara con y sin `--ignore`.

**¿Por qué `context-trace --repo` queda esperando?** Al final pregunta `Generate a 'project.json'? [Y/n]` por stdin. En scripts/CI añade `< /dev/null`.

---

## 6. Chat y archivos

**¿El chat siempre crea/edita/borra archivos con solo pedirlo?** Con modelos cloud (Claude, GPT, Gemini), de forma fiable. Con modelos locales pequeños (p. ej. un Qwen 3B en LM Studio) no siempre: a veces responden «no puedo crear archivos» en vez de invocar `apply_file_changes`, aun con la instrucción reforzada de `mova chat`. No es del motor sino del modelo: prueba uno más grande o pide explícitamente «usa `apply_file_changes` para crear…».

**¿Funciona igual en todas las puertas?** Mismo motor. El CLI pide confirmación interactiva (menú o y/n); MCP y HTTP, sin terminal, **devuelven la propuesta como texto y exigen una segunda llamada explícita** (p. ej. `delete_path` con `confirm:"true"`); nunca aplican solos. Referencia: [`FUNCTIONS.md`](FUNCTIONS.md), [`PROJECT_JSON.md`](PROJECT_JSON.md) (`apply`).

**¿Quién escribe `memory.md`?** `mova chat` (con `"memory": true`) y la tool `save_memory`. **`mova run` no la escribe.**

---

## 7. Instalación y Windows

**¿Cómo instalo y desinstalo?** `make install` (requiere Go ≥ 1.24) o `installers/<tu SO>/…`. Para quitar binario, `PATH` y `MOVA_PROJECT_ROOT`: [`uninstallers/`](../../../uninstallers/README.md) (probado en Linux; macOS/Windows sin ejecutar: usa `--dry-run`).

**Corrección heredada (rutas de Windows):** un `"repo"` absoluto con letra de unidad (`E:\\…`) podía fallar al resolver `"focus": ["."]` y `"exclude"` si la letra variaba en mayúscula/minúscula. Corregido según el FAQ anterior; no re-verificado en esta revisión.

**¿Dónde está lo verificado y lo no verificado?** [`docs/VERIFICATION.md`](../../VERIFICATION.md).
