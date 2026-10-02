# Núcleo KISS + DRY
Antes de escribir: lee la tarea y traza el flujo real completo (archivos que toca, quién llama a qué). La escalera acorta la solución, nunca la lectura.
Escalera (detente en el primer peldaño que resuelve):
1. ¿La plataforma nativa (HTML/CSS/SQL, constraint de BD) lo resuelve? Úsala.
2. ¿La librería estándar del lenguaje lo resuelve? Úsala.
3. ¿Ya existe en el proyecto (helper, util, tipo, patrón, dependencia instalada)? Búscalo y reutilízalo.
4. ¿Cabe en una línea? Una línea.
5. Solo entonces, el mínimo código propio.
Bug: antes de editar, busca todos los llamadores de la función; corrige una vez en la raíz común (un guard compartido < uno por llamador), no en la vista del síntoma.
DRY: una regla vive en un solo lugar. Eliminar antes que agregar; aburrido antes que ingenioso.
Perezoso ≠ frágil: entre dos opciones del mismo tamaño, la correcta en casos límite. Nunca omitir validación en límites de confianza, manejo de errores que evita pérdida de datos, seguridad (secrets/auth/PII), accesibilidad ni lo pedido explícitamente.
Atajo con techo conocido: `lazy: <límite> → <mejora>`. Lógica no trivial: un assert/test mínimo, sin frameworks.
