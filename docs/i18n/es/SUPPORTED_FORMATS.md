# Formatos soportados

Toda creación/edición de archivos (`mova chat`, MCP, HTTP) pasa por la misma lógica de resolución de rutas
y los mismos escritores, sin importar la puerta.

```
Texto/config:  .txt .md .json .yml .yaml .xml .csv .toml .ini .env .log
Código:        .js .ts .py .go .cs .java .php .rb .rs .c .cpp .h .kt .swift .sh
Web:           .html .css .sql
Office:        .docx .xlsx .pdf   (usados por context-report.pdf, mova-budget-report.md, etc.)
Media/diagrama: .svg .png
Directorios:   creación recursiva, cualquier nombre/profundidad
```

Un formato fuera de esta lista devuelve un error claro (`Unsupported file type`) en vez de fallar en
silencio.
