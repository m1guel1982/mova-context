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