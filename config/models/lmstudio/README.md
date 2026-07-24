Este proveedor todavía no tiene ningún modelo configurado.

Ya no existe un `config.json` separado por proveedor — creá acá el
primer `<modelo>.json` a mano con conexión + parámetros juntos, por
ejemplo `config/models/lmstudio/mi-modelo.json`:

```json
{
  "provider": "lmstudio",
  "type": "openai-compatible",
  "base_url": "http://localhost:1234",
  "timeout_seconds": 120,

  "tipo": "llm",
  "model": "nombre-exacto-que-espera-lm-studio",
  "temperature": 0.2,
  "num_predict": 512
}
```

Después: `mova config lmstudio` (ya funciona con la carpeta vacía) y
`mova chat` usando ese `<modelo>` — ver docs/i18n/es/COMMANDS.md.
