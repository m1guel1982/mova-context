# Plan de Pruebas de QA y Ejecución: Egress Audit & Token Budget Control

> **Nota para Evaluadores / Testers:** Este documento contiene la suite de pruebas completa ejecutada para validar los controles de gobernanza de contexto, sanitización de PII y aislamiento Air-Gap de Mova Context utilizando un cliente MCP (ej. Cline con Claude Anthropic Haiku). Puedes replicar estos comandos y escenarios en tu propio entorno para verificar los resultados y el cumplimiento de auditoría.

---

## Fase 1 — Con `dry_run: true` (Modo Guardrail / Bloqueo Total)

**Premisa de Fase:** Interceptación preventiva y bloqueo de herramientas de egreso hacia LLMs externas o lectura de artefactos restringidos.

### Test 1 — `chat_completion` (Egresos Críticos)
* **Comando:** Usando la herramienta `chat_completion` de `mova-context`, para el proyecto `02-pii-compliance-governance`, pedile que resuma el contexto.
* **Criterios:**

 **PASO:** Devuelve `[MOVA EGRESS AUDIT]`, `dry_run=true`, tokens evaluados reales y `0 tokens enviados`, con directiva de seguridad que impide fallbacks del asistente.
 **FALLO:** Expone un resumen real o la LLM adivina el contenido usando herramientas locales.
* **Estado:** ✅ PASADO (7.787 tokens evaluados, 0 enviados; directiva acatada sin lectura de disco).

### Test 2 — `get_full_context`
* **Comando:** Traé el contexto completo del proyecto `02-pii-compliance-governance` con `get_full_context` de `mova-context`.
* **Criterios:** 

**PASO:** Retorna únicamente el mensaje de auditoría `[MOVA EGRESS AUDIT]`. 
**FALLO:** Filtra estructura de archivos, `agents`, `skills` o contenido del repositorio.
* **Estado:** ✅ PASADO (Interceptado de forma hermética).

### Test 3 — `get_memory` (Capa de Memoria en Air-Gap)
* **Comando:** Mostrame la memoria del proyecto `02-pii-compliance-governance` con `get_memory`.
* **Criterios:** 

**PASO:** Retorna el mensaje `[MOVA EGRESS AUDIT]`, incluso si `memory.md` estuviera vacío (0 tokens). 
**FALLO:** Muestra el contenido original de la memoria o un mensaje estándar de "memoria vacía".
* **Estado:** ✅ PASADO (Corte de auditoría aplicado consistentemente en la puerta MCP).

### Test 4 — `read_file` (Inspección de Fuga de Datos)
* **Comando:** Leé el archivo `customers.json` del proyecto `02-pii-compliance-governance` con `read_file`.
* **Criterios:** 

**PASO:** Bloqueo total sin exposición de datos personales (sin RUTs, emails o nombres). 
**FALLO:** Se filtra cualquier fragmento o clave JSON con datos sensibles.
* **Estado:** ✅ PASADO (Acceso bloqueado por política de auditoría).

### Test 5 — Control Negativo: `search_context` (Fuera del Air-Gap)
* **Comando:** Buscá "PII" en el conocimiento de `mova-context` con `search_context`.
* **Criterios:** 

**PASO:** Devuelve resultados de búsqueda normales (fragmentos indexados de `agents/skills`). 
**FALLO:** La búsqueda es interceptada por el guardrail de auditoría.
* **Estado:** ✅ PASADO (Operación fuera del perímetro ejecutada correctamente).

### Test 6 — Evidencia en Disco (`egress.md`)
* **Comando:** Revisar el reporte generado en disco para verificar que guarde evidencia de auditoría sin exponer PII crudo.
* **Criterios:** 

**PASO:** Genera `egress.md` preservando la estructura JSON pero aplicando enmascarado `[PII_xxxxxxxx]`.
 **FALLO:** El reporte contiene datos personales sin sanitizar.
* **Estado:** ✅ PASADO (PII enmascarado correctamente con tokens de entropía).

---

## Fase 2 — Con `dry_run: false` (Delegación / Inferencia Activa)

**Premisa de Fase:** Con el guardrail deshabilitado o pasivo, la ejecución debe delegarse de forma transparente al proveedor host.

### Test 7 — Delegación Real al Host
* **Comando:** Usando `chat_completion` de `mova-context` para el proyecto `02-pii-compliance-governance`, preguntale qué tipo de datos personales hay en el proyecto.
* **Criterios:**

 **PASO:** Inferencia real entregada por la LLM (identifica campos tipo `rut`, `email`, `telefono`).
  **FALLO:** Error de conexión a modelos locales o bloqueo por auditoría.
* **Estado:** ✅ PASADO (Delegación correcta al proveedor host).

### Test 8 — `get_full_context` Normal
* **Comando:** Traé el contexto completo del proyecto `02-pii-compliance-governance` con `get_full_context`.
* **Criterios:** 

**PASO:** Devuelve el contexto completo (`agents` + `skills` + `prompt` + `focus`). 
**FALLO:** Respuesta parcial o bloqueo no deseado.
* **Estado:** ✅ PASADO (Entrega de contexto completa y no interceptada).