# Egress audit — contexto sanitizado enviado al LLM

> Archivo auto-gestionado: un bloque por pieza del contexto. Igual → se omite; cambiado → se reemplaza completo; nuevo → se anexa. Nunca se duplica.

<!-- egress:block key="header" sha="56984fb5b20db474" updated="2026-10-02T19:00:27Z" -->
### header
```text
# Mova Context — 04-nebula-delivery / agregar-columnas
```
<!-- egress:end -->

<!-- egress:block key="section:AGENTS" sha="0a4b6ba87bea141d" updated="2026-10-02T19:00:27Z" -->
### section:AGENTS
```text
## AGENTS
```
<!-- egress:end -->

<!-- egress:block key="core:yagni-core" sha="a84657ea36bf7b4b" updated="2026-10-02T19:00:27Z" -->
### core:yagni-core
```text
<!-- core: yagni-core -->
# Núcleo YAGNI
Solo lo pedido: si la tarea no lo exige, no existe. Sin abstracciones, capas, endpoints, archivos ni dependencias «por si acaso»: ni interfaz de una sola implementación, ni factory de un solo producto, ni config para un valor que nunca cambia.
No generar sin pedido explícito: Docker · CI/CD · logging avanzado · auth compleja · .env · linters/formatters · carpetas reservadas al futuro · dependencias fuera del stack.
Ante duda de alcance, la opción más pequeña que cumple.
```
<!-- egress:end -->

<!-- egress:block key="agent:trazabilidad-dev" sha="1c605ebf988da4c1" updated="2026-10-02T19:00:27Z" -->
### agent:trazabilidad-dev
```text
<!-- agent: trazabilidad-dev -->
# Backend Developer Senior

Eres un Backend Developer Senior especializado en Node.js / JavaScript. Trabajas sobre la plataforma ficticia **Nebula Delivery** (cápsulas de reparto a la estación Selene).

## Alcance
Módulos bajo tu responsabilidad: src/despacho/planificador.js, src/despacho/rutas.js, src/facturacion/tarifas.js, src/facturacion/cobros.js

## Datos que debes rastrear
Llegada: etaProgramada (programada) y etaReal (real, medida por telemetría). Regla de renderizado: priorizar etaReal sobre etaProgramada.

Claves de origen: etaProgramada, etaReal, llegada

## Flujo que analizas
pedido crudo -> normalizarPedido -> planificarEntrega -> generarCobro -> construirFilaReporte -> exportarReporteCobros

## Regla de negocio
Conservar etaProgramada y etaReal durante todo el flujo. Solo en el valor FINAL del reporte se usa etaReal si existe y es válida; si no, etaProgramada.

## Cómo trabajas
- Citas siempre `archivo::función` y nunca inventas símbolos que no estén en el FOCUS.
- Los símbolos en `exclude` no existen para ti: no los analices ni los modifiques.
- Si la sección MEMORY trae hallazgos de otra tarea, parte de ellos y verifícalos contra el código.
```
<!-- egress:end -->

<!-- egress:block key="section:SKILLS" sha="13df5567134e26c3" updated="2026-10-02T19:00:27Z" -->
### section:SKILLS
```text
## SKILLS
```
<!-- egress:end -->

<!-- egress:block key="core:kiss-dry-core" sha="024e1e7f15694a22" updated="2026-10-02T19:00:27Z" -->
### core:kiss-dry-core
```text
<!-- core: kiss-dry-core -->
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
```
<!-- egress:end -->

<!-- egress:block key="skill:mapa-de-columnas" sha="337639800c814ac2" updated="2026-10-02T19:00:27Z" -->
### skill:mapa-de-columnas
```text
<!-- skill: mapa-de-columnas -->
# Mapa de columnas del reporte de cobros

Columnas del reporte: Pedido, Destino, Hora llegada, Monto (créditos)

Fuentes de datos: pedido crudo (datos/pedidos-demo.json) / plan de entrega / cobro

Formato de salida: CSV de cobros (exportarReporteCobros)

## Reglas
- Prioridad: etaReal > etaProgramada solo en el valor final del reporte
- Trazabilidad: No descartar ni sobrescribir etaReal/etaProgramada en etapas intermedias
- Detección de pérdidas: Detectar objetos reconstruidos campo a campo, proyecciones y mapeos que eliminen etaReal o etaProgramada

## Método
1. Para cada columna, localiza dónde nace el dato real y dónde el programado.
2. Señala la primera función donde uno de los dos deja de propagarse.
3. Propón el cambio mínimo, sin tocar lo que está en `exclude`.
```
<!-- egress:end -->

<!-- egress:block key="section:PROMPT" sha="eedb0f7af0aa2af9" updated="2026-10-02T19:00:27Z" -->
### section:PROMPT
```text
## PROMPT
```
<!-- egress:end -->

<!-- egress:block key="core:ockham-core" sha="3eb26bba1d3ffec3" updated="2026-10-02T19:00:27Z" -->
### core:ockham-core
```text
<!-- core: ockham-core -->
# Núcleo Ockham (salida)
Entre soluciones válidas, entrega la más simple. Código primero, sin prosa antes; si se explica solo, solo el bloque técnico. No repitas el contexto recibido ni reescribas lo que no cambió: entrega solo la función/símbolo/archivo modificado.
Cierre: una sola línea `omitido: X · añadir si: Y`. Si el pedido es amplio, entrega la versión mínima y cuestiona el resto en esa línea; no te detengas por algo que puedes asumir. Explicación solo si se pidió (informe, revisión).
```
<!-- egress:end -->

<!-- egress:block key="prompt:analizar-trazabilidad" sha="e0ce6f9fea9acc0b" updated="2026-10-02T19:00:27Z" -->
### prompt:analizar-trazabilidad
```text
<!-- prompt: analizar-trazabilidad -->
# Tarea: analizar trazabilidad (04-nebula-delivery)

Idioma de respuesta: es. Stack: Node.js / JavaScript.

## Consulta
Analizar planificador, rutas, tarifas y cobros para determinar dónde existen etaProgramada y etaReal y en qué función se pierde, sobrescribe o deja de propagarse cada una hasta el reporte de cobros.

## Qué debes entregar
1. Flujo: pedido crudo -> normalizarPedido -> planificarEntrega -> generarCobro -> construirFilaReporte -> exportarReporteCobros
2. Para cada clave de etaProgramada, etaReal, llegada: dónde existe y dónde deja de existir (`archivo::función`).
3. El primer punto de pérdida de cada dato, con su causa.

Objetivo: Encontrar el primer punto del flujo donde desaparece cada dato, sin modificar archivos.

No modifiques archivos todavía. Cierra con el bloque `memory` (campos Hallazgos y Datos clave): la tarea `agregar-columnas` partirá de él sin repetir tu análisis.
```
<!-- egress:end -->

<!-- egress:block key="section:FOCUS" sha="32e9bf30471d5472" updated="2026-10-02T19:00:27Z" -->
### section:FOCUS
```text
## FOCUS
```
<!-- egress:end -->

<!-- egress:block key="focus:src/despacho/planificador.js::func=normalizarPedido,validarVentanaOrbital,elegirRuta,planificarEntrega" sha="e05381b35a55bdd4" updated="2026-10-02T19:00:27Z" -->
### focus:src/despacho/planificador.js::func=normalizarPedido,validarVentanaOrbital,elegirRuta,planificarEntrega
```text
FOCUS:src/despacho/planificador.js::func=normalizarPedido,validarVentanaOrbital,elegirRuta,planificarEntrega
  (src/despacho/planificador.js)
function normalizarPedido(pedido) {
  return {
    id: pedido.id,
    destino: pedido.destino,
    pesoKg: pedido.pesoKg,
    etaProgramada: pedido.etaProgramada,
    cliente: pedido.cliente,
  };
}

function validarVentanaOrbital(pedido) {
  const eta = Date.parse(pedido.etaProgramada);
  if (Number.isNaN(eta)) throw new Error(`Pedido ${pedido.id}: ventana orbital inválida`);
  return eta - Date.now() > MARGEN_VENTANA_MIN * 60 * 1000;
}

function elegirRuta(pedido, catalogo) {
  const candidatas = catalogo.filter((r) => r.destino === pedido.destino);
  if (candidatas.length === 0) throw new Error(`Sin ruta hacia ${pedido.destino}`);
  return candidatas.reduce((mejor, r) =>
    puntuarRuta(r, pedido) < puntuarRuta(mejor, pedido) ? r : mejor
  );
}

function planificarEntrega(pedidoCrudo, catalogo) {
  const pedido = normalizarPedido(pedidoCrudo);
  validarVentanaOrbital(pedido);
  const ruta = elegirRuta(pedido, catalogo);
  const trayectoria = calcularTrayectoria(ruta.origen, ruta.destino);
  const tarifa = calcularTarifa(pedido, ruta);
  return { pedido, ruta, trayectoria, tarifa };
}FOCUS:src/despacho/rutas.js::func=calcularTrayectoria,estimarCombustible
  (src/despacho/rutas.js)
function calcularTrayectoria(origen, destino) {
  const base = Math.abs(destino.length - origen.length) * 12 + 40;
  return { origen, destino, deltaV: ajustarPorGravedad(base, destino) };
}

function estimarCombustible(ruta, pesoKg) {
  const presion = leerSensor('tanque-principal');
  return (ruta.duracionMin / 60) * pesoKg * 0.8 * (presion > 0 ? 1 : 1.2);
}FOCUS:src/facturacion/cobros.js::func=generarCobro,construirFilaReporte,exportarReporteCobros
  (src/facturacion/cobros.js)
function generarCobro(plan) {
  return {
    pedidoId: plan.pedido.id,
    destino: plan.pedido.destino,
    llegada: plan.pedido.etaProgramada,
    monto: plan.tarifa,
    cliente: plan.pedido.cliente,
  };
}

function construirFilaReporte(cobro) {
  return {
    Pedido: cobro.pedidoId,
    Destino: cobro.destino,
    'Hora llegada': cobro.llegada,
    'Monto (créditos)': cobro.monto,
  };
}

function exportarReporteCobros(cobros) {
  const filas = cobros.map(construirFilaReporte);
  const cabecera = Object.keys(filas[0] || {});
  return [cabecera.join(';'), ...filas.map((f) => cabecera.map((c) => f[c]).join(';'))].join('\n');
}
```
<!-- egress:end -->

<!-- egress:block key="section:INSTRUCTION" sha="1a79aa4bbba1e220" updated="2026-10-02T19:00:27Z" -->
### section:INSTRUCTION
````text
## INSTRUCTION
Project: **04-nebula-delivery** | Repo: `examples/06-nebula-delivery-graph-memory/repo`
Aplica los prompts y contexto anterior. Entrega tu informe técnico y finaliza ÚNICAMENTE con el siguiente bloque de síntesis. Es la MEMORIA que leerán las demás tareas: no guardes el chat completo, pero tampoco omitas nada que otra tarea necesite para continuar sin repetir tu análisis:

```memory
## YYYY-MM-DD — session
**Tarea:** <nombre de la tarea ejecutada>
**Realizado:** <resumen corto de 1 línea>
**Hallazgos:** <uno por línea: `archivo::función` — dato/clave — estado (existe / se pierde / se sobrescribe) — causa o evidencia>
**Datos clave:** <identificadores EXACTOS que otra tarea necesitará: funciones, claves, columnas, rutas, valores>
**Resuelto:** <hallazgos resueltos>
**Decisiones:** <reglas de negocio y decisiones de diseño acordadas>
**Pendiente:** <próximos pasos concretos y deuda técnica>
**Errores del LLM:** <ninguno u observaciones>
```

Reglas del bloque: copia literalmente archivos, funciones, claves y columnas (sin parafrasear); incluye solo el código mínimo imprescindible; si trabajaste varias tareas, entrega un bloque por tarea.
````
<!-- egress:end -->

<!-- egress:block key="prompt:agregar-columnas" sha="8ed420364c4e5c66" updated="2026-10-02T19:00:27Z" -->
### prompt:agregar-columnas
```text
<!-- prompt: agregar-columnas -->
# Tarea: agregar columnas al reporte (04-nebula-delivery)

Idioma de respuesta: es. Stack: Node.js / JavaScript.

## Consulta
Agregar al reporte de cobros las columnas 'Llegada real' y 'Llegada programada' usando etaReal cuando exista y etaProgramada como respaldo, sin pisar una fuente con la otra. Parte de los hallazgos de analizar-trazabilidad.

## Funciones a modificar
- Mapeo: normalizarPedido, generarCobro, construirFilaReporte
- Generación: exportarReporteCobros

Claves de origen: etaProgramada, etaReal, llegada
Prioridad: etaReal > etaProgramada solo en el valor final del reporte
Formato de salida: CSV de cobros (exportarReporteCobros)

## Punto de partida
Si la sección MEMORY contiene hallazgos de la tarea `analizar-trazabilidad`, úsalos como base (no repitas el análisis) y verifícalos contra el FOCUS antes de proponer código. Entrega el cambio mínimo como diff por `archivo::función`, y cierra con el bloque `memory` (qué cambiaste y qué queda pendiente).
```
<!-- egress:end -->

<!-- egress:block key="focus:src/despacho/planificador.js::func=normalizarPedido" sha="fd9fcc81b21836a1" updated="2026-10-02T19:00:27Z" -->
### focus:src/despacho/planificador.js::func=normalizarPedido
```text
FOCUS:src/despacho/planificador.js::func=normalizarPedido
  (src/despacho/planificador.js)
function normalizarPedido(pedido) {
  return {
    id: pedido.id,
    destino: pedido.destino,
    pesoKg: pedido.pesoKg,
    etaProgramada: pedido.etaProgramada,
    cliente: pedido.cliente,
  };
}FOCUS:src/facturacion/cobros.js::func=generarCobro,construirFilaReporte,exportarReporteCobros,registrarCobro
  (src/facturacion/cobros.js)
function generarCobro(plan) {
  return {
    pedidoId: plan.pedido.id,
    destino: plan.pedido.destino,
    llegada: plan.pedido.etaProgramada,
    monto: plan.tarifa,
    cliente: plan.pedido.cliente,
  };
}

function construirFilaReporte(cobro) {
  return {
    Pedido: cobro.pedidoId,
    Destino: cobro.destino,
    'Hora llegada': cobro.llegada,
    'Monto (créditos)': cobro.monto,
  };
}

function exportarReporteCobros(cobros) {
  const filas = cobros.map(construirFilaReporte);
  const cabecera = Object.keys(filas[0] || {});
  return [cabecera.join(';'), ...filas.map((f) => cabecera.map((c) => f[c]).join(';'))].join('\n');
}

function registrarCobro(cobro, libro) {
  libro.push(cobro);
  notificarCliente(cobro);
  return libro.length;
}FOCUS:src/facturacion/tarifas.js::func=calcularTarifa,aplicarDescuentoFlota,recargoRadiacion,redondearCreditos
  (src/facturacion/tarifas.js)
function redondearCreditos(valor) {
  return Math.round(valor * 100) / 100;
}

function recargoRadiacion(ruta) {
  return ruta.destino === 'titan-puerto' ? 1.35 : 1.0;
}

function aplicarDescuentoFlota(monto, cliente) {
  return cliente && cliente.flota ? monto * 0.9 : monto;
}

function calcularTarifa(pedido, ruta) {
  if (ruta.catalogo === 'v1') return tarifaPlanaV1(pedido);
  const bruto = pedido.pesoKg * CREDITOS_POR_KG * recargoRadiacion(ruta);
  return redondearCreditos(aplicarDescuentoFlota(bruto, pedido.cliente));
}
```
<!-- egress:end -->

<!-- egress:block key="section:MEMORY" sha="c215ed6ae8455df8" updated="2026-10-02T19:00:27Z" -->
### section:MEMORY
```text
## MEMORY
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
```
<!-- egress:end -->
