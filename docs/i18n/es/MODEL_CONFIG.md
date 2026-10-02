# Configuración de modelos — `max_tokens` (tope de respuesta)

Cada modelo se describe en `config/models/<proveedor>/<modelo>.json`.

## Un único campo estándar: `max_tokens`

`max_tokens` es el tope de tokens de **respuesta** y vale **igual para todos los proveedores**; Mova lo traduce al parámetro nativo de cada API:

| Proveedor (`type`) | Parámetro nativo que se envía | Default si no se declara |
|---|---|---|
| `openai` / OpenRouter / LM Studio / vLLM | `max_tokens` | 512 |
| `anthropic` | `max_tokens` | 1024 |
| `google` | `generationConfig.maxOutputTokens` | 2048 |
| `ollama` | `options.num_predict` | el de Ollama |

`num_predict` sigue aceptándose como **alias** (si ambos existen, gana `max_tokens`). Los archivos existentes con cualquiera de los dos funcionan sin cambios.
 

## Modelos con razonamiento

El razonamiento consume tokens de salida. Usa un `max_tokens` holgado (≥16384; DeepSeek vía OpenRouter: 32768) o limita el razonamiento con el campo opcional `reasoning`, que se reenvía tal cual a APIs openai-compatible que lo soportan (OpenRouter):

```json
{ "type": "openai", "max_tokens": 32768, "reasoning": { "effort": "low" } }
```
 