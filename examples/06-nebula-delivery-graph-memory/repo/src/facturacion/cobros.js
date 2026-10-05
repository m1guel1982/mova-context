'use strict';
const { notificarCliente } = require('./notificaciones');

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
}

module.exports = { generarCobro, construirFilaReporte, exportarReporteCobros, registrarCobro, conciliarCobros };
