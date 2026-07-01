# CLI — Mova Context

The CLI packages a project's context into a single block of text ready to use with any LLM.

Precompiled binaries for Linux, macOS (Intel + Apple Silicon), and Windows in `cli/dist/`.

---

## Commands

```bash
mova list                                    # list available projects
mova run [project] [task]                    # generate context
mova memory [project] "llm response"         # save session to memory.md
mova init [name]                             # create new project
mova mcp start [--port 3000]                 # start MCP server
mova search "query"                          # search knowledge
mova memory-archive [project]               # archive old memory entries
```

---

## Typical usage

```bash
# 1. See available projects
mova list

# 2. Generate project context
mova run privacy-law analyze-contract > context.txt

# 3. Copy context.txt and paste into Claude / ChatGPT / Gemini

# 4. Save the LLM response to memory.md
mova memory privacy-law "$(pbpaste)"
```

---

## Redirect to file

```bash
mova run my-project my-task > context.txt
```

`context.txt` contains the full context ready to paste into any web LLM.

---

## Initialize a new project

```bash
mova init my-company
```

Creates the basic structure:

```text
projects/my-company/
├── project.json
└── memory.md
```

---

## MCP server

```bash
mova mcp start           # default port 3000
mova mcp start --port 4000
```

Exposes all project contexts as an MCP server compatible with Claude Desktop and other tools.

---

## Environment variables

```bash
MOVA_ROOT=/path/to/mova-context   # repo root directory
MOVA_PORT=3000                    # MCP server port
```

---

## Installation

```bash
# macOS / Linux
chmod +x cli/dist/mova-linux-amd64
sudo mv cli/dist/mova-linux-amd64 /usr/local/bin/mova

# Windows
# Add cli/dist/mova-windows-amd64.exe to PATH
```
