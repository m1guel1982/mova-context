# Supported Formats

**Recommended entry point:** for creating or editing any file (or directory) in
this list, use the single `save` tool / `/save` chat command — it picks the
right generator automatically from the extension, so you never need to know
which tool below handles which format. Every tool listed here (`write_file`,
`generate_word_contract`, `generate_pdf_document`, `generate_excel_report`,
`generate_vector_graphic`, `create_directory`) still works exactly as before
and is kept for backward compatibility — this document stays the
authoritative, per-format reference for what each one (and `save`, which
shares the same underlying writers) actually supports. See
`docs/i18n/{es,en}/COMMANDS.md`'s "`save`" section for the full contract
(`path`/`directory`/`content`/`append`/`overwrite`).

This is the authoritative list of every file/document format that `mova chat`,
MCP (stdio and HTTP), and the HTTP transport can **create, edit, read, or
generate**, and through which tool. If a format isn't listed here, it isn't
supported — `write_file`/`patch_file`/`save` return a clear `Unsupported file type`
error (in English) for anything outside the allowlist below.

All three transports (stdio MCP, HTTP MCP, `chat_completion`) go through the
exact same `Process()` dispatcher (`mcp/server.go`) and the exact same
`documents/` package underneath — there is no separate code path per
transport, so this list applies identically to all of them.

---

## 0. Where files & directories are created (in plain terms)

Every create/edit tool below (`write_file`, `patch_file`, `create_directory`,
`save`, and every `generate_*` tool) resolves its target the same simple way:

1. **You gave an absolute path** (`/home/user/carpeta`, `C:/carpeta`,
   `C:\carpeta`, or a `\\server\share` UNC path) → used exactly as given,
   on Linux, macOS, or Windows. A Windows-style path asked of a
   Linux/macOS server returns a clear error instead of silently failing —
   there is no `C:\` drive there.
2. **You gave nothing** → the project's `repo` from `project.json`
   (the existing default).
3. **You gave a bare folder name** (e.g. "config", with no `/`) → Mova
   searches the project's `repo` tree for an existing folder with that
   exact name.
   - Found nowhere → created fresh under `repo`.
   - Found once → reused.
   - **Found more than once → Mova asks which one you meant**, listing
     every full path, instead of guessing.
4. **You gave an explicit relative path** (`output/reports`,
   `output/reports/file.md`) → resolved directly under `repo`, no search —
   an explicit multi-segment path is unambiguous by construction.

For file tools, only the **folder portion** of what you gave is
search-worthy — the file name itself is never treated as a folder to look
for. `create_directory` is recursive (like `mkdir -p`): it creates every
missing parent folder in one call.

---

## 1. Text & source-code files — `read_file` / `write_file` / `patch_file`

Implemented in `documents/textfile.go`. `read_file` has no extension
restriction (it reads any file). `write_file` and `patch_file` are
restricted to the allowlist below — anything else is rejected with:

```
Unsupported file type: <ext>. Supported extensions: <the list below>
```

### Plain / structured text

| Extension | Format | Validated before write? |
|---|---|---|
| `.txt` | Plain text | no |
| `.md` | Markdown | no |
| `.json` | JSON | **yes** — `encoding/json` |
| `.yml` / `.yaml` | YAML | no |
| `.xml` | XML | **yes** — `encoding/xml` (well-formedness) |
| `.csv` | CSV | **yes** — `encoding/csv` (consistent field count per record) |
| `.toml` | TOML | no |
| `.ini` | INI config | no |
| `.env` | Env file | no |
| `.log` | Log file | no |

### Programming languages

| Extension | Language | Validated before write? |
|---|---|---|
| `.js` | JavaScript | no |
| `.ts` | TypeScript | no |
| `.py` | Python | no |
| `.go` | Go | **yes** — `go/parser` (real syntax check, stdlib only) |
| `.cs` | C# | no |
| `.java` | Java | no |
| `.php` | PHP | no |
| `.rb` | Ruby | no |
| `.rs` | Rust | no |
| `.c` | C | no |
| `.cpp` | C++ | no |
| `.h` | C/C++ header | no |
| `.kt` | Kotlin | no |
| `.swift` | Swift | no |
| `.sh` | Shell script | no |

### Web

| Extension | Format | Validated before write? |
|---|---|---|
| `.html` | HTML | no |
| `.css` | CSS | no |
| `.sql` | SQL | no |

"Validated" means content is checked for correctness with a **standard-library-only**
check before the file is written — no third-party parser/compiler is used
anywhere in this project. Formats without a cheap stdlib validator are
written as-is, the same way any plain text editor would.

---

## 2. Office documents & media — dedicated tools

Implemented in `documents/docx.go`, `documents/xlsx.go`, `documents/pdf.go`,
`documents/svg.go`, `documents/read_layer.go`, `documents/diffusion.go`.
These are binary/structured formats, so each has its own tool rather than
going through `write_file`:

| Extension | Format | Create | Read | Tool(s) |
|---|---|---|---|---|
| `.docx` | Microsoft Word (OOXML) | yes | yes | `generate_word_contract` (markdown → docx) / `read_document_layer` |
| `.xlsx` | Microsoft Excel (OOXML) | yes | yes | `generate_excel_report` (typed JSON → xlsx) / `read_document_layer` |
| `.pdf` | PDF 1.4 | yes | best-effort | `generate_pdf_document` (HTML/CSS text → pdf) / `read_document_layer` |
| `.svg` | Scalable Vector Graphics | yes | — (read via `read_file`, it's plain XML text) | `generate_vector_graphic` |
| `.png` / `.jpg` | Raster image | yes (via local diffusion server) | — | `trigger_diffusion_image` |

`.docx`, `.xlsx`, and `.pdf` are hand-written using only Go's standard
library (`archive/zip`, `encoding/xml`) — no third-party office-format
dependency. `.pdf` reading is best-effort text extraction (FlateDecode
streams + `Tj`/`TJ` operators): it reliably reads PDFs this tool generates
and most simple text PDFs, but it is not a full PDF parser, so scanned or
exotically-encoded PDFs may return no text. `.png`/`.jpg` generation
requires a local diffusion server (AUTOMATIC1111-compatible) configured at
`config/models/diffusion/config.json` — Mova Context does not run the
diffusion model itself, only the HTTP call and file save.

---

## 3. Knowledge & config formats — internal to the engine

These aren't user-facing "create a file" formats, but they are files the
engine itself reads as part of every `mova run` / `get_full_context` call:

| Extension | Used for |
|---|---|
| `.json` | `project.json` (project config), `config/models/*/*.json` (provider/model config) |
| `.md` | agents, skills, prompts, `memory.md`, `workflow.md`, all documentation |
| `.sql` | schema lookups via the `SQLResolver` focus resolver (`CREATE TABLE` extraction) |

---

## 4. Directories — `create_directory`

Implemented in `documents/directory.go`. Creates a directory and every
missing parent in one call (`os.MkdirAll` — recursive, "mkdir -p"
semantics), cross-platform on Linux, macOS, and Windows. Resolves its
`path` argument exactly like the rules in section 0 above — absolute path,
bare-name disk search with disambiguation, explicit relative path, or the
project's `repo` by default. A no-op if the directory already exists.

---

## Quick reference — every extension in one place

```
Text/config:  .txt .md .json .yml .yaml .xml .csv .toml .ini .env .log
Code:         .js .ts .py .go .cs .java .php .rb .rs .c .cpp .h .kt .swift .sh
Web:          .html .css .sql
Office:       .docx .xlsx .pdf
Media:        .svg .png .jpg
Directories:  create_directory (recursive, any name/depth)
```

29 text/code/web extensions via `write_file`/`patch_file`/`read_file`, plus
5 office/media formats via their dedicated tools, plus recursive directory
creation via `create_directory` — all reachable identically from
`mova chat`, MCP (stdio), MCP (HTTP), and the HTTP transport, and all
resolved through the same absolute/search/relative/default path logic in
section 0.
