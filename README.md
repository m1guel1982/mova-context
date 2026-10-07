# Mova Context

**Especificación de contexto por tarea, validada y con evidencia — antes de la inferencia.**
**Per-task context specification, validated and evidenced — before inference.**

- [Español](docs/i18n/es/README.md)
- [English](docs/i18n/en/README.md)

```
project.json (focus / exclude / policies)  →  cierre de dependencias  →  contexto determinista y sanitizado  →  runs/<run_id>/{context.txt, manifest.json, events.jsonl}
```

Mova es un binario local (CLI · `mova chat` · MCP stdio · HTTP en loopback). **No es** un gateway, un DLP, un RAG ni un coding agent: gobierna lo que pasa **por Mova**, y las lecturas del agente solo si instalas sus hooks. El perímetro exacto está en [PERIMETER](docs/i18n/es/GOVERNANCE_CONTROLS.md) · [EN](docs/i18n/en/GOVERNANCE_CONTROLS.md).

```bash
mova run 04-nebula-delivery recalcular-tarifas   # bloquea: calcularTarifa depende de un archivo excluido
mova run 04-nebula-delivery agregar-columnas     # libera un contexto cerrado y deja runs/<run_id>/
```

Desinstalar / Uninstall: [`uninstallers/`](uninstallers/README.md)
