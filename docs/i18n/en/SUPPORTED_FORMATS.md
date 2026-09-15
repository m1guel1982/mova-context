# Supported formats

All file creation/editing (`mova chat`, MCP, HTTP) goes through the same path-resolution logic and the
same writers, regardless of the door.

```
Text/config:   .txt .md .json .yml .yaml .xml .csv .toml .ini .env .log
Code:          .js .ts .py .go .cs .java .php .rb .rs .c .cpp .h .kt .swift .sh
Web:           .html .css .sql
Office:        .docx .xlsx .pdf   (used by context-report.pdf, mova-budget-report.md, etc.)
Media/diagram: .svg .png
Directories:   recursive creation, any name/depth
```

A format outside this list returns a clear error (`Unsupported file type`) instead of silently failing.
