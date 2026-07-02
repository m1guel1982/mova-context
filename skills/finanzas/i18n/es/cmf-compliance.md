KISS + DRY: ver `kiss-dry-core.md`

# Regulación aplicable: CMF — Comisión para el Mercado Financiero (Chile)

> Este skill provee el conocimiento de dominio que usa el agente `compliance-officer`.
> El agente define cómo evaluar. Este skill define qué evaluar.

## Marco normativo principal

| Norma | Materia |
|-------|---------|
| Ley 18.045 | Mercado de Valores |
| Ley 18.046 | Sociedades Anónimas |
| Ley 19.913 | Prevención de Lavado de Activos (UAF) |
| NCG 461 CMF | Gestión de Riesgo Operacional |
| NCG 380 CMF | Ciberseguridad entidades financieras |
| Basilea III | Suficiencia de capital (aplicado por CMF) |

## Obligaciones clave por tipo de entidad

### Bancos e instituciones financieras (Ley 21.000 + NCG CMF)
```
□ Capital mínimo regulatorio (Basilea III: 10.5% capital/activos ponderados)
□ Ratio de liquidez LCR ≥ 100%
□ Gestión de riesgo operacional documentada (NCG 461)
□ Plan de continuidad del negocio (BCP) actualizado
□ Política de ciberseguridad aprobada por directorio (NCG 380)
□ Reporte de incidentes de seguridad a CMF en 24 horas
```

### Corredoras de bolsa / AGF
```
□ Patrimonio mínimo mantenido
□ Política de conflictos de interés documentada
□ Segregación de activos de clientes
□ Registro de operaciones por 10 años (Ley 18.045 Art. 29)
□ Divulgación de información esencial al mercado (hecho esencial)
```

### Prevención de Lavado de Activos (Ley 19.913 — UAF)
```
□ Oficial de cumplimiento designado
□ Programa de cumplimiento AML documentado
□ Debida diligencia de clientes (KYC) aplicada
□ Reporte de operaciones sospechosas (ROS) a UAF
□ Capacitación anual del personal
□ Umbral de reporte: transacciones ≥ USD 10.000 o equivalente
```

## Checklist por tipo de documento

### Política / Manual interno
```
□ Aprobación de directorio u órgano equivalente
□ Responsable designado para cada proceso crítico
□ Versión y fecha de última actualización
□ Proceso de revisión periódica establecido
□ Referencias a normativas CMF aplicables
```

### Contrato con cliente (cuenta corriente, inversión, crédito)
```
□ Tasas y comisiones expresadas en CAE y precio total (Ley 20.555)
□ Derechos del consumidor financiero declarados
□ Proceso de reclamo y SERNAC Financiero mencionado
□ Sin cláusulas abusivas (Ley 19.496)
```

## Sanciones de referencia

| Infracción | Sanción |
|------------|---------|
| No reportar hecho esencial | Multa + suspensión |
| Incumplimiento AML (UAF) | Hasta UF 5.000 por infracción |
| Deficiencia de capital | Intervención CMF |
| No reportar incidente seguridad | Multa NCG 380 |

## Formato de hallazgo obligatorio

```
HALLAZGO: [descripción breve]
NORMA: [Ley XX / NCG XXX CMF / Art. XX]
RIESGO: Alto / Medio / Bajo
CORRECCIÓN: [texto o acción concreta]
```
