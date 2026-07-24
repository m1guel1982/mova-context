# Skill: Consentimiento Granular (Ley 21.719 / equivalentes tipo GDPR)
KISS+DRY: ver `../engineering/kiss-dry-core.md`.

Regla base: el consentimiento debe ser libre, específico, informado e inequívoco. Cada finalidad de tratamiento de datos requiere su propio consentimiento — no se puede agrupar en un solo "acepto".

# Señales de infracción a buscar en un flujo (checkout, registro, formulario)

| Señal | Por qué infringe |
|---|---|
| Un solo checkbox para T&C + política de privacidad + tratamiento de datos | Agrupa finalidades distintas — el titular no puede aceptar una sin las otras |
| Checkbox pre-marcado por defecto | El consentimiento no es libre si ya viene "aceptado" |
| No se puede completar la acción sin aceptar tratamientos no esenciales | Consentimiento condicionado — prohibido |
| Texto legal sin resumen accesible | No es "informado" si el titular no entiende qué acepta |
| No hay forma de revocar el consentimiento después | Falta control continuo del titular |

# Corrección mínima a proponer

Separar en checkboxes independientes:

1. **T&C del servicio** — obligatorio solo si es imprescindible para completar la transacción
2. **Política de privacidad** — informativo, enlace visible, no bloquea la acción
3. **Tratamiento de datos con fines secundarios** (marketing, perfilamiento, terceros) — opcional, desmarcado por defecto, nunca bloquea la transacción

# Referencia (Chile, Ley 21.719)

Multas de 5.000 a 20.000 UTM según gravedad. Aplica a cualquier responsable de tratamiento — sin importar el tamaño, la antigüedad o el stack del sistema.
