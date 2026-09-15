# PROMPT: auto-patch

Propósito: Analizar el problema planteado y generar el parche del archivo especificado inmediatamente.

---

## REGLAS DE SALIDA ESTRICTAS

1. La PRIMERA LÍNEA de tu respuesta DEBE SER EXACTAMENTE la apertura del bloque de código Markdown indicando el lenguaje del archivo (ej. ```javascript, ```go, ```python, etc.).
2. PROHIBIDO generar comentarios, explicaciones, cabeceras o etiquetas antes del bloque de código (NO uses etiquetas como `<<<FILE: ...>>>` ni prefijos de archivo).
3. El archivo debe ser COMPLETO y 100% funcional. Prohibido usar omisiones o comentarios del tipo `// ... resto del código`.
4. Cierra el archivo de código con ```.
5. Cualquier explicación o resumen opcional DEBE ir ÚNICAMENTE después del cierre del bloque de código.

---

## TAREA A EJECUTAR

Archivo a modificar: {{PROJECT_PATH}}
Instrucción del usuario: {{USER_INSTRUCTION}}

Código actual / Contexto:
{{FILE_CONTENT}}

---

Genera el parche ahora comenzando directamente con el bloque de código Markdown (```):