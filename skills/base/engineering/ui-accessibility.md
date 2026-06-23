# Objetivo
Revisar accesibilidad. Standard: WCAG 2.1 AA
KISS+DRY: ver `kiss-dry-core.md`.

# Verificaciones
* Contraste 4.5:1 texto normal, 3:1 texto grande (≥18px)
* Alt text en imagen informativa; `alt=""` en decorativa
* Navegación completa por teclado: Tab, Enter, Escape
* Focus siempre visible
* Labels asociados a cada formulario
* Roles ARIA correctos en elementos no semánticos

# Anti-patrones
ARIA redundante sobre elemento semántico · `div` clickeable sin `role`/keyboard handler · modal sin focus trap · info transmitida solo por color

# Output
Violaciones con referencia WCAG y fix en código.
