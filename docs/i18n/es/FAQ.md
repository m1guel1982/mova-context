## Preguntas Frecuentes (FAQ)

### 1. ¿Cómo maneja Mova Context la precisión en el conteo de tokens, especialmente entre diferentes tokenizadores como Claude y GPT?

El conteo local de tokens utiliza un tokenizador BPE embebido y sin dependencias de red a través de `tiktoken-go` bajo la codificación `cl100k_base` (el estándar de GPT-4). Las diferencias entre proveedores se resuelven mediante tres pilares de diseño:

* **Varianza mínima en la práctica:** Aunque Anthropic y OpenAI utilizan codificaciones de vocabulario distintas, los algoritmos BPE modernos dividen el código, esquemas JSON y texto técnico con proporciones de caracteres por token casi idénticas. En cargas de trabajo reales, la variación entre `cl100k_base` y el tokenizador de Claude se mantiene consistentemente por debajo del 3–5%, siendo más que suficiente para validar límites de presupuesto pre-ejecución.
* **Bucle de retroalimentación de la API:** Mova no depende únicamente de la estimación local. Cada vez que se completa una llamada al modelo, lee el conteo real de `prompt_tokens` devuelto por la API del proveedor (Anthropic, OpenAI, Gemini) y registra la diferencia en `mova-token-history.json`. Esto calibra las estimaciones futuras contra datos reales sin sobrecargar el binario con dependencias específicas para cada tokenizador.
* **Sanitización previa al conteo:** Antes de contar tokens, el Token Firewall colapsa espacios duplicados, secuencias de logs y estandariza los reemplazos de PII (`[PII_a1b2c3d4]`). Limpiar el ruido estructural primero garantiza que el conteo de tokens sea estable y predecible sin importar el modelo final.

---

### 2. ¿Es posible personalizar la información que aparece en los diagramas de arquitectura o se muestra todo por defecto?

El motor de diagramación es totalmente dinámico y se personaliza según la configuración del proyecto:

* **Estructura basada en configuración:** Todo lo que se grafica en el diagrama está impulsado directamente por los archivos de configuración (`project.json`) definidos en la estructura del proyecto.
* **Renderizado bajo demanda:** El motor lee las definiciones al momento de la ejecución y renderiza de forma limpia únicamente los agentes, fuentes de datos, reglas de privacidad y métricas que fueron efectivamente configurados y ejecutados para esa canalización específica.
* **Modularidad:** Cada agente mantiene su propia configuración de contexto, permitiendo aislar flujos locales (`llama3.2:3b` vía Ollama) de flujos en la nube (`gemini-3-flash-preview`) dentro del mismo grupo de ejecución.



### 3. ¿Cómo funciona el Cache Layout Guard y cómo evita que se rompa el "Prompt Caching" en conversaciones dinámicas?

El Cache Layout Guard está diseñado para maximizar el uso de *prompt caching* en proveedores como Anthropic, OpenAI y Gemini al estructurar el prompt en dos secciones claramente delimitadas desde el byte 0:

* **Prefijo Estático (Offset 0):** Agrupa las piezas permanentes del contexto (`agents` + `skills` + prompts base + reglas de `workflow.md`). Mova garantiza que este bloque superior se mantenga idéntico byte por byte entre ejecuciones y turnos del chat. Dado que los proveedores evalúan el caché de arriba hacia abajo (*exact prefix matching*), mantener este bloque pesado al inicio evita la invalidación del caché.
* **Sufijo Dinámico:** Toda la información cambiante —como la resolución de `focus` (archivos específicos), el historial de `memory.md` y los nuevos turnos de la conversación— se acopla exclusivamente **después** del bloque estático.
* **Verificación de Integridad:** Mova genera un hash del estado del layout para verificar que el prefijo estático no haya sufrido alteraciones entre llamadas, registrando los deltas de tokens en `mova-token-history.json` sin comprometer la velocidad ni la tasa de acierto del caché (*cache hit ratio*).

---

### 4. ¿Qué tipo de herramienta es Mova Context, exactamente? ¿Un framework, un agente, un CLI?

Ninguna de esas categorías por sí sola lo describe bien — es más preciso pensarlo en dos capas separables:

* **Capa 1 — Convención de archivos (siempre disponible, cero dependencias):** `workflow.md` + `agents/` + `skills/` + `prompts/` + `project.json` + `memory.md`. Esto es, en esencia, un **formato de workflow/prompt estructurado en Markdown**. Cualquier agente de código que ya sepa leer archivos (Claude Code, Cursor, un simple copy-paste a un chat web) puede seguirlo sin instalar nada — en ese sentido, funciona como **workflow**, como **definición de agente**, como **skill**, o como **prompt** reutilizable, según cómo se lo use.
* **Capa 2 — Motor en Go (opcional):** un binario que automatiza trabajar con esa misma convención — arma el contexto, lo audita, lo sanitiza, estima su costo, y opcionalmente lo envía a un modelo (local o Cloud) por CLI, Chat, MCP o HTTP. Esta capa es la que agrega gobernanza de contexto, presupuesto de tokens, y control de inferencia — pero nunca reemplaza ni oculta la Capa 1.

Dicho de otra forma: Mova Context es, primero, una **convención liviana**; y, en segundo lugar y de forma opcional, un **motor de gobernanza y optimización de contexto para LLMs** construido sobre esa convención. No es un framework de agentes (no orquesta razonamiento ni decide qué hacer por el modelo — eso es trabajo del LLM), ni un simple CLI de utilidades (mantiene estado, presupuesto y protección de datos entre llamadas).

---

### 5. ¿Qué arquitectura y qué patrón de software se usó para construirlo, y por qué?

El motor en Go sigue **Arquitectura Hexagonal (Ports & Adapters)**, con un núcleo (`core/`) que no depende de nada más que la librería estándar de Go:

* **El puerto:** la interfaz `core.Adapter` (`GetProject`, `GetKnowledge`, `Search`, `AppendMemory`, ...) — el motor entero razona en términos de este contrato, nunca de cómo se implementa.
* **Los adaptadores:** `core.FileAdapter` (lectura de archivos — el caso por defecto) y `adapters.DBAdapter` (Postgres/MongoDB) implementan el mismo puerto de forma intercambiable. Agregar un tercer backend de almacenamiento es implementar la interfaz una vez, sin tocar ni una línea del motor.
* **Los mismos "puertos de entrada" para los cuatro doors:** CLI, Chat, MCP y HTTP no son cuatro implementaciones distintas de la lógica de negocio — son cuatro adaptadores de entrada delgados que llaman exactamente a las mismas funciones (`budget.BuildGatedContext`, `mcp.Process`, etc.). `http/server.go` es, literalmente, un envoltorio delgado sobre `mcp.Process()` — no hay una segunda implementación del protocolo.

¿Por qué este patrón y no, por ejemplo, MVC o Clean Architecture? Tres razones concretas:

1. **El dominio (`core/`) tiene que ser trivialmente portable.** Al depender solo de la librería estándar, el núcleo nunca se acopla a un formato de almacenamiento, a un proveedor de modelo, ni a un transporte — eso es justo lo que Ports & Adapters garantiza por diseño.
2. **Cuatro puntos de entrada, una sola lógica.** El requisito de "mismo comportamiento en CLI/Chat/MCP/HTTP" es exactamente el problema que Ports & Adapters resuelve: los puertos de entrada (adaptadores primarios) son intercambiables sin duplicar reglas de negocio.
3. **Extensibilidad sin romper nada existente.** Cada "Extension Point" documentado en `SOURCE.md` (Adapters de almacenamiento, Focus Resolvers, Model Providers, Save Writers) es, en el fondo, la misma idea aplicada en distintos puntos: definir el contrato una vez, permitir múltiples implementaciones intercambiables detrás de él.

En la práctica esto es una variante pragmática de Hexagonal/Ports & Adapters — no una implementación académica estricta con capas de "entities/use-cases/interface-adapters" nombradas explícitamente como en Clean Architecture, sino su mismo principio central (el dominio no conoce el mundo exterior; el mundo exterior se adapta al dominio) aplicado con el mínimo de ceremonia necesaria para un binario Go de un solo módulo.

---

### 6. ¿Qué significa "gobernanza de contexto" en Mova Context, concretamente?

Gobernanza de contexto es el conjunto de reglas y mecanismos que deciden **qué información entra al contexto que se le envía a un modelo, en qué forma, y con qué controles** — antes de que ese contexto salga de la máquina del usuario:

* **Qué entra:** `focus` (archivos/directorios/símbolos específicos, en vez de todo el repositorio), agents/skills/prompts explícitamente declarados en `project.json` — nunca una inclusión implícita de "todo lo que haya en el disco".
* **En qué forma:** el Sanitizer (deduplicación de párrafos exactos, colapso de logs repetidos, remoción opcional de comentarios/líneas en blanco) reduce ruido estructural antes de contar tokens o enviar nada.
* **Con qué controles:** PII Masking (opcional, algoritmo estructural — forma de palabra + entropía de Shannon, ver `config/policy.json`), el Budget Gate (`max_tokens` como techo duro, ver `budget.EnforceLimit`), y el Circuit Breaker de gasto (`max_tokens_per_run` / `max_monthly_usd`, ver `budget/spend.go`).

Todo esto corre en la Capa 2 (el motor), siempre en la máquina que arma el contexto — nunca en el servidor remoto que eventualmente recibe el resultado ya sanitizado (ver pregunta 12 y `PROJECT_JSON.md § Arquitectura distribuida`).

---

### 7. ¿Cómo protege Mova Context la información personal (PII) antes de que llegue a un modelo externo?

El PII Masking (opcional, `budget.pii_masking.enabled`) es **estructural, no basado en diccionarios ni reglas de idioma**: analiza la forma del token (proporción de dígitos, separadores, símbolos, longitud, mayúsculas sostenidas) y su entropía de Shannon, combinando ambos en un puntaje (`min_score` en `config/policy.json`, 0.62 por defecto). Un token que supera ese umbral se reemplaza por una etiqueta determinística (`[PII_a1b2c3d4]`) antes de que el contexto salga hacia cualquier modelo, local o Cloud.

Esto es una herramienta técnica, no un reemplazo de una política de privacidad o una revisión legal — el disclaimer completo está en `COMMANDS.md § PII Masking`. Su valor concreto es hacer visible y auditable, con un número (`N token(s) pseudonymized`), qué tanto de lo que se envía tiene forma de dato personal — algo que hoy, en la mayoría de integraciones de LLM, queda invisible dentro de una llamada a una API.

---

### 8. ¿Cómo ayuda Mova Context a controlar costos y optimizar el presupuesto de tokens?

El "Token Firewall" es la combinación de tres mecanismos independientes, todos corriendo **antes** de la llamada real al modelo:

* **Estimación local** (`mova budget`): cuenta tokens con un tokenizador BPE embebido (`tiktoken-go`, `cl100k_base`), sin llamada de red, y calcula el costo aproximado contra `config/prices.json` para cada modelo configurado — permitiendo comparar antes de decidir cuál usar.
* **Budget Gate** (`max_tokens`): un techo duro de contenido — si el contexto ensamblado lo supera, la ejecución se detiene antes de gastar nada, con una sugerencia concreta (activar `focus`, reducir agents/skills, etc.).
* **Circuit Breaker de gasto** (`max_tokens_per_run` / `max_monthly_usd`): un segundo techo, independiente del anterior, que sí persiste entre corridas — corta o advierte cuando el gasto mensual acumulado de un proyecto se acerca a un límite en dólares.

Además, el **Feedback Loop** (`mova-token-history.json`) compara la estimación local contra el conteo real que cada proveedor Cloud devuelve, calibrando la precisión de la estimación con el tiempo — y el **Cache Layout Guard** (ver pregunta 3) maximiza el aprovechamiento del *prompt caching* nativo de cada proveedor, que es, en la práctica, el mayor ahorro de costo disponible en cargas de trabajo repetitivas.

---

### 9. ¿Cómo funciona el flujo de datos completo, de punta a punta, cuando se ejecuta `mova run` o `mova chat`?

1. **Resolución de proyecto** (`runtime.FindRoot` + `core.Adapter.GetProject`): se localiza `project.json` y se cargan sus referencias a agents/skills/prompts.
2. **Ensamblado** (`core.BuildContext` / Focus Resolvers): se arma el contexto crudo — agents + skills + prompt + memoria + focus (si está configurado).
3. **Sanitización** (`sanitize.Apply`, opcionalmente cacheada por `budget.SanitizeCached`): deduplicación, colapso de logs, remoción opcional de comentarios/blancos.
4. **PII Masking** (opcional): reemplazo estructural de tokens con forma de dato personal.
5. **Budget Gate**: verificación contra `max_tokens`; si excede, se detiene aquí — nada se envía todavía.
6. **Circuit Breaker de gasto**: verificación contra los techos de `budget.spend`; si `on_exceed: "abort"` y se excede, se detiene aquí también.
7. **Envío** (`models.Session.Send`/`SendStream`): recién en este paso el payload final, ya sanitizado, sale de la máquina hacia `base_url` (local o remoto).
8. **Feedback Loop**: si el proveedor es Cloud, el conteo real de tokens que devuelve se registra en `mova-token-history.json` para calibrar futuras estimaciones.

Los pasos 1 a 6 ocurren **siempre** en la máquina que ejecuta el comando — nunca en un servidor remoto, incluso cuando el paso 7 apunta a uno (ver pregunta 12).

---

### 10. ¿Cómo se calcula el "presupuesto" (`mova budget`) exactamente? ¿Qué número da y qué tan confiable es?

`mova budget <proyecto> [tarea]` arma el mismo contexto que `mova run` (pasos 1 a 4 de la pregunta 9), cuenta sus tokens con `tiktoken-go` bajo `cl100k_base`, y calcula el costo estimado en USD para cada modelo configurado en `config/prices.json`, usando la fórmula `(tokens / divisor_de_unidad) * precio_de_entrada`. El resultado se escribe en `mova-budget-report.md` junto con un aviso obligatorio: es una **estimación**, no el costo exacto que cobrará cada proveedor — la variación real observada contra distintos tokenizadores de proveedores Cloud se mantiene, en la práctica, por debajo del 3–5% (ver pregunta 1), suficiente para decisiones de presupuesto pre-ejecución pero no para facturación exacta.

---


### 11. ¿Qué pasa con mis datos si uso un servidor remoto centralizado (Oracle Cloud, AWS, un servidor propio)?

Nada de esto cambia por usar un endpoint remoto — es la misma separación de responsabilidades de siempre, solo que `base_url` (en el `.json` del modelo, nunca en `project.json`) apunta a otra máquina en vez de `localhost`:

* La lectura del repositorio, el Sanitizer, el PII Masking, el Budget Gate y el Circuit Breaker corren **siempre** en la máquina que ejecuta el comando — el servidor remoto nunca los ejecuta, porque nunca recibe el repositorio ni `project.json`.
* El servidor remoto solo recibe el payload final ya sanitizado, actúa como coprocesador de inferencia *stateless* (no guarda nada del contenido del proyecto), y devuelve la respuesta del modelo.
* Se recomienda enrutar ese tráfico por una red privada (Tailscale, WireGuard, o la red virtual del proveedor Cloud) en vez de una IP pública sin protección — ver `DEPLOY.md § Seguridad de red` para el detalle completo.

Ver también `PROJECT_JSON.md § Arquitectura distribuida (endpoints remotos)` para la tabla completa de qué corre dónde.

---

### 12. ¿Cómo soporta Mova Context alta concurrencia (múltiples usuarios/llamadas al mismo tiempo)?

El motor está preparado para servir múltiples invocaciones simultáneas desde cualquiera de sus cuatro puertas:

* **HTTP/MCP** (`http/server.go`): cada request corre en su propia goroutine, acotada por un semáforo configurable (`MOVA_HTTP_MAX_CONCURRENCY`, por defecto 4× núcleos de CPU) para evitar consumo de recursos sin límite bajo ráfagas de tráfico, más timeouts de lectura/escritura para que un cliente lento no retenga un slot indefinidamente.
* **Multiagente** (`orchestrator.RunGroup`): los agentes de un mismo grupo corren en paralelo a través de un *worker pool* acotado (`MOVA_MAX_CONCURRENCY`, por defecto los núcleos de CPU disponibles, con techo de 8), en vez de uno por uno.
* **Estado compartido protegido:** `mova-token-history.json`, `mova-spend.json` y `mova-context-cache.json` — los tres archivos que distintas invocaciones concurrentes del mismo proyecto podrían leer/escribir al mismo tiempo — están serializados con un mutex por-ruta (ver `budget/filelock.go`), así que dos llamadas concurrentes nunca pueden pisarse una a la otra y perder una actualización de gasto o de historial.

Esto es lo que hace viable el despliegue centralizado de la pregunta 12: una sola instancia puede atender a un equipo completo sin condiciones de carrera en su propio estado interno.
