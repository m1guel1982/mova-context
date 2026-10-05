# Tarea: agregar columnas al reporte ({{PROJECT_NAME}})

Idioma de respuesta: {{LANG}}. Stack: {{STACK}}.

## Consulta
{{QUERY}}

## Funciones a modificar
- Mapeo: {{MAPPER_FUNCS}}
- Generación: {{GENERATOR_FUNCS}}

Claves de origen: {{SOURCE_KEYS}}
Prioridad: {{PRIORITY_RULE}}
Formato de salida: {{OUTPUT_FORMAT}}

## Punto de partida
Si la sección MEMORY contiene hallazgos de la tarea `analizar-trazabilidad`, úsalos como base (no repitas el análisis) y verifícalos contra el FOCUS antes de proponer código. Entrega el cambio mínimo como diff por `archivo::función`, y cierra con el bloque `memory` (qué cambiaste y qué queda pendiente).
