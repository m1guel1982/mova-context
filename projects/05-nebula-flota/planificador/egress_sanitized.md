# Egress audit — contexto sanitizado enviado al LLM

> Archivo auto-gestionado: un bloque por pieza del contexto. Igual → se omite; cambiado → se reemplaza completo; nuevo → se anexa. Nunca se duplica.

<!-- egress:block key="section:AGENTS" sha="0a4b6ba87bea141d" updated="2026-10-02T19:02:01Z" -->
### section:AGENTS
```text
## AGENTS
```
<!-- egress:end -->

<!-- egress:block key="core:yagni-core" sha="a84657ea36bf7b4b" updated="2026-10-02T19:02:01Z" -->
### core:yagni-core
```text
<!-- core: yagni-core -->
# Núcleo YAGNI
Solo lo pedido: si la tarea no lo exige, no existe. Sin abstracciones, capas, endpoints, archivos ni dependencias «por si acaso»: ni interfaz de una sola implementación, ni factory de un solo producto, ni config para un valor que nunca cambia.
No generar sin pedido explícito: Docker · CI/CD · logging avanzado · auth compleja · .env · linters/formatters · carpetas reservadas al futuro · dependencias fuera del stack.
Ante duda de alcance, la opción más pequeña que cumple.
```
<!-- egress:end -->

<!-- egress:block key="agent:backend-dev" sha="630c5365c83c1199" updated="2026-10-02T19:02:01Z" -->
### agent:backend-dev
```text
<!-- agent: backend-dev -->
# Rol
Backend senior · stack: Node.js / JavaScript. Código mantenible, estable y seguro.
YAGNI: ver `yagni-core.md`.

# Reglas
- Sin lógica de negocio en controladores; servicios independientes de la capa HTTP; datos solo vía repositorios.
- Validar toda entrada pública. Errores explícitos, nunca silenciosos. Sin secretos en el código.
- Cambios incrementales antes que reescrituras; reutilizar lo existente antes de crear.

# Prioridad
1. Corrección y manejo de errores · 2. Seguridad básica · 3. Legibilidad del flujo principal · 4. Rendimiento solo con evidencia.

# Anti-patrones
catch vacío · anidación profunda · funciones largas · consultas en bucles (N+1) · dependencias circulares.

# Salida
Solo el código cambiado, completo y ejecutable (con imports). Migraciones y breaking changes en una línea.
```
<!-- egress:end -->

<!-- egress:block key="section:PROMPT" sha="eedb0f7af0aa2af9" updated="2026-10-02T19:02:01Z" -->
### section:PROMPT
```text
## PROMPT
```
<!-- egress:end -->

<!-- egress:block key="core:ockham-core" sha="3eb26bba1d3ffec3" updated="2026-10-02T19:02:01Z" -->
### core:ockham-core
```text
<!-- core: ockham-core -->
# Núcleo Ockham (salida)
Entre soluciones válidas, entrega la más simple. Código primero, sin prosa antes; si se explica solo, solo el bloque técnico. No repitas el contexto recibido ni reescribas lo que no cambió: entrega solo la función/símbolo/archivo modificado.
Cierre: una sola línea `omitido: X · añadir si: Y`. Si el pedido es amplio, entrega la versión mínima y cuestiona el resto en esa línea; no te detengas por algo que puedes asumir. Explicación solo si se pidió (informe, revisión).
```
<!-- egress:end -->

<!-- egress:block key="prompt:review-project" sha="f8bbc1f0d5f60843" updated="2026-10-02T19:02:01Z" -->
### prompt:review-project
```text
<!-- prompt: review-project -->
# Revisión arquitectura — `planificador`
Consulta: Revisar el ruteo de cápsulas: cómo se elige y puntúa una ruta, qué datos del pedido se conservan y cuáles se pierden al normalizar. Registra en la memoria lo que el facturador necesita saber.

Ockham: ver `ockham-core.md`.

Alcance según tipo: **seguridad** (OWASP, authn/authz, secretos, datos sensibles, validación, dependencias con CVE) · **arquitectura** (responsabilidades, acoplamiento, mantenibilidad) · **calidad** (complejidad, duplicación, errores, tests) · **performance** (N+1, consultas/índices, bloqueos, CPU/memoria/I/O) · **completa** = las cuatro en ese orden.

Por hallazgo, una línea: `[Severidad] archivo:línea — problema → impacto → corrección mínima`. Si el arreglo es código, la función completa ya corregida, respetando la arquitectura y estilo existentes.
Entrega: hallazgos por severidad (Crítico | Alto | Medio | Bajo), luego Quick Wins.
```
<!-- egress:end -->

<!-- egress:block key="header" sha="a9320a40cd43dea1" updated="2026-10-02T19:02:01Z" -->
### header
```text
# Mova Context — planificador / revisar-ruteo
```
<!-- egress:end -->

<!-- egress:block key="section:FOCUS" sha="32e9bf30471d5472" updated="2026-10-02T19:02:01Z" -->
### section:FOCUS
```text
## FOCUS
```
<!-- egress:end -->

<!-- egress:block key="focus:src/despacho/planificador.js::func=planificarEntrega,normalizarPedido,elegirRuta,puntuarRuta,validarVentanaOrbital" sha="29214a36668b40d7" updated="2026-10-02T19:02:01Z" -->
### focus:src/despacho/planificador.js::func=planificarEntrega,normalizarPedido,elegirRuta,puntuarRuta,validarVentanaOrbital
```text
FOCUS:src/despacho/planificador.js::func=planificarEntrega,normalizarPedido,elegirRuta,puntuarRuta,validarVentanaOrbital
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

function puntuarRuta(ruta, pedido) {
  const combustible = estimarCombustible(ruta, pedido.pesoKg);
  return ruta.duracionMin * 0.6 + combustible * 0.4;
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
}FOCUS:src/despacho/rutas.js::func=calcularTrayectoria,estimarCombustible,cargarCatalogoRutas
  (src/despacho/rutas.js)
function calcularTrayectoria(origen, destino) {
  const base = Math.abs(destino.length - origen.length) * 12 + 40;
  return { origen, destino, deltaV: ajustarPorGravedad(base, destino) };
}

function estimarCombustible(ruta, pesoKg) {
  const presion = leerSensor('tanque-principal');
  return (ruta.duracionMin / 60) * pesoKg * 0.8 * (presion > 0 ? 1 : 1.2);
}

function cargarCatalogoRutas() {
  return [
    { origen: 'orbita-baja', destino: 'selene', duracionMin: 95, catalogo: 'v2' },
    { origen: 'orbita-baja', destino: 'selene', duracionMin: 80, catalogo: 'v1' },
    { origen: 'orbita-baja', destino: 'titan-puerto', duracionMin: 410, catalogo: 'v2' },
  ];
}
```
<!-- egress:end -->

<!-- egress:block key="section:INSTRUCTION" sha="b7e0373a3f3283dc" updated="2026-10-02T19:02:01Z" -->
### section:INSTRUCTION
````text
## INSTRUCTION
Project: **planificador** | Repo: `examples/06-nebula-delivery-graph-memory/repo`
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
