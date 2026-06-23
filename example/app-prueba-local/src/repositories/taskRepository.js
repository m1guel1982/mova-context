// lazy: node:sqlite experimental (Node 22+) — sin dependencias externas de DB
// ceiling: API experimental, puede cambiar en versiones futuras de Node
// upgrade path: reemplazar por better-sqlite3 si se necesita Node <22 o estabilidad de API
const { DatabaseSync } = require('node:sqlite');

const db = new DatabaseSync(process.env.DB_PATH || ':memory:');

db.exec(`
  CREATE TABLE IF NOT EXISTS tareas (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    titulo     TEXT    NOT NULL,
    completada INTEGER NOT NULL DEFAULT 0,
    creada_en  TEXT    NOT NULL DEFAULT (datetime('now'))
  )
`);

const stmts = {
  insertar:    db.prepare('INSERT INTO tareas (titulo) VALUES (?)'),
  listar:      db.prepare('SELECT * FROM tareas ORDER BY creada_en DESC'),
  buscar:      db.prepare('SELECT * FROM tareas WHERE id = ?'),
  completar:   db.prepare('UPDATE tareas SET completada = 1 WHERE id = ?'),
};

function crear(titulo) {
  const { lastInsertRowid } = stmts.insertar.run(titulo);
  return stmts.buscar.get(lastInsertRowid);
}

function listar()       { return stmts.listar.all(); }
function buscarPorId(id){ return stmts.buscar.get(id) ?? null; }
function completar(id)  { stmts.completar.run(id); return stmts.buscar.get(id); }

module.exports = { crear, listar, buscarPorId, completar };
