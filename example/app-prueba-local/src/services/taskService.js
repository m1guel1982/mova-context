const repo = require('../repositories/taskRepository');

function crearTarea(titulo) {
  if (!titulo || typeof titulo !== 'string' || titulo.trim() === '') {
    throw Object.assign(new Error('titulo es requerido y no puede estar vacío'), { status: 400 });
  }
  return repo.crear(titulo.trim());
}

function listarTareas()    { return repo.listar(); }

function completarTarea(id) {
  const tarea = repo.buscarPorId(id);
  if (!tarea) throw Object.assign(new Error(`Tarea ${id} no encontrada`), { status: 404 });
  if (tarea.completada) throw Object.assign(new Error(`Tarea ${id} ya completada`), { status: 409 });
  return repo.completar(id);
}

module.exports = { crearTarea, listarTareas, completarTarea };
