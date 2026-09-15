# mova — Avanzado

## Variables de entorno

| Variable | Uso |
|---|---|
| `MOVA_PROJECT_ROOT` / `MOVA_PROJECT_PATH` | Fuerza la raíz del proyecto en vez de buscar `workflow.md` hacia arriba. |
| `MOVA_ADAPTER` / `MOVA_DSN` | Cambia el backend de almacenamiento (`file` por defecto, o Postgres/MongoDB vía DSN). |
| `MOVA_POLICY_AUTHOR` | Autor de política para CI/CD cuando `project.json`/`config/policy.json` no lo declaran — ver Matriz de Auditoría #3. |
| `MOVA_MAX_CONCURRENCY` / `MOVA_HTTP_MAX_CONCURRENCY` | Límite de goroutines concurrentes (CLI / servidor HTTP). |

## Instalación

```bash
make install   # compila y copia el binario a $(go env GOPATH)/bin/mova
```

Con esa carpeta en el `PATH`, `mova` corre desde cualquier directorio.

## Multiagente — grupos de agentes

Un `project.json` con `"is_group": true` y `"members": [...]` ejecuta cada proyecto miembro como un agente,
en orden. Ver `mova agents list` / `mova agents run <grupo>`.

## La cascada de políticas de gobernanza

`config/policy.json` enumera archivos de `config/policy/` (`security.json`, `review.json`,
`compliance.json`, `pii.json`), cargados en orden — agregar/quitar una entrada no requiere recompilar Go.

## `--diagram` — formatos y destino

`--export svg,png,pdf` acepta uno o varios formatos separados por coma. `--path` acepta un directorio
(nombra el archivo automáticamente como `<proyecto>.<formato>`) o, si se pide un único formato, la ruta
completa de un archivo con esa extensión (ej. `--path ./diagramas/evidencia.png`).
