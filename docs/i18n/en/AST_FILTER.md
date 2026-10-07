# AST filter — `file::kind=name1,name2`

Inside a `focus` or `exclude` target: extracts (`focus`) or removes (`exclude`) ONLY those symbols from
`file`, using a real AST parser (Tree-Sitter, not text heuristics). It's additive — a normal `focus`
(`"file.py"`, `"src/**/*.go"`) still works the same.

## Supported `kind`

| `kind` | Matches | Alias |
|---|---|---|
| `func` | function/method/constructor | `method` |
| `class` | class (struct/enum/trait in Rust; `type` in Go) | — |
| `interface`, `struct`, `enum`, `trait`, `type` | that exact construct | — |
| `namespace` | namespace/package/module | `package` |
| `const` | constant | — |
| `var` | variable | `variable`, `property` |
| `impl` | `impl` block (Rust) | — |
| `macro` | macro | — |

A `kind` not supported by the file's language doesn't break anything — that target just matches nothing.

## Real example (see `examples/03-tokenomics-context-trace`)

```json
{ "exclude": ["checkout.js::func=procesarPagoLegacy"] }
```

Analyzes `checkout.js` in full and only removes that function's body — the rest of the file stays intact.

## Dependency graph

The same `file::kind=names` entries in `focus` and `exclude` feed the dependency graph (`graph` in the task): see `PROJECT_JSON.md` § Dependency graph.

## Dependency closure

Before releasing a context, Mova uses the graph's parser to check that no symbol in `focus` calls or references an **excluded** symbol (`exclude`). If one does, it is a conflict: with `dependency_policy: "block"` (default) nothing is released; with `"warn"` it is released and the conflict is recorded in `manifest.json`; `accept_missing` accepts it with a written reason. Dependencies that stay **outside** the context are reported too (non-blocking).

Limits: static calls and references only (no dynamic dispatch, reflection or string-built calls); only relative imports are followed; a name that cannot be resolved is not reported.
