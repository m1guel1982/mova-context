# Caso completo: Presidio + bge-m3 + Ley 21.719

## El problema

Una empresa recibe documentos con datos personales (contratos laborales, políticas de privacidad,
formularios de clientes) y necesita evaluarlos contra la **Ley 21.719** de Chile.

No puede pegar esos documentos directamente en Claude o GPT:

```
Contrato de Juan Pérez, RUT 12.345.678-9, email jperez@empresa.cl
→ datos personales reales viajan a un servidor externo
→ posible infracción Art. 20 Ley 21.719 (medidas de seguridad)
→ posible brecha GDPR si el proveedor está en la UE
```

## La solución

Flujo de tres pasos antes de que el LLM vea cualquier dato:

```
DOCUMENTO ORIGINAL
       │
       ▼
┌─────────────────────┐
│  Microsoft Presidio  │  ← detecta: PERSON, RUT, EMAIL, PHONE, etc.
│  (corre local)       │  ← nunca sale de tu máquina
└─────────────────────┘
       │ texto anonimizado
       ▼
┌─────────────────────┐
│  bge-m3 (Ollama)    │  ← vectoriza el texto anonimizado
│  embedding local     │  ← busca los agents/skills relevantes
└─────────────────────┘
       │ contexto seleccionado
       ▼
┌─────────────────────┐
│  Mova Context        │  ← arma el contexto con agents + skills + prompt
│  mova run            │
└─────────────────────┘
       │ contexto.txt
       ▼
┌─────────────────────┐
│  Claude / GPT /      │  ← recibe solo texto anonimizado
│  Llama (local)       │  ← nunca ve RUTs, emails, nombres reales
└─────────────────────┘
       │
       ▼
ANÁLISIS DE CUMPLIMIENTO LEY 21.719
```

**Dato clave:** el LLM recibe `<PERSON>`, `<RUT>`, `<EMAIL>` — nunca los datos reales.
Presidio y bge-m3 corren 100% en tu infraestructura.

---

## Qué se instala

### 1. Python 3.9+ y Presidio

```bash
pip install presidio-analyzer presidio-anonymizer

# Modelo de lenguaje español (spaCy)
python -m spacy download es_core_news_lg

# Modelo inglés (si procesas documentos en inglés también)
python -m spacy download en_core_web_lg
```

### 2. Ollama + modelos locales

```bash
# Instalar Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Modelo de embedding multilingüe (español + inglés)
ollama pull bge-m3             # 1.2 GB — funciona con ES y EN sin configuración extra

# Reranker (par natural de bge-m3)
ollama pull bge-reranker-v2-m3 # 568 MB — mejora precisión de búsqueda

# LLM para generación (opcional si usas Claude/GPT vía API)
ollama pull llama3.1           # 4.7 GB — si quieres todo 100% local
```

### 3. Mova Context CLI

```bash
# Windows
curl -LO https://github.com/tu-org/mova-context/releases/latest/download/mova-windows-amd64.exe

# macOS / Linux
curl -LO https://github.com/tu-org/mova-context/releases/latest/download/mova-linux-amd64
chmod +x mova-linux-amd64 && sudo mv mova-linux-amd64 /usr/local/bin/mova
```

### 4. Verificar instalación

```bash
python -c "from presidio_analyzer import AnalyzerEngine; print('Presidio OK')"
ollama list   # debe mostrar bge-m3 y bge-reranker-v2-m3
mova list     # debe mostrar privacidad-presidio
```

---

## Documento de prueba

Guarda este archivo como `contrato-original.txt`:

```text
CONTRATO DE TRABAJO

Entre la empresa Acme SpA, representada por María González Rojas, RUT 8.765.432-1,
en adelante "el Empleador", y don Carlos Soto Muñoz, RUT 15.234.567-8,
domiciliado en Av. Providencia 1234, Santiago, email csoto@gmail.com,
teléfono +56 9 8765 4321, en adelante "el Trabajador".

CLÁUSULA 1 — OBJETO
El Trabajador prestará servicios de desarrollo de software a partir del 1 de febrero de 2026.

CLÁUSULA 2 — REMUNERACIÓN
Remuneración mensual de $2.500.000.

CLÁUSULA 3 — DATOS PERSONALES
Los datos del Trabajador serán almacenados en nuestros sistemas y compartidos
con nuestro proveedor de RRHH ubicado en Estados Unidos para procesamiento de nómina.

Firmado en Santiago, 21 de enero de 2026.
```

---

## Paso 1 — Anonimizar con Presidio

Guarda esto como `anonimizar.py`:

```python
from presidio_analyzer import AnalyzerEngine, RecognizerRegistry
from presidio_analyzer.nlp_engine import NlpEngineProvider
from presidio_anonymizer import AnonymizerEngine
from presidio_anonymizer.entities import OperatorConfig
import sys

# Configurar motor NLP para español
provider = NlpEngineProvider(nlp_configuration={
    "nlp_engine_name": "spacy",
    "models": [{"lang_code": "es", "model_name": "es_core_news_lg"}]
})
nlp_engine = provider.create_engine()

analyzer  = AnalyzerEngine(nlp_engine=nlp_engine, supported_languages=["es"])
anonymizer = AnonymizerEngine()

# Entidad RUT personalizada (Chile)
from presidio_analyzer import PatternRecognizer, Pattern
rut_recognizer = PatternRecognizer(
    supported_entity="RUT",
    patterns=[
        Pattern("RUT con puntos y guión", r"\b\d{1,2}\.\d{3}\.\d{3}-[\dkK]\b", 0.9),
        Pattern("RUT sin puntos",         r"\b\d{7,8}-[\dkK]\b",                 0.8),
    ],
    supported_language="es"
)
analyzer.registry.add_recognizer(rut_recognizer)

# Leer documento
text = open(sys.argv[1]).read()

# Detectar PII
results = analyzer.analyze(
    text=text,
    language="es",
    entities=["PERSON","EMAIL_ADDRESS","PHONE_NUMBER","LOCATION","DATE_TIME","RUT","CREDIT_CARD"],
    score_threshold=0.6
)

# Anonimizar
operators = {
    "PERSON":        OperatorConfig("replace", {"new_value": "<PERSON>"}),
    "EMAIL_ADDRESS": OperatorConfig("replace", {"new_value": "<EMAIL>"}),
    "PHONE_NUMBER":  OperatorConfig("replace", {"new_value": "<TELEFONO>"}),
    "LOCATION":      OperatorConfig("replace", {"new_value": "<DIRECCION>"}),
    "DATE_TIME":     OperatorConfig("replace", {"new_value": "<FECHA>"}),
    "RUT":           OperatorConfig("replace", {"new_value": "<RUT>"}),
    "CREDIT_CARD":   OperatorConfig("replace", {"new_value": "<TARJETA>"}),
}

anonymized = anonymizer.anonymize(text=text, analyzer_results=results, operators=operators)

# Reporte de entidades detectadas
print("=== ENTIDADES DETECTADAS ===")
for r in sorted(results, key=lambda x: x.start):
    print(f"  {r.entity_type:20} score={r.score:.2f}  [{text[r.start:r.end]}]")

print("\n=== TEXTO ANONIMIZADO ===")
print(anonymized.text)

# Guardar resultado
with open("contrato-anonimizado.txt", "w") as f:
    f.write(anonymized.text)
print("\n→ Guardado en contrato-anonimizado.txt")
```

```bash
python anonimizar.py contrato-original.txt
```

**Salida esperada:**

```
=== ENTIDADES DETECTADAS ===
  PERSON               score=0.85  [María González Rojas]
  RUT                  score=0.90  [8.765.432-1]
  PERSON               score=0.85  [Carlos Soto Muñoz]
  RUT                  score=0.90  [15.234.567-8]
  LOCATION             score=0.80  [Av. Providencia 1234, Santiago]
  EMAIL_ADDRESS        score=0.95  [csoto@gmail.com]
  PHONE_NUMBER         score=0.85  [+56 9 8765 4321]
  DATE_TIME            score=0.85  [1 de febrero de 2026]
  DATE_TIME            score=0.85  [21 de enero de 2026]

=== TEXTO ANONIMIZADO ===
CONTRATO DE TRABAJO

Entre la empresa Acme SpA, representada por <PERSON>, RUT <RUT>,
en adelante "el Empleador", y don <PERSON>, RUT <RUT>,
domiciliado en <DIRECCION>, email <EMAIL>,
teléfono <TELEFONO>, en adelante "el Trabajador".

CLÁUSULA 1 — OBJETO
El Trabajador prestará servicios de desarrollo de software a partir del <FECHA>.

CLÁUSULA 2 — REMUNERACIÓN
Remuneración mensual de $2.500.000.

CLÁUSULA 3 — DATOS PERSONALES
Los datos del Trabajador serán almacenados en nuestros sistemas y compartidos
con nuestro proveedor de RRHH ubicado en Estados Unidos para procesamiento de nómina.

Firmado en Santiago, <FECHA>.

→ Guardado en contrato-anonimizado.txt
```

---

## Paso 2 — Generar contexto con Mova

```bash
mova run privacidad-presidio analizar-contrato > contexto.txt
```

**Verificar que los tres cores están presentes:**

```bash
grep "<!-- core:" contexto.txt
```

```
<!-- core: yagni-core -->
<!-- core: kiss-dry-core -->
<!-- core: ockham-core -->
```

---

## Paso 3 — Enviar al LLM

**Opción A — Claude / ChatGPT (pegar en el chat):**

```bash
# Combinar contexto + documento anonimizado
cat contexto.txt > prompt-final.txt
echo "" >> prompt-final.txt
echo "---" >> prompt-final.txt
echo "## DOCUMENTO A ANALIZAR" >> prompt-final.txt
cat contrato-anonimizado.txt >> prompt-final.txt

# Copiar al clipboard
cat prompt-final.txt | pbcopy       # macOS
cat prompt-final.txt | xclip        # Linux
cat prompt-final.txt | clip         # Windows
```

Pegar en Claude o ChatGPT y enviar.

**Opción B — Llama 3.1 local (100% offline):**

```bash
cat prompt-final.txt | ollama run llama3.1
```

**Opción C — vía MCP (automatizado):**

```bash
# Terminal 1
mova mcp start --port 3000

# Terminal 2 — enviar documento anonimizado directamente
DOCUMENTO=$(cat contrato-anonimizado.txt)
curl -s -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d "{\"tool\":\"run_context\",\"arguments\":{\"project\":\"privacidad-presidio\",\"task\":\"analizar-contrato\",\"document\":\"$DOCUMENTO\"}}" \
  | jq .
```

---

## Respuesta esperada del LLM

```
## VERIFICACIÓN DE ANONIMIZACIÓN
Estado: COMPLETA
No se detectan datos personales visibles en el texto.

## MAPA DE ENTIDADES DETECTADAS
| Etiqueta   | Ocurrencias | Tipo de dato     |
|------------|-------------|------------------|
| <PERSON>   | 2           | Nombre completo  |
| <RUT>      | 2           | RUT (Chile)      |
| <DIRECCION>| 1           | Domicilio        |
| <EMAIL>    | 1           | Correo electrónico |
| <TELEFONO> | 1           | Teléfono         |
| <FECHA>    | 2           | Fecha            |

## EVALUACIÓN DE BASE LEGAL
- <PERSON>, <RUT>, <DIRECCION>, <TELEFONO>: Art. 13 b) — ejecución del contrato ✓
- <EMAIL>: Art. 13 b) — ejecución del contrato ✓
- <FECHA>: Art. 13 b) — ejecución del contrato ✓

## HALLAZGOS

### HALLAZGO 1
- Descripción: Transferencia de datos a proveedor en EE.UU. sin garantías declaradas
- Artículo: Art. 25 Ley 21.719 — Transferencia internacional de datos
- Riesgo: **Alto**
- Corrección: Agregar cláusula: "La transferencia se realiza bajo Cláusulas Contractuales
  Tipo aprobadas, garantizando nivel de protección equivalente al exigido por esta ley."

### HALLAZGO 2
- Descripción: No se informa al trabajador sobre sus derechos ARCOP
- Artículo: Art. 14 Ley 21.719 — Deber de información
- Riesgo: **Alto**
- Corrección: Agregar cláusula: "El Trabajador tiene derecho a acceder, rectificar,
  cancelar y oponerse al tratamiento de sus datos. Para ejercer estos derechos,
  contactar a <EMAIL_DPO>."

### HALLAZGO 3
- Descripción: No se declara plazo de retención de los datos
- Artículo: Art. 14 c) Ley 21.719
- Riesgo: **Medio**
- Corrección: "Los datos serán conservados durante la vigencia del contrato y
  hasta 5 años después de su término, según exige la legislación laboral."

## ¿REQUIERE EIPD?
**Evaluar** — la transferencia internacional a EE.UU. y el procesamiento automatizado
de nómina pueden requerir Evaluación de Impacto (Art. 22). Consultar con DPO.

## RESUMEN EJECUTIVO
Anonimización completa. 3 hallazgos: 2 de riesgo Alto (transferencia internacional
sin garantías, ausencia de información de derechos) y 1 Medio (plazo de retención).
Acción prioritaria: agregar cláusulas de transferencia internacional y derechos ARCOP.
```

---

## Paso 4 — Guardar memoria de la sesión

```bash
mova memory privacidad-presidio '```memory
## 2026-01-21 — análisis contrato laboral
**Hecho:** analizado contrato-original.txt, 9 entidades PII detectadas, 3 hallazgos
**Resuelto:** anonimización completa verificada
**Pendiente:** corregir cláusula de transferencia internacional y agregar derechos ARCOP
**Decisiones:** todos los contratos nuevos deben incluir cláusula ARCOP antes de firmar
**LLM Errors:** ninguno
```'
```

---

## Estructura de archivos completa

```text
mova-context/
├── agents/privacidad/i18n/
│   ├── es/
│   │   ├── yagni-core.md
│   │   ├── analista-presidio.md      ← interpreta etiquetas Presidio
│   │   └── abogado-privacidad.md     ← evalúa Ley 21.719
│   └── en/
│       ├── yagni-core.md
│       ├── presidio-analyst.md
│       └── privacy-lawyer.md
│
├── skills/privacidad/i18n/
│   ├── es/
│   │   ├── kiss-dry-core.md
│   │   ├── deteccion-pii.md          ← cómo interpretar Presidio
│   │   └── cumplimiento-ley-21719.md ← checklist Art. 13-30
│   └── en/
│       ├── kiss-dry-core.md
│       ├── pii-detection.md
│       └── gdpr-compliance.md
│
├── prompts/privacidad/i18n/
│   ├── es/
│   │   ├── ockham-core.md
│   │   ├── analizar-documento-anonimizado.md
│   │   └── evaluar-cumplimiento.md
│   └── en/
│       ├── ockham-core.md
│       └── analyze-anonymized-document.md
│
└── projects/privacidad-presidio/
    ├── project.json                  ← LLM + embedding + reranker
    └── memory.md
```

---

## Variantes del project.json

### Todo local (sin internet, máxima privacidad)

```json
{
  "llm_profile": {
    "type": "local",
    "provider": "ollama",
    "model": "llama3.1",
    "base_url": "http://localhost:11434"
  },
  "embedding": {
    "provider": "ollama",
    "model": "bge-m3",
    "base_url": "http://localhost:11434"
  },
  "reranker": {
    "provider": "ollama",
    "model": "bge-reranker-v2-m3",
    "base_url": "http://localhost:11434",
    "min_score": 0.7
  }
}
```

### Claude para generación, embeddings locales (híbrido recomendado)

```json
{
  "llm_profile": {
    "type": "powerful",
    "provider": "claude",
    "model": "claude-sonnet-4-6"
  },
  "embedding": {
    "provider": "ollama",
    "model": "bge-m3",
    "base_url": "http://localhost:11434"
  }
}
```

---

## Lo que nunca cambia entre variantes

```text
workflow.md                            ← igual
agents/privacidad/i18n/es/*.md         ← igual
skills/privacidad/i18n/es/*.md         ← igual
prompts/privacidad/i18n/es/*.md        ← igual
anonimizar.py                          ← igual
```

Solo cambia `llm_profile` / `embedding` / `reranker` en `project.json`.
