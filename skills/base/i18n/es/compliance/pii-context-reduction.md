# Skill: Reducción técnica de PII antes de enviar contexto a un LLM

Este skill documenta una capacidad OPCIONAL y desactivada por defecto de Mova Context: **PII Masking** (`mova.local/sanitize`, archivos `pii.go`/`pii_policy.go`). No es una lista de instrucciones para razonar sobre privacidad en abstracto — describe una funcionalidad real del motor que este proyecto puede activar.

## Qué hace (alcance)

Antes de contar tokens o enviar el contexto a cualquier modelo (local o cloud), Mova puede reemplazar en Focus/Memory los tokens cuya **forma estructural** (proporción de dígitos, separadores tipo `.`/`-`/`/`, presencia de `@`, longitud, mayúsculas) y **entropía de Shannon** superen un umbral configurable, por un pseudónimo determinístico `[PII_xxxxxxxx]` (hash FNV-1a truncado). El mismo valor original siempre produce el mismo pseudónimo — permite ver "esto se repite" sin reconstruir el valor original a partir del tag.

El algoritmo es **agnóstico al idioma**: no usa diccionarios, listas de palabras ni reglas gramaticales de español/inglés/otro idioma. Las mismas señales matemáticas (entropía) y estructurales (forma de la palabra) disparan igual sobre un RUT, un teléfono o un email, sin importar el idioma del texto alrededor.

## Cómo se activa

1. En `project.json` (o a nivel de task), dentro de `"budget"`:
   ```json
   "budget": { "pii_masking": { "enabled": true } }
   ```
   Por defecto está **desactivado** — un proyecto que nunca declara este campo no ve ningún cambio de comportamiento.
2. Los umbrales/pesos/formato del tag viven en `config/policy.json` (`pii_masking`), nunca hardcodeados en Go — se pueden ajustar sin recompilar.
3. El resultado aparece en `mova-budget-report.md`, sección "PII Masking" (`mova budget <proyecto>`), y el contexto que Mova arma con `mova run`/`mova chat`/jobs/MCP/HTTP ya sale con los tokens reemplazados si esta etapa masqueó algo.

## Qué NO hace (límites — leer antes de usar)

* **No es anonimización jurídica.** Es una mitigación heurística estructural — no cumple, por sí sola, ningún estándar legal de anonimización.
* **No garantiza detectar el 100% de la información personal.** Un nombre común sin dígitos ni separadores estructurales puede no alcanzar el umbral y no ser enmascarado (falso negativo). Por el mismo motivo, también puede enmascarar tokens que NO son PII pero comparten una forma estructural parecida — por ejemplo, fechas con separadores tipo `2024-07-30` (falso positivo).
* **No reemplaza asesoría legal, una política interna de privacidad, ni un programa de cumplimiento de la Ley 21.719, GDPR u otra normativa.**
* No reemplaza el resto del Token Firewall (Sanitizer, Circuit Breaker) — es una etapa adicional y opcional, no un sustituto.

## Cuándo usarlo en este dominio (compliance)

Útil quen un proyecto de Mova Context construye contexto a partir de datos de clientes (CRM, ERP, sistemas legados) y ese contexto se va a enviar a un LLM externo — reduce, técnicamente, cuánta información identificable viaja fuera del entorno local, sin bloquear el flujo de trabajo. Combinar siempre con: preferir un modelo local cuando sea posible, revisar qué campos son realmente necesarios para la consulta (ver agente `ai-privacy-reviewer`), y no depender únicamente de esta etapa para cumplimiento normativo.
