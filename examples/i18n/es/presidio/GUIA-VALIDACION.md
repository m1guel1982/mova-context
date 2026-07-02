# Guía de validación — Presidio + Ley 21.719

Checklist para verificar que el caso completo funciona correctamente.

---

## PARTE 1 — Instalación

### ✅ Presidio

```bash
python -c "
from presidio_analyzer import AnalyzerEngine
from presidio_anonymizer import AnonymizerEngine
print('Presidio OK')
"
```

**Resultado esperado:** `Presidio OK`
**Error común:** `ModuleNotFoundError` → `pip install presidio-analyzer presidio-anonymizer`

```bash
python -m spacy info es_core_news_lg | grep "Name"
```

**Resultado esperado:** `Name: es_core_news_lg`
**Error común:** modelo no encontrado → `python -m spacy download es_core_news_lg`

---

### ✅ Ollama

```bash
ollama list
```

**Resultado esperado:**
```
NAME                         SIZE
bge-m3                       1.2 GB
bge-reranker-v2-m3           568 MB
```

**Error común:** vacío → `ollama pull bge-m3 && ollama pull bge-reranker-v2-m3`

```bash
# Verificar que Ollama responde
curl -s http://localhost:11434/api/tags | python -m json.tool | grep "name"
```

---

### ✅ Mova Context

```bash
mova list | grep privacidad-presidio
```

**Resultado esperado:**
```
  privacidad-presidio    [es] Análisis de documentos con PII anonimizada...
    tasks: analizar-contrato, analizar-terminos, evaluar-formulario, evaluar-politica
```

**Error común:** no aparece → ejecutar desde el directorio `mova-context/`

---

## PARTE 2 — Anonimización con Presidio

### ✅ Test básico de detección

```bash
python -c "
from presidio_analyzer import AnalyzerEngine, RecognizerRegistry
from presidio_analyzer.nlp_engine import NlpEngineProvider
from presidio_analyzer import PatternRecognizer, Pattern

provider = NlpEngineProvider(nlp_configuration={
    'nlp_engine_name': 'spacy',
    'models': [{'lang_code': 'es', 'model_name': 'es_core_news_lg'}]
})
analyzer = AnalyzerEngine(nlp_engine=provider.create_engine(), supported_languages=['es'])

# Agregar reconocedor de RUT
rut = PatternRecognizer(
    supported_entity='RUT',
    patterns=[Pattern('rut', r'\b\d{1,2}\.\d{3}\.\d{3}-[\dkK]\b', 0.9)],
    supported_language='es'
)
analyzer.registry.add_recognizer(rut)

texto = 'Contactar a Juan Pérez, RUT 12.345.678-9, email jperez@test.cl'
resultados = analyzer.analyze(text=texto, language='es', score_threshold=0.6)
for r in resultados:
    print(f'{r.entity_type}: {texto[r.start:r.end]} (score={r.score:.2f})')
"
```

**Resultado esperado:**
```
PERSON: Juan Pérez (score=0.85)
RUT: 12.345.678-9 (score=0.90)
EMAIL_ADDRESS: jperez@test.cl (score=0.95)
```

**Error común:** `PERSON` no detectado → el modelo spaCy no está instalado correctamente

---

### ✅ Test de anonimización completa

```bash
echo "María González, RUT 8.765.432-1, vive en Av. Las Condes 123, Santiago" \
  | python -c "
import sys
from presidio_analyzer import AnalyzerEngine, RecognizerRegistry
from presidio_analyzer.nlp_engine import NlpEngineProvider
from presidio_analyzer import PatternRecognizer, Pattern
from presidio_anonymizer import AnonymizerEngine
from presidio_anonymizer.entities import OperatorConfig

provider = NlpEngineProvider(nlp_configuration={
    'nlp_engine_name': 'spacy',
    'models': [{'lang_code': 'es', 'model_name': 'es_core_news_lg'}]
})
analyzer  = AnalyzerEngine(nlp_engine=provider.create_engine(), supported_languages=['es'])
anonymizer = AnonymizerEngine()

rut = PatternRecognizer(
    supported_entity='RUT',
    patterns=[Pattern('rut', r'\b\d{1,2}\.\d{3}\.\d{3}-[\dkK]\b', 0.9)],
    supported_language='es'
)
analyzer.registry.add_recognizer(rut)

texto = sys.stdin.read().strip()
results = analyzer.analyze(text=texto, language='es', score_threshold=0.6)
ops = {
    'PERSON':   OperatorConfig('replace', {'new_value': '<PERSON>'}),
    'RUT':      OperatorConfig('replace', {'new_value': '<RUT>'}),
    'LOCATION': OperatorConfig('replace', {'new_value': '<DIRECCION>'}),
}
anon = anonymizer.anonymize(text=texto, analyzer_results=results, operators=ops)
print(anon.text)
"
```

**Resultado esperado:**
```
<PERSON>, RUT <RUT>, vive en <DIRECCION>
```

---

## PARTE 3 — Contexto Mova

### ✅ Estructura de archivos

```bash
find mova-context/agents/privacidad mova-context/skills/privacidad \
     mova-context/prompts/privacidad -name "*.md" | sort
```

**Resultado esperado:**
```
mova-context/agents/privacidad/i18n/en/presidio-analyst.md
mova-context/agents/privacidad/i18n/en/privacy-lawyer.md
mova-context/agents/privacidad/i18n/en/yagni-core.md
mova-context/agents/privacidad/i18n/es/abogado-privacidad.md
mova-context/agents/privacidad/i18n/es/analista-presidio.md
mova-context/agents/privacidad/i18n/es/yagni-core.md
mova-context/skills/privacidad/i18n/es/cumplimiento-ley-21719.md
mova-context/skills/privacidad/i18n/es/deteccion-pii.md
mova-context/skills/privacidad/i18n/es/kiss-dry-core.md
mova-context/prompts/privacidad/i18n/es/analizar-documento-anonimizado.md
mova-context/prompts/privacidad/i18n/es/evaluar-cumplimiento.md
mova-context/prompts/privacidad/i18n/es/ockham-core.md
```

---

### ✅ Generación de contexto

```bash
mova run privacidad-presidio analizar-contrato | grep "<!-- "
```

**Resultado esperado:**
```
<!-- core: yagni-core -->
<!-- agent: analista-presidio -->
<!-- agent: abogado-privacidad -->
<!-- core: kiss-dry-core -->
<!-- skill: deteccion-pii -->
<!-- skill: cumplimiento-ley-21719 -->
<!-- core: ockham-core -->
<!-- prompt: analizar-documento-anonimizado -->
```

**Validación:** los 3 cores presentes exactamente 1 vez cada uno.

```bash
# Contar ocurrencias de cada core (debe ser 1)
mova run privacidad-presidio analizar-contrato | grep "<!-- core:" | sort | uniq -c
```

**Resultado esperado:**
```
      1 <!-- core: kiss-dry-core -->
      1 <!-- core: ockham-core -->
      1 <!-- core: yagni-core -->
```

---

### ✅ Todas las tareas disponibles

```bash
for task in analizar-contrato evaluar-politica evaluar-formulario analizar-terminos; do
  resultado=$(mova run privacidad-presidio $task | grep "<!-- prompt:" | head -1)
  echo "$task → $resultado"
done
```

**Resultado esperado:**
```
analizar-contrato  → <!-- prompt: analizar-documento-anonimizado -->
evaluar-politica   → <!-- prompt: evaluar-cumplimiento -->
evaluar-formulario → <!-- prompt: evaluar-cumplimiento -->
analizar-terminos  → <!-- prompt: analizar-documento-anonimizado -->
```

---

## PARTE 4 — Flujo completo integrado

### ✅ Pipeline end-to-end

```bash
# 1. Anonimizar el documento de prueba
python anonimizar.py contrato-original.txt

# 2. Verificar que no quedan datos reales
grep -E "\d{1,2}\.\d{3}\.\d{3}-[\dkK]" contrato-anonimizado.txt && echo "⚠ RUT sin anonimizar" || echo "✓ Sin RUTs visibles"
grep -E "[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}" contrato-anonimizado.txt && echo "⚠ Email sin anonimizar" || echo "✓ Sin emails visibles"

# 3. Generar contexto
mova run privacidad-presidio analizar-contrato > contexto.txt

# 4. Combinar
cat contexto.txt contrato-anonimizado.txt > prompt-final.txt

# 5. Ver tamaño del prompt final
wc -w prompt-final.txt
```

**Resultado esperado (paso 2):**
```
✓ Sin RUTs visibles
✓ Sin emails visibles
```

---

## PARTE 5 — Memoria

```bash
# Guardar resultado de la sesión
mova memory privacidad-presidio '```memory
## 2026-01-21 — validación
**Hecho:** pipeline completo validado, 9 entidades detectadas, 3 hallazgos
**Pendiente:** corregir contrato con hallazgos encontrados
**LLM Errors:** ninguno
```'

# Verificar que se guardó
mova memory-read privacidad-presidio | head -10

# Verificar que aparece en el próximo contexto
mova run privacidad-presidio analizar-contrato | grep -A5 "## MEMORY"
```

---

## PARTE 6 — MCP (opcional)

```bash
# Terminal 1
mova mcp start --port 3000

# Terminal 2
curl -s -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{"tool":"run_context","arguments":{"project":"privacidad-presidio","task":"analizar-contrato"}}' \
  | jq '.content' | head -5
```

**Resultado esperado:** JSON con el contexto completo.

---

## Resumen del checklist

| Paso | Comando | ✓ |
|------|---------|---|
| Presidio instalado | `python -c "from presidio_analyzer..."` | □ |
| Modelo spaCy ES | `python -m spacy info es_core_news_lg` | □ |
| bge-m3 en Ollama | `ollama list \| grep bge-m3` | □ |
| proyecto visible | `mova list \| grep privacidad` | □ |
| cores cargados (×1) | `mova run ... \| grep core: \| uniq -c` | □ |
| RUT detectado | `python -c "..."` → `RUT: 12.345.678-9` | □ |
| email detectado | `python -c "..."` → `EMAIL_ADDRESS: ...` | □ |
| sin datos reales post-anon | `grep -E "..."` → `✓ Sin RUTs` | □ |
| prompt-final.txt generado | `wc -w prompt-final.txt` | □ |
| memoria guardada | `mova memory-read privacidad-presidio` | □ |

**Si todos los ítems son ✓ → el caso está funcionando correctamente.**
