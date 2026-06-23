const { test } = require('node:test');
const assert = require('node:assert/strict');
const service = require('./src/services/taskService');

test('crear tarea válida', () => {
  const t = service.crearTarea('Comprar leche');
  assert.equal(t.titulo, 'Comprar leche');
  assert.equal(t.completada, 0);
  assert.ok(t.id > 0);
});

test('crear sin título lanza 400', () => {
  assert.throws(() => service.crearTarea(''), err => err.status === 400);
});

test('crear con null lanza 400', () => {
  assert.throws(() => service.crearTarea(null), err => err.status === 400);
});

test('listar devuelve array', () => {
  assert.ok(Array.isArray(service.listarTareas()));
});

test('completar tarea existente', () => {
  const t = service.crearTarea('Para completar');
  assert.equal(service.completarTarea(t.id).completada, 1);
});

test('completar inexistente lanza 404', () => {
  assert.throws(() => service.completarTarea(99999), err => err.status === 404);
});

test('completar ya completada lanza 409', () => {
  const t = service.crearTarea('Doble completar');
  service.completarTarea(t.id);
  assert.throws(() => service.completarTarea(t.id), err => err.status === 409);
});
