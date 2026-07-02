# Caso: finanzas-cmf — El mismo agente, diferente regulación

## El punto clave

```
privacidad-presidio:   compliance-officer + deteccion-pii + cumplimiento-ley-21719
finanzas-cmf:          compliance-officer + cmf-compliance + aml-kyc
laboral (próximo):     compliance-officer + codigo-del-trabajo
salud (próximo):       compliance-officer + ley-20584-pacientes + cumplimiento-ley-21719
tributario (próximo):  compliance-officer + sii-tributario
```

**El agente no cambió. Solo los skills.**

---

## Documento de prueba

Guarda como `politica-aml-original.txt`:

```text
POLÍTICA DE PREVENCIÓN DE LAVADO DE ACTIVOS
Versión 2.1 — Enero 2026
InverFácil Corredores de Bolsa SpA

1. OFICIAL DE CUMPLIMIENTO
El oficial de cumplimiento designado es Roberto Fuentes Mora, RUT 14.567.890-2,
email rfuentes@inverfacil.cl, teléfono +56 2 2345 6789.

2. DEBIDA DILIGENCIA
Se solicitará a cada cliente nuevo: cédula de identidad vigente, declaración
de origen de fondos y clasificación de riesgo. Para clientes con más de $50.000.000
en activos se aplicará debida diligencia reforzada.

3. REPORTE A UAF
Se reportarán operaciones en efectivo superiores a $5.000.000.

4. CAPACITACIÓN
El personal recibirá capacitación anual.
```

---

## Paso 1 — Anonimizar (misma lógica que el caso de privacidad)

```bash
python anonimizar.py politica-aml-original.txt
```

**Salida:** `politica-aml-anonimizada.txt`

```text
POLÍTICA DE PREVENCIÓN DE LAVADO DE ACTIVOS
Versión 2.1 — Enero 2026
InverFácil Corredores de Bolsa SpA

1. OFICIAL DE CUMPLIMIENTO
El oficial de cumplimiento designado es <PERSON>, RUT <RUT>,
email <EMAIL>, teléfono <TELEFONO>.

2. DEBIDA DILIGENCIA
Se solicitará a cada cliente nuevo: cédula de identidad vigente, declaración
de origen de fondos y clasificación de riesgo. Para clientes con más de $50.000.000
en activos se aplicará debida diligencia reforzada.

3. REPORTE A UAF
Se reportarán operaciones en efectivo superiores a $5.000.000.

4. CAPACITACIÓN
El personal recibirá capacitación anual.
```

---

## Paso 2 — Generar contexto

```bash
mova run finanzas-cmf evaluar-politica-aml > contexto.txt

# Verificar agente y skills cargados
grep "<!-- " contexto.txt
```

```
<!-- core: yagni-core -->
<!-- agent: compliance-officer -->
<!-- core: kiss-dry-core -->
<!-- skill: cmf-compliance -->
<!-- skill: aml-kyc -->
<!-- core: ockham-core -->
<!-- prompt: evaluar-cumplimiento-financiero -->
```

El mismo `compliance-officer` que analiza contratos laborales y políticas de privacidad.

---

## Paso 3 — Combinar y enviar al LLM

```bash
cat contexto.txt politica-aml-anonimizada.txt > prompt-final.txt

# Claude/GPT: pegar en el chat
# Llama local:
ollama run llama3.1 "$(cat prompt-final.txt)"
```

---

## Respuesta esperada del LLM

```
## TIPO DE DOCUMENTO
Política AML/KYC — Prevención de Lavado de Activos

## REGULACIÓN APLICADA
Ley 19.913 (UAF Chile) + GAFI Recommendations 2023

## CHECKLIST
| Obligación | Norma | Estado | Riesgo |
|------------|-------|--------|--------|
| Oficial de cumplimiento designado | Ley 19.913 Art. 3 | ✓ | - |
| Debida diligencia estándar definida | GAFI Rec. 10 | ✓ | - |
| Debida diligencia reforzada (criterio) | GAFI Rec. 10 | Parcial | Medio |
| Umbral de reporte UAF | Ley 19.913 Art. 3 | ✗ | Alto |
| Identificación de PEP | GAFI Rec. 12 | ✗ | Alto |
| Frecuencia de capacitación documentada | Ley 19.913 Art. 3 | Parcial | Medio |
| Señales de alerta documentadas | GAFI Rec. 20 | ✗ | Alto |
| Proceso de ROS (reporte operación sospechosa) | Ley 19.913 Art. 3 | ✗ | Alto |

## HALLAZGOS

### HALLAZGO 1 — Umbral de reporte incorrecto
- Descripción: El umbral definido es $5.000.000 CLP, pero la Ley 19.913 exige
  reportar operaciones en efectivo ≥ USD 10.000 (equivalente aprox. $10.000.000 CLP)
- Norma: Ley 19.913 Art. 3 / Circular UAF 049
- Riesgo: **Alto**
- Corrección: "Se reportarán a la UAF todas las operaciones en efectivo iguales
  o superiores a USD 10.000 o su equivalente en moneda nacional, dentro de las
  24 horas siguientes a la operación."

### HALLAZGO 2 — Sin identificación de PEP
- Descripción: La política no menciona el tratamiento de Personas Políticamente
  Expuestas, obligatorio bajo GAFI Rec. 12 y Ley 18.045 Art. 52 bis
- Norma: GAFI Recomendación 12 / Ley 18.045 Art. 52 bis
- Riesgo: **Alto**
- Corrección: Agregar sección: "Se aplicará debida diligencia reforzada a todos
  los clientes identificados como PEP, sus familiares directos y colaboradores
  cercanos, independiente del monto de activos."

### HALLAZGO 3 — Sin catálogo de señales de alerta
- Descripción: La política no incluye señales de alerta (red flags) que permitan
  al personal identificar operaciones sospechosas
- Norma: GAFI Recomendación 20
- Riesgo: **Alto**
- Corrección: Agregar anexo con señales de alerta mínimas (ver skill aml-kyc
  sección "Señales de alerta").

### HALLAZGO 4 — Criterio de debida diligencia reforzada incompleto
- Descripción: Solo se considera el monto de activos ($50M). No se consideran
  origen geográfico, estructura societaria ni tipo de cliente
- Norma: GAFI Recomendación 10
- Riesgo: **Medio**
- Corrección: Ampliar criterios: "Se aplicará DDA reforzada a: (a) clientes
  con activos ≥ $50.000.000, (b) clientes de jurisdicciones GAFI no cooperantes,
  (c) estructuras con beneficiario final no identificable."

### HALLAZGO 5 — Capacitación sin frecuencia ni evidencia
- Descripción: Se menciona "capacitación anual" pero sin mecanismo de evidencia
  (lista de asistencia, evaluación, registro)
- Norma: Ley 19.913 Art. 3
- Riesgo: **Medio**
- Corrección: "La capacitación anual incluirá evaluación escrita. El Oficial de
  Cumplimiento mantendrá registro de asistencia y resultados por mínimo 5 años."

## ¿REQUIERE REVISIÓN EXPERTA?
**Sí** — 4 hallazgos de riesgo Alto. La política no puede entrar en vigencia
sin corregir los umbrales de reporte y agregar identificación de PEP.

## RESUMEN EJECUTIVO
Política con estructura básica pero con 4 incumplimientos graves (Ley 19.913 + GAFI).
Principal riesgo: umbral de reporte incorrecto y ausencia de protocolo PEP.
Acción prioritaria: corregir umbral UAF y agregar sección PEP antes de aprobar.
```

---

## Lo que demostró este ejemplo

```
ANTES del refactor:
  abogado-privacidad.md  →  solo sirve para privacidad
  abogado-finanzas.md    →  habría que crear uno nuevo
  abogado-laboral.md     →  otro nuevo
  ...

DESPUÉS del refactor:
  compliance-officer.md  →  sirve para todos los dominios
  + cumplimiento-ley-21719.md   →  privacidad Chile
  + cmf-compliance.md           →  finanzas Chile
  + aml-kyc.md                  →  lavado de activos
  + codigo-del-trabajo.md       →  laboral Chile
  + ley-20584-pacientes.md      →  salud Chile
  + sii-tributario.md           →  tributario Chile
  + [cualquier regulación nueva] → agregar un skill
```

DRY aplicado a agentes: **un agente, N dominios**.
