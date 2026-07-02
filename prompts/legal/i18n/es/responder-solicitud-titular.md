# Responder Solicitud de Titular — Ley 21.719

El cliente ha enviado la siguiente solicitud:

{{MENSAJE_CLIENTE}}

**Canal:** {{CANAL}}

---

## Instrucciones

1. Identifica qué derecho está ejerciendo el titular según los Arts. 17-28 de {{LEY}}
2. Indica el plazo legal que tiene la empresa para responder
3. Especifica si se requiere verificación de identidad previa
4. Redacta una respuesta apropiada para el canal indicado (máximo 200 palabras)
5. Proporciona el registro de auditoría en formato JSON

**Formato de respuesta:**

```
DERECHO EJERCIDO: [artículo y nombre del derecho]
PLAZO LEGAL: [días hábiles]
VERIFICACIÓN: [sí/no y por qué]
RESPUESTA AL TITULAR: [texto de la respuesta]
AUDITORÍA: [JSON con case_id, type, deadline, legal_basis]
```
