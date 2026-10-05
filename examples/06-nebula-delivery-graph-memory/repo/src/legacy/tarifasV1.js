'use strict';
// Tarifa plana heredada (catálogo v1). Pendiente de retiro.

function tarifaPlanaV1(pedido) {
  return pedido.pesoKg * 5.0;
}

function migrarTarifaV1(registroViejo) {
  return { ...registroViejo, catalogo: 'v2', monto: registroViejo.monto * 0.95 };
}

module.exports = { tarifaPlanaV1, migrarTarifaV1 };
