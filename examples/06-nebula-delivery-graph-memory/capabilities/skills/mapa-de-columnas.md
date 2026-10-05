# Mapa de columnas del reporte de cobros

Columnas del reporte: {{REPORT_COLUMNS}}

Fuentes de datos: {{DATA_SOURCES}}

Claves de origen: {{SOURCE_KEYS}}

Formato de salida: {{OUTPUT_FORMAT}}

## Reglas
- Prioridad: {{PRIORITY_RULE}}
- Trazabilidad: {{TRACEABILITY_RULE}}
- Detección de pérdidas: {{LOSS_DETECTION}}

## Método
1. Para cada columna, localiza dónde nace el dato real y dónde el programado.
2. Señala la primera función donde uno de los dos deja de propagarse.
3. Propón el cambio mínimo, sin tocar lo que está en `exclude`.
