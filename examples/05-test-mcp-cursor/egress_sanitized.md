
## Egress audit — execution 18d8055a5d5603a8-79cccba3b23ad25f

- execution_id: `18d8055a5d5603a8-79cccba3b23ad25f`
- timestamp: 2026-09-23T18:05:37Z (UTC)

### Sanitized context sent to the LLM

```text


---
## AGENTS

<!-- core: yagni-core -->
# Regla núcleo: YAGNI

No asumas necesidades futuras. No crees abstracciones, endpoints ni estructuras si la tarea actual no lo solicita explícitamente. Si el prompt no lo pide, no existe.

Nunca generar sin pedido explícito: Docker · docker-compose · CI/CD · logging avanzado · autenticación compleja · archivos .env · linters · formatters · carpetas vacías reservadas para el futuro · dependencias no declaradas en el stack.

<!-- agent: ai-privacy-reviewer -->
# Rol
AI Privacy Reviewer. Evalúa, sobre el FOCUS entregado y la consulta original, qué información es realmente necesaria para que un modelo de IA (local o cloud) responda esa consulta puntual, y qué información debería evitarse enviar innecesariamente a un proveedor externo — razonando como Data Protection Officer + Arquitecto de IA.

# Reglas
* Partir SIEMPRE de la consulta original — "necesario" se define en función de esa consulta, no del dataset completo
* Marcar como innecesario cualquier dato que no cambie la respuesta a la consulta (ej. metadata técnica interna, IDs de sesión, huellas de dispositivo, historiales sin relación con la pregunta)
* Si el FOCUS contiene datos que sí son necesarios pero muy sensibles (ej. RUT completo, email, teléfono), sugerir preferir un modelo local para esa tarea o preparar un contexto reducido/pseudonimizado antes de usar un LLM cloud — nunca asumir que Mova Context ya hizo esa reducción por sí solo, salvo que el reporte de Context Governance / PII Masking lo confirme explícitamente
* No prometer ni sugerir cumplimiento automático de la Ley 21.719 u otra normativa — esta evaluación es una ayuda técnica, no asesoría legal
* Cuando el proyecto tenga PII Masking configurado (ver `budget.pii_masking` en project.json y `config/policy.json`), señalar que es una mitigación heurística estructural (forma de palabra + entropía), no una anonimización jurídica, y que no garantiza detectar el 100% de la información personal

# Formato de respuesta
```txt
Consulta original:
Información necesaria para responderla:
Información NO necesaria (candidata a reducir/omitir):
Recomendación (modelo local / contexto reducido para cloud / ambos):
Advertencia de límites (heurística, no garantía legal):
```


---
## SKILLS

<!-- core: kiss-dry-core -->
# Regla núcleo: KISS + DRY

KISS: cada skill resuelve una sola cosa, de la forma más directa posible, usando lo que ya existe antes de proponer algo nuevo.

DRY: una regla o instrucción existe en un solo lugar. El resto la referencia, no la copia.

<!-- skill: pii-context-reduction -->
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
* No reemplaza el resto del Context Governance (Sanitizer, Circuit Breaker) — es una etapa adicional y opcional, no un sustituto.

## Cuándo usarlo en este dominio (compliance)

Útil quen un proyecto de Mova Context construye contexto a partir de datos de clientes (CRM, ERP, sistemas legados) y ese contexto se va a enviar a un LLM externo — reduce, técnicamente, cuánta información identificable viaja fuera del entorno local, sin bloquear el flujo de trabajo. Combinar siempre con: preferir un modelo local cuando sea posible, revisar qué campos son realmente necesarios para la consulta (ver agente `ai-privacy-reviewer`), y no depender únicamente de esta etapa para cumplimiento normativo.


---
## PROMPT

<!-- core: ockham-core -->
# Regla núcleo: Navaja de Ockham (salida)

Entre múltiples soluciones válidas, ejecuta siempre la más simple y compacta. Prohibida la prosa explicativa antes o después del código. Si el código es autoexplicativo, entrega solo el bloque técnico. Cada palabra ahorrada es eficiencia de tokens.

<!-- prompt: analizar-contexto-clientes-ia -->
# Analizar contexto de clientes para IA — 02-pii-compliance-governance
Proyecto: `02-pii-compliance-governance` · Normativa de referencia: `Ley 21.719 (Chile)`
Ockham: ver `../engineering/ockham-core.md`.

# Consulta original del negocio
> Busca la información relevante de los clientes y dime qué datos personales tenemos, para qué los usamos y qué sistemas o procesos están involucrados.

# Objetivo
Sobre el FOCUS entregado (datos de clientes desde JSON/PDF/DOCX, simulando un CRM/ERP/sistema legado), responder la consulta original desde el rol que te fue asignado (ver tu agente: Data Analyst, Purpose Analyst o AI Privacy Reviewer), usando exclusivamente el formato de respuesta definido en ese agente.

# Pasos
1. Leer el FOCUS completo — no asumir datos que no aparezcan literalmente en él
2. Aplicar el formato de respuesta de tu rol a cada hallazgo relevante
3. Citar siempre la fuente (archivo/sección) de cada hallazgo
4. Si tu rol es AI Privacy Reviewer, cerrar explícitamente con qué información no sería necesaria enviar a un LLM externo para responder la consulta original
5. No afirmar que este análisis por sí solo garantiza cumplimiento de Ley 21.719 (Chile) — es una ayuda técnica, no asesoría legal

# Dónde queda el reporte de este análisis (importante)
Este proyecto tiene configurado `budget_path`/`token_history_path` propios — al correr `mova budget 02-pii-compliance-governance` (o el job programado del proyecto), Mova Context genera automáticamente:
- `mova-budget-report.md` en la carpeta de este agente, con el detalle de tokens, Sanitizer, PII Masking (si está activado) y Circuit Breaker.
- Un reporte narrativo adicional en `reports/analisis-ley21719_{date}.md`, vía el job programado de este proyecto (ver `jobs` en `project.json`).

Estos reportes se generan igual sin importar el canal usado (CLI `mova run`/`mova budget`, `mova chat`, `mova jobs run`, HTTP/API, o MCP) — es el mismo motor detrás de los cinco.

# Formato de respuesta
Usar exactamente el formato definido en tu agente — no inventar un formato nuevo aquí.
# Mova Context — 02-pii-compliance-governance / analizar
Generated: 2026-09-23 15:05 | Repo: examples/02-pii-compliance-governance/repo | Lang: es | LLM: not set | Profile: powerful


---
## [PII_65366faf]
FOCUS:customers.json
{
  "_nota": "Datos 100% FICTICIOS generados para este ejemplo. Ningún nombre, RUT, correo, teléfono o dirección corresponde a una persona real. Simula un export típico de CRM/ERP/sistema legado.",
  "_note_en": "100% [PII_4740c3d9] data generated for this example. No name, RUT, email, phone, or address corresponds to a real person. Simulates a typical CRM/ERP/legacy-system export.",
  "clientes": [
    {
      "id": "[PII_bd6e667b]",
      "nombre": "Andrea Fuentes Rojas",
      "rut": "[PII_3437c915]",
      "email": "[PII_39fe126c]",
      "telefono": "+56 9 [PII_e50a123d] [PII_5cc501d8]",
      "direccion": "Av. Los Alerces [PII_eb5a3bf0], depto 302, Ñuñoa, Santiago",
      "fecha_registro": "[PII_ba1facb1]",
      "historial_interacciones": [
        "[PII_ba1facb1] - Alta de cuenta vía formulario web",
        "[PII_c8259c60] - Solicitud de cambio de plan (canal: call center)",
        "[PII_80fd6568] - Reclamo por cobro duplicado, resuelto en 3 días"
      ],
      "consentimiento": {
        "marketing_directo": true,
        "fecha_consentimiento": "[PII_ba1facb1]",
        "canal": "checkbox formulario web",
        "puede_revocarse": true
      },
      "finalidad_tratamiento": [
        "gestión de la relación comercial",
        "facturación y cobranza",
        "comunicaciones de marketing (con consentimiento)"
      ],
      "sistemas_procesos": ["CRM ventas", "Facturación", "Call center"],
      "metadata_tecnica_interna": {
        "app_build": "mova-crm-v4.2.1",
        "device_fingerprint": "[PII_2183c960]",
        "session_id": "[PII_f4fd8454]",
        "integration_trace_id": "[PII_49e0eac5]"
      }
    },
    {
      "id": "[PII_bd6e657b]",
      "nombre": "Marcelo Iturra Peña",
      "rut": "[PII_9f48a219]",
      "email": "[PII_8f4c4623]",
      "telefono": "+56 9 [PII_efa1c1e1] [PII_1f9735f1]",
      "direccion": "Camino Real 220, La Serena",
      "fecha_registro": "[PII_28c17f6b]",
      "historial_interacciones": [
        "[PII_28c17f6b] - Alta de cuenta en sucursal física",
        "[PII_e9d510a1] - Actualización de dirección postal",
        "[PII_84ecb058] - Consulta sobre portabilidad de datos"
      ],
      "consentimiento": {
        "marketing_directo": false,
        "fecha_consentimiento": "[PII_28c17f6b]",
        "canal": "formulario físico",
        "puede_revocarse": true
      },
      "finalidad_tratamiento": [
        "gestión de la relación comercial",
        "atención de solicitudes de titulares (Ley [PII_90a9d44f])"
      ],
      "sistemas_procesos": ["CRM ventas", "Portal de solicitudes de titulares"],
      "metadata_tecnica_interna": {
        "app_build": "mova-crm-v4.2.1",
        "device_fingerprint": "[PII_d2960e0a]",
        "session_id": "[PII_3a57ab4d]",
        "integration_trace_id": "[PII_5275fec5]"
      }
    },
    {
      "id": "[PII_bd6e647b]",
      "nombre": "Javiera Contreras Muñoz",
      "rut": "[PII_0fbdbf86]",
      "email": "[PII_a2438da2]",
      "telefono": "+56 9 [PII_bf45b5ea] [PII_32e7983d]",
      "direccion": "Pasaje Las Dalias 88, Concepción",
      "fecha_registro": "[PII_3763817a]",
      "historial_interacciones": [
        "[PII_3763817a] - Alta de cuenta vía app móvil",
        "[PII_962806bb] - Solicitud de soporte técnico",
        "[PII_b68e9c81] - Encuesta de satisfacción respondida"
      ],
      "consentimiento": {
        "marketing_directo": true,
        "fecha_consentimiento": "[PII_3763817a]",
        "canal": "checkbox app móvil",
        "puede_revocarse": true
      },
      "finalidad_tratamiento": [
        "gestión de la relación comercial",
        "soporte técnico",
        "comunicaciones de marketing (con consentimiento)"
      ],
      "sistemas_procesos": ["App móvil", "Mesa de ayuda", "CRM ventas"],
      "metadata_tecnica_interna": {
        "app_build": "mova-crm-v4.2.1",
        "device_fingerprint": "[PII_1f6b0e37]",
        "session_id": "[PII_1790e3aa]",
        "integration_trace_id": "[PII_5a3f2ec5]"
      }
    },
    {
      "id": "[PII_bd6e637b]",
      "nombre": "Rodrigo Salinas Bravo",
      "rut": "[PII_b19717b2]",
      "email": "[PII_9c8e7685]",
      "telefono": "+56 9 [PII_95cda5cf] [PII_cd0622e1]",
      "direccion": "Av. Alemania 300, Temuco",
      "fecha_registro": "[PII_412d826d]",
      "historial_interacciones": [
        "[PII_412d826d] - Alta de cuenta vía convenio empresa",
        "[PII_7989884a] - Cambio de método de pago",
        "[PII_d17dd871] - Solicitud de eliminación de datos (Ley [PII_90a9d44f])"
      ],
      "consentimiento": {
        "marketing_directo": false,
        "fecha_consentimiento": "[PII_412d826d]",
        "canal": "convenio corporativo",
        "puede_revocarse": true
      },
      "finalidad_tratamiento": [
        "gestión de la relación comercial",
        "facturación y cobranza",
        "atención de solicitudes de titulares (Ley [PII_90a9d44f])"
      ],
      "sistemas_procesos": ["CRM ventas", "Facturación", "Portal de solicitudes de titulares"],
      "metadata_tecnica_interna": {
        "app_build": "mova-crm-v4.2.1",
        "device_fingerprint": "[PII_c800b2ed]",
        "session_id": "[PII_28cf643c]",
        "integration_trace_id": "[PII_62d442c5]"
      }
    },
    {
      "id": "[PII_bd6e627b]",
      "nombre": "Camila Ossandón Lagos",
      "rut": "[PII_cff870c1]",
      "email": "[PII_007a0ea5]",
      "telefono": "+56 9 [PII_65c399d8] [PII_59f6fa02]",
      "direccion": "Calle Los Notros 77, Puerto Montt",
      "fecha_registro": "[PII_7350df5f]",
      "historial_interacciones": [
        "[PII_7350df5f] - Alta de cuenta vía formulario web",
        "[PII_d7918589] - Renovación de plan anual",
        "[PII_f2526a5f] - Consulta sobre finalidad de tratamiento de datos"
      ],
      "consentimiento": {
        "marketing_directo": true,
        "fecha_consentimiento": "[PII_7350df5f]",
        "canal": "checkbox formulario web",
        "puede_revocarse": true
      },
      "finalidad_tratamiento": [
        "gestión de la relación comercial",
        "comunicaciones de marketing (con consentimiento)",
        "analítica interna de uso del servicio"
      ],
      "sistemas_procesos": ["CRM ventas", "Plataforma de analítica interna"],
      "metadata_tecnica_interna": {
        "app_build": "mova-crm-v4.2.1",
        "device_fingerprint": "[PII_5193516d]",
        "session_id": "[PII_bd1b6773]",
        "integration_trace_id": "[PII_6c358ac5]"
      }
    },
    {
      "id": "[PII_bd6e617b]",
      "nombre": "Felipe Herrera Vidal",
      "rut": "[PII_b6e2b42a]",
      "email": "[PII_118329e4]",
      "telefono": "+56 9 [PII_5a0b7a02] [PII_855a4a35]",
      "direccion": "Av. Balmaceda 512, Antofagasta",
      "fecha_registro": "[PII_184cb980]",
      "historial_interacciones": [
        "[PII_184cb980] - Alta de cuenta en sucursal física",
        "[PII_c3a18876] - Actualización de teléfono de contacto",
        "[PII_f0f58f60] - Reclamo por envío de publicidad sin consentimiento"
      ],
      "consentimiento": {
        "marketing_directo": false,
        "fecha_consentimiento": "[PII_184cb980]",
        "canal": "formulario físico",
        "puede_revocarse": true
      },
      "finalidad_tratamiento": [
        "gestión de la relación comercial",
        "atención de solicitudes de titulares (Ley [PII_90a9d44f])"
      ],
      "sistemas_procesos": ["CRM ventas", "Portal de solicitudes de titulares", "Call center"],
      "metadata_tecnica_interna": {
        "app_build": "mova-crm-v4.2.1",
        "device_fingerprint": "[PII_e51745c9]",
        "session_id": "[PII_e7046108]",
        "integration_trace_id": "[PII_e9c626c4]"
      }
    },
    {
      "id": "[PII_bd6e607b]",
      "nombre": "Valentina Riquelme Soto",
      "rut": "[PII_fd7da01c]",
      "email": "[PII_5db0715b]",
      "telefono": "+56 9 [PII_f5a2760b] [PII_e1ebe9ea]",
      "direccion": "Pasaje El Roble 45, Rancagua",
      "fecha_registro": "[PII_12656e68]",
      "historial_interacciones": [
        "[PII_12656e68] - Alta de cuenta vía app móvil",
        "[PII_84f01958] - Solicitud de portabilidad de datos",
        "[PII_c59109ab] - Consulta sobre qué sistemas usan sus datos"
      ],
      "consentimiento": {
        "marketing_directo": true,
        "fecha_consentimiento": "[PII_12656e68]",
        "canal": "checkbox app móvil",
        "puede_revocarse": true
      },
      "finalidad_tratamiento": [
        "gestión de la relación comercial",
        "comunicaciones de marketing (con consentimiento)",
        "atención de solicitudes de titulares (Ley [PII_90a9d44f])"
      ],
      "sistemas_procesos": ["App móvil", "CRM ventas", "Portal de solicitudes de titulares"],
      "metadata_tecnica_interna": {
        "app_build": "mova-crm-v4.2.1",
        "device_fingerprint": "[PII_08e474b0]",
        "session_id": "[PII_26a76550]",
        "integration_trace_id": "[PII_f3425ec4]"
      }
    },
    {
      "id": "[PII_bd6e5f7b]",
      "nombre": "Ignacio Pizarro Cárdenas",
      "rut": "[PII_1813da8a]",
      "email": "[PII_84958051]",
      "telefono": "+56 9 [PII_f4665df0] [PII_65db9ed8]",
      "direccion": "Av. Colón 990, Viña del Mar",
      "fecha_registro": "[PII_e7c6bbff]",
      "historial_interacciones": [
        "[PII_e7c6bbff] - Alta de cuenta en sucursal física",
        "[PII_84519488] - Migración a plan corporativo",
        "[PII_005dec89] - Consulta sobre conservación de datos históricos"
      ],
      "consentimiento": {
        "marketing_directo": false,
        "fecha_consentimiento": "[PII_e7c6bbff]",
        "canal": "formulario físico",
        "puede_revocarse": true
      },
      "finalidad_tratamiento": [
        "gestión de la relación comercial",
        "facturación y cobranza"
      ],
      "sistemas_procesos": ["CRM ventas", "Facturación"],
      "metadata_tecnica_interna": {
        "app_build": "mova-crm-v4.2.1",
        "device_fingerprint": "[PII_291e5b3a]",
        "session_id": "[PII_7aae9b57]",
        "integration_trace_id": "[PII_e49c34dc]"
      }
    },
    {
      "id": "[PII_bd6e5e7b]",
      "nombre": "Daniela Castro Yáñez",
      "rut": "[PII_a24e95cf]",
      "email": "[PII_611d8dcb]",
      "telefono": "+56 9 [PII_32e7983d] [PII_29f06f0b]",
      "direccion": "Calle Las Camelias 63, Talca",
      "fecha_registro": "[PII_8ede3b17]",
      "historial_interacciones": [
        "[PII_8ede3b17] - Alta de cuenta vía formulario web",
        "[PII_84b060c2] - Solicitud de rectificación de datos (dirección errónea)",
        "[PII_6772ada2] - Consulta sobre transferencia de datos a proveedor de IA"
      ],
      "consentimiento": {
        "marketing_directo": true,
        "fecha_consentimiento": "[PII_8ede3b17]",
        "canal": "checkbox formulario web",
        "puede_revocarse": true
      },
      "finalidad_tratamiento": [
        "gestión de la relación comercial",
        "comunicaciones de marketing (con consentimiento)",
        "atención de solicitudes de titulares (Ley [PII_90a9d44f])"
      ],
      "sistemas_procesos": ["CRM ventas", "Portal de solicitudes de titulares"],
      "metadata_tecnica_interna": {
        "app_build": "mova-crm-v4.2.1",
        "device_fingerprint": "[PII_837f4f05]",
        "session_id": "[PII_dfb87164]",
        "integration_trace_id": "[PII_dcd314dc]"
      }
    },
    {
      "id": "[PII_bd6b617b]",
      "nombre": "Sebastián Molina Torres",
      "rut": "[PII_b6cdc94d]",
      "email": "[PII_e04845c7]",
      "telefono": "+56 9 [PII_cd24a1e1] [PII_5a07fb02]",
      "direccion": "Av. Argentina 145, Iquique",
      "fecha_registro": "[PII_bda11568]",
      "historial_interacciones": [
        "[PII_bda11568] - Alta de cuenta en sucursal física",
        "[PII_2473835d] - Actualización de correo de contacto",
        "[PII_c29d016f] - Consulta por entrada en vigencia de la Ley [PII_90a9d44f]"
      ],
      "consentimiento": {
        "marketing_directo": false,
        "fecha_consentimiento": "[PII_bda11568]",
        "canal": "formulario físico",
        "puede_revocarse": true
      },
      "finalidad_tratamiento": [
        "gestión de la relación comercial",
        "facturación y cobranza",
        "atención de solicitudes de titulares (Ley [PII_90a9d44f])"
      ],
      "sistemas_procesos": ["CRM ventas", "Facturación", "Portal de solicitudes de titulares"],
      "metadata_tecnica_interna": {
        "app_build": "mova-crm-v4.2.1",
        "device_fingerprint": "[PII_511e0872]",
        "session_id": "[PII_a11e024e]",
        "integration_trace_id": "[PII_f6928cdc]"
      }
    }
  ]
}
FOCUS:customer-profile.pdf
Ficha ampliada de clientes - Sistema legado (ejemplo ficticio) Este documento simula una ficha extendida que, en muchas empresas, vive en un sistema legado separado del CRM principal (por ejemplo, un modulo de atencion al cliente o un antiguo [PII_b1d42a60]. Complementa a customers.json con historial detallado de tres clientes seleccionados. Todos los datos son FICTICIOS. Cliente [PII_bd6e667b] - Andrea Fuentes Rojas RUT: [PII_ab33b475] Email: [PII_39fe126c]. Telefono: +56 9 [PII_e50a123d] [PII_4d6c821a] Notas de atencion: cliente activa desde [PII_18371f0b], consulto por cambio de plan y reclamo de cobro duplicado ya resuelto. Prefiere contacto por correo. Finalidad de uso: gestion comercial y facturacion. Cliente [PII_bd6e637b] - Rodrigo Salinas Bravo RUT: [PII_002649f8] Email: [PII_9c8e7685]. Telefono: +56 9 [PII_95cda5cf] [PII_5611770d] Notas de atencion: solicito eliminacion de datos personales al amparo de la Ley [PII_90a9d44f] el [PII_679d0680] Solicitud en curso en el area de cumplimiento. Finalidad de uso: gestion comercial, facturacion y atencion de solicitudes de titulares. Cliente [PII_bd6e5e7b] - Daniela Castro Yanez RUT: [PII_a73feee1] Email: [PII_611d8dcb]. Telefono: +56 9 [PII_32e7983d] [PII_f09a9c1d] Notas de atencion: consulto especificamente si sus datos serian enviados a un proveedor de IA externo y bajo que finalidad. Pendiente de respuesta formal del area de privacidad.FOCUS:privacy-policy.docx
Política de privacidad (ejemplo ficticio) — Ley [PII_90a9d44f]
Este documento es un ejemplo FICTICIO, creado únicamente para esta demo de Mova Context. No corresponde a una empresa real ni debe usarse como política de privacidad válida.
1. Qué datos personales tratamos
Nombre completo, RUT, correo electrónico, teléfono, dirección, historial de interacciones con nuestros canales de atención, y preferencias de consentimiento para comunicaciones de marketing.
2. Para qué los utilizamos
Gestión de la relación comercial: identificar al cliente y administrar su cuenta.
Facturación y cobranza: emitir boletas/facturas y procesar pagos.
Atención de solicitudes de titulares: responder solicitudes de acceso, rectificación, eliminación y portabilidad conforme a la Ley [PII_d2aab6d8]
Comunicaciones de marketing: solo cuando el titular dio su consentimiento explícito, revocable en cualquier momento.
Soporte técnico: resolver consultas o incidencias reportadas por el cliente.
3. Qué sistemas o procesos los utilizan
CRM de ventas, sistema de facturación, mesa de ayuda / call center, app móvil, portal de solicitudes de titulares, y plataforma de analítica interna de uso del servicio.
4. Conservación de los datos
Los datos se conservan mientras la relación comercial esté vigente y, tras su término, por el plazo mínimo que exija la normativa tributaria y comercial aplicable. Vencido ese plazo, los datos que ya no tengan una base de tratamiento vigente deben eliminarse o anonimizarse.
5. Comunicaciones comerciales
Solo se envían comunicaciones de marketing directo a quienes dieron consentimiento explícito y específico para ese fin — nunca agrupado con la aceptación de otros términos. El consentimiento puede revocarse en cualquier momento por el mismo canal en que se otorgó.
6. Atención de solicitudes de titulares
Toda persona titular de datos puede solicitar acceso, rectificación, eliminación, oposición o portabilidad de sus datos a través del portal de solicitudes de titulares. Las solicitudes se responden dentro de los plazos que establece la Ley [PII_d2aab6d8]
7. Uso de modelos de inteligencia artificial
Cuando un proceso interno utiliza un modelo de IA (local o en la nube) para apoyar la atención o el análisis de datos de clientes, el principio aplicado es enviar únicamente la información mínima necesaria para responder la consulta puntual — nunca el registro completo de un cliente por defecto. Los detalles técnicos de cómo se prepara y reduce ese contexto no forman parte de esta política; ver la documentación técnica del sistema correspondiente.
8. Nota importante
Este documento fue redactado únicamente con fines demostrativos y educativos para el ejemplo &#[PII_569022e3]; de Mova Context. No constituye asesoría legal ni una política de privacidad válida para ninguna organización real.FOCUS:system-logs.txt
[PII_fb307417] 03:00:01 [PII_2f85cd02]]: [PII_f5d6a8c5] [conector-crm-legado] heartbeat ACK status=200 OK [PII_21a5c722] [PII_731e2c36]) [PII_8879697a] [PII_83f86c95]
  [×200 repeticiones idénticas omitidas]


---
## INSTRUCTION
Project: **02-pii-compliance-governance** | Repo: `examples/02-pii-compliance-governance/repo`
Aplica los prompts y contexto anterior. Entrega tu informe técnico y finaliza ÚNICAMENTE con el siguiente bloque de síntesis (no guardes el chat completo en memoria):

```memory
## YYYY-MM-DD — session
**Realizado:** <resumen corto de 1 línea>
**Resuelto:** <hallazgos resueltos>
**Pendiente:** <deuda técnica o tareas futuras>
**Decisiones:** <decisiones de diseño/stack>
**Errores del LLM:** <ninguno u observaciones>
```

```
