// api.js — endpoint mínimo de la API de pedidos, usado como contexto
// de ejemplo para la puerta MCP (ver README.md de este ejemplo).
function crearPedido(clienteId, monto) {
  if (monto <= 0) {
    throw new Error("monto inválido");
  }
  return { id: "ord_" + Date.now(), clienteId, monto, estado: "creado" };
}

module.exports = { crearPedido };
