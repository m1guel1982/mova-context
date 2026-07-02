# Ejemplo PostgreSQL — Mova Context con adaptador de base de datos

Mova Context puede leer agents, skills y prompts desde PostgreSQL.
El workflow, los projects y la lógica del CLI no cambian.

## Lo único que cambia: project.json

```json
{
  "project": "mi-proyecto",
  "lang": "es",
  "adapter": "db",
  "dsn": "postgres://usuario:password@localhost:5432/mova_db?sslmode=disable"
}
```

## Configuración

### 1. Crear la base de datos

```bash
createdb mova_db
psql mova_db < schema/postgresql.sql
```

El schema está en `mova-context/schema/postgresql.sql`.

### 2. Poblar con los archivos existentes

```bash
# Ejemplo: insertar un agent
psql mova_db -c "
INSERT INTO knowledge (kind, domain, lang, name, content)
VALUES ('agent', 'base', 'es', 'backend-dev',
  \$\$(contenido de agents/base/i18n/es/engineering/backend-dev.md)\$\$);
"
```

O usar un script de migración:

```bash
# Script de ejemplo (adaptar según el SO)
for f in agents/base/i18n/es/engineering/*.md; do
  name=$(basename "$f" .md)
  content=$(cat "$f")
  psql mova_db -c "INSERT INTO knowledge (kind,domain,lang,name,content)
    VALUES ('agent','base','es','$name',\$\$$content\$\$) ON CONFLICT DO NOTHING;"
done
```

### 3. Ejecutar

```bash
# Con DSN en project.json
mova run mi-proyecto mi-task > contexto.txt

# O con variables de entorno (override)
MOVA_ADAPTER=db MOVA_DSN="postgres://..." mova run mi-proyecto mi-task > contexto.txt
```

## Lectura y escritura de memoria

```bash
# La memoria también se almacena en la DB cuando adapter=db
mova memory mi-proyecto "```memory
## 2026-01-21 — sesión
**Hecho:** módulo X implementado
```"
```

## Schema de referencia

Ver `mova-context/schema/postgresql.sql` para la definición completa de tablas.

Tablas principales:
- `knowledge` — agents, skills, prompts
- `projects` — configuración de proyectos
- `memory` — historial de sesiones

## Conclusión

Solo cambia el `adapter` y el `dsn` en `project.json`.
El workflow, los agents, skills y prompts son exactamente los mismos.
