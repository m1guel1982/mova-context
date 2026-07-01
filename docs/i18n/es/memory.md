# Comandos de Memoria

Mova Context almacena la memoria activa del proyecto en `memory.md` y puede archivar automáticamente las entradas antiguas según la política de retención configurada.

## Leer Memoria

Muestra la memoria activa del proyecto.

```bash
mova memory-read [project]
```

---

## Limpiar Memoria

Permite eliminar la memoria utilizando distintos criterios.

### Eliminar toda la memoria

Elimina la memoria activa y todo el historial archivado.

```bash
mova memory-clear [project]
```

### Eliminar solo la memoria archivada

Conserva `memory.md` y elimina todos los meses archivados.

```bash
mova memory-clear [project] --archived
```

### Eliminar un día específico

Elimina todas las entradas correspondientes a una fecha determinada.

```bash
mova memory-clear [project] --date 2024-06-15
```

### Eliminar un rango de fechas

Elimina todas las entradas comprendidas entre dos fechas.

```bash
mova memory-clear [project] --from 2024-06-01 --to 2024-06-30
```

### Conservar la memoria activa

Elimina únicamente la memoria archivada, manteniendo `memory.md`.

```bash
mova memory-clear [project] --keep-active
```

---

# Configurar el Archivado Automático

Permite habilitar o deshabilitar el archivado automático y configurar el tiempo de retención.

### Habilitar el archivado automático

```bash
mova memory-config [project] enable
```

### Deshabilitar el archivado automático

```bash
mova memory-config [project] disable
```

### Configurar la retención

Define la cantidad de días que las entradas permanecerán en la memoria activa antes de archivarse.

```bash
mova memory-config [project] days N
```

Ejemplo:

```bash
mova memory-config mi-proyecto days 30
```
