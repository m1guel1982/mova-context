# Guía de validación — Mova Context

Checklist completo para verificar que todo funciona correctamente.

---

## 1. CLI básico

### `mova list`

```bash
mova list
```

**Resultado esperado:**

```text
  acme-demo              [es] API interna multi-tenant de ejemplo
    tasks: audit-auth, audit-payments, new-module, review-complete
  ley-21719              [es] Ley 21.719 — Protección de Datos Personales (Chile)
    tasks: analizar-contrato, evaluar-cumplimiento, redactar-politica, responder-titular
  omnicanal-demo         [es] Acme Services — Omnicanal Empresarial
    tasks: cobranza-tardia, cobranza-temprana, ivr-menu, ventas-webchat, ventas-whatsapp
  pruebas-locales        [es] Ejemplo mínimo oficial — ...
    tasks: agregar-modulo, auditar-modulo, configurar-cicd, crear-proyecto, generar-tests
```

**Validación:** aparecen los 4 proyectos.  
**Error común:** ejecutar desde fuera del directorio `mova-context` (no encuentra `workflow.md`).

---

## 2. Generación de contexto

### `mova run`

```bash
mova run pruebas-locales crear-proyecto
```

**Resultado esperado:** contexto con secciones AGENTS, SKILLS, PROMPT, INSTRUCTION.

```bash
# Verificar que los tres core están presentes
mova run pruebas-locales crear-proyecto | grep "<!-- core:"
```

**Salida esperada:**

```text
<!-- core: yagni-core -->
<!-- core: kiss-dry-core -->
<!-- core: ockham-core -->
```

**Error común:** contexto vacío → verificar que `project.json` usa el esquema correcto (`agents.domain`, `agents.use`, etc.).

---

## 3. Proyectos

### ley-21719

```bash
mova run ley-21719 analizar-contrato | head -20
```

Debe mostrar: `abogado-datos` agent + `ley-21719-obligaciones` skill.

### omnicanal-demo

```bash
mova run omnicanal-demo ivr-menu | head -20
mova run omnicanal-demo ventas-whatsapp | head -20
```

Ambos deben ejecutar sin error.

### acme-demo

```bash
mova run acme-demo review-complete | grep "<!-- agent:"
```

Debe mostrar: `backend-dev`, `security-architect`, `acme-backend`, `architect`.

### pruebas-locales

```bash
mova run pruebas-locales crear-proyecto | grep "<!-- "
```

---

## 4. Archivos Core

```bash
# yagni-core debe estar en agents
mova run pruebas-locales crear-proyecto | grep -A2 "yagni-core"

# kiss-dry-core debe estar en skills
mova run pruebas-locales crear-proyecto | grep -A2 "kiss-dry-core"

# ockham-core debe estar en prompt
mova run pruebas-locales crear-proyecto | grep -A2 "ockham-core"

# Nunca se cargan dos veces
mova run pruebas-locales crear-proyecto | grep "core:" | sort | uniq -c
# Cada core debe aparecer exactamente 1 vez
```

---

## 5. Memoria

```bash
# Guardar memoria
mova memory pruebas-locales '```memory
## 2026-01-21 — test
**Hecho:** validación OK
**Pendiente:** nada
```'

# Leer memoria
mova memory-read pruebas-locales

# Verificar que aparece en el siguiente run
mova run pruebas-locales crear-proyecto | grep -A5 "## MEMORY"
```

---

## 6. MCP

```bash
# Terminal 1: iniciar servidor
mova mcp start --port 3000

# Terminal 2: probar
curl -s -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{"tool":"list_projects","arguments":{}}' | jq .

curl -s -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{"tool":"run_context","arguments":{"project":"pruebas-locales","task":"crear-proyecto"}}' \
  | jq .content | head -20
```

**Resultado esperado:** JSON con el contexto completo.

---

## 7. Ollama

```bash
# Cambiar llm_profile en project.json
# "type": "local", "provider": "ollama", "model": "llama3.1"

mova run pruebas-locales crear-proyecto | grep "Profile:"
# Debe mostrar: Profile: local/ollama:llama3.1

# El contexto debe tener listas numeradas en lugar de bullets
mova run pruebas-locales crear-proyecto | grep "^1\."
```

---

## 8. PostgreSQL

```bash
# Verificar que el adaptador acepta DSN
MOVA_ADAPTER=db MOVA_DSN="postgres://user:pass@localhost/mova_db" mova run pruebas-locales crear-proyecto
```

**Error esperado si no hay DB:** `db connect failed ... falling back to file`  
**Resultado esperado con DB activa:** contexto completo desde PostgreSQL.

---

## 9. Claude / GPT

```bash
# Generar contexto y copiar al clipboard
mova run pruebas-locales crear-proyecto > contexto.txt
```

Pegar `contexto.txt` como primer mensaje en Claude o ChatGPT.  
El modelo debe responder generando código según los agents y skills cargados.

---

## 10. Búsqueda

```bash
mova search "datos personales" legal
mova search "jwt"
mova search "ventas"
```

**Resultado esperado:** lista de archivos relevantes con excerpts.

---

## 11. Init (nuevo proyecto)

```bash
mova init mi-nuevo-proyecto
ls projects/mi-nuevo-proyecto/
# project.json  memory.md
cat projects/mi-nuevo-proyecto/project.json
```

---

## 12. Context Compiler (`mova compile`)

```bash
mova compile compiler-demo revisar-ordenes
cat projects/compiler-demo/contexto.txt
```

**Resultado esperado:** archivo compacto (sin encabezados `## AGENTS`) con `FOCUS:CreateOrder()` mostrando solo esa función, `FOCUS:manual.md` con el manual completo, y `FOCUS:Artículo 6` con solo esa sección.

```bash
# Verificar que nunca envía el proyecto completo cuando hay focus
grep -c "FOCUS:" projects/compiler-demo/contexto.txt
# esperado: 3 (uno por elemento de focus)
```

**Error común:** `task not found` — revisar `default_task` en `projects/compiler-demo/project.json`.
**Error común:** el símbolo no aparece — la búsqueda es literal; revisar mayúsculas/nombre exacto en el código fuente.

Ver [context-compiler.md](context-compiler.md) para el detalle de Fase 1 y Fase 2.

---

## Resumen del checklist

| Ítem                         | Comando                              | ✓ |
|------------------------------|--------------------------------------|---|
| list muestra 4 proyectos     | `mova list`                          | □ |
| run genera contexto          | `mova run pruebas-locales crear-proyecto` | □ |
| yagni-core presente          | `grep "<!-- core: yagni"`            | □ |
| kiss-dry-core presente       | `grep "<!-- core: kiss"`             | □ |
| ockham-core presente         | `grep "<!-- core: ockham"`           | □ |
| omnicanal sin errores        | `mova run omnicanal-demo ivr-menu`   | □ |
| memoria funciona             | `mova memory-read pruebas-locales`   | □ |
| MCP responde                 | curl al puerto 3000                  | □ |
| Ollama profile se aplica     | `grep "Profile: local"`              | □ |
| compile genera contexto.txt  | `mova compile compiler-demo`         | □ |
| focus poda sin enviar todo   | `grep -c "FOCUS:" contexto.txt`      | □ |
