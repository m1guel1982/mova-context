const API_KEY = process.env.API_KEY || 'dev-key-local';

function authMiddleware(req, res, next) {
  const key = req.headers['x-api-key'];
  if (!key || key !== API_KEY) {
    return res.status(401).json({ error: 'API key inválida o ausente' });
  }
  next();
}

module.exports = { authMiddleware };
