# mova CLI — Manual de uso y notas de desarrollo

Un programa de línea de comandos que empaqueta el contexto de un proyecto Mova Context en un único bloque de texto, listo para pegar en cualquier LLM web (Claude, ChatGPT, Gemini, u otro).

---

## El problema que resuelve

Mova Context organiza el conocimiento de un proyecto en archivos de texto (agents, skills, prompts, memory). Cuando usas Cline u otra herramienta con acceso a archivos, esos archivos se cargan automáticamente. Cuando usas un LLM web (claude.ai, chat.openai.com, gemini.google.com), no hay acceso a tu filesystem — tendrías que copiar y pegar cada archivo a mano en el orden correcto, inyectar las variables, y no olvidar nada.

`mova run` hace exactamente eso: lee el `project.json`, carga los archivos en el orden que define el workflow, inyecta las variables, y genera un único bloque listo para pegar.

---

## Requisitos

| Requisito | Versión mínima |
|---|---|
| Sistema operativo | Windows 10+, macOS 10.15+, Linux (cualquier distro moderna) |
| Arquitectura | x86_64 o Apple Silicon (ARM64) |
| Go (solo si compilas desde fuente) | 1.22+ |

El binario precompilado no requiere instalar nada más — ni Go, ni Node.js, ni ningún runtime.

---

## Instalación y primeros pasos por sistema operativo

### Windows

**Paso 1 — Descargar el binario**

Descarga `mova-windows-amd64.exe` desde la carpeta `cli/dist/` del repositorio.

**Paso 2 — Ubicarlo**

Puedes ejecutarlo directamente desde donde lo descargaste, o moverlo a una carpeta que esté en el PATH para usarlo desde cualquier terminal.

Opción rápida sin tocar el PATH — colócalo en la raíz de mova-context:
```
mova-context\
├── mova.exe        ← aquí
├── workflow.md
├── agents\
...
```

**Paso 3 — Ejecutar**

Abre PowerShell o CMD, navega a la carpeta de mova-context y ejecuta:
```powershell
cd E:\nuevosProyectos21012026Mova\misProyectos\mova-context-test\mova-context
mova list
mova run pruebas-locales crear-proyecto > contexto.txt (o el nombre que se estime conveniente)
```
**Crear proyecto base :**
  mova init proyecto1
--Salida Creado projects/proyecto1/project.json

**Copiar output a a un archivo (para pegarlo en el LLM):**
```powershell
mova run pruebas-locales crear-proyecto > contexto.txt (o el nombre que se estime conveniente)
```

**Actualizar memory.md desde el portapapeles:**
```powershell
.\mova.exe memory pruebas-locales "$(Get-Clipboard)"
```

**Si Windows bloquea el archivo** ("Windows protegió tu equipo"):
- Haz clic derecho en el `.exe` → Propiedades → marca "Desbloquear" → Aceptar
- O en PowerShell: `Unblock-File -Path .\mova.exe`

---

### macOS

**Paso 1 — Descargar el binario correcto**

- Mac con chip Intel → `mova-macos-amd64`
- Mac con chip Apple Silicon (M1, M2, M3) → `mova-macos-arm64`

No sabes cuál tienes → menú Apple → "Acerca de este Mac" → si dice "Apple M..." es ARM64, si dice "Intel" es amd64.

**Paso 2 — Dar permisos y moverlo al PATH**

```bash
# Renombrar (opcional, para simplificar)
mv mova-macos-arm64 mova

# Dar permiso de ejecución
chmod +x mova

# Quitar la cuarentena de Gatekeeper (necesario la primera vez)
xattr -d com.apple.quarantine mova

# Mover al PATH para usarlo desde cualquier carpeta
sudo mv mova /usr/local/bin/mova
```

Si prefieres no moverlo al PATH, colócalo en la raíz de mova-context y úsalo como `./mova`.

**Paso 3 — Ejecutar**

```bash
cd /ruta/a/mova-context
mova list
mova run pruebas-locales crear-proyecto > contexto.txt (o el nombre que se estime conveniente)
```

 

**Actualizar memory.md desde el portapapeles:**
```bash
mova memory pruebas-locales "$(pbpaste)"
```

**Si macOS dice "no se puede abrir porque no se puede verificar al desarrollador":**
```bash
xattr -d com.apple.quarantine mova
```
O: Sistema → Privacidad y Seguridad → al final verás el binario bloqueado → "Abrir de todas formas".

---

### Linux

**Paso 1 — Descargar el binario**

Descarga `mova-linux-amd64` desde `cli/dist/`.

**Paso 2 — Dar permisos y moverlo al PATH**

```bash
chmod +x mova-linux-amd64

# Mover al PATH (requiere sudo)
sudo mv mova-linux-amd64 /usr/local/bin/mova

# O sin sudo, en el PATH del usuario
mkdir -p ~/.local/bin
mv mova-linux-amd64 ~/.local/bin/mova
# Asegúrate de que ~/.local/bin esté en tu PATH:
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc && source ~/.bashrc
```

**Paso 3 — Ejecutar**

```bash
cd /ruta/a/mova-context
mova list
mova run pruebas-locales crear-proyecto > contexto.txt (o el nombre que se estime conveniente)
```

**Copiar output al portapapeles:**
```bash
# Con xclip
mova run pruebas-locales crear-proyecto | xclip -selection clipboard

# Con xsel
mova run pruebas-locales crear-proyecto | xsel --clipboard --input
```

**Actualizar memory.md desde el portapapeles:**
```bash
mova memory pruebas-locales "$(xclip -selection clipboard -o)"
```

---

### Compilar desde el código fuente (cualquier SO)

Requiere Go 1.22+. Descarga desde [go.dev/dl](https://go.dev/dl/).

```bash
cd cli/
go build -o mova .        # Linux / macOS
go build -o mova.exe .    # Windows

# Compilar todos los binarios de una vez
make build
```

---


## Uso

Ejecuta `mova` siempre desde dentro del repositorio `mova-context` (la carpeta que contiene `workflow.md`).

### `mova list`

Lista todos los proyectos disponibles y sus tareas.

```bash
mova list
```

Ejemplo de salida:
```
Proyectos disponibles:
  pruebas-locales — Proyecto ficticio mínimo (API de tareas)
    Tasks: agregar-modulo-categorias, auditar-modulo-tareas, configurar-cicd, crear-proyecto, generar-tests, refactor-ponytail

```

---

### `mova run [proyecto] [tarea]`

Genera el contexto completo listo para pegar en un LLM.

```bash
# Con proyecto y tarea explícitos
mova run pruebas-locales crear-proyecto

# Con tarea por defecto (la declarada en default_task del project.json)
mova run pruebas-locales

# Si solo hay un proyecto en el repo, se detecta automáticamente
mova run
```

El output es texto que se puede copiar directamente al chat del LLM. Incluye, en orden:

1. Agents (base globales → custom globales → base de la task → custom de la task)
2. Skills (mismo orden)
3. Prompt (base → custom)
4. Memoria de sesiones anteriores (si existe `memory.md`)
5. Instrucción final: qué hacer y cómo responder para que la memoria se pueda actualizar

**Flujo completo en la práctica:**

```bash
# 1. Generar el contexto
mova run pruebas-locales crear-proyecto

# 2. Copiar el output completo

# 3. Pegarlo en Claude / ChatGPT / Gemini y trabajar

# 4. Cuando el LLM responde, copiar su respuesta completa

# 5. Actualizar memory.md con esa respuesta
mova memory pruebas-locales "pegar aquí la respuesta completa del LLM"
```

**Atajos para pegar desde el portapapeles directamente:**

```bash
# macOS
mova memory pruebas-locales "$(pbpaste)"

# Linux (requiere xclip instalado)
mova memory pruebas-locales "$(xclip -selection clipboard -o)"

# Windows PowerShell
mova memory pruebas-locales "$(Get-Clipboard)"
```

---

### `mova memory [proyecto] "respuesta"`

Extrae el bloque de memoria de la respuesta del LLM y actualiza `memory.md`.

```bash
mova memory pruebas-locales "respuesta completa del LLM aquí"
```

El LLM debe incluir en su respuesta un bloque con este formato (la instrucción final de `mova run` ya se lo pide):

````
```memory
## 2026-06-22 — sesión
**Hecho:** Se generó la API de tareas con 3 endpoints
**Resuelto:** Estructura de carpetas, server.js, taskController
**Pendiente:** Tests y módulo de categorías
**Decisiones:** SQLite en archivo local, sin Docker
**Errores LLM:** Ninguno
```
````

`mova memory` extrae ese bloque y lo agrega al inicio de `projects/[proyecto]/memory.md`, con la sesión más reciente siempre arriba.

---

## Variables automáticas

Toda clave declarada en `variables` del `project.json` se inyecta automáticamente en agents, skills y prompts. No hay lista fija — cualquier clave nueva que agregues funciona.

```json
"variables": {
  "stack": "Node.js + Express",
  "mi_variable_nueva": "cualquier valor"
}
```

Se convierte en `{{STACK}}` y `{{MI_VARIABLE_NUEVA}}` en los archivos `.md`.

Tres variables siempre disponibles sin declararlas:
- `{{PROJECT}}` — nombre del proyecto
- `{{REPO}}` — directorio de trabajo
- `{{TASK}}` — nombre de la tarea activa

---

## Directorio de trabajo (`repo`)

El código generado por el LLM debe ir en el directorio que indica `"repo"` en `project.json`:

```json
"repo": "."                                          // dentro de mova-context
"repo": "../app-prueba-local"                        // carpeta hermana
"repo": "E:/proyectos/mi-proyecto"                   // ruta absoluta Windows
"repo": "/home/usuario/proyectos/mi-proyecto"        // ruta absoluta Linux/macOS
```

`mova run` incluye esta ruta en el contexto generado para que el LLM sepa dónde crear los archivos cuando trabajes con una herramienta con acceso a filesystem (Cline, Claude Code, etc.).

---

## Cómo se desarrolló

Esta sección documenta las decisiones de diseño del CLI para que cualquiera pueda entenderlo, modificarlo o portarlo a otro lenguaje.

### Por qué Go

- Compila a un binario estático sin dependencias — el usuario descarga un archivo y ya funciona, sin instalar runtimes
- Multiplataforma desde un solo comando de compilación cruzada
- La librería estándar cubre todo lo necesario: lectura de archivos, JSON, strings, argumentos de CLI

### Estructura del código (`main.go`)

El programa tiene un solo archivo. No hay paquetes externos. Estructura:

```
main.go
├── Estructuras JSON (Project, Task, AgentSkillBlock, PromptBlock)
├── movaRoot()          — encuentra la carpeta raíz buscando workflow.md
├── resolveFile()       — busca un .md en base/ y custom/ recursivamente
├── normalize()         — convierte snake_case a {{UPPER_CASE}}
├── mergeVars()         — fusiona variables globales y de task
├── injectVars()        — reemplaza {{VARIABLE}} en el texto
├── loadSection()       — carga agents o skills en orden correcto
├── loadPrompt()        — carga prompt base → custom
├── cmdRun()            — comando principal: construye el contexto completo
├── cmdMemory()         — extrae bloque memory y actualiza memory.md
├── cmdList()           — lista proyectos disponibles
└── main()              — despacho de comandos
```

### Decisiones de diseño

**Sin flags, solo argumentos posicionales.** `mova run proyecto tarea` es más rápido de escribir que `mova --project=proyecto --task=tarea`, y cubre el 100% de los casos de uso.

**El output va a stdout.** Permite redirigir a un archivo (`mova run proyecto > contexto.txt`) o copiar con cualquier herramienta del sistema operativo, sin que el programa tenga que manejar el portapapeles.

**Si un archivo no existe, se ignora y se continúa.** Mismo comportamiento que define `workflow.md`: la ausencia de un archivo no es un error fatal.

**La normalización de variables es automática.** Cualquier clave `snake_case` en el JSON se convierte a `{{UPPER_CASE}}` sin lista predefinida. Esto significa que agregar una variable nueva al `project.json` funciona de inmediato sin tocar el código.

**La memoria se actualiza manualmente.** El LLM no puede escribir archivos en tu máquina — el usuario ejecuta `mova memory` con la respuesta. Esto es una limitación real, pero es la única forma de funcionar en el contexto de LLMs web sin acceso al filesystem.

**`movaRoot()` sube por el árbol buscando `workflow.md`.** Esto permite ejecutar `mova` desde cualquier subdirectorio del repo sin tener que hacer `cd` a la raíz primero.

### Cómo agregar un nuevo comando

Agregar `mova init [proyecto]` que cree `projects/[proyecto]/project.json` desde una plantilla:

```go
case "init":
    if len(os.Args) < 3 {
        fmt.Fprintln(os.Stderr, "Uso: mova init [nombre-proyecto]")
        os.Exit(1)
    }
    cmdInit(root, os.Args[2])
```

Y la función:
```go
func cmdInit(root, name string) {
    dir := filepath.Join(root, "projects", name)
    os.MkdirAll(dir, 0755)
    template := `{"project": "` + name + `", "description": "", "repo": ".", "default_task": "", "variables": {}, "agents": {"base": [], "custom": []}, "skills": {"base": [], "custom": []}, "tasks": {}}`
    os.WriteFile(filepath.Join(dir, "project.json"), []byte(template), 0644)
    fmt.Printf("Creado projects/%s/project.json\n", name)
}
```

---

## Limitaciones conocidas

- **La memoria se actualiza a mano**: el usuario debe ejecutar `mova memory` después de cada sesión. No hay sincronización automática.
- **El output puede ser largo**: si un proyecto tiene muchos agents/skills, el contexto generado puede superar el límite de contexto de algunos LLMs web. En ese caso, usa solo las skills y agents de la task específica en vez de cargarlos todos globalmente.
- **Sin validación de variables faltantes en el output**: si un `.md` usa `{{VARIABLE}}` y esa variable no está declarada, el placeholder queda sin reemplazar en el output. El programa lo informa, pero no detiene la ejecución.
- **Sin modo interactivo**: `mova` es un generador de texto, no un agente. La interacción ocurre en el LLM web, no en el CLI.
