const express = require('express');
const taskRouter = require('./src/controllers/taskController');
const { authMiddleware } = require('./src/middlewares/auth');

const app = express();
const PORT = process.env.PORT || 3000;

app.use(express.json());
app.use(authMiddleware);
app.use('/api/v1/tareas', taskRouter);

app.listen(PORT, () => {
  console.log(`tareas-api corriendo en http://localhost:${PORT}`);
});

module.exports = app; // lazy: exportado para tests sin levantar puerto
