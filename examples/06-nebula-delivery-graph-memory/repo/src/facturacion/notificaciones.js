'use strict';

function formatearAviso(cobro) {
  return `Pedido ${cobro.pedidoId} a ${cobro.destino}: ${cobro.monto} créditos`;
}

function notificarCliente(cobro) {
  const aviso = formatearAviso(cobro);
  return { para: cobro.cliente && cobro.cliente.email, aviso };
}

module.exports = { notificarCliente, formatearAviso };
