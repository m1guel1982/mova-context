# MCP — Mova Context v3

Mova Context puede exponerse como un servidor compatible con el protocolo **MCP (Model Context Protocol)**. Esto permite que Claude Desktop, Cursor, Windsurf y otras herramientas compatibles lean el contexto de tus proyectos directamente y ejecuten búsquedas, eliminando la necesidad de copiar y pegar manualmente.

El servidor soporta dos modalidades de transporte: **Stdio** (entrada/salida estándar para integración nativa con clientes de IA) y **HTTP** (JSON-RPC para pruebas con Postman o desarrollo web).

---

## Iniciar el servidor

### Modo HTTP (Por defecto - Ideal para Postman y desarrollo)
Levanta un servidor web en el puerto especificado (por defecto `3000`).


# Bash

mova mcp start
mova mcp start --port 4000
Modo Stdio (Requerido para Claude Desktop y editores de código)
La IA ejecuta el binario localmente y se comunica de manera directa y ultra rápida a través de la consola estándar.

# Bash

mova mcp start --stdio

Qué expone el servidor MCP
Contexto Completo: Ensamble de agentes, habilidades, prompts y memoria activa de un proyecto mediante una sola llamada.

Búsqueda Global: Consultas indexadas sobre toda la base de conocimiento (agents/skills/prompts).

Acceso a Memoria Histórica: Capacidad de leer la memoria viva (memory.md) o los archivos históricos consolidados.

Guía de Trabajo: Acceso directo al flujo operacional del repositorio (workflow.md).

# Configuración en Claude Desktop

Para integrar Mova en Claude Desktop, debes configurar el servidor utilizando el transporte --stdio. Modifica tu archivo claude_desktop_config.json agregando lo siguiente:

JSON
{
  "mcpServers": {
    "mova-context": {
      "command": "mova",
      "args": ["mcp", "start", "--stdio"]
    }
  }
}


Nota: Asegúrate de que el binario mova esté disponible en las variables de entorno de tu sistema (PATH), o reemplaza "mova" por la ruta absoluta de tu ejecutable.

Herramientas expuestas (Tools)

El servidor implementa formalmente el método tools/list de MCP y expone las siguientes herramientas a través de tools/call:

Herramienta	Argumentos (* = Requerido)	Descripción

get_full_context	project*, task	Retorna el contexto ensamblado completo (= mova run). Es la herramienta primaria.

get_knowledge	kind, domain, name*, lang	Obtiene un agente, skill o prompt específico por su nombre y dominio.

get_memory	project*	Lee la memoria activa actual de un proyecto (memory.md).

get_memory_all	project*	Lee la memoria activa consolidada junto con todo el histórico archivado.

get_workflow	lang	Devuelve la guía de flujo operativo workflow.md desde la raíz.

search_context	query*, domain	Busca palabras clave o conceptos en toda la base de conocimiento disponible.

Por qué usar MCP en lugar de copiar y pegar

# Sin MCP:
  mova run proyecto tarea > contexto.txt
  → Abrir archivo contexto.txt
  → Copiar manualmente miles de líneas
  → Pegar en el chat de la IA corriendo el riesgo de saturar el prompt

# Con MCP:
  Le dices a Claude: "Revisa la tarea crear-auth en el proyecto backend"
  → Claude detecta que tiene la herramienta 'get_full_context'
  → Ejecuta el comando en segundo plano de forma transparente
  → Obtiene el contexto fresco, limpio y procesado al instante