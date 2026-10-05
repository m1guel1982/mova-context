# Ejemplo 07 — Multiagente: la flota de Nebula Delivery

> Mismo universo ficticio del [ejemplo 06](../06-nebula-delivery-graph-memory/README.md), ahora con **tres agentes especializados** que revisan el mismo repo con su propio `focus`/`exclude`, su propio grafo y su propio egress, y que **se pasan la posta por una memoria compartida**.

## Cómo es un multiagente en Mova

Un **grupo** es una carpeta en `projects/` con un `config.json` que lista agentes; **cada agente es un proyecto normal** (`projects/<grupo>/<agente>/project.json`) con su presupuesto, tareas, focus, memoria, grafo y egress. No hay un formato especial: un agente es un proyecto direccionado como `<grupo>/<agente>`.

```
projects/05-nebula-flota/
├── config.json                  ← el orquestador: {"group", "description", "agents": [...]}
├── memory.md                    ← memoria COMPARTIDA del grupo
├── planificador/  project.json · grafo.png · egress_sanitized.md   → revisa el ruteo
├── facturador/    project.json · grafo.png · egress_sanitized.md   → mejora cobros y tarifas
└── auditor/       project.json · grafo.png · egress_sanitized.md   → audita PII (pii_masking activo)
```

| Agente | Tarea | Foco | Excluye |
|---|---|---|---|
| `planificador` | `revisar-ruteo` | `planificador.js`, `rutas.js` | `depurarTrazaPlanificador`, `telemetria/sensores.js` |
| `facturador` | `mejorar-cobros` | `cobros.js`, `tarifas.js` | `tarifasV1.js`, `recargoRadiacionLegacy` |
| `auditor` | `auditar-datos` | `notificaciones.js`, `cobros.js`, `pedidos-demo.json` | `tarifasV1.js` |

**Memoria compartida:** los tres agentes declaran `"memory": "../memory.md"` (relativa a la carpeta de cada agente), así que escriben y leen el **mismo** `projects/05-nebula-flota/memory.md`. Cada entrada lleva la tarea que la generó.

## Cómo se ejecuta

```bash
mova agents list 05-nebula-flota              # los 3 agentes del grupo
mova agents run  05-nebula-flota              # arma el contexto de los 3 (SIN llamar al LLM) y genera sus grafos
mova agents run  05-nebula-flota facturador   # solo uno

# Conversar con un agente, uno tras otro: cada uno lee lo que dejó el anterior
mova chat 05-nebula-flota planificador        # deja su hallazgo en la memoria compartida
mova chat 05-nebula-flota facturador          # recibe ese hallazgo en MEMORY y parte de él
mova chat 05-nebula-flota auditor             # recibe ambos; su contexto sale con PII enmascarado
mova chat 05-nebula-flota                     # sin agente: lista los agentes disponibles
```

**Por MCP / HTTP** (mismo orquestador):

```bash
mova mcp start --port 3000

# listar / ejecutar agentes (herramientas MCP list_agents · run_agent)
curl -X POST localhost:3000/agents/run -H 'Content-Type: application/json' \
     -d '{"group":"05-nebula-flota","agent":"facturador"}'

# conversar con un agente por JSON-RPC: el proyecto es "<grupo>/<agente>"
curl -X POST localhost:3000/mcp -H 'Content-Type: application/json' -d '{
  "jsonrpc":"2.0","id":1,"method":"tools/call",
  "params":{"name":"chat_completion","arguments":{"project":"05-nebula-flota/planificador","message":"revisa el ruteo"}}}'
```

## Qué demuestra

- **Orquestación sin magia:** `config.json` + proyectos normales; los mismos comandos (`run`, `chat`, MCP, HTTP) sirven para un agente y para un grupo.
- **Especialización:** cada agente ve solo su porción del código (su propio `focus`/`exclude`) y por eso su contexto y su costo son pequeños.
- **Traspaso por memoria:** el `facturador` recibe en `MEMORY` el hallazgo del `planificador` (pérdida de `etaReal` en `normalizarPedido`) y el `auditor` recibe ambos. Con `memory.md` abierto verás tres entradas, una por tarea.
- **Gobernanza por agente:** el `auditor` activa `budget.pii_masking`; en su `egress_sanitized.md` el email y el documento del cliente salen como `[PII_xxxxxxxx]`.
- **Un grafo por agente** (`grafo.png`), sin gastar tokens.

## Notas honestas

- `mova agents run` **no llama al modelo**: arma y entrega el contexto de cada agente (en paralelo, con un tope de trabajadores). Por eso no escribe memoria; la memoria se registra al conversar (`mova chat`) o vía `chat_completion`.
- El orden de traspaso lo define **el orden en que conversas** con los agentes; el grupo no impone una cadena.
- Este ejemplo reutiliza el código de `examples/06-nebula-delivery-graph-memory/repo` y el modelo `nebula-demo`.
- Los `memory.md`, `egress_sanitized.md` y grafos de muestra salieron del binario real con **respuestas de modelo simuladas** (coherentes con el código). La ruta del subtítulo de cada PNG es la de la máquina donde se generó.
