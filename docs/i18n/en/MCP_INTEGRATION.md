# Mova with Claude Code, Codex and HTTP

[README](README.md) · [Español](../es/MCP_INTEGRATION.md)

## 1. Register Mova (stdio)

```bash
claude mcp add --transport stdio --scope project --env MOVA_PROJECT_ROOT=<mova path> mova-context -- mova mcp start
```

`mova mcp start` defaults to stdio. With this, the agent **may** call `get_full_context`, `read_file`, `estimate_budget`… but nothing forces it to: its own tools (Read, Bash, Grep) stay outside Mova.

## 2. Extend the perimeter with hooks (Claude Code)

Claude Code supports `mcp_tool` hooks on `PreToolUse` and `PostToolUse`. Mova exposes two tools for that, sharing exactly the same policy as `read_file`:

- `check_read` (PreToolUse): returns `permissionDecision: "deny"`, with the reason, for paths outside the repo, excluded, or outside focus (`read_scope: focus`). When the read is allowed it returns `{}`: Mova never auto-approves, so the normal permission flow is kept.
- `sanitize_tool_output` (PostToolUse): for `Read`, replaces the output with the governed view (focused symbols only, excluded symbols stripped, secrets and PII sanitized) via `updatedToolOutput`. For other tools (Bash, Grep), it sanitizes the text.

Example `.claude/settings.json`. Adjust field names to the `mcp_tool` hook syntax of your Claude Code version:

```json
{
  "hooks": {
    "PreToolUse": [{ "matcher": "Read|Grep|Glob", "hooks": [{
      "type": "mcp_tool", "server": "mova-context", "tool": "check_read",
      "input": { "project": "04-nebula-delivery", "tool_name": "${tool_name}", "tool_input": "${tool_input}" } }] }],
    "PostToolUse": [{ "matcher": "Read|Bash", "hooks": [{
      "type": "mcp_tool", "server": "mova-context", "tool": "sanitize_tool_output",
      "input": { "project": "04-nebula-delivery", "tool_name": "${tool_name}", "tool_input": "${tool_input}", "tool_response": "${tool_response}" } }] }]
  }
}
```

Every decision is recorded in `projects/<p>/runs/<session run>/events.jsonl`.

**Limits:**
- @-mentions inject content without a tool call: use Claude Code permission `deny` rules for those too.
- Grep and Glob are checked at directory level only.
- A host that does not run hooks is not governed.

**Verification status:** the responses of both tools are covered by Mova tests. The end-to-end integration inside Claude Code is **not** verified yet.

## 3. Codex

Codex hooks can deny in PreToolUse and, in PostToolUse, replace a tool result with a block message. They cannot yet replace an MCP tool output with a sanitized version. Codex also reads files through shell commands, so enforcing `read_scope` would require interpreting commands, which Mova does not do.

In Codex, Mova is useful to build and validate the spec (`mova run`, closure check, evidence) and for the context the agent requests over MCP. It does not govern Codex's own reads.

## 4. HTTP

```bash
mova mcp start --http --port 3000                          # listens on 127.0.0.1
MOVA_HTTP_TOKEN=… mova mcp start --http --bind 0.0.0.0     # non-loopback: token required
```

- Without `MOVA_HTTP_TOKEN`, a non-loopback bind refuses to start.
- With a token, every request must send `Authorization: Bearer <token>`.
- Non-localhost `Origin` headers are rejected.

The server exposes reads **and writes** of the project repo: do not publish it on a network without a token.
