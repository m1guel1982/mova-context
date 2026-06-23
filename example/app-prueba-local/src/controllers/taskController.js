const { Router } = require('express');
const service = require('../services/taskService');
const router = Router();

router.post('/', (req, res) => {
  try   { res.status(201).json(service.crearTarea(req.body.titulo)); }
  catch (err) { res.status(err.status || 500).json({ error: err.message }); }
});

router.get('/', (_req, res) => { res.json(service.listarTareas()); });

router.patch('/:id/completar', (req, res) => {
  try   { res.json(service.completarTarea(Number(req.params.id))); }
  catch (err) { res.status(err.status || 500).json({ error: err.message }); }
});

module.exports = router;
