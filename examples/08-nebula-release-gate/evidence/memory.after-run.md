## 2026-10-05 00:52 — tarea: revisar-cierre
<!-- mova:entry task=revisar-cierre sha=2e2ff9fdec68 -->
**Tarea:** revisar-cierre
**Decisiones:** release NO pasa hasta copiar etaReal en normalizarPedido y acotar cliente en generarCobro

---

## 2026-10-05 00:52 — tarea: auditar-pii
<!-- mova:entry task=auditar-pii sha=e945b5ea1f35 -->
**Tarea:** auditar-pii
**Hallazgos:**
- src/facturacion/cobros.js::generarCobro — cliente — se copia completo al cobro sin necesidad
- src/facturacion/notificaciones.js::notificarCliente — email — solo se necesita al enviar
**Pendiente:** acotar cliente en generarCobro

---

## 2026-10-05 00:52 — tarea: mapear-perdidas
<!-- mova:entry task=mapear-perdidas sha=7a95d86e4c05 -->
**Tarea:** mapear-perdidas
**Hallazgos:**
- src/despacho/planificador.js::normalizarPedido — etaReal — se pierde — reconstruye el pedido campo a campo y no copia etaReal
**Pendiente:** el guardián debe revisar qué más se copia completo (cliente)

---

