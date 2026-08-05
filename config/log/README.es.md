# Configuración de logging — `config/log/logging.json`

El sistema de logging de Mova Context viene **deshabilitado por
defecto**. No se escribe nada en disco hasta que se defina
explícitamente `"enabled": true` en `config/log/logging.json`. Todas
las puertas de entrada (CLI, chat, HTTP, MCP) leen este mismo archivo a
través de un único componente compartido (`mova.local/logging`), así
que el comportamiento es idéntico sin importar cómo se inicie Mova.

Si el archivo no existe, no se puede leer, o tiene un JSON inválido,
Mova simplemente asume "logging deshabilitado" — una configuración de
logging rota nunca debe impedir que Mova funcione.

## Ubicación del archivo

```
config/log/logging.json
```

## Ejemplo completo

```json
{
  "enabled": true,
  "structured": false,
  "level": "info",
  "categories": {
    "jobs": true,
    "orchestrator": true,
    "memory": true,
    "save": true,
    "delete": true,
    "budget": true,
    "chat": true,
    "mcp": true,
    "http": true,
    "cli": true
  },
  "file": {
    "path": "logs/mova.log",
    "auto_create": true
  },
  "rotation": {
    "mode": "daily",
    "custom_days": 7
  },
  "retention": {
    "policy": "daily",
    "custom_days": 30
  }
}
```

## Parámetros

### `enabled`
- **Descripción**: interruptor maestro de todo el sistema de logging.
- **Valores permitidos**: `true` | `false`
- **Por defecto**: `false`
- **Ejemplo**: `"enabled": true`
- **Recomendación**: dejarlo en `false` en un checkout nuevo; activarlo
  solo en los entornos donde realmente se necesite un registro (por
  ejemplo, un servidor corriendo `mova jobs start`).

### `structured`
- **Descripción**: cambia el formato de cada línea del archivo entre
  texto plano y un objeto JSON por línea.
- **Valores permitidos**: `true` (JSON por línea) | `false` (texto plano)
- **Por defecto**: `false`
- **Ejemplo (plano)**: `2026-07-30 02:00:01 [info] [jobs] finished job for project=ventas_online/vendedor (3 step(s))`
- **Ejemplo (estructurado)**: `{"time":"2026-07-30T02:00:01Z","level":"info","category":"jobs","message":"finished job..."}`
- **Recomendación**: usar `true` si se planea alimentar los logs a una
  herramienta de agregación (Loki, ELK, Datadog...); el texto plano es
  más simple para revisar con `tail -f`.

### `level`
- **Descripción**: severidad mínima que se escribe en el archivo de
  log. Todo lo que quede por debajo de este nivel se descarta
  silenciosamente.
- **Valores permitidos**: `"debug"` | `"info"` | `"warning"` | `"error"`
  (severidad creciente, en ese orden)
- **Por defecto**: `"info"`
- **Ejemplo**: `"level": "warning"` — solo se guardan warnings y errores.
- **Recomendación**: usar `"debug"` solo para depurar; es muy
  detallado. `"info"` es el valor correcto para operación normal.

### `categories`
- **Descripción**: interruptores por subsistema. Un objeto `categories`
  **ausente o vacío** significa "registrar todas las categorías" una vez
  que `enabled` es `true`. Un objeto **explícito** solo registra las
  categorías puestas en `true` — poner todas las categorías en `false`
  deshabilita toda la salida de logging sin tener que tocar `enabled`
  (útil para silenciar todo temporalmente sin perder el resto de la
  configuración).
- **Claves permitidas**: `jobs`, `orchestrator`, `memory`, `save`,
  `delete`, `budget`, `chat`, `mcp`, `http`, `cli` (cualquier subconjunto)
- **Valores permitidos por clave**: `true` | `false`
- **Por defecto**: todas las categorías habilitadas (objeto vacío)
- **Ejemplo**: `{"jobs": true, "memory": true}` — solo se registran
  trazas del Job Engine y de Memory; todo lo demás queda en silencio.
- **Recomendación**: empezar con todo habilitado; acotar una vez que se
  sepa qué subsistema realmente se necesita observar.

### `file.path`
- **Descripción**: dónde se escribe el archivo de log activo. Las rutas
  relativas se resuelven respecto a la raíz de Mova (el directorio que
  contiene `workflow.md`); las rutas absolutas se usan tal cual.
- **Valores permitidos**: cualquier ruta de archivo válida
- **Por defecto**: `"logs/mova.log"`
- **Ejemplo**: `"file": {"path": "/var/log/mova/mova.log"}`

### `file.auto_create`
- **Descripción**: si Mova crea el archivo de log (y sus directorios
  padre) automáticamente la primera vez que se registra algo.
- **Valores permitidos**: `true` | `false`
- **Por defecto**: `true`
- **Recomendación**: dejarlo en `true` salvo que el despliegue ya cree
  `logs/` con permisos específicos.

### `rotation.mode`
- **Descripción**: cada cuánto se rota el archivo de log activo hacia
  un respaldo con fecha (`mova-2026-07-30.log`).
- **Valores permitidos**: `"daily"` | `"weekly"` | `"monthly"` |
  `"yearly"` | `"custom"`
- **Por defecto**: `"daily"`
- **Ejemplo**: `"rotation": {"mode": "weekly"}`

### `rotation.custom_days`
- **Descripción**: intervalo de rotación, en días, usado solo cuando
  `rotation.mode` es `"custom"`.
- **Valores permitidos**: cualquier entero positivo
- **Por defecto**: `1` si se omite mientras `mode` es `"custom"`
- **Ejemplo**: `"rotation": {"mode": "custom", "custom_days": 3}` —
  rota cada 3 días.

### `retention.policy`
- **Descripción**: cuánto tiempo se conservan los archivos de log
  rotados antes de eliminarse automáticamente.
- **Valores permitidos**: `"daily"` (1 día) | `"weekly"` (7 días) |
  `"monthly"` (30 días) | `"yearly"` (365 días) | `"custom"`
- **Por defecto**: `"daily"`
- **Recomendación**: `"monthly"` es razonable para servidores en
  producción; `"daily"` está bien para desarrollo local.

### `retention.custom_days`
- **Descripción**: ventana de retención, en días, usada solo cuando
  `retention.policy` es `"custom"`.
- **Valores permitidos**: cualquier entero positivo
- **Por defecto**: `30` si se omite mientras `policy` es `"custom"`
- **Ejemplo**: `"retention": {"policy": "custom", "custom_days": 90}` —
  conserva 90 días de logs rotados, borra automáticamente lo más viejo.

## Notas

- Los archivos rotados viejos se eliminan automáticamente según
  `retention` — no hay que hacer limpieza manual.
- Los fallos de logging (disco lleno, permisos denegados...) nunca
  interrumpen la operación que se estaba registrando; Mova simplemente
  omite esa línea de log.
- Ver `docs/SOURCE.md` § Logging para el paquete interno
  (`mova.local/logging`) que este archivo configura.
