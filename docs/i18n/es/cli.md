# CLI — Mova Context

La CLI empaqueta el contexto de un proyecto en un único bloque de texto listo para usar con cualquier LLM.

Binarios precompilados para Linux, macOS (Intel + Apple Silicon) y Windows en `cli/dist/`.

---

## Comandos 

```bash
mova list                                    # listar proyectos disponibles
mova run [proyecto] [tarea]                  # generar contexto
mova memory [proyecto] "respuesta del LLM"   # guardar sesión en memory.md
mova init [nombre]                           # crear nuevo proyecto
mova mcp start [--port 3000]                 # iniciar servidor MCP
mova search "consulta"                       # buscar en el conocimiento
mova memory-archive [proyecto]               # archivar entradas de memoria antiguas
```

---

## Uso típico

```bash
# 1. Ver qué proyectos existen
mova list

# 2. Generar el contexto del proyecto
mova run ley-21719 analizar-contrato > contexto.txt

# 3. Copiar contexto.txt y pegar en Claude / ChatGPT / Gemini

# 4. Guardar la respuesta del LLM en memory.md
mova memory ley-21719 "$(pbpaste)"
```

---

## Con redirección a archivo

```bash
mova run mi-proyecto mi-tarea > contexto.txt
```

El archivo `contexto.txt` contiene el contexto completo listo para pegar en cualquier LLM web.

---

## Inicializar un nuevo proyecto

```bash
mova init mi-empresa
```

Crea la estructura básica:

```text
projects/mi-empresa/
├── project.json
└── memory.md
```

---

## Servidor MCP

```bash
mova mcp start           # puerto 3000 por defecto
mova mcp start --port 4000
```

Expone el contexto de todos los proyectos como servidor MCP compatible con Claude Desktop y otras herramientas.

---

## Variables de entorno

```bash
MOVA_ROOT=/ruta/a/mova-context   # directorio raíz del repo
MOVA_PORT=3000                   # puerto del servidor MCP
```

---

## Instalación

```bash
# macOS / Linux
chmod +x cli/dist/mova-linux-amd64
sudo mv cli/dist/mova-linux-amd64 /usr/local/bin/mova

# Windows
# Agregar cli/dist/mova-windows-amd64.exe al PATH
```
