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
