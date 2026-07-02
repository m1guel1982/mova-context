KISS + DRY: ver `kiss-dry-core.md`

# Regulación aplicable: Ley 21.719 — Protección de Datos Personales (Chile)

> Este skill provee el conocimiento de dominio que usa el agente `compliance-officer`.
> El agente define cómo evaluar. Este skill define qué evaluar.

## Bases legales válidas para el tratamiento (Art. 13)

```
a) Consentimiento explícito del titular
b) Ejecución de contrato del que el titular es parte
c) Obligación legal del responsable
d) Interés vital del titular o de terceros
e) Interés legítimo del responsable (NO aplica a datos sensibles)
f) Misión de interés público
```

## Datos sensibles — requieren consentimiento explícito siempre (Art. 16)

```
Origen racial o étnico · opiniones políticas · convicciones religiosas
Datos biométricos · datos de salud · vida u orientación sexual
Datos genéticos · antecedentes penales
```

## Obligaciones del responsable

| Obligación | Artículo | Qué verificar |
|------------|----------|---------------|
| Informar al titular | Art. 14 | finalidad, responsable, destinatarios, derechos, plazo |
| Derechos ARCOP | Art. 16 | canal habilitado para Acceso, Rectificación, Cancelación, Oposición, Portabilidad |
| Medidas de seguridad | Art. 20 | proporcionales al riesgo y tipo de dato |
| EIPD | Art. 22 | obligatoria para tratamientos de alto riesgo |
| Registro de actividades | Art. 30 | mantener registro actualizado |
| Transferencia internacional | Art. 25 | garantías equivalentes a la ley chilena |

## Sanciones de referencia

| Infracción | Sanción máxima |
|------------|---------------|
| Tratar sin base legal | 5.000 UTM |
| No informar al titular | 2.500 UTM |
| No implementar seguridad | 5.000 UTM |
| Transferir sin garantías | 5.000 UTM |
| Datos sensibles sin consentimiento | 10.000 UTM |

## Checklist por tipo de documento

### Contrato laboral / de servicios
```
□ Finalidad declarada (Art. 14 a)
□ Base legal identificada (Art. 13)
□ Derechos ARCOP informados (Art. 16)
□ Plazo de retención (Art. 14 c)
□ Transferencias a terceros con garantías (Art. 25)
□ Datos sensibles: consentimiento explícito (Art. 16)
```

### Política de privacidad / términos
```
□ Identidad del responsable y DPO si corresponde
□ Finalidades específicas (no genéricas)
□ Destinatarios de los datos
□ Canal para ejercer derechos
□ Transferencias internacionales y mecanismos
□ Derecho a retirar el consentimiento
□ Decisiones automatizadas (si aplica)
```

### Formulario de recolección
```
□ Consentimiento claro, sin letra chica
□ Casilla por separado para cada finalidad
□ Sin casillas pre-marcadas
□ Campos obligatorios vs. opcionales identificados
□ Enlace a política de privacidad completa
```

## Formato de hallazgo obligatorio

```
HALLAZGO: [descripción breve]
ARTÍCULO: Art. XX Ley 21.719
RIESGO: Alto / Medio / Bajo
CORRECCIÓN: [texto de cláusula sugerida o acción concreta]
```
