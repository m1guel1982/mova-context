# Filtro AST — `archivo::kind=nombre1,nombre2`

Dentro de un target de `focus` o `exclude`: extrae (`focus`) o quita (`exclude`) SOLO esos símbolos de
`archivo`, con un parser AST real (Tree-Sitter, no heurísticas de texto). Es un agregado — un `focus`
normal (`"archivo.py"`, `"src/**/*.go"`) sigue funcionando igual.

## `kind` soportados

| `kind` | Filtra | Alias |
|---|---|---|
| `func` | función/método/constructor | `method` |
| `class` | clase (struct/enum/trait en Rust; `type` en Go) | — |
| `interface`, `struct`, `enum`, `trait`, `type` | esa construcción exacta | — |
| `namespace` | namespace/paquete/módulo | `package` |
| `const` | constante | — |
| `var` | variable | `variable`, `property` |
| `impl` | bloque `impl` (Rust) | — |
| `macro` | macro | — |

Un `kind` no soportado por el lenguaje del archivo no rompe nada — ese target simplemente no encuentra
nada.

## Ejemplo real (ver `examples/03-tokenomics-context-trace`)

```json
{ "exclude": ["checkout.js::func=procesarPagoLegacy"] }
```

Analiza `checkout.js` completo y solo quita el cuerpo de esa función — el resto del archivo queda intacto.
