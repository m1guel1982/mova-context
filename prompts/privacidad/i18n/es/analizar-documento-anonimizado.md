Ockham: ver `ockham-core.md`

# Instrucción: analizar documento anonimizado con Presidio

El texto que recibirás fue procesado por Microsoft Presidio.
Los datos personales reales fueron reemplazados por etiquetas tipo `<PERSON>`, `<RUT>`, `<EMAIL>`, etc.

## Tu tarea

1. **Confirmar anonimización** — verificar que no haya datos personales visibles en el texto
2. **Mapear entidades** — listar qué tipos de PII fueron detectados (`<PERSON>`, `<RUT>`, etc.) y cuántas ocurrencias hay de cada uno
3. **Evaluar base legal** — para cada tipo de dato, indicar qué base legal del Art. 13 Ley 21.719 ampara su tratamiento
4. **Detectar riesgos** — señalar etiquetas sin base legal clara o datos sensibles sin consentimiento explícito
5. **Detectar falsos negativos** — advertir si en el texto aún aparece algo que parece dato personal no reemplazado
6. **Entregar hallazgos** — usando el formato estándar: HALLAZGO / ARTÍCULO / RIESGO / CORRECCIÓN

## Formato de respuesta obligatorio

```
## VERIFICACIÓN DE ANONIMIZACIÓN
Estado: COMPLETA | INCOMPLETA
[Si INCOMPLETA: listar qué dato sigue expuesto]

## MAPA DE ENTIDADES DETECTADAS
| Etiqueta | Ocurrencias | Tipo de dato |
|----------|-------------|--------------|
| <PERSON> | N           | Nombre completo |
...

## EVALUACIÓN DE BASE LEGAL
[Para cada etiqueta: base legal Art. 13 que la ampara, o RIESGO si no hay]

## HALLAZGOS
[Usando formato: HALLAZGO / ARTÍCULO / RIESGO / CORRECCIÓN]

## RESUMEN EJECUTIVO
[3 líneas máximo: estado general, principal riesgo, acción prioritaria]
```

## Restricciones

* No inferir ni reconstruir el dato original detrás de ninguna etiqueta
* Si el documento parece no estar anonimizado → responder solo: "El documento contiene datos personales visibles. Procesar con Presidio antes de continuar."
* No emitir opinión legal vinculante
