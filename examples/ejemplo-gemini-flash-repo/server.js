// server.js — API mínima de ejemplo (genérica a propósito) para el
// proyecto "ejemplo-gemini-flash". Representa cualquier backend
// chico: Node/Express, Python/Flask, Go — el bug de abajo es el mismo
// tipo de problema sea cual sea el stack real.
const express = require("express");
const app = express();
app.use(express.json());

//  BUG 1: secreto hardcodeado en el código fuente en vez de venir de
// una variable de entorno / secret manager.
const API_SECRET = "sk_live_51Hc9f2KZ9pQ7aB3d";

//  BUG 2: sin validación de entrada — "amount" e "email" se usan tal
// cual llegan del request, sin chequear tipo, rango, ni formato.
app.post("/api/charge", (req, res) => {
  const { amount, email } = req.body;
  charge(amount, email, API_SECRET);
  res.json({ ok: true });
});

//  BUG 3: sin autenticación/autorización — cualquiera puede pegarle a
// este endpoint y ver el listado completo de clientes.
app.get("/api/customers", (req, res) => {
  res.json(db.customers.findAll());
});

function charge(amount, email, secret) {
  // (stub de ejemplo)
}

app.listen(3000);
