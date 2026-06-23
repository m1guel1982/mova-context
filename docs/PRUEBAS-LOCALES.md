# Guía de pruebas — Validar Mova Context en local

Esta guía valida el concepto completo de Mova Context **sin tocar ningún proyecto real**, usando un proyecto ficticio (`pruebas-locales`) que ya viene incluido en este repo. Sirve para cualquier persona que quiera probar de cero, en cualquier sistema operativo (Windows, macOS, Linux) y con cualquier herramienta de IA.

Esta guía usa Cline como ejemplo concreto porque es con lo que se probó primero, pero el procedimiento es el mismo sin importar qué herramienta uses — ver sección 2.5 para los detalles.

---

## 0. Qué vas a probar

| Prueba | Qué demuestra |
|---|---|
| 1. Crear proyecto nuevo | `workflow.md` resuelve `project.json` y arma un proyecto desde un prompt base |
| 2. Modificar proyecto existente | Se reutiliza contexto ya cargado (agentes/skills) sobre código que ya existe |
| 3. DevOps | Un agente y prompt distintos (`devops-engineer`, `cicd-setup`) se cargan sin tocar los anteriores |
| 4. QA | Una skill (`generate-tests`) se combina con un agente (`qa-engineer`) sobre el mismo proyecto |
| 5. Ponytail | El modo minimalista se activa como prompt opt-in, sin romper nada de lo anterior |
| 6. Memoria | `memory.md` se crea solo, se lee en la sesión siguiente, y el modelo retoma sin que tengas que reexplicar nada |

Todo esto se hace sobre **un solo proyecto ficticio**: una API mínima de tareas (to-do list). No es un proyecto real, es el mínimo necesario para que las pruebas sean rápidas y verificables.

---

## 1. Requisitos

- Una herramienta de IA con acceso a archivos del proyecto (esta guía usa Cline + Gemini Flash como ejemplo — ver sección 2.5 si usas otra)
- Node.js 18+ instalado (para correr el proyecto ficticio que se va a generar)
- Este repositorio clonado o descargado en tu máquina

No necesitas Docker ni ninguna infraestructura adicional.

---

## 2. Instalar y configurar Cline con Gemini Flash (ejemplo usado en esta guía)

### 2.1 Obtener una API key de Gemini

1. Ve a [Google AI Studio](https://aistudio.google.com/apikey)
2. Inicia sesión con tu cuenta de Google
3. Haz clic en **"Create API Key"**
4. Copia la key — la vas a necesitar en el paso siguiente

### 2.2 Instalar Cline en VS Code

1. Abre VS Code
2. Ve a la pestaña de Extensiones (ícono de cuadrados en la barra lateral)
3. Busca **"Cline"**
4. Instala la extensión oficial (publisher: `saoudrizwan`)

### 2.3 Conectar Cline con Gemini Flash

1. Abre el panel de Cline (ícono en la barra lateral de VS Code)
2. Haz clic en el ícono de configuración (⚙️)
3. En **"API Provider"** selecciona **"Google Gemini"**
4. Pega tu API key en el campo **"Gemini API Key"**
5. En **"Model"** selecciona el modelo Flash disponible (el nombre exacto cambia con el tiempo — elige el que diga "flash" en la lista)
6. Guarda

### 2.4 Verificar que funciona

Abre el chat de Cline y escribe:

```text
Hola, responde solo "Cline + Gemini Flash funcionando"
```

Si responde, ya tienes todo listo para empezar las pruebas.

### 2.5 ¿Y si uso Claude, Codex, otro proveedor en Cline, o un LLM local?

Da exactamente igual. Mova Context no depende de Cline ni de Gemini en ningún punto — depende de dos cosas únicamente:

1. Que la herramienta pueda **leer archivos del proyecto** (`workflow.md`, `project.json`, los `.md` de agents/skills/prompts)
2. Que pueda **seguir instrucciones en lenguaje natural** del tipo `Lee workflow.md → pruebas-locales → crear-proyecto`

Eso lo cumplen por igual:

- **Claude** (claude.ai con archivos subidos, Claude Code, Claude Cowork)
- **Codex** o cualquier asistente dentro de un IDE que lea el repo
- **Cline con otro proveedor** (Claude, GPT, OpenRouter) — solo cambia el paso 2.3, seleccionas otro "API Provider"
- **Un LLM local** (Ollama, LM Studio u otro) siempre que la herramienta que lo envuelve pueda leer archivos del proyecto

Si tu herramienta **no tiene acceso a archivos** (por ejemplo, un chat web donde no puedes subir ni apuntar a una carpeta), `workflow.md` ya contempla ese caso: en vez de leer automáticamente, te va a pedir que pegues el contenido de los archivos que necesita. Si te pasa eso durante las pruebas, simplemente copia y pega el archivo que te pida — el resultado es el mismo, solo cambia quién mueve el archivo.

A partir de la sección 3, todo el resto de esta guía es exactamente igual sin importar qué herramienta hayas elegido aquí.

---

## 3. Preparar el espacio de pruebas

1. Abre en VS Code la carpeta raíz de este repositorio (la que contiene `workflow.md`, `agents/`, `skills/`, `prompts/`, `projects/`)
2. Confirma que existe `projects/pruebas-locales/project.json` — es el proyecto ficticio que vas a usar en todas las pruebas
3. No necesitas crear ni editar nada todavía

Estructura que deberías ver:

```text
mova-context/
├── agents/
├── skills/
├── prompts/
├── projects/
│   └── pruebas-locales/
│       └── project.json
└── workflow.md
```

---

## 4. Prueba 1 — Crear proyecto nuevo

**Objetivo:** confirmar que Cline/Gemini lee `workflow.md`, resuelve `project.json` de `pruebas-locales`, carga el prompt `create-project` y genera la estructura mínima de una API de tareas.

**Qué vas a pedir en el chat de Cline:**

```text
Lee workflow.md → pruebas-locales → crear-proyecto
```

**Qué debería pasar (resultado esperado):**

1. El modelo busca y lee `workflow.md`
2. Localiza `projects/pruebas-locales/project.json`
3. Resuelve la task `crear-proyecto` (carga el prompt base `create-project`, los agentes `yagni-core` + `backend-dev`, las skills `kiss-dry-core` + `lazy-minimalism`)
4. Genera: estructura de carpetas, 3 endpoints (crear/listar/completar tarea), sin base de datos pesada (usa SQLite o un archivo JSON, según lo que el agente decida con criterio YAGNI)
5. **No debería** crear autenticación compleja, no debería crear microservicios, no debería agregar Docker si no se lo pediste — eso sería una señal de que YAGNI no se está aplicando

**Cómo verificar que funcionó:**
- [ ] Existen los archivos de código generados
- [ ] El proyecto corre con `node` sin pasos extra no solicitados
- [ ] No hay carpetas ni abstracciones que tú no pediste explícitamente

**Cómo probarlo de verdad (no solo mirar el código):**

1. Pídele a la herramienta el comando exacto para arrancar el servidor (algo como `node server.js` o `npm start`, según lo que haya generado) y ejecútalo en una terminal
2. Con el servidor corriendo, en otra terminal prueba los 3 endpoints generados, por ejemplo:

```text
curl -X POST http://localhost:3000/api/v1/tareas -H "Content-Type: application/json" -d "{\"titulo\":\"Probar Mova Context\"}"
curl http://localhost:3000/api/v1/tareas
curl -X PATCH http://localhost:3000/api/v1/tareas/1/completar
```

(Ajusta el puerto y la ruta exacta a lo que el modelo haya generado — pídeselo si no lo sabes: "¿qué endpoints creaste y en qué puerto corre?")

3. Si las 3 llamadas responden sin error y la tarea creada aparece al listar, la Prueba 1 quedó validada de extremo a extremo, no solo "el código se ve bien"

**Qué anotar:** si el modelo generó algo no pedido (ej. un sistema de logging elaborado), anótalo — es la métrica de qué tan bien se está aplicando YAGNI en la práctica.

---

## 5. Prueba 2 — Modificar proyecto existente

**Objetivo:** confirmar que, sobre el código ya generado en la Prueba 1, el modelo puede agregar un módulo nuevo reutilizando el mismo contexto, sin que tengas que re-explicar el stack.

**Qué vas a pedir:**

```text
Lee workflow.md → pruebas-locales → agregar-modulo-categorias
```

**Qué debería pasar:**

1. Carga el prompt `create-module` con la variable `module = categorias`
2. Agrega un módulo de categorías a la API existente, siguiendo el mismo patrón (Service + Repository) que ya estableció el agente `backend-dev` en la Prueba 1
3. No debería reescribir ni romper el módulo de tareas ya creado

**Cómo verificar que funcionó:**
- [ ] El módulo de tareas (Prueba 1) sigue funcionando igual
- [ ] El módulo nuevo de categorías sigue la misma convención de carpetas que el anterior
- [ ] No tuviste que volver a explicar el stack ni las convenciones — el modelo las dedujo del código ya existente y del agente cargado

**Cómo probarlo de verdad:**

1. Reinicia el servidor (para cargar el módulo nuevo) y repite las llamadas de la Prueba 1 — deben seguir funcionando igual que antes
2. Prueba el endpoint nuevo de categorías, por ejemplo:

```text
curl -X POST http://localhost:3000/api/v1/categorias -H "Content-Type: application/json" -d "{\"nombre\":\"Trabajo\"}"
curl http://localhost:3000/api/v1/categorias
```

3. Si ambos módulos (tareas y categorías) responden bien al mismo tiempo, sin que tocar uno haya roto el otro, la Prueba 2 quedó validada

---

## 6. Prueba 3 — DevOps

**Objetivo:** confirmar que un agente distinto (`devops-engineer`) y un prompt distinto (`cicd-setup`) se cargan correctamente sin interferir con lo anterior, y que las variables del `project.json` (`ci_provider`, `cloud_provider`) se inyectan bien.

**Qué vas a pedir:**

```text
Lee workflow.md → pruebas-locales → configurar-cicd
```

**Qué debería pasar:**

1. Carga el agente `devops-engineer` (que no se usó en las pruebas 1 y 2)
2. Genera un pipeline simple (lint → test → build) usando `{{CI_PROVIDER}}` = GitHub Actions, tomado de las variables del proyecto
3. No debería pedir credenciales de la nube (`cloud_provider` está configurado como "ninguno" en `project.json` a propósito, para que el agente lo respete)

**Cómo verificar que funcionó:**
- [ ] El pipeline generado menciona GitHub Actions, no otro proveedor inventado
- [ ] No incluye despliegue a una nube específica (porque `cloud_provider: "ninguno"` se lo dijo explícitamente)
- [ ] Esto demuestra que las variables de `project.json` sí llegan al prompt

**Cómo probarlo de verdad:**

1. Si tienes el repo en GitHub (aunque sea uno de prueba, vacío), copia el archivo de pipeline generado a `.github/workflows/`, súbelo, y revisa en la pestaña "Actions" de GitHub si corre sin errores
2. Si no quieres usar GitHub para esta prueba, al menos ejecuta a mano, en tu terminal, los mismos pasos que el pipeline dice que hace (`npm install`, el comando de lint, el comando de test) y confirma que cada uno termina sin error — eso es lo que el pipeline automatizaría

---

## 7. Prueba 4 — QA

**Objetivo:** confirmar que un agente (`qa-engineer`) y una skill (`generate-tests`) se combinan correctamente sobre el mismo proyecto.

**Qué vas a pedir:**

```text
Lee workflow.md → pruebas-locales → generar-tests
```

**Qué debería pasar:**

1. Carga el agente `qa-engineer` + la skill `generate-tests`
2. Genera tests para el módulo de tareas: caso exitoso, input inválido, caso límite, error del sistema (según la "Cobertura mínima" que define la skill)
3. Los tests deberían poder correr con `{{TEST_FRAMEWORK}}` = `node:test` (definido en `project.json`)

**Cómo verificar que funcionó:**
- [ ] Los tests cubren al menos: caso exitoso + caso inválido + caso límite (lee la skill `generate-tests` y compara contra lo que efectivamente entregó)
- [ ] Los tests corren sin instalar un framework externo no solicitado
- [ ] Ningún test "siempre pasa" (anti-patrón explícito en el agente `qa-engineer`)

**Cómo probarlo de verdad:**

1. Corre los tests generados:

```text
node --test
```

2. Confirma que todos pasan en verde
3. Rompe algo a propósito (por ejemplo, comenta una línea de validación en el módulo de tareas) y vuelve a correr `node --test` — al menos un test debería fallar. Si nada falla, los tests no estaban probando lo que decían probar

---

## 8. Prueba 5 — Ponytail (modo minimalista)

**Objetivo:** confirmar que activar `ponytail` como prompt custom no rompe nada de lo anterior, y que el modelo marca explícitamente sus simplificaciones.

**Qué vas a pedir:**

```text
Lee workflow.md → pruebas-locales → refactor-ponytail
Aplica esto al módulo de categorías: simplifica si hay algo que no se necesita.
```

**Qué debería pasar:**

1. El modelo revisa el módulo de categorías con la escalera de decisión de `lazy-minimalism` (¿hace falta? ¿lo cubre la librería estándar? etc.)
2. Si simplifica algo, debería marcarlo con un comentario `# lazy:` indicando el límite del atajo
3. No debería eliminar validación de entrada, manejo de errores ni nada de la lista de "no es perezoso en" — si lo hace, es una falla a anotar

**Cómo verificar que funcionó:**
- [ ] Si hubo recorte, aparece el comentario `# lazy:` con el límite explicado
- [ ] La validación de entrada sigue intacta
- [ ] El código quedó más corto o igual, nunca más largo

---

## 9. Prueba 6 — Memoria (la más importante)

**Objetivo:** confirmar el beneficio central de Mova Context — que no tengas que reexplicar contexto en la sesión siguiente.

### Parte A — Generar memoria

Al final de cualquiera de las pruebas anteriores, pide:

```text
Actualiza memory.md de pruebas-locales con lo que hicimos en esta sesión.
```

**Qué debería pasar:** se crea (o actualiza) `projects/pruebas-locales/memory.md` siguiendo el formato que define `workflow.md`:

```md
## YYYY-MM-DD — sesión
**Hecho:**
**Resuelto:**
**Pendiente:**
**Decisiones:**
**Errores LLM:**
```

### Parte B — Reutilizar memoria (cierra la sesión y abre una nueva)

1. Cierra por completo la sesión actual (cierra el chat, o cierra VS Code y vuelve a abrirlo, según tu herramienta)
2. Abre una conversación nueva
3. Pide:

```text
Lee workflow.md → pruebas-locales
¿En qué quedamos la última vez?
```

**Qué debería pasar:** el modelo lee `memory.md` y responde con lo que se hizo, **sin que tú hayas tenido que volver a explicar nada del proyecto**.

**Cómo verificar que funcionó (la prueba más importante de todas):**
- [ ] El modelo menciona correctamente lo que se hizo en la sesión anterior
- [ ] No tuviste que volver a pegar contexto del proyecto
- [ ] Compara cuánto tuviste que escribir tú en esta prueba vs. cuánto hubieras tenido que escribir explicando todo desde cero — esa diferencia es el ahorro real (ver guía de medición de tokens más abajo)

---

## 10. Cómo medir ahorro de tokens y precisión con estas mismas pruebas

Usa las 6 pruebas de arriba para sacar un número real, no estimado:

1. Para cada prueba, cuenta cuántas palabras/líneas tuviste que escribir tú en el chat (eso es tu input real)
2. Compara contra cuánto habrías tenido que escribir explicando el mismo contexto sin Mova Context (stack, convenciones, qué se hizo antes)
3. Para precisión: marca cada prueba como ✅ (resultado correcto a la primera) o ❌ (tuviste que corregir)

```text
Ahorro aproximado = (palabras que hubieras escrito sin Mova Context) − (palabras que escribiste con Mova Context)
Precisión = pruebas ✅ / 6
```

Repite las 6 pruebas 2-3 veces en sesiones distintas para que el número no sea de una sola muestra.

---

## 11. Checklist final — resumen de todo lo validado

- [ ] Prueba 1 — Crear proyecto nuevo
- [ ] Prueba 2 — Modificar proyecto existente
- [ ] Prueba 3 — DevOps
- [ ] Prueba 4 — QA
- [ ] Prueba 5 — Ponytail
- [ ] Prueba 6 — Memoria (generación + reutilización en sesión nueva)

Si las 6 quedan en ✅, el concepto completo de Mova Context queda validado de punta a punta, sin haber tocado ningún proyecto real.

---

## 11b. Prueba adicional — Focus (opcional pero recomendada)

**Objetivo:** confirmar que `focus` limita el trabajo del modelo a archivos o funciones específicas, sin que analice el proyecto completo.

**Qué vas a pedir:**

```text
Lee workflow.md → pruebas-locales → auditar-modulo-tareas
```

Esta task tiene declarado en `project.json`:
```json
"focus": ["taskController.js", "taskService.js", "crearTarea()"]
```

**Qué debería pasar:**

1. El modelo informa explícitamente que va a trabajar solo sobre esos tres elementos
2. No analiza `taskRepository.js`, `auth.js` ni el resto del proyecto
3. Si `taskController.js` apareciera en más de una carpeta, el modelo pregunta cuál antes de continuar

**Cómo verificar que funcionó:**
- [ ] El reporte de auditoría menciona solo `taskController.js`, `taskService.js` y la función `crearTarea()`
- [ ] No hay análisis de archivos fuera del focus declarado

---

## 12. Después de probar

Si elegiste una herramienta distinta a Cline+Gemini para estas pruebas (ver sección 2.5), ya comprobaste en la práctica que el resultado es el mismo. Eso es, de hecho, todo el punto de Mova Context: la convención vive en archivos de texto, no en una herramienta específica.
