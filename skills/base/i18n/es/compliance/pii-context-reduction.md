# Skill: PII enmascarada en el contexto
Mova puede reemplazar datos personales por pseudónimos `[PII_xxxxxxxx]` (mismo valor → mismo tag).
- Trata cada tag como valor opaco: no intentes adivinar, reconstruir ni completar el original. Mismo tag = mismo dato.
- El enmascarado es heurístico (forma + entropía): puede dejar PII sin marcar (falso negativo) o marcar no-PII como fechas `2024-07-30` (falso positivo). No es anonimización jurídica ni garantiza cumplir {{REGULATION}}.
- Si ves PII sin enmascarar, no la repitas en tu respuesta salvo que la consulta lo exija, y adviértelo.
