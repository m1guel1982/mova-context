# mova(1) — Comandos (unificado: COMMANDS + COMMANDS_ADVANCED)

Estilo MAN page. Tools MCP / rutas HTTP / comandos slash del chat: ver `FUNCTIONS.md` (matriz única).

## NAME
`mova` — gobernanza pre-inferencia del contexto para LLMs: seleccionar, sanitizar, enmascarar PII, presupuestar, auditar.

## SYNOPSIS
`mova <comando> [proyecto] [tarea] [flags]` — `[proyecto]` = carpeta bajo `projects/`.

## COMMANDS

### run
Ensambla el contexto de un proyecto (Agents+Skills+Prompt+Focus+Memory); opcionalmente dibuja evidencia.
`mova run <proyecto> [tarea] [--count] [--diagram --export svg,png,pdf --path <dir|archivo.ext>]`
- `--count`: solo estima tokens/USD, sin imprimir contexto ni escribir reporte; acepta un grupo multiagente.
- `--export`: lista separada por comas. `--path`: directorio (archivo `<proyecto>.<fmt>`) o, con un formato, ruta completa.
```bash
mova run 02-pii-compliance-governance --diagram --export png --path ./evidencia.png
```

### context-trace (alias `trace`)
Auditoría previa a la inferencia: qué se seleccionó, sanitizó, tokens, costo, quién/qué lo autorizó.
`mova context-trace <proyecto> [--export md|pdf] [--diagram] [--policies_include <l>] [--policies_exclude <l>]`
`mova context-trace --repo <url|ruta> [--task <texto>] [--ignore <globs>] [--prune-docstrings]` (discovery, sin `project.json`)
`--prune-docstrings` elimina comentarios/docstrings del código seleccionado.
- `--repo` termina con `Generate a 'project.json'? [Y/n]` por stdin: responde, o usa `< /dev/null` en scripts/CI (si no, espera).
- Políticas: lista con nombres simples (búsqueda recursiva en `config/policy/`) o rutas completas; pisan `project.json` y `config/policy.json`.
```bash
mova trace --repo https://github.com/usuario/repo --task "revisar login" --ignore "docs/**,*.lock"
```

### budget
Desglose de tokens/USD por proveedor de la tarea activa; escribe `mova-budget-report.md`. 100 % local (tiktoken-go).
`mova budget <proyecto> [tarea] [--focus]` — `--focus` compara además repo completo vs. solo focus.

### chat
REPL interactivo con un modelo local/cloud; el contexto de Mova es el system prompt.
`mova chat <proyecto> [tarea|all]` · `mova chat <grupo> <agente>` · `mova chat <grupo>` (lista agentes)
- Sin tarea y con varias en `project.json` → carga todas. `all` lo fuerza.
- Dentro: `/tasks` `/task <n|all>` `/run <n>` `/budget` `/diagram` `/context-trace` `/memory` `/save` `/delete` `/tools` `/clear`, `set -model <n>`, `exit`.
- `"memory"` activo → cada respuesta sustancial deja un bloque de síntesis en `memory.md`.
- `"apply"` activo → los bloques ```` ```lang:ruta::func() ```` propuestos se listan; eliges `[s]` todos · `[1..N]` · `[n]` ninguno. Los archivos existentes se respaldan en `<archivo>.mova-<fecha>.bak` (`"backup": false` lo desactiva). Lo `exclude`d nunca se toca.
- Si el modelo gasta su cupo de salida sin devolver texto, se muestra un error con los tokens consumidos (ver `MODEL_CONFIG.md`).

### mcp start
Levanta el servidor MCP (Claude Code, Cursor, Windsurf) o el HTTP para curl/Postman.
`mova mcp start [--stdio] [--port 3000]` — sin `--stdio` sirve HTTP (puerto por defecto 3000).
```json
{ "mcpServers": { "mova": { "command": "mova", "args": ["mcp", "start", "--stdio"] } } }
```
```bash
curl -s localhost:3000/health     # vida del servidor
```

### agents list | agents run
Orquestador multiagente. `mova agents list <grupo>` · `mova agents run <grupo> [agente|--all]`
Arma el contexto y el grafo de cada agente **sin llamar al modelo** (en paralelo, con tope de trabajadores).
Un **grupo** es `projects/<grupo>/config.json` = `{"group","description","agents":[...]}`; cada agente es un
proyecto normal en `projects/<grupo>/<agente>/project.json`, direccionado `<grupo>/<agente>`.
Si se omite `agents`, se descubren las subcarpetas. (No existe el campo `is_group`/`members`.)
```bash
mova agents run 05-nebula-flota facturador
```

### familia memory
`mova memory <p> "texto"` guardar · `memory-read <p> [--all] [--month AAAA-MM]` leer · `memory-archive <p> [--days N]` (def. 30)
· `memory-clear <p> [--archived|--keep-active|--date D|--from D --to D] [--yes]` · `memory-config <p> enable|disable|days N|confirm true|false`.

### list · init · search
`mova list` proyectos · `mova init <nombre>` crea `projects/<nombre>/project.json` mínimo · `mova search "consulta" [dominio]`.

### config · show · install · model-list · remove
Modelos locales: `mova config <proveedor>` · `mova show config [modelo]` · `mova install a,b` · `mova model-list` · `mova remove a,b`.
Los perfiles viven en `config/models/<proveedor>/*.json`; cada proyecto elige con `llm_profile.config`.

## ENVIRONMENT
| Variable | Efecto |
|---|---|
| `MOVA_PROJECT_ROOT` | Fuerza la raíz en vez de buscar `workflow.md` hacia arriba (necesario si un cliente MCP lanza `mova` en otro directorio). |
| `MOVA_PROJECT_PATH` | Igual, y omite por completo la búsqueda de `workflow.md`. |
| `MOVA_POLICY_AUTHOR` | Autoridad de política (`project.json` → `config/policy.json` → esta → `system:default`). |
| `MOVA_ADAPTER=db` `MOVA_DSN=postgres://…` | Adaptador PostgreSQL en vez de archivos. |

## POLICY CASCADE
`config/policy.json` enumera archivos de `config/policy/` (`security`, `review`, `compliance`, `pii_*`) cargados en orden;
agregar/quitar una entrada no requiere recompilar.

## INSTALL
Desde la raíz del repo (requiere Go ≥ 1.24): `make install` — compila el binario del host con `CGO_ENABLED=0`, lo copia a
`$(go env GOPATH)/bin/mova` y agrega esa carpeta al perfil del shell. `make build-all` compila en `dist/` para
Windows, Linux (amd64, arm64) y macOS (amd64, arm64). Instaladores: `installers/{linux,macos,windows}`.

## EXAMPLES
```bash
mova list
mova budget 03-tokenomics-context-trace --focus
mova run --count 05-nebula-flota
mova mcp start --port 3000 &  curl -s localhost:3000/health
```

## SEE ALSO
`FUNCTIONS.md` · `PROJECT_JSON.md` · `GOVERNANCE_CONTROLS.md` · `ARTIFACTS.md` · `AST_FILTER.md` · `CONTEXT-TRACE.md`
