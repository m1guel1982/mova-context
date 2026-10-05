'use strict';
const { tarifaPlanaV1 } = require('../legacy/tarifasV1');

const CREDITOS_POR_KG = 4.5;

function redondearCreditos(valor) {
  return Math.round(valor * 100) / 100;
}

function recargoRadiacion(ruta) {
  return ruta.destino === 'titan-puerto' ? 1.35 : 1.0;
}

function recargoRadiacionLegacy(ruta) {
  return ruta.destino === 'titan-puerto' ? 1.2 : 1.05;
}

function aplicarDescuentoFlota(monto, cliente) {
  return cliente && cliente.flota ? monto * 0.9 : monto;
}

function calcularTarifa(pedido, ruta) {
  if (ruta.catalogo === 'v1') return tarifaPlanaV1(pedido);
  const bruto = pedido.pesoKg * CREDITOS_POR_KG * recargoRadiacion(ruta);
  return redondearCreditos(aplicarDescuentoFlota(bruto, pedido.cliente));
}

module.exports = { calcularTarifa, aplicarDescuentoFlota, recargoRadiacion, redondearCreditos };
