// help.go — texto de `mova` sin argumentos. Separado de main.go a
// propósito: main.go es el dispatcher, este archivo es solo texto.
package main

func usage() {
	consolePrint(`mova — Mova Context v3

  mova run           [project] [task]        generate context for LLM
  mova memory        [project] "response"    save session to memory.md
  mova memory-read   [project]               print active memory
    --all                                    include archives
    --month 2024-01                          specific archive month
  mova memory-archive [project]              archive old entries
    --days N                                 keep N days active (default 30)
  mova list                                  list all projects
  mova init          [name]                  create project
  mova search        "query" [domain]        search knowledge

  mova budget        [project] [task]        estimate token/USD cost of a project's context
    --focus                                  also compare full repo vs. focus-only cost
    Reads config/prices.json (hot-reloaded), writes mova-budget-report.md.
    100% local — tiktoken-go only, no LLM call, nothing leaves this machine.

  mova mcp start                             start MCP server
    --port 3000                              run as HTTP server (default)
    --stdio                                  run as Stdio server (for Claude/Cursor)
  mova memory-clear  [project]               delete ALL memory
    --archived                               delete only archived months
    --keep-active                            delete archives, keep memory.md
    --date 2024-06-15                        delete a specific day
    --from 2024-06-01 --to 2024-06-30        delete date range
    --yes                                    skip confirmation
  mova memory-config [project] [action] [value]
    enable | disable                         toggle auto-archive
    days N                                   set retention days (1, 10, 30, 90...)
    confirm true|false                       toggle confirmation on delete

  Local models (Ollama, LM Studio, vLLM...) — see docs/i18n/en/COMMANDS.md
  mova config        <provider>              set the active provider (ollama, lmstudio...)
  mova show          config [model]          show active provider, or one model's config
  mova install       llama3.1,mistral        install models (shows a progress bar)
  mova model-list                            list installed models
  mova remove        llama3.1,mistral        remove installed models
  mova chat          [project] [task]        interactive chat with a local model
    set -model <name>                        switch models mid-chat, keeps history
    /memory                                  save last exchange to memory.md
    exit | quit                              leave the chat

  MOVA_ADAPTER=db  MOVA_DSN=postgres://... mova run project task

  MOVA_PROJECT_ROOT=/path/to/project  mova mcp start --stdio
    Needed when an MCP client (Claude Desktop, Cursor) launches mova from
    a working directory outside the project. MOVA_PROJECT_PATH also works
    and skips the workflow.md search entirely. See docs/i18n/en/COMMANDS.md.
`)
}
