# FAQ

**¿Qué es Mova Context exactamente?** Un binario Go (CLI + servidor MCP) que se coloca entre "el contexto
está listo" y "se envía al LLM". No es un framework de agentes ni un editor de código.

**¿Qué hace concretamente "Gobernanza de Contexto"?** Decide y deja evidencia de: qué contexto se
seleccionó, qué se sanitizó/enmascaró (PII, secretos), cuánto costaría, y quién/qué lo autorizó — antes de
la llamada al LLM. No decide nada durante o después de la inferencia.

**¿Cómo protege PII antes de que llegue a un modelo?** Enmascarado estructural heurístico (forma + entropía,
`budget.pii_masking.enabled`), no un diccionario de nombres. No es 100% infalible ni sustituye revisión
legal — ver `docs/i18n/es/ARTIFACTS.md`.

**¿Cómo ayuda con el costo?** `mova budget`/`context-trace` estiman tokens (tokenizer real `cl100k_base`)
y costo contra `config/prices.json` por proveedor, antes de gastar nada.

**¿Qué pasa si excedo el presupuesto?** `budget.on_exceed` en `project.json` (`warn` o `block`) — el
circuit breaker corta o advierte antes del envío, no después.

**¿Mis datos salen de mi máquina?** Solo si tu `llm_profile` apunta a un proveedor cloud. Con `ollama`/
`lm-studio` (local), nada sale — ver el campo `TargetModel` en cualquier reporte para confirmarlo.

**¿Por qué `pii-audit-log.json` a veces trae ceros en `security`?** Ese detalle fino solo se llena en modo
*discovery* (repo remoto sin `project.json`). En modo proyecto, el resultado real de PII Masking está en
`mova-budget-report.md`.
