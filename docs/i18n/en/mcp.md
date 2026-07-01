# MCP — Mova Context v3

Mova Context can be exposed as a server compatible with the **MCP (Model Context Protocol)**. This allows Claude Desktop, Cursor, Windsurf, and other MCP-compatible tools to access your project context directly and perform searches, eliminating the need to manually copy and paste context.

The server supports two transport modes: **Stdio** (standard input/output for native AI client integration) and **HTTP** (JSON-RPC for testing with Postman or web development).

---

# Start the Server

## HTTP Mode (Default – Ideal for Postman and Web Development)

Starts a web server on the specified port (default: `3000`).

```bash
mova mcp start
mova mcp start --port 4000
```

## Stdio Mode (Required for Claude Desktop and Code Editors)

The AI launches the binary locally and communicates directly through standard input/output for fast, native integration.

```bash
mova mcp start --stdio
```

---

# What the MCP Server Exposes

### Full Context

Assembles agents, skills, prompts, and active project memory into a single response.

### Global Search

Performs indexed searches across the entire knowledge base (agents, skills, and prompts).

### Historical Memory Access

Reads the current project memory (`memory.md`) or the complete archived memory history.

### Workflow Guide

Provides direct access to the repository workflow guide (`workflow.md`).

---

# Claude Desktop Configuration

To integrate Mova Context with Claude Desktop, configure the server using the `--stdio` transport.

Edit your `claude_desktop_config.json` file and add:

```json
{
  "mcpServers": {
    "mova-context": {
      "command": "mova",
      "args": ["mcp", "start", "--stdio"]
    }
  }
}
```

> **Note:** Make sure the `mova` executable is available in your system `PATH`. Otherwise, replace `"mova"` with the absolute path to the executable.

---

# Exposed Tools

The server fully implements the MCP `tools/list` method and exposes the following tools through `tools/call`.

| Tool | Arguments (* = Required) | Description |
|------|---------------------------|-------------|
| `get_full_context` | `project*`, `task` | Returns the fully assembled project context (equivalent to `mova run`). This is the primary tool. |
| `get_knowledge` | `kind`, `domain`, `name*`, `lang` | Retrieves a specific agent, skill, or prompt by name and domain. |
| `get_memory` | `project*` | Reads the current active project memory (`memory.md`). |
| `get_memory_all` | `project*` | Reads the active project memory together with the complete archived history. |
| `get_workflow` | `lang` | Returns the repository workflow guide (`workflow.md`) from the project root. |
| `search_context` | `query*`, `domain` | Searches keywords or concepts across the entire knowledge base. |

---

# Why Use MCP Instead of Copy and Paste?

### Without MCP

```text
mova run project task > context.txt
→ Open context.txt
→ Manually copy thousands of lines
→ Paste them into the AI chat, risking prompt saturation
```

### With MCP

```text
You tell Claude:
"Review the create-auth task in the backend project."

→ Claude detects that the get_full_context tool is available.
→ It invokes the tool automatically in the background.
→ It instantly receives fresh, clean, and fully assembled project context.
```