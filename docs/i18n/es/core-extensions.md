# Core + Extensions

> Documentación: [Español](core-extensions.md) · [English](../en/core-extensions.md)

---

## Qué es

Mova Context existe en dos ediciones que comparten un único núcleo:

```text
Core (Open Source, GitHub)          ← workflow.md, project.json, agents,
                                        skills, prompts, memory, MCP, CLI,
                                        adaptadores, i18n, Context Compiler

Extensions (binarios comerciales)   ← se instalan encima del Core
                                        sin modificarlo ni reemplazarlo
```

No hay dos arquitecturas. Hay un Core que funciona solo, y Extensions opcionales que lo amplían.

---

## Cómo funciona

El CLI (`mova`) es siempre el mismo binario para ambas ediciones. Al iniciar:

```text
1. Cargar Core
2. Leer project.json
3. Inicializar workflow.md
4. Detectar licencia (si existe)
5. Detectar módulos instalados
6. Registrar únicamente las funcionalidades adicionales
7. Continuar ejecución normal
```

Si no hay licencia ni módulos instalados, el CLI funciona exactamente igual que la edición Open Source. Nunca genera errores por su ausencia — es el estado por defecto de todo proyecto Mova Context.

La dirección de dependencia siempre es:

```text
Extensions → Core     (correcto)
Core → Extensions     (nunca)
```

El Core define contratos (la interfaz `Adapter` en `src/cli/adapter.go`, el bloque `contextCompiler` en `project.json`, los puntos de extensión del Context Compiler). Las Extensions los implementan o los amplían — nunca los modifican.

---

## Qué permanece en Open Source

Todo lo que hace funcionar un proyecto de punta a punta: `workflow.md`, `project.json`, agents/skills/prompts base, el adaptador de archivos, adaptadores PostgreSQL/MongoDB, el servidor MCP básico, el Context Compiler completo (Fase 1 y Fase 2), y toda la documentación bilingüe. Ver [`OPEN_SOURCE_STRATEGY.md`](../../../OPEN_SOURCE_STRATEGY.md) para el detalle completo.

## Qué incorpora la edición comercial

Conectores empresariales (Salesforce, SAP, ServiceNow...), UI de administración, sincronización avanzada, métricas de uso, asistentes preentrenados por industria y soporte multi-tenant. Nunca reemplazan una pieza del Core — se registran como módulos adicionales durante el arranque del CLI.

---

## Cómo actualizar una instalación Open Source a la edición comercial

```text
1. Instalar los módulos comerciales (binarios oficiales) + la licencia
2. Reiniciar el CLI
3. Las funcionalidades adicionales quedan disponibles automáticamente
```

Nunca reinstalar el proyecto. Nunca modificar `workflow.md`, `project.json`, agents, skills, prompts, memory o la estructura i18n. El ejecutable del CLI no cambia.

---

## Cómo mantener compatibilidad futura

- Todo proyecto creado con el Core debe abrirse sin cambios cuando existan Extensions instaladas.
- Todo proyecto que solo use funcionalidades del Core debe seguir funcionando si se desinstalan las Extensions.
- `project.json` sigue siendo la única fuente de configuración; `workflow.md`, el único punto de entrada.
- Toda funcionalidad nueva del Core (como el Context Compiler) se diseña primero para funcionar sola, sin depender de ninguna Extension — así ocurrió aquí: `contextCompiler` es 100% Core.

---

## Validación

| Comando | Resultado esperado |
|---|---|
| `mova run [proyecto]` sin ninguna Extension instalada | Funciona igual que la edición Open Source |
| `mova list` | Lista los proyectos sin importar si hay Extensions instaladas |
| Eliminar Extensions instaladas y volver a ejecutar cualquier comando | El CLI sigue funcionando, sin errores por su ausencia |
