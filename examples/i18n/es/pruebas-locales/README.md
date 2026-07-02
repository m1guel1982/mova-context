# Ejemplo mínimo oficial — pruebas-locales

Proyecto de referencia para validar Mova Context de punta a punta.

## Qué incluye

- API de tareas (Node.js + Express + SQLite)
- 5 tasks: `crear-proyecto`, `agregar-modulo`, `auditar-modulo`, `configurar-cicd`, `generar-tests`
- Agentes: `backend-dev`, `security-architect`, `devops-engineer`, `qa-engineer`
- Skills: `lazy-minimalism`, `generate-tests`
- Restricciones YAGNI estrictas: sin Docker, sin `.env`, sin CI/CD innecesario

## Ejecución

```bash
# Generar contexto
mova run pruebas-locales crear-proyecto > contexto.txt

# Pegar contexto.txt en Claude/GPT y ejecutar.
# Al finalizar la sesión, guardar memoria:
mova memory pruebas-locales "```memory
## 2026-01-21 — sesión
**Hecho:** server.js con POST /tasks, GET /tasks, PATCH /tasks/:id
**Pendiente:** módulo categorías
```"

# Siguiente tarea
mova run pruebas-locales agregar-modulo > contexto2.txt
```

## Validación rápida

```bash
# Los tres core deben aparecer en el contexto
mova run pruebas-locales crear-proyecto | grep "<!-- core:"
```

Salida esperada:

```text
<!-- core: yagni-core -->
<!-- core: kiss-dry-core -->
<!-- core: ockham-core -->
```

## Estructura de salida esperada

```text
.
├── server.js
├── package.json
└── database.js
```

Sin carpetas vacías. Sin Dockerfile. Sin configuraciones no solicitadas.
