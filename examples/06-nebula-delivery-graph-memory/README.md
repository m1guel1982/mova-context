# Ejemplo 06 — cierre de dependencias (proyecto `04-nebula-delivery`)

[English](README_EN.md)

Repo ficticio de 182 líneas (Node.js). Es pequeño a propósito: aquí no importa el ahorro de tokens. Lo que importa es si la especificación de cada tarea está **cerrada** respecto de lo que excluye.

```bash
mova run 04-nebula-delivery analizar-trazabilidad   # liberado
mova run 04-nebula-delivery agregar-columnas        # liberado
mova run 04-nebula-delivery recalcular-tarifas      # BLOQUEADO
```

`recalcular-tarifas` selecciona `calcularTarifa`, pero excluye `src/legacy/tarifasV1.js`. `calcularTarifa` llama a `tarifaPlanaV1` cuando `ruta.catalogo === 'v1'`, así que Mova no libera el contexto:

```
[recalcular-tarifas] src/facturacion/tarifas.js::calcularTarifa -> src/legacy/tarifasV1.js::tarifaPlanaV1 (call, excluido)
```

Hay tres maneras de resolverlo, y cada una queda registrada en `manifest.json`:
1. Incluir `src/legacy/tarifasV1.js::func=tarifaPlanaV1` en el `focus`.
2. Aceptarlo con una razón: `"accept_missing": [{"symbol": "tarifaPlanaV1", "reason": "el catálogo v1 está fuera de alcance"}]`.
3. Usar `"dependency_policy": "warn"`: se libera, y el conflicto queda en la evidencia.

En las tareas liberadas, `manifest.json → dependency_closure.outside` lista lo que queda fuera del contexto sin estar excluido. Por ejemplo, `elegirRuta → puntuarRuta` o `estimarCombustible → leerSensor`. Es información para que quien escribe el focus decida, no un bloqueo.

**Lo que NO demuestra:** que el focus sea el correcto para la tarea, ni que un modelo resuelva mejor la tarea con este contexto. Eso requiere probar tareas reales con un grupo de control (ver la auditoría). Tampoco detecta llamadas dinámicas.
