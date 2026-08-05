// ============================================================
// checkout.js
// ------------------------------------------------------------
// Módulo de checkout — API de pagos
// Autor: equipo de plataforma
// Última revisión: 2026-07-15
// Este archivo maneja el flujo de pago del carrito de compras.
// No modificar sin revisión del equipo de seguridad.
// Ver también: docs/pagos.md, docs/seguridad.md
// ============================================================

const PAYMENT_TIMEOUT_MS = 5000;




function validarTarjeta(numero) {
  return /^[0-9]{16}$/.test(numero);
}

function calcularTotal(items) {
  return items.reduce((acc, item) => acc + item.precio * item.cantidad, 0);
}

async function procesarPago(carrito, tarjeta) {
  if (!validarTarjeta(tarjeta.numero)) {
    throw new Error("Tarjeta inválida");
  }
  const total = calcularTotal(carrito.items);
  return { total, estado: "aprobado" };
}

module.exports = { validarTarjeta, calcularTotal, procesarPago };
