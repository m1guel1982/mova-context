# Objetivo
Generar tests consistentes y mínimos.
KISS+DRY: ver `kiss-dry-core.md`.

# Estructura
```
describe('Modulo') { describe('metodo()') {
  it('should [resultado] when [condición]')
  it('should throw [error] when [condición]')
}}
```

# Cobertura mínima por función
Caso exitoso · input inválido/nulo · caso límite (vacío, 0) · error del sistema (DB down, 404)

# Reglas
Un `expect` por test cuando sea posible · datos hardcodeados explícitos, sin factories mágicas · mocks solo para HTTP/DB/filesystem · nombre del test = documentación
