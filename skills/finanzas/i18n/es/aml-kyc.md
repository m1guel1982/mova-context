KISS + DRY: ver `kiss-dry-core.md`

# Skill: AML / KYC — Prevención de Lavado de Activos y Financiamiento del Terrorismo

> Complementa `cmf-compliance.md` para análisis específico de AML/KYC.
> Usar ambos skills juntos para evaluación financiera completa.

## Marco aplicable
- Ley 19.913 (Chile) — Unidad de Análisis Financiero (UAF)
- FATF/GAFI Recommendations 2023
- Ley 18.045 Art. 52 bis — Personas Políticamente Expuestas (PEP)

## Debida diligencia de clientes (KYC)

### Estándar (clientes nuevos)
```
□ Verificación de identidad con documento oficial vigente
□ Verificación de RUT en Registro Civil
□ Declaración de origen lícito de fondos
□ Perfil de riesgo del cliente (bajo / medio / alto)
□ Propósito de la relación comercial declarado
```

### Reforzada (clientes de alto riesgo)
```
□ PEP: familiar o colaborador de funcionario público (aplicar siempre)
□ País de origen en lista GAFI de jurisdicciones no cooperantes
□ Operaciones en efectivo superiores a USD 10.000
□ Estructura societaria compleja o paraísos fiscales
□ Fuente de fondos documentada con evidencia adicional
□ Aprobación de gerencia o compliance para onboarding
```

## Señales de alerta (Red Flags)

```
Operaciones fraccionadas por debajo del umbral de reporte
Inconsistencia entre perfil económico y montos operados
Urgencia inusual o resistencia a proporcionar información
Uso de múltiples cuentas para una sola transacción
Terceros que pagan en nombre del cliente sin relación clara
Cambios frecuentes de beneficiario final sin justificación
```

## Umbrales de reporte a UAF

| Operación | Umbral | Plazo |
|-----------|--------|-------|
| Efectivo (compra/venta divisa) | ≥ USD 10.000 | 24 horas |
| Transferencia internacional | ≥ USD 10.000 | 24 horas |
| Operación sospechosa (cualquier monto) | Sin umbral | Inmediato |

## Formato de hallazgo

```
HALLAZGO: [descripción]
NORMA: Ley 19.913 Art. XX / GAFI Rec. XX
RIESGO: Alto / Medio / Bajo
CORRECCIÓN: [acción concreta]
```
