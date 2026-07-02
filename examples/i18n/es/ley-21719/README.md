# Ejemplo 1 — Ley 21.719 de Protección de Datos (Chile)

> **Este es el caso insignia de Mova Context.**
> Demuestra cómo desacoplar completamente el conocimiento legal del software.

---

## El problema

El 29 de agosto de 2024 se promulgó en Chile la **Ley 21.719 de Protección de Datos Personales**.

Entró en vigencia en 2026.

Esta ley afecta a toda empresa o persona que trate datos de personas naturales en Chile: bancos, retail, telecomunicaciones, salud, seguros, tecnología, startups, y cualquier otro sector.

### Antes de la ley

Una empresa típica tenía su conocimiento legal distribuido así:

```text
✗ En la cabeza del abogado de turno
✗ En documentos Word sin versionar
✗ En correos electrónicos de hace dos años
✗ En configuraciones específicas del proveedor de IA
✗ Reconstruido en cada sesión de trabajo
```

Cuando el abogado cambiaba, el conocimiento se perdía.

Cuando el modelo de IA cambiaba, el contexto debía reconstruirse.

Cuando la ley se actualizaba, nadie sabía cuántos sistemas afectaba.

### Con Mova Context

```text
✓ Ley 21.719 vive en archivos Markdown versionados
✓ El mismo conocimiento sirve para todos los sistemas
✓ El backend nunca cambia
✓ Solo cambia el conocimiento cuando la ley cambia
✓ Cualquier modelo puede usarlo (Claude, GPT, Gemini, Ollama)
```

---

## Sistemas afectados en una empresa típica

Una empresa mediana puede tener todos estos sistemas tratando datos personales:

| Sistema | Canal | ¿Afectado por la ley? |
|---------|-------|----------------------|
| Web corporativa | Web | ✓ |
| App móvil | Mobile | ✓ |
| API REST | API | ✓ |
| Backend legacy | Interno | ✓ |
| Call Center | Telefónico | ✓ |
| WhatsApp Business | Mensajería | ✓ |
| IVR (menú de voz) | Telefónico | ✓ |
| CRM | Interno | ✓ |
| Portal de clientes | Web | ✓ |
| Asistente interno de RRHH | Interno | ✓ |

### El problema real

Antes de Mova Context, cada sistema tenía su propia implementación del conocimiento legal:

```text
Sistema Web     → un desarrollador configuró el LLM con su interpretación
Call Center     → el equipo de operaciones tiene un manual diferente
App móvil       → el equipo mobile tiene otra versión
IVR             → el equipo de telefonía tiene su propio flujo
```

Cuatro equipos. Cuatro interpretaciones. Cuatro problemas de cumplimiento.

### Con Mova Context

```text
Un solo conocimiento → todos los sistemas
```

El backend de cada sistema no cambia.
Las reglas de cada sistema no cambian.
Solo el conocimiento de Mova Context se actualiza cuando la ley cambia.

---

## Cómo funciona — Flujo completo

### Configuración

```json
// projects/ley-21719/project.json
{
  "project": "ley-21719",
  "lang": "es",
  "adapter": "file",
  "llm": "claude",

  "agents": { "domain": "legal", "use": ["abogado-datos"] },
  "skills": { "domain": "legal", "use": ["ley-21719-obligaciones", "derechos-titulares"] },

  "tasks": {
    "analizar-contrato": { "prompt": "analizar-contrato-datos" },
    "evaluar-cumplimiento": { "prompt": "evaluar-cumplimiento" },
    "responder-titular": { "prompt": "responder-solicitud-titular" }
  }
}
```

---

### Caso End-to-End: Cliente solicita acceso a sus datos

**Canal:** WhatsApp Business

---

#### Paso 1 — Mensaje del cliente

```
Cliente: "Hola, quiero saber qué datos tienen de mí y cómo los usan"
```

---

#### Paso 2 — workflow.md selecciona el contexto

```text
1. Lee project.json → ley-21719
2. Detecta lang: "es"
3. Detecta llm: "claude"
4. Detecta task: "responder-titular"
5. Carga agents/legal/i18n/es/abogado-datos.md
6. Carga skills/legal/i18n/es/ley-21719-obligaciones.md
7. Carga skills/legal/i18n/es/derechos-titulares.md
8. Carga memory.md
```

---

#### Paso 3 — Agent seleccionado

```markdown
# Rol
Abogado especialista en protección de datos personales bajo Ley 21.719 (Chile).

# Comportamiento
* Citar el artículo específico de Ley 21.719 en cada observación
* La autoridad de supervisión es: Agencia de Protección de Datos Personales
```

---

#### Paso 4 — Skill seleccionada (extracto relevante)

```markdown
# Derechos de los Titulares — Ley 21.719

## Derecho de acceso
El titular puede solicitar:
* Confirmación de si se tratan sus datos
* Copia de los datos tratados
* Información sobre el origen, finalidad, destinatarios y plazo de conservación

## Plazos
Respuesta inicial: 30 días hábiles
```

---

#### Paso 5 — Prompt construido

```markdown
El cliente ha enviado la siguiente solicitud por WhatsApp:

"Hola, quiero saber qué datos tienen de mí y cómo los usan"

Identifica:
1. Qué derecho está ejerciendo el titular (Art. 17-28 Ley 21.719)
2. Qué debe responder la empresa y en qué plazo
3. Redacta una respuesta apropiada para WhatsApp (máximo 200 palabras)
4. Indica si se requiere verificación de identidad previa

Variables: tipo_solicitud=acceso
```

---

#### Paso 6 — Contexto enviado al LLM

El LLM recibe el agent + la skill + el prompt + la memoria de sesiones anteriores.

---

#### Paso 7 — Respuesta del LLM

```
Hola [Nombre],

Gracias por contactarnos. Identificamos que estás ejerciendo tu **derecho de acceso** 
según el Art. 17 de la Ley 21.719 de Protección de Datos Personales de Chile.

Tenemos un plazo de **30 días hábiles** para responder.

Para procesar tu solicitud necesitamos verificar tu identidad. 
Te enviaremos un formulario de verificación a tu correo registrado.

Una vez verificada tu identidad, recibirás:
• Confirmación de los datos que tratamos
• Información sobre el propósito de cada dato
• Con quién compartimos tus datos
• Por cuánto tiempo los conservamos

Este proceso es completamente **gratuito**.

Número de caso: [CASO-2026-XXXX]

Equipo de Privacidad
```

---

#### Paso 8 — Acción del backend

```python
# El backend NUNCA cambió. Solo recibe la respuesta del LLM y la envía.

def send_whatsapp_response(customer_id: str, message: str, case_id: str):
    case = create_privacy_case(
        customer_id=customer_id,
        type="access_request",
        channel="whatsapp",
        case_id=case_id,
        deadline=business_days(30)
    )
    whatsapp_api.send(customer_id, message)
    audit_log.record(case)
    return case
```

**El backend es el mismo que existía antes de la ley.**
Solo se agregó el campo `type="access_request"` y el `deadline`.

---

#### Paso 9 — Respuesta final al usuario

El cliente recibe la respuesta por WhatsApp con el número de caso y los próximos pasos.

---

#### Paso 10 — Auditoría

```json
{
  "case_id": "CASO-2026-0847",
  "type": "access_request",
  "channel": "whatsapp",
  "customer_id": "CL-12345",
  "received_at": "2026-06-29T10:30:00Z",
  "deadline": "2026-08-12",
  "legal_basis": "Art. 17 Ley 21.719",
  "status": "pending_identity_verification",
  "llm": "claude",
  "context_version": "2026-06-01"
}
```

---

#### Paso 11 — Por qué el backend nunca cambió

```text
ANTES de la ley:
  Backend → recibe mensaje → envía respuesta

DESPUÉS de la ley (con Mova Context):
  Backend → recibe mensaje → LLAMA AL LLM CON CONTEXTO → envía respuesta

El backend añadió una llamada al LLM.
El LLM tiene el conocimiento de la Ley 21.719.
El backend no necesita saber nada de la ley.
```

---

## El mismo flujo con diferentes adaptadores

### Con archivos (default)

```bash
mova run ley-21719 responder-titular
```

El conocimiento vive en `skills/legal/i18n/es/ley-21719-obligaciones.md`.

### Con PostgreSQL

```json
{ "adapter": "postgresql", "dsn": "postgres://..." }
```

El conocimiento vive en la tabla `knowledge`. El workflow es idéntico.

### Con MongoDB

```json
{ "adapter": "mongodb", "dsn": "mongodb://..." }
```

El conocimiento vive en la colección `knowledge`. El workflow es idéntico.

**Solo cambia el adaptador. Todo lo demás permanece igual.**

---

## El mismo flujo con diferentes modelos

Solo cambia `"llm"` en `project.json`. Los agentes, skills y prompts son idénticos.

| Modelo | project.json |
|--------|-------------|
| Claude | `"llm": "claude"` |
| GPT-4 | `"llm": "gpt"` |
| Gemini | `"llm": "gemini"` |
| Ollama (local) | `"llm": "ollama"` |
| Llama | `"llm": "openrouter", "model": "meta-llama/llama-3.3-70b"` |
| DeepSeek | `"llm": "openrouter", "model": "deepseek/deepseek-r1"` |
| Mistral | `"llm": "openrouter", "model": "mistralai/mistral-large"` |

---

## Tareas disponibles

| Task | Descripción |
|------|-------------|
| `analizar-contrato` | Analizar un contrato bajo la Ley 21.719 |
| `evaluar-cumplimiento` | Evaluar el estado de cumplimiento de la empresa |
| `redactar-politica` | Redactar una política de privacidad conforme a la ley |
| `responder-titular` | Redactar respuesta a solicitud de un titular |

---

## A quién le sirve este ejemplo

| Rol | Qué obtiene |
|-----|-------------|
| **Desarrollo** | El backend no cambia. Solo agrega una llamada al LLM |
| **Arquitectura** | El conocimiento legal está desacoplado del código |
| **Jurídico** | Actualiza la ley en un solo lugar, se propaga a todos los sistemas |
| **Compliance** | Trazabilidad completa de cada decisión legal tomada por el LLM |
| **Gerencia** | Un sistema que cumple la ley en todos los canales simultáneamente |
| **Comercial** | Argumento de venta: cumplimiento legal sin reescribir sistemas |
| **Product Owner** | Features de cumplimiento sin deuda técnica |

---

## Archivos de este ejemplo

```text
projects/ley-21719/
├── project.json                        ← configuración del proyecto
└── memory.md                           ← historial de sesiones

agents/legal/i18n/es/
└── abogado-datos.md                    ← quién es el modelo

skills/legal/i18n/es/
├── ley-21719-obligaciones.md           ← qué sabe el modelo sobre la ley
└── derechos-titulares.md              ← derechos específicos de los titulares

prompts/legal/i18n/es/
├── analizar-contrato-datos.md          ← tarea: analizar contratos
├── evaluar-cumplimiento.md             ← tarea: evaluar cumplimiento
├── redactar-politica-privacidad.md     ← tarea: redactar política
└── responder-solicitud-titular.md      ← tarea: responder al titular
```

---

## Ejecutar este ejemplo

```bash
# Clonar el repositorio
git clone https://github.com/tu-usuario/mova-context

# Ver las tareas disponibles
mova list

# Generar contexto para analizar un contrato
mova run ley-21719 analizar-contrato > contexto.txt

# Copiar contexto.txt y pegar en Claude, ChatGPT o Gemini

# Guardar la respuesta del LLM
mova memory ley-21719 "$(pbpaste)"
```
