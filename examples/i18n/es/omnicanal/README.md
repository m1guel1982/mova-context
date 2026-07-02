# Ejemplo 2 — Omnicanal Empresarial

> Demuestra cómo una empresa usa el mismo conocimiento en todos sus canales de atención.

---

## El principio

```text
Un solo conocimiento → todos los canales
```

El backend nunca cambia.
Los canales nunca contienen reglas.
Todo el conocimiento vive en Mova Context.

---

## Canales demostrados

| Canal | Sistema | ¿Quién atiende? |
|-------|---------|-----------------|
| WhatsApp | Mensajería | LLM con agent ventas |
| WebChat | Web | LLM con agent ventas |
| IVR | Telefonía | LLM con agent IVR |
| Call Center | Telefonía | Agente humano + LLM de apoyo |
| CRM | Interno | LLM con agent CRM |
| Mesa de Ayuda | Interno | LLM con agent soporte |
| Portal Web | Web | LLM con agent autoservicio |

---

## Configuración

```json
// projects/omnicanal-demo/project.json
{
  "project": "omnicanal-demo",
  "description": "Acme Services — Omnicanal Empresarial",
  "lang": "es",
  "adapter": "file",
  "llm": "claude",

  "variables": {
    "empresa": "Acme Services",
    "producto": "Plan Enterprise",
    "precio_base": "USD 299/mes",
    "descuento_maximo": "25%",
    "periodo_prueba": "30 días"
  },

  "tasks": {
    "ventas-whatsapp": {
      "agents": { "domain": "callcenter", "use": ["ejecutivo-ventas"] },
      "skills": { "domain": "callcenter", "use": ["politica-ventas", "objeciones"] },
      "prompt": "respuesta-ventas",
      "variables": { "canal": "whatsapp", "etapa": "primer-contacto" }
    },
    "ventas-webchat": {
      "agents": { "domain": "callcenter", "use": ["ejecutivo-ventas"] },
      "skills": { "domain": "callcenter", "use": ["politica-ventas", "objeciones"] },
      "prompt": "respuesta-ventas",
      "variables": { "canal": "webchat", "etapa": "primer-contacto" }
    },
    "ivr-menu": {
      "agents": { "domain": "callcenter", "use": ["asistente-ivr"] },
      "skills": { "domain": "callcenter", "use": ["rutas-ivr"] },
      "prompt": "respuesta-ivr"
    },
    "cobranza-temprana": {
      "agents": { "domain": "callcenter", "use": ["ejecutivo-ventas"] },
      "skills": { "domain": "callcenter", "use": ["politica-cobranza"] },
      "prompt": "respuesta-cobranza",
      "variables": { "etapa_deuda": "temprana", "dias_mora": "20" }
    }
  }
}
```

---

## Flujo: Cliente escribe por WhatsApp y luego llama al IVR

### Canal 1 — WhatsApp

```
Cliente: "Hola, me interesa el Plan Enterprise"
```

```text
workflow.md → task: ventas-whatsapp
  ↓ carga agents/callcenter/i18n/es/ejecutivo-ventas.md
  ↓ carga skills/callcenter/i18n/es/politica-ventas.md
  ↓ carga skills/callcenter/i18n/es/objeciones.md
  ↓ ejecuta
  ↓ respuesta personalizada para WhatsApp
```

```
Respuesta: "¡Hola! Me alegra tu interés. El Plan Enterprise incluye [beneficios].
El precio base es USD 299/mes con un período de prueba gratuito de 30 días.
¿Te gustaría agendar una demo?"
```

### Canal 2 — IVR (mismo cliente llama 5 minutos después)

```
Menú IVR: "Para ventas, presione 1. Para soporte, presione 2."
Cliente: [presiona 1]
```

```text
workflow.md → task: ivr-menu → sub-task: ventas
  ↓ carga agents/callcenter/i18n/es/asistente-ivr.md
  ↓ carga skills/callcenter/i18n/es/rutas-ivr.md
  ↓ ejecuta
  ↓ respuesta de voz para IVR
```

```
IVR: "Gracias por llamar a Acme Services. Veo que recientemente
consultaste por el Plan Enterprise. ¿Deseas hablar con un
ejecutivo de ventas o prefieres recibir más información por WhatsApp?"
```

**El mismo conocimiento de ventas funciona en ambos canales.**
**El IVR sabe del contacto por WhatsApp porque ambos leen memory.md.**

---

## Por qué el backend nunca cambia

```text
ANTES de Mova Context:
  WhatsApp Backend  → tiene sus propias reglas de ventas
  IVR Backend       → tiene sus propias reglas de ventas
  CRM Backend       → tiene sus propias reglas de ventas

  3 equipos. 3 versiones de las mismas reglas. 3 problemas de consistencia.

CON Mova Context:
  WhatsApp Backend  → llama a LLM con contexto de ventas de Mova
  IVR Backend       → llama a LLM con contexto de ventas de Mova
  CRM Backend       → llama a LLM con contexto de ventas de Mova

  1 conocimiento. 3 canales. Consistencia garantizada.
```

---

## Archivos de este ejemplo

```text
projects/omnicanal-demo/
├── project.json
└── memory.md

agents/callcenter/i18n/es/
├── ejecutivo-ventas.md
└── asistente-ivr.md

skills/callcenter/i18n/es/
├── politica-ventas.md
├── politica-cobranza.md
├── objeciones.md
└── rutas-ivr.md

prompts/callcenter/i18n/es/
├── respuesta-ventas.md
├── respuesta-cobranza.md
└── respuesta-ivr.md
```

---

## Ejecutar este ejemplo

```bash
# Simular una llamada de ventas por WhatsApp
mova run omnicanal-demo ventas-whatsapp > contexto.txt

# Simular un menú IVR
mova run omnicanal-demo ivr-menu > contexto-ivr.txt

# Simular cobranza temprana
mova run omnicanal-demo cobranza-temprana > contexto-cobranza.txt
```
