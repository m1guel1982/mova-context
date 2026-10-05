## 2026-10-02 19:02 — tarea: auditar-datos
<!-- mova:entry task=auditar-datos sha=638926be667a -->
**Tarea:** auditar-datos
**Realizado:** auditoría de PII en cobros y notificaciones
**Hallazgos:**
- `src/facturacion/cobros.js::generarCobro` — cliente — se copia completo al cobro sin necesidad
- `src/facturacion/notificaciones.js::notificarCliente` — email — expuesto en el aviso
**Datos clave:** cliente.email, cliente.documento, cliente.nombre, generarCobro, notificarCliente
**Decisiones:** conservar solo cliente.flota en el cobro; el email se resuelve al enviar
**Pendiente:** acotar el objeto cliente en generarCobro
**Errores del LLM:** ninguno

---

## 2026-10-02 19:02 — tarea: mejorar-cobros
<!-- mova:entry task=mejorar-cobros sha=2a6ad9ff8a2c -->
**Tarea:** mejorar-cobros
**Realizado:** cambio mínimo en cobros para mostrar la llegada real con respaldo
**Hallazgos:**
- `src/facturacion/cobros.js::generarCobro` — llegada — pasa a etaReal || etaProgramada
**Datos clave:** generarCobro, construirFilaReporte, etaReal, etaProgramada
**Decisiones:** sin clases ni capas nuevas; mapeador de columnas solo si se agregan más columnas de tiempo
**Pendiente:** el planificador debe copiar etaReal en normalizarPedido
**Errores del LLM:** ninguno

---

## 2026-10-02 19:02 — tarea: revisar-ruteo
<!-- mova:entry task=revisar-ruteo sha=266c2bfc921f -->
**Tarea:** revisar-ruteo
**Realizado:** revisión de arquitectura de planificador.js y rutas.js
**Hallazgos:**
- `src/despacho/planificador.js::normalizarPedido` — etaReal — se pierde — lista blanca de campos
- `src/despacho/planificador.js::planificarEntrega` — flujo — normaliza antes de rutear, el resto de la cadena nunca ve campos descartados
**Datos clave:** normalizarPedido, planificarEntrega, elegirRuta, puntuarRuta, etaReal
**Decisiones:** no se toca el algoritmo de puntuación
**Pendiente:** el facturador debe conservar etaReal en el reporte de cobros; propagar el campo desde normalizarPedido
**Errores del LLM:** ninguno

---

