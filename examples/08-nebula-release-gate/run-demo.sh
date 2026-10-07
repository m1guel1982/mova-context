#!/usr/bin/env bash
# Ejemplo 08 — puerta de release multiagente. Solo usa herramientas MCP/HTTP existentes:
# list_agents · run_agent · save_memory · get_full_context · estimate_budget. Ninguna llama a un LLM.
# El "anfitrión" (aquí este script; en la vida real Claude Code/Cursor) escribe los hallazgos con save_memory.
set -euo pipefail
G=08-nebula-release-gate; PORT=${PORT:-3000}; URL=http://localhost:$PORT/mcp
call() { # call <tool> <json-arguments>
  curl -s "$URL" -H 'Content-Type: application/json' \
    -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"$1\",\"arguments\":$2}}"; }

mova mcp start --http --port "$PORT" >/dev/null 2>&1 & MOVA_PID=$!; trap 'kill $MOVA_PID 2>/dev/null' EXIT
for _ in $(seq 20); do curl -sf "localhost:$PORT/health" >/dev/null && break; sleep 0.5; done

echo "== 1. Agentes del grupo";            call list_agents  "{\"group\":\"$G\"}"; echo
echo "== 2. Presupuesto del grupo (suma)"; call estimate_budget "{\"project\":\"$G\"}"; echo

echo "== 3. Agente 1 — mapeador: arma contexto y registra hallazgo"
call run_agent "{\"group\":\"$G\",\"agent\":\"mapeador\"}" | head -c 400; echo
call save_memory "{\"project\":\"$G/mapeador\",\"task\":\"mapear-perdidas\",\"entry\":\"**Tarea:** mapear-perdidas\n**Hallazgos:**\n- src/despacho/planificador.js::normalizarPedido — etaReal — se pierde — reconstruye el pedido campo a campo y no copia etaReal\n**Pendiente:** el guardián debe revisar qué más se copia completo (cliente)\"}"; echo

echo "== 4. Agente 2 — guardian-pii recibe el hallazgo previo en MEMORY"
call get_full_context "{\"project\":\"$G/guardian-pii\"}" | grep -o 'normalizarPedido — etaReal[^\\]*' | head -1 || echo "(sin MEMORY: revisa 'memory' en project.json)"
call save_memory "{\"project\":\"$G/guardian-pii\",\"task\":\"auditar-pii\",\"entry\":\"**Tarea:** auditar-pii\n**Hallazgos:**\n- src/facturacion/cobros.js::generarCobro — cliente — se copia completo al cobro sin necesidad\n- src/facturacion/notificaciones.js::notificarCliente — email — solo se necesita al enviar\n**Pendiente:** acotar cliente en generarCobro\"}"; echo

echo "== 5. Agente 3 — revisor lee ambos hallazgos y cierra"
call get_memory "{\"project\":\"$G/revisor\"}" | head -c 600; echo
call save_memory "{\"project\":\"$G/revisor\",\"task\":\"revisar-cierre\",\"entry\":\"**Tarea:** revisar-cierre\n**Decisiones:** release NO pasa hasta copiar etaReal en normalizarPedido y acotar cliente en generarCobro\"}"; echo
echo "== Memoria compartida: projects/$G/memory.md"
