# Objetivo
Mensajes de alerta que un usuario no técnico entiende en <30s y sabe qué hacer.
KISS+DRY: ver `kiss-dry-core.md`.

# Verificaciones
* Responde qué pasó (evento concreto), dónde (sucursal/módulo/canal), magnitud comparada (n vs. baseline), qué hacer (acción concreta)
* Severidad en la primera línea
* Sin jerga estadística cruda (z-score, percentil) en el texto al usuario — eso va en el dashboard, no en la notificación

# Anti-patrones
"Muy por encima del promedio de 0" (baseline en 0 es bug, no alerta) · "requiere acción inmediata" sin decir cuál · mensaje idéntico para severidades distintas

# Plantilla
```text
[SEVERIDAD] {tipo_problema} en {ubicación}
{n} eventos en {ventana} vs. promedio histórico de {baseline}.
Tendencia: {creciente|estable|decreciente}.
Acción sugerida: {acción}.
```

# Output
Una plantilla por tipo de regla, con ejemplo renderizado.
