'use strict';
// Nebula Delivery — planificador de cápsulas de reparto hacia la estación Selene.
const { calcularTrayectoria, estimarCombustible } = require('./rutas');
const { calcularTarifa } = require('../facturacion/tarifas');

const MARGEN_VENTANA_MIN = 15;

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
}

function depurarTrazaPlanificador(plan) {
  console.log('[debug planificador]', JSON.stringify(plan, null, 2));
}

module.exports = { planificarEntrega, normalizarPedido, elegirRuta, depurarTrazaPlanificador };
