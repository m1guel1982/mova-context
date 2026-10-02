# Revisión {{REVIEW_TYPE}} — `{{PROJECT}}`
Consulta: {{QUERY}}

Ockham: ver `ockham-core.md`.

Alcance según tipo: **seguridad** (OWASP, authn/authz, secretos, datos sensibles, validación, dependencias con CVE) · **arquitectura** (responsabilidades, acoplamiento, mantenibilidad) · **calidad** (complejidad, duplicación, errores, tests) · **performance** (N+1, consultas/índices, bloqueos, CPU/memoria/I/O) · **completa** = las cuatro en ese orden.

Por hallazgo, una línea: `[Severidad] archivo:línea — problema → impacto → corrección mínima`. Si el arreglo es código, la función completa ya corregida, respetando la arquitectura y estilo existentes.
Entrega: hallazgos por severidad (Crítico | Alto | Medio | Bajo), luego Quick Wins.
