# Verification log (what was actually run)

Date: 2026-10-05 · Host: Linux amd64 sandbox · Go 1.24.13 · dependencies resolved with `GOPROXY=direct` (the `replace` mirrors of `golang.org/x/*` in `src/go.mod` to `github.com/golang/*` worked; `go.mod`/`go.sum` unchanged).

## Verified (executed)
| Item | Result |
|---|---|
| `make build` in a clean copy | OK → `dist/mova` (41 MB), `mova list` runs |
| `go test -tags <GO_TAGS> ./...` | 21 packages `ok`, 3 without tests, **0 FAIL** (re-run with `-count=1` after the final identifier renames in tests) |
| `mova run --count 02-pii-compliance-governance` | 7153 tokens (cl100k_base) |
| `mova run 02-… --diagram --export png` | PNG generated: 20,014 → 7,153 tok (−64 %); Focus 20,101 → 5,325 (−74 %); PII 171/1,694 pseudonymized; 199 repeated lines collapsed |
| `mova budget 04-nebula-delivery --focus` | Focus 1,965 → 779 tok (−60.4 %); task 2: 598 (−69.6 %) (tiny fictional repo) |
| `mova run --count 08-nebula-release-gate` / `agents list` / `agents run --all` | 3 agents, 4,994 tok total |
| MCP stdio `tools/list` | 26 tools; all documented in `FUNCTIONS.md` |
| HTTP+MCP flow `examples/08/run-demo.sh` | OK; shared memory written/read; output in `examples/08-*/evidence/` |
| `mova context-trace --repo <local path>` and `<GitHub URL>` | OK; asks `[Y/n]` on stdin (use `< /dev/null`) |
| `mova trace 03-…`, `memory-read`, `show config nebula-demo` | OK |

## Built but NOT executed
`linux-arm64`, `macos-amd64`, `macos-arm64` (Mach-O), `windows-amd64.exe` (PE32+): compile cleanly with `CGO_ENABLED=0`; no machine available to run them. Do not claim support until `ci.yml` is green on those runners.

## Not verified
`mova chat` against a real model · `/save` `/delete` `/diagram` HTTP routes · `chat_completion` · `release.yml` and `ci.yml` on GitHub · PII precision/recall · Windows installers.

## Defects found and fixed
1. `COMMANDS` docs described `is_group`/`members`; code uses `projects/<group>/config.json` (`group`,`agents`).
2. Broken links (`source.md`, `context-trace.md`, `MCP_HTTP_TOOLS.md`), `cd src && make install` (Makefile is at the root).
3. Missing model profile `nebula-demo` (documented, absent) → created `config/models/ollama/nebula-demo.json` (local Ollama).
4. `.gitignore` ignored `/Makefile`; `make test` ignored build tags; `install` did not build.
5. Example 08 initially used `dry_run: true`, which blocks MCP reads → `false` (see `FUNCTIONS.md`).
6. A demo GIF with wrong figures was regenerated from the real output above.
7. Private-case identifiers removed from a unit-test fixture, `PROJECT_JSON.md` and example text.
