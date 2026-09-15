Puedes ejecutar mova context-trace directamente sobre cualquier repositorio público de GitHub o sobre una ruta local. Si ejecutas el siguiente comando, Mova clonará en un directorio temporal el repositorio target, aplicará los filtros de exclusión e ignorados, podará docstrings innecesarios para optimizar el presupuesto de tokens y generará la salida formateada en output-mova trace-fastApi/


mova context-trace \
  --repo https://github.com/fastapi/fastapi \
  --export pdf \
  --task "solve_dependencies get_dependant in fastapi/dependencies/utils.py" \
  --prune-docstrings \
  --ignore "docs/**, tests/**, *.lock, docs_src/**, .github/**, docs/en/**"


Esta es la salida por consola, junto con los reportes y el diagrama generados automáticamente en el mismo directorio:



[ 0%] Preparing target repository...
[ 15%] Target ready for analysis: C:\Users\mandr\AppData\Local\Temp\mova-trace-2147425420
[ 85%] Temporary directory removed: C:\Users\mandr\AppData\Local\Temp\mova-trace-2147425420
[ 90%] Generating reports...
[100%] Done.
Mova Context Trace (Remote Analysis)
----------------------------------
INPUT
  Repository           https://github.com/fastapi/fastapi
  Branch               master
  Task                 solve_dependencies get_dependant in fastapi/dependencies/utils.py
  Agent                mova-cli
  Target model         n/a
  Policy author        system:default
  Active --ignore patterns: docs/**, tests/**, *.lock, docs_src/**, .github/**, docs/en/**

FOCUS
  Included             3139 file(s)  (repository scope considered, before content-level discovery)
  Excluded             29 file(s) (suggestion: .git)
  Note: 3011 of the 3139 in-scope file(s) could not be read as text/image during discovery (binary/unreadable/empty) and are reported as EXCLUDED below - Focus is the SCOPE considered, Discovery is what was actually analyzed.

CONTEXT GOVERNANCE (audit mode - no active project.json)
  Sanitizer            DISABLED (Audit Mode - see GOVERNANCE below for per-file SANITIZED/BLOCKED decisions)
  PII Masking          ON  (14 file(s) with sensitive-data indicators (see files_with_any_indicator vs files_would_sanitize in pii-audit-log.json))
  Cache Layout Guard   OFF
  Circuit Breaker      OFF

GOVERNANCE
+--------------------------------------------------------------------+
| STATUS: DISCOVERY ONLY (Default Global Policy)                     |
|                                                                    |
| DISCOVERED            128 files          207,095 tok               |
| CANDIDATE              14 files          117,187 tok               |
| ALLOWED                 0 files                0 tok               |
| WOULD SANITIZE         14 files          117,187 tok               |
| BLOCKED                 0 files                0 tok               |
| EXCLUDED            3,125 files           89,908 tok               |
|                                                                    |
| SENDABLE CONTEXT BREAKDOWN (audit mode - nothing written to disk): |
|   Repository context (scanned)            207,095 tok              |
|   Policy allowed (safe now)                     0 tok              |
|   Would require sanitization              117,187 tok              |
|   Actually transformed/redacted                 0 tok              |
|                                                                    |
| Safe to send RIGHT NOW: 0 tok (100.0% reduction)                   |
| Sendable AFTER sanitization is applied: 117,187 tok (43.4% reducti |
|                                                                    |
| Files with an indicator: 14 = would_sanitize 14 + blocked 0 + dete |
|                                                                    |
| Est. cost - safe now (lowest): No cost (local execution) (anthropi |
| Est. cost - after sanitization (lowest): No cost (local execution) |
| Local (Ollama/LM Studio/vLLM): $0 in both scenarios                |
+--------------------------------------------------------------------+

BUDGET
  N/A (no active project.json)

  Total tokens: 117,187 (encoding: cl100k_base)

TOP DIRECTORIES BY TOKEN USAGE
  Directory                            Tokens    Files  % total
  fastapi                             104,178       11   88.9%
  scripts                              13,009        3   11.1%
  TOTAL                               117,187     3139  100.0%

PROJECTED COST (estimate if the 117,187 candidate tokens were sent - audit mode, nothing sent - not the whole repository)
  anthropic (claude-haiku) $0.09 USD
  anthropic (claude-sonnet) $0.35 USD
  google (gemini-flash)    $0.0088 USD
  google (gemini-pro)      $0.15 USD
  local (lm-studio)        No cost (local execution)
  local (ollama)           No cost (local execution)
  local (vllm)             No cost (local execution)
  openai (gpt-4.1)         $0.23 USD
  openai (gpt-4o)          $0.29 USD
  openai (gpt-4o-mini)     $0.02 USD
  Local (Ollama/LM Studio/vLLM)  $0 (local execution)
  All figures above are technical ESTIMATES (public price lists + cl100k_base), not absolute costs.

RELEVANCE
  Task: solve_dependencies get_dependant in fastapi/dependencies/utils.py
  Top-ranked files (BM25 + structural + AST signal - see docs' FAQ for the exact formula):
  #1   fastapi/dependencies/utils.py                      score 1018.95  role producer
  #2   fastapi/_compat/v2.py                              score 3.00     role producer
  #3   fastapi/applications.py                            score 3.00     role producer
  #4   fastapi/encoders.py                                score 3.00     role producer
  #5   fastapi/openapi/utils.py                           score 3.00     role producer
  #6   fastapi/security/http.py                           score 3.00     role producer
  #7   fastapi/security/oauth2.py                         score 3.00     role producer
  #8   fastapi/security/open_id_connect_url.py            score 3.00     role producer
  #9   fastapi/utils.py                                   score 3.00     role producer
  #10  scripts/docs.py                                    score 3.00     role producer


OUTPUT
  context-report.pdf
  context-diagram.png
  pii-audit-log.json

EXECUTION
  Execution ID       18d59b76aade82d0-1b096cefb5a060ff
  Repository state   50113da

  All artifacts were generated from this execution result.

Generate a 'project.json' file with the detected suggestions? [Y/n]:
