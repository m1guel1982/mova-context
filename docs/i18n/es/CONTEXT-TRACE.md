# context-trace — cómo se toma la decisión de contexto

`context-trace` responde, antes de gastar un solo token real: cuánto va a costar, qué datos privados
podrían colarse, y por qué regla se tomó cada decisión. Es el punto de control entre "tengo una carpeta de
código" y "esto está a punto de ir a un LLM".

## Cada archivo pasa por uno de estos 5 estados

| Estado | Significa |
|---|---|
| `DISCOVERED` | Encontrado durante el escaneo. |
| `CANDIDATE` | Dentro del alcance (`focus`/relevancia). |
| `ALLOWED` | Nada sospechoso — pasa tal cual. |
| `SANITIZED` | Se encontró algo (API key, email...) y se enmascaró antes de contar como enviable. |
| `BLOCKED` | Algo crítico (llave privada) — el archivo **completo** queda fuera. |
| `EXCLUDED` | Fuera por razones no relacionadas a seguridad (binario, build, no relevante a la tarea). |

Nunca se sanea nada en silencio: cada decisión queda con la regla exacta que la tomó, en `context-report.md`
y `pii-audit-log.json`.

## Dos modos

```bash
mova context-trace <proyecto>              # modo proyecto: usa project.json (focus/exclude/task)
mova context-trace --repo <url>             # modo discovery: escanea todo, sin project.json
```

# Escaneo local especificando tarea y podando docstrings
mova context-trace --repo C:\testMovaContext\fastapi --task "solve_dependencies get_dependant in fastapi/dependencies/utils.py" --prune-docstrings

# Escaneo desde repositorio remoto con exportación a PDF y patrones de exclusión
mova context-trace --repo [https://github.com/fastapi/fastapi](https://github.com/fastapi/fastapi) --export pdf --task "fix dependency injection" --ignore "docs/**, tests/**, *.lock, .github/**"


El modo proyecto nunca cuenta tokens dos veces — reusa `mova budget`. El modo discovery cuenta archivo
por archivo (para el desglose "tokens por directorio").

Salida siempre: `context-report.md`/`.pdf` + `context-diagram.png` (con `--diagram`) + `pii-audit-log.json`.
