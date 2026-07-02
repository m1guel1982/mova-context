# MCP Examples — Mova Context v3

The server exposes the **Mova Context** engine through the official **MCP (Model Context Protocol)** specification based on **JSON-RPC 2.0**.

The protocol is identical regardless of the transport being used. The only difference is how the JSON-RPC messages are delivered.

Supported transports:

* **HTTP** – Ideal for testing with **Postman**, **curl**, or any HTTP client.
* **Stdio** – Used by MCP clients such as Claude Desktop, Cursor, Windsurf, and other compatible applications.

---

# Starting the Server

## HTTP Transport

Start the server in HTTP mode:

```bash
mova mcp start
```

or specify a custom port:

```bash
mova mcp start --port 3000
```

### Endpoint

```text
POST http://localhost:3000/mcp
Content-Type: application/json
```

---

## Stdio Transport

Start the server in Stdio mode:

```bash
mova mcp start --stdio
```

> **Note:** Stdio does not expose an HTTP endpoint. Messages are exchanged directly through standard input/output (stdin/stdout).

---

# Available MCP Methods

The server currently implements the following root methods:

| Method       | Description                          |
| ------------ | ------------------------------------ |
| `initialize` | Initializes the MCP session.         |
| `tools/list` | Returns the list of available tools. |
| `tools/call` | Executes a specific tool.            |

The JSON-RPC messages below are identical for both HTTP and Stdio transports.

---

# Example 1 — initialize

## JSON-RPC Request

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {}
}
```

## HTTP (curl)

```bash
curl -s -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' \
  | jq .
```

## Stdio

Start the server:

```bash
mova mcp start --stdio
```

Then send the same JSON-RPC message **as a single line**:

```text
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
```

---

# Example 2 — tools/list

## JSON-RPC Request

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {}
}
```

## HTTP (curl)

```bash
curl -s -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
  | jq .
```

## Stdio

```text
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
```

---

# Example 3 — tools/call → get_full_context

## JSON-RPC Request

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "get_full_context",
    "arguments": {
      "project": "ley-21719",
      "task": "analyze-contract"
    }
  }
}
```

## Sample Response

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "# Mova Context — ley-21719 / analyze-contract\n..."
      }
    ]
  }
}
```

## HTTP (curl)

```bash
curl -s -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_full_context","arguments":{"project":"ley-21719","task":"analyze-contract"}}}' \
  | jq .
```

## Stdio

```text
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_full_context","arguments":{"project":"ley-21719","task":"analyze-contract"}}}
```

---

# Example 4 — tools/call → search_context

## JSON-RPC Request

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "search_context",
    "arguments": {
      "query": "personal data",
      "domain": "legal"
    }
  }
}
```

## HTTP (curl)

```bash
curl -s -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"search_context","arguments":{"query":"personal data","domain":"legal"}}}' \
  | jq .
```

## Stdio

```text
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"search_context","arguments":{"query":"personal data","domain":"legal"}}}
```

---

# Postman Collection

Save the following content as **mova-mcp.postman_collection.json** and import it into Postman.

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
          "raw": "{\n  \"jsonrpc\": \"2.0\",\n  \"id\": 3,\n  \"method\": \"tools/call\",\n  \"params\": {\n    \"name\": \"get_full_context\",\n    \"arguments\": {\n      \"project\": \"local-tests\",\n      \"task\": \"create-project\"\n    }\n  }\n}"
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

# Operational Notes

## Stdio Message Format

When testing the server manually over **Stdio**, each JSON-RPC request **must be sent as a single-line JSON object**.

Sending formatted multi-line JSON directly into the terminal will cause the server to interpret each line as a separate message, resulting in:

* `-32700 (Parse error)`

Correct:

```text
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
```

Incorrect:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {}
}
```

## Strict JSON-RPC Format

All requests must conform to the JSON-RPC 2.0 specification. Missing or malformed required fields (`jsonrpc`, `id`, or `method`) will produce errors such as:

* `-32601 (Method not found)`
* `-32700 (Parse error)`

## Security

The HTTP server bundled with the CLI does not implement authentication or authorization. It is recommended to expose it only on local interfaces (`localhost` or `127.0.0.1`).

## Result Parity

The `get_full_context` tool returns exactly the same assembled context generated by:

```bash
mova run
```
