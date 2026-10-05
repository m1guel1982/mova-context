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

<!-- egress:block key="section:SKILLS" sha="13df5567134e26c3" updated="2026-10-02T19:02:01Z" -->
### section:SKILLS
```text
## SKILLS
```
<!-- egress:end -->

<!-- egress:block key="core:kiss-dry-core" sha="024e1e7f15694a22" updated="2026-10-02T19:02:01Z" -->
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

<!-- egress:block key="skill:minimal-tests" sha="71939cfff64b532e" updated="2026-10-02T19:02:01Z" -->
### skill:minimal-tests
```text
<!-- skill: minimal-tests -->
# Skill: Tests mínimos
Un test por comportamiento observable y por caso límite relevante (vacío, nulo, límites, error), no por línea. Sin mocks salvo fronteras externas. El nombre describe el comportamiento esperado. Usa el framework que el proyecto ya usa (o el estándar del lenguaje). La lógica trivial no requiere test.
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

<!-- egress:block key="prompt:fix-or-improve" sha="bc318294f5f1ae6e" updated="2026-10-02T19:02:01Z" -->
### prompt:fix-or-improve
```text
<!-- prompt: fix-or-improve -->
# Corregir o mejorar — `facturador`
Stack: Node.js / JavaScript

Ockham: ver `ockham-core.md`.

Consulta: Corregir el reporte de cobros para que muestre la llegada real cuando exista (respaldo: la programada). Parte de los hallazgos del planificador que están en MEMORY y verifícalos contra el código.

Sobre el FOCUS entregado (no supongas código que no ves):
1. Identifica la causa raíz; si el bug se ve en una vista, corrígelo en el punto común de los invocadores.
2. Aplica la escalera de `kiss-dry-core` (nativo → estándar → ya instalado → código propio).
3. No introduzcas: No crear clases ni capas nuevas: cambios mínimos en funciones existentes.
4. Escala a una solución más compleja solo si: Si se agregan más columnas de tiempo al reporte, extraer un mapeador de columnas.

Salida: por cada cambio, `archivo::símbolo` y el bloque con la función completa ya corregida; cierre según `ockham-core`.
```
<!-- egress:end -->

<!-- egress:block key="header" sha="a0cc387656067121" updated="2026-10-02T19:02:01Z" -->
### header
```text
# Mova Context — facturador / mejorar-cobros
```
<!-- egress:end -->

<!-- egress:block key="section:FOCUS" sha="32e9bf30471d5472" updated="2026-10-02T19:02:01Z" -->
### section:FOCUS
```text
## FOCUS
```
<!-- egress:end -->

<!-- egress:block key="focus:src/facturacion/cobros.js::func=generarCobro,construirFilaReporte,exportarReporteCobros,registrarCobro,conciliarCobros" sha="70fe7536e243ce79" updated="2026-10-02T19:02:01Z" -->
### focus:src/facturacion/cobros.js::func=generarCobro,construirFilaReporte,exportarReporteCobros,registrarCobro,conciliarCobros
```text
FOCUS:src/facturacion/cobros.js::func=generarCobro,construirFilaReporte,exportarReporteCobros,registrarCobro,conciliarCobros
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
}

function conciliarCobros(cobros, libro) {
  const ids = new Set(libro.map((c) => c.pedidoId));
  return cobros.filter((c) => !ids.has(c.pedidoId));
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

<!-- egress:block key="section:MEMORY" sha="b8d964eff9a372a0" updated="2026-10-02T19:02:01Z" -->
### section:MEMORY
```text
## MEMORY
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
```
<!-- egress:end -->

<!-- egress:block key="section:INSTRUCTION" sha="f16be84e4aed434c" updated="2026-10-02T19:02:01Z" -->
### section:INSTRUCTION
````text
## INSTRUCTION
Project: **facturador** | Repo: `examples/06-nebula-delivery-graph-memory/repo`
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
