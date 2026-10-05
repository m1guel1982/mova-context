'use strict';
const { leerSensor } = require('../telemetria/sensores');

const G_LUNAR = 1.62;

function ajustarPorGravedad(delta, cuerpo) {
  return cuerpo === 'luna' ? delta * (G_LUNAR / 9.81) : delta;
}

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

module.exports = { calcularTrayectoria, estimarCombustible, cargarCatalogoRutas };
