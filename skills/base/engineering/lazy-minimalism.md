# Skill: Lazy Minimalism
Aplica `yagni-core.md` + `kiss-dry-core.md` como escalera de decisión antes de escribir código.

# Escalera (detenerse en el primer peldaño que resuelve)
1. ¿Esto necesita existir? (YAGNI)
2. ¿La librería estándar ya lo resuelve?
3. ¿Una funcionalidad nativa de la plataforma lo cubre?
4. ¿Una dependencia ya instalada lo resuelve?
5. ¿Cabe en una línea?
6. Solo entonces: el mínimo código que funcione

# Reglas
Sin abstracción no pedida · sin dependencia nueva evitable · sin boilerplate no solicitado · eliminar antes que agregar · menor número de archivos posible · entre dos enfoques del mismo tamaño, el correcto en casos límite (perezoso ≠ frágil)

# No es perezoso en
Validación en límites de confianza · manejo de errores que evita pérdida de datos · seguridad (secrets/auth/PII) · accesibilidad · calibración de hardware real · lo pedido explícitamente

# Atajos intencionales
Marcar con `# lazy:` indicando límite conocido (lock global, O(n²), heurística ingenua) y camino de mejora.

# Verificación mínima
Toda lógica no trivial deja un assert o test mínimo, sin frameworks. One-liners triviales no requieren test.

# Formato de respuesta
```txt
Peldaño: [1-6]
Código:
lazy: [si aplica]
Verificación: [assert/test, o "trivial"]
```

# Relación con ponytail
Versión genérica de `prompts/custom/ponytail.md`. Criterio de activación: `docs/GUIDE.md#cuándo-usar-ponytail`.
