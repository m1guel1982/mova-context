# Qué se ejecutó realmente

Entorno: Linux amd64, Go 1.27.1 (el `go.mod` declara 1.24.4), binario compilado desde `src/cli`. macOS, Windows y arm64 **no** se ejecutaron en esta verificación.

| Verificación | Resultado |
|---|---|
| `go test ./...` | Todos los paquetes pasan, incluidos los tests nuevos: límite del repo (`documents`), política de lectura y hooks (`mcp/documents_policy_test.go`), loop de tools (`TestRunLoopTool_GovernsToolResults`), sanitización por bloque (`sanitize/govern_test.go`), cierre de dependencias (`graph/closure_test.go`), evidencia de escritura única (`evidence`), guard HTTP (`http/guard_test.go`) |
| `read_file` por MCP stdio sobre `src/legacy/tarifasV1.js` (excluido en 04) | Denegado (`exclude`) |
| `read_file` sobre `datos/pedidos-demo.json` (fuera del focus en 04) | Denegado (`focus_scope`) |
| `read_file` sin `project` con ruta absoluta a datos con PII | Denegado (`project` obligatorio) |
| `read_file` sobre `/etc/passwd` | Denegado (fuera del repo) |
| `mova run 04-nebula-delivery analizar-trazabilidad` / `agregar-columnas` | Liberados, cada uno con su `runs/<run_id>/` |
| `mova run 04-nebula-delivery recalcular-tarifas` | Bloqueado: `calcularTarifa -> tarifaPlanaV1 (call, excluido)`; `context.txt` vacío y `decision.gate = dependency_closure` |
| `mova run 02-pii-compliance-governance` dos veces | Mismo `context.sha256` en ambos runs; 0/10 nombres, 0 direcciones, 0 emails, 0 RUT y 0 IP en claro en `context.txt`; 4 bloques `FOCUS:` separados |
| `mova mcp start --http --bind 0.0.0.0` sin `MOVA_HTTP_TOKEN` | Se rehúsa a arrancar (test) |
| `examples/08-nebula-release-gate/run-demo.sh` | Ejecutado sobre HTTP loopback; las 3 entradas de memoria quedan con `source=host` |

**No verificado:**
- La integración de punta a punta de los hooks `check_read` / `sanitize_tool_output` dentro de Claude Code (solo están probadas las respuestas de las tools).
- La precisión y el recall del enmascarado PII.
- Que un modelo resuelva mejor una tarea con el contexto de Mova que sin él.
