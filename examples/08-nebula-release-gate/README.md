# Ejemplo 08 — Puerta de release multiagente (sin LLM propio)

Tres agentes (`mapeador` → `guardian-pii` → `revisor`) sobre el repo ficticio de Nebula Delivery. **No agrega API ni funciones**: usa `config.json` + proyectos normales + las tools MCP existentes.

| Agente | Foco | Control |
|---|---|---|
| `mapeador` | `normalizarPedido`, `planificarEntrega` | dónde se pierde `etaReal` |
| `guardian-pii` | `notificarCliente`, `generarCobro`, `pedidos-demo.json` | `pii_masking` activo |
| `revisor` | `generarCobro`, `construirFilaReporte` | lee los dos hallazgos previos |

Los tres comparten `projects/08-nebula-release-gate/memory.md` (`"memory": "../memory.md"`) y usan `egress_audit.dry_run: false` **a propósito**: verificado, con `dry_run: true` el anfitrión no puede leer contexto ni memoria por MCP (recibe `tokens_sent: 0`). Sin `llm_profile` no hay proveedor al que enviar nada.

## Ejecutar
```bash
mova agents run 08-nebula-release-gate      # contexto + grafo de los 3, sin LLM
bash examples/08-nebula-release-gate/run-demo.sh   # orquestación por MCP/HTTP
```
`run-demo.sh` hace de agente anfitrión: `list_agents` → `estimate_budget` → `run_agent` → `save_memory` (uno por agente). En la práctica ese rol lo cumple Claude Code o Cursor.

## Notas honestas
- Sin `llm_profile`, Mova nunca llama al modelo; el anfitrión razona y registra con `save_memory`. Los hallazgos del script son **escritos a mano** (coherentes con el código), no salida de un modelo.
- `mova agents run` no escribe memoria. El orden lo impone quien orquesta.
- **Ejecutado** con el binario Linux amd64: salida en `evidence/run-demo.output.txt` y memoria resultante en `evidence/memory.after-run.md` (cada entrada marcada `source=host`: la escribió el script, Mova no vio ningún modelo). Cada `get_full_context`/`run_agent` deja su run en `projects/08-nebula-release-gate/<agente>/runs/`.
- Comprobado solo en Linux amd64.
