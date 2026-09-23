# Controles que cortan el proceso — `debug`, `policies`, `on_exceed`, `dry_run`

Cuatro campos de `project.json`. Los cuatro funcionan **igual en CLI, `mova chat`, MCP y HTTP** — una sola implementación, cuatro puertas.

## `debug: true` — ver la ruta exacta de cada política

```json
"debug": true
```

Con esto, `mova context-trace` agrega una línea por cada política incluida/excluida, con su **ruta absoluta real**:

```
[debug] incluida: security.json -> /ruta/real/config/policy/security.json
[debug] excluida: pii_permissive.json -> /ruta/real/config/policy/pii_permissive.json (excluida por configuración)
```

Sin `debug`, esas líneas no aparecen — el resto del reporte es idéntico. `debug` nunca cambia una decisión, solo la explica.

## `policies` — qué reglas se cargan

```json
"policies": { "include": ["security.json", "pii_strict.json"], "exclude": ["pii_permissive.json"] }
```

Ver `PROJECT_JSON.md § policies` para la resolución de rutas y la precedencia completa (CLI > `project.json` > `config/policy.json`).

## `budget.on_exceed` — qué pasa si te pasas del presupuesto

| Valor | Efecto |
|---|---|
| `"warn"` (default) | Avisa en consola, **continúa** y llama al LLM igual. |
| `"abort"` / `"block"` | **Corta antes de llamar al LLM.** 0 tokens salen. Mismo corte en las 4 puertas. |

## `egress_audit.dry_run` — probar todo sin llamar al LLM

```json
"egress_audit": { "dry_run": true, "output_file": "egress_sanitized.md" }
```

Con `dry_run: true`: se arma el contexto, se sanitiza, se audita — y **nunca se llama al proveedor, en ninguna herramienta/comando que exponga contexto** (`chat_completion`, `get_full_context`, `get_memory`, `get_memory_all`, `get_workflow`, `read_file`, `read_document_layer` y `mova run` por igual — ver `PROJECT_JSON.md § Air-gap real`). Devuelve una respuesta de éxito indicando que fue un dry-run, más una **directiva explícita anti-elusión** dirigida al modelo/agente que la recibe (ver la nota "Límites honestos" más abajo — esa directiva es una instrucción reforzada, no una garantía técnica). Sirve para probar todo el pipeline de gobernanza **sin tener un modelo corriendo** — ver `MCP_HTTP_TESTING.md` para probar así, gratis, sin GPU.

La directiva y el encabezado del bloque de auditoría son 100% personalizables y multiidioma —
viven en `config/lang/{es,en}.json` (`reports.egress_airgap_message` y
`reports.egress_airgap_directive`) y se recargan **en caliente**, sin reiniciar Mova: editás el
archivo, el cambio se aplica dentro de ~1 segundo. Si borrás la clave `egress_airgap_directive`
(o la dejás vacía), el bloqueo sigue funcionando igual — solo desaparece la directiva extra, nunca
se rompe el proceso ni se filtra el nombre de la clave.

### Límites : el `dry_run` es lo que Mova controla, no lo que el anfitrión hace después

Mova garantiza su propia parte del contrato: mientras `dry_run: true` esté activo, el contexto real
**nunca sale del proceso de Mova** — ni al proveedor, ni en el resultado de la herramienta — y queda
evidencia en disco de cada intento. Lo que Mova **no puede garantizar** es qué hace el **modelo/agente
anfitrión** (Cursor, Claude Code, Grok, etc.) con el bloqueo que recibe: un anfitrión insistente puede
intentar leer otros archivos locales (`context-report.md`, `project.json`, memoria, etc.) para
"reconstruir" el contexto por su cuenta, en vez de detenerse — esto ya ocurrió en pruebas reales. La
directiva anti-elusión reduce la probabilidad de eso (le dice explícitamente al modelo que no lo
haga), pero no puede impedirlo técnicamente: Mova no tiene control sobre el proceso del anfitrión una
vez que este decide invocar otras herramientas por su cuenta. Tratá el `dry_run` como "Mova nunca te
da el contexto real", no como "es imposible que el anfitrión consiga el contexto por otra vía".

## Sin `llm_profile` — Mova nunca llama a un modelo por su cuenta

Si `project.json` no declara `llm_profile`, Mova no intenta contactar ningún proveedor local/cloud. En MCP/HTTP, `chat_completion` devuelve el contexto ya gobernado para que el **LLM anfitrión** (Cursor, Claude Code, etc.) responda. En `mova chat` se avisa explícitamente qué modelo global se usa en su lugar.

## Verificado en las 4 puertas (no solo documentado)

| Puerta | `debug` | `on_exceed: block` | `dry_run: true` |
|---|---|---|---|
| CLI (`mova context-trace` / `mova chat` / `mova run`) | ✅ | ✅ corta | ✅ corta |
| MCP (stdio) | ✅ | ✅ corta | ✅ corta |
| HTTP (`POST /mcp`) | ✅ | ✅ corta | ✅ corta |

Ejemplo real, ejecutado contra `examples/02-pii-compliance-governance` (que trae los 4 campos activados):

```bash
mova context-trace 02-pii-compliance-governance   # muestra [debug] con rutas reales
```
