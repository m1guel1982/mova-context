## 2026-10-02 19:00 — tarea: agregar-columnas
<!-- mova:entry task=agregar-columnas sha=87b6174deac2 -->
**Tarea:** agregar-columnas
**Realizado:** propuesta de cambio para columnas Llegada real / Llegada programada
**Hallazgos:**
- `src/despacho/planificador.js::normalizarPedido` — etaReal — corregido — ahora se copia
- `src/facturacion/cobros.js::generarCobro` — llegada — etaReal si existe, si no etaProgramada
**Datos clave:** llegadaReal, llegadaProgramada, 'Llegada real', 'Llegada programada'
**Decisiones:** 'Hora llegada' aplica etaReal > etaProgramada; las dos columnas nuevas muestran cada fuente sin mezclar
**Pendiente:** agregar prueba mínima con un pedido con etaReal y otro sin ella (NB-1001 / NB-1002)
**Errores del LLM:** ninguno

---

## 2026-10-02 19:00 — tarea: analizar-trazabilidad
<!-- mova:entry task=analizar-trazabilidad sha=3c8c4ec2858d -->
**Tarea:** analizar-trazabilidad
**Realizado:** trazado de etaProgramada y etaReal desde el pedido crudo hasta el CSV de cobros
**Hallazgos:**
- `src/despacho/planificador.js::normalizarPedido` — etaReal — se pierde — reconstruye el objeto y no copia etaReal (primer punto de pérdida)
- `src/despacho/planificador.js::planificarEntrega` — etaReal — nunca llega aguas abajo — normaliza antes de rutas, tarifa y cobro
- `src/facturacion/cobros.js::generarCobro` — etaProgramada — se renombra llegada — no existe etaReal
- `src/facturacion/cobros.js::construirFilaReporte` — columna 'Hora llegada' — usa solo la programada
**Datos clave:** etaProgramada, etaReal, llegada, normalizarPedido, generarCobro, construirFilaReporte, exportarReporteCobros
**Decisiones:** regla etaReal>etaProgramada solo en el valor final del reporte; conservar ambas en el flujo
**Pendiente:** conservar etaReal en normalizarPedido; agregar llegadaReal/llegadaProgramada en generarCobro; nuevas columnas en construirFilaReporte
**Errores del LLM:** ninguno

---

