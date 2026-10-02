# Corregir o mejorar — `{{PROJECT}}`
Stack: {{STACK}}

Ockham: ver `ockham-core.md`.

Consulta: {{QUERY}}

Sobre el FOCUS entregado (no supongas código que no ves):
1. Identifica la causa raíz; si el bug se ve en una vista, corrígelo en el punto común de los invocadores.
2. Aplica la escalera de `kiss-dry-core` (nativo → estándar → ya instalado → código propio).
3. No introduzcas: {{SKIPPED_ABSTRACTIONS}}
4. Escala a una solución más compleja solo si: {{UPGRADE_TRIGGER}}

Salida: por cada cambio, `archivo::símbolo` y el bloque con la función completa ya corregida; cierre según `ockham-core`.
