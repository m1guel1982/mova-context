# config/mcp_clients/ — ready-to-copy MCP client snippets

Full guide with concrete examples: `docs/i18n/{es,en}/MCP_INTEGRATION.md`.

Each file here is a **complete, valid** config file for its target
client — copy it verbatim to the path below, then replace
`MOVA_PROJECT_ROOT`'s placeholder with this installation's absolute
path (the directory that contains this repo's own `workflow.md`).

| File | Copy to | Client |
|---|---|---|
| `claude_desktop.json` | macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`<br>Windows: `%APPDATA%\Claude\claude_desktop_config.json` | Claude Desktop |
| `claude_code.json` | `.mcp.json` at your OTHER project's repo root (project-scoped) | Claude Code |
| `cursor.json` | `.cursor/mcp.json` at your OTHER project's repo root | Cursor |
| `windsurf.json` | `.windsurf/mcp.json` at your OTHER project's repo root | Windsurf |
| `vscode.json` | `.vscode/mcp.json` at your OTHER project's repo root | VS Code (Copilot Chat, agent mode) |

If the target file already has other MCP servers configured, merge
the `"mova-context"` entry into the existing `"mcpServers"`/`"servers"`
object instead of overwriting the file.

**Why `MOVA_PROJECT_ROOT` instead of relying on the working directory:**
every one of these hosts launches `mova mcp start --stdio` as a
subprocess from ITS OWN working directory (your other project's repo,
not this one) — Mova's own root auto-detection (`workflow.md` search)
would fail there. Setting `MOVA_PROJECT_ROOT` is a direct override
(see `runtime.FindRoot`) that always wins, regardless of cwd.

**ChatGPT is not in this list on purpose:** as of this writing, ChatGPT
only accepts remote (HTTPS) MCP servers — it cannot spawn a local
`stdio` process the way the clients above do. Mova's own `--stdio`
transport isn't reachable from ChatGPT without deploying it behind a
public HTTPS endpoint first (a tunnel like `cloudflared`/`ngrok` for
testing, a real deployment for anything else) — see
`MCP_INTEGRATION.md`'s "ChatGPT / remote HTTP" section for the honest
current state and the workaround.
