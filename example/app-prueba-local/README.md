# tareas-api

API mínima de tareas generada con [Mova Context](https://github.com/tu-usuario/mova-context).

**Requiere Node.js 22+. Sin dependencias de base de datos — usa `node:sqlite` nativo.**

## Arrancar

```bash
npm install
npm start
```

## Tests

```bash
npm test
```

## Endpoints

```bash
# Crear tarea
curl -X POST http://localhost:3000/api/v1/tareas \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dev-key-local" \
  -d '{"titulo":"Mi primera tarea"}'

# Listar tareas
curl http://localhost:3000/api/v1/tareas \
  -H "X-API-Key: dev-key-local"

# Completar tarea
curl -X PATCH http://localhost:3000/api/v1/tareas/1/completar \
  -H "X-API-Key: dev-key-local"
```

## Variables de entorno

| Variable | Default | Descripción |
|---|---|---|
| `PORT` | `3000` | Puerto del servidor |
| `API_KEY` | `dev-key-local` | Clave de autenticación |
| `DB_PATH` | `:memory:` | Ruta SQLite (`:memory:` = sin persistencia) |

## Generado con

```bash
mova run pruebas-locales crear-proyecto
```
