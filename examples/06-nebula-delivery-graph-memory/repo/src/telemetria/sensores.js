'use strict';

function normalizarLectura(valor, unidad) {
  return unidad === 'kpa' ? valor : valor * 100;
}

function leerSensor(id) {
  const crudo = { 'tanque-principal': 3.2 }[id] ?? 0;
  return normalizarLectura(crudo, 'bar');
}

module.exports = { leerSensor, normalizarLectura };
