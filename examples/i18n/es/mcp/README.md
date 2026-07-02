# Ejemplos MCP — Mova Context v3

El servidor expone el motor de **Mova Context** mediante la especificación oficial de **MCP (Model Context Protocol)** basada en **JSON-RPC 2.0**.

Soporta dos mecanismos de transporte:

- **HTTP**: ideal para pruebas manuales con **Postman**, **curl** o cualquier cliente HTTP.
- **Stdio**: utilizado por clientes MCP como Claude Desktop, Cursor, Windsurf y otros clientes compatibles.

---

# Transporte HTTP

El modo HTTP expone un endpoint JSON-RPC que permite ejecutar manualmente cualquiera de las herramientas MCP.

## Iniciar el servidor

```bash
mova mcp start
```

o especificando un puerto:

```bash
mova mcp start --port 3000
```

## Endpoint

```text
POST http://localhost:3000/mcp
Content-Type: application/json
```

---

# Métodos MCP disponibles

Todas las peticiones HTTP deben enviarse siguiendo el formato **JSON-RPC 2.0**.

Actualmente el servidor implementa los siguientes métodos:

| Método | Descripción |
|---------|-------------|
| `initialize` | Inicializa la sesión MCP. |
| `tools/list` | Lista todas las herramientas disponibles. |
| `tools/call` | Ejecuta una herramienta específica. |

---

# Ejemplo 1 — initialize

## Request

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {}
}
```

## CURL

```bash
curl -s -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' \
  | jq .
```

---

# Ejemplo 2 — tools/list

## Request

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {}
}
```

## CURL

```bash
curl -s -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
  | jq .
```

---

# Ejemplo 3 — tools/call → get_full_context

## Request

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "get_full_context",
    "arguments": {
      "project": "ley-21719",
      "task": "analizar-contrato"
    }
  }
}
```

## Response

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "# Mova Context — ley-21719 / analizar-contrato\n..."
      }
    ]
  }
}
```

## CURL

```bash
curl -s -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_full_context","arguments":{"project":"ley-21719","task":"analizar-contrato"}}}' \
  | jq .
```

---

# Ejemplo 4 — tools/call → search_context

## Request

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "search_context",
    "arguments": {
      "query": "datos personales",
      "domain": "legal"
    }
  }
}
```

## CURL

```bash
curl -s -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"search_context","arguments":{"query":"datos personales","domain":"legal"}}}' \
  | jq .
```

---

# Colección de Postman

Copia y guarda el siguiente contenido como **mova-mcp.postman_collection.json** para importar toda la suite de pruebas JSON-RPC directamente en Postman.

```json
{
  "info": {
    "name": "Mova Context MCP v3",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "1. Initialize Protocol",
      "request": {
        "method": "POST",
        "url": "http://localhost:3000/mcp",
        "header": [
          {
            "key": "Content-Type",
            "value": "application/json"
          }
        ],
        "body": {
          "mode": "raw",
          "raw": "{\n  \"jsonrpc\": \"2.0\",\n  \"id\": 1,\n  \"method\": \"initialize\",\n  \"params\": {}\n}"
        }
      }
    },
    {
      "name": "2. List Available Tools",
      "request": {
        "method": "POST",
        "url": "http://localhost:3000/mcp",
        "header": [
          {
            "key": "Content-Type",
            "value": "application/json"
          }
        ],
        "body": {
          "mode": "raw",
          "raw": "{\n  \"jsonrpc\": \"2.0\",\n  \"id\": 2,\n  \"method\": \"tools/list\",\n  \"params\": {}\n}"
        }
      }
    },
    {
      "name": "3. Get Full Context",
      "request": {
        "method": "POST",
        "url": "http://localhost:3000/mcp",
        "header": [
          {
            "key": "Content-Type",
            "value": "application/json"
          }
        ],
        "body": {
          "mode": "raw",
          "raw": "{\n  \"jsonrpc\": \"2.0\",\n  \"id\": 3,\n  \"method\": \"tools/call\",\n  \"params\": {\n    \"name\": \"get_full_context\",\n    \"arguments\": {\n      \"project\": \"pruebas-locales\",\n      \"task\": \"crear-proyecto\"\n    }\n  }\n}"
        }
      }
    },
    {
      "name": "4. Search Context",
      "request": {
        "method": "POST",
        "url": "http://localhost:3000/mcp",
        "header": [
          {
            "key": "Content-Type",
            "value": "application/json"
          }
        ],
        "body": {
          "mode": "raw",
          "raw": "{\n  \"jsonrpc\": \"2.0\",\n  \"id\": 4,\n  \"method\": \"tools/call\",\n  \"params\": {\n    \"name\": \"search_context\",\n    \"arguments\": {\n      \"query\": \"jwt\"\n    }\n  }\n}"
        }
      }
    }
  ]
}
```

---

# Transporte Stdio

El modo **Stdio** está pensado para clientes MCP como Claude Desktop, Cursor o Windsurf.

## Iniciar el servidor

```bash
mova mcp start --stdio
```

En este modo **no existe un endpoint HTTP** y, por lo tanto, **no pueden utilizarse los ejemplos con curl o Postman**, ya que la comunicación se realiza mediante la entrada y salida estándar (stdin/stdout).

Una vez configurado el cliente MCP, las herramientas (`initialize`, `tools/list` y `tools/call`) serán invocadas automáticamente por el cliente cuando el modelo las necesite.

---

# Notas Operacionales

## Formato estricto

Al consumir el servidor mediante HTTP, todas las peticiones deben cumplir el formato **JSON-RPC 2.0**. Omitir o estructurar incorrectamente los campos obligatorios (`jsonrpc`, `id` o `method`) provocará respuestas como:

- `-32601 (Method not found)`
- `-32700 (Parse error)`

## Seguridad

El servidor HTTP integrado en la CLI no incorpora autenticación ni autorización. Se recomienda exponerlo únicamente en interfaces locales (`localhost` o `127.0.0.1`).

## Paridad de resultados

La herramienta `get_full_context` devuelve exactamente el mismo contexto ensamblado que genera:

```bash
mova run
```