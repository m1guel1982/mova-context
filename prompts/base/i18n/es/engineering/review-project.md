# Revisar proyecto completo

Proyecto: `{{PROJECT}}`
Tipo: `{{REVIEW_TYPE}}` (seguridad | arquitectura | calidad | performance | completa)

Ockham: ver `ockham-core.md`.

## Alcance

### Seguridad
Evaluar:
- OWASP Top 10
- Autenticación y autorización
- Gestión de secretos y credenciales
- Datos sensibles
- Validación de entradas
- Dependencias vulnerables (CVE si es posible)

### Arquitectura
Evaluar:
- Separación de responsabilidades
- Acoplamiento y cohesión
- Patrones utilizados
- Organización del proyecto
- Mantenibilidad y escalabilidad

### Calidad
Evaluar:
- Complejidad
- Duplicación
- Manejo de errores
- Legibilidad
- Cobertura y calidad de tests existentes

### Performance
Evaluar:
- N+1
- Consultas ineficientes
- Índices faltantes
- Bloqueos evidentes
- Uso innecesario de CPU, memoria o I/O

### Tipo `completa`
Analizar las cuatro áreas anteriores en el siguiente orden:
1. Seguridad
2. Arquitectura
3. Calidad
4. Performance

## Para cada hallazgo

- Severidad (Crítico | Alto | Medio | Bajo)
- Archivo afectado
- Línea(s) afectada(s) (si pueden determinarse)
- Descripción del problema
- Impacto
- Solución recomendada
- Esfuerzo estimado (horas)

Si la solución requiere modificar código, retornar el código fuente completo de la función, clase o archivo afectado, ya corregido, manteniendo la arquitectura y el estilo existentes.

## Entrega

1. Una sección por área.
2. Hallazgos ordenados por severidad.
3. Quick Wins.
4. Resumen final en una tabla:

| Área | Archivo | Línea(s) | Severidad | Problema | Estado |
|------|----------|-----------|-----------|----------|--------|

---

## REGLA FINAL STRICTA - BLOQUE DE MEMORIA (OBLIGATORIO)

Al FINAL ABSOLUTO de tu respuesta, debes escribir OBLIGATORIAMENTE un bloque delimitado exactamente por ```memory y ```.
NO omitas este bloque bajo ninguna circunstancia.

Estructura obligatoria al final de la respuesta:

```memory
- [Severidad] Archivo:Breve resumen del hallazgo