KISS + DRY: ver `kiss-dry-core.md`

# Detección de PII con Microsoft Presidio

## Entidades detectadas por Presidio (relevantes para Chile)

| Entidad Presidio | Dato real | Sensible |
|-----------------|-----------|----------|
| `PERSON` | Nombre completo | No |
| `EMAIL_ADDRESS` | Correo electrónico | No |
| `PHONE_NUMBER` | Teléfono | No |
| `LOCATION` | Dirección, ciudad, región | No |
| `DATE_TIME` | Fecha de nacimiento, fechas médicas | Depende |
| `CREDIT_CARD` | Número de tarjeta | No (financiero) |
| `IBAN_CODE` | Cuenta bancaria | No (financiero) |
| `MEDICAL_LICENSE` | Número colegio médico | No |
| `RUT` | Rol Único Tributario *(custom)* | No |
| `HEALTH_DATA` | Diagnósticos, medicamentos *(custom)* | **Sí** |

## Nivel de confianza (score)

Presidio asigna score 0.0–1.0 a cada detección:

```
score ≥ 0.85  → alta confianza, anonimizar siempre
score 0.6–0.85 → revisar manualmente antes de procesar
score < 0.6   → probable falso positivo, descartar
```

## Falsos negativos frecuentes (Presidio puede omitirlos)

```
- RUTs escritos con puntos pero sin guión: 12345678K
- Nombres en mayúsculas: JUAN PÉREZ GONZÁLEZ
- Apodos o nombres de usuario dentro del texto
- Números de ficha o código interno que identifican a una persona
- Domicilios descritos narrativamente: "vive frente a la Plaza de Armas"
```

## Operadores de anonimización recomendados

```python
# Reemplazar con etiqueta tipo
operators = {
    "PERSON":        OperatorConfig("replace", {"new_value": "<PERSON>"}),
    "EMAIL_ADDRESS": OperatorConfig("replace", {"new_value": "<EMAIL>"}),
    "PHONE_NUMBER":  OperatorConfig("replace", {"new_value": "<TELEFONO>"}),
    "LOCATION":      OperatorConfig("replace", {"new_value": "<DIRECCION>"}),
    "DATE_TIME":     OperatorConfig("replace", {"new_value": "<FECHA>"}),
    "RUT":           OperatorConfig("replace", {"new_value": "<RUT>"}),
}
```

## Regla de verificación post-anonimización

Antes de enviar al LLM, verificar que el texto anonimizado no contiene:
- Secuencias de 8-9 dígitos seguidas de letra (RUT sin formato)
- Patrones `@dominio.com`
- Secuencias de 16 dígitos (tarjeta)
- Nombres propios obvios (lista mínima de nombres comunes)
