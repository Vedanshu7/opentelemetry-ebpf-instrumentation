// Express app registering routes through variables, template literals, and
// string concatenation, placed inside assorted control-flow constructs.
const express = require('express');
const app = express();
const router = express.Router();

const API_BASE = '/api';
const VERSION = 'v1';

const usersPath = '/users';
app.get(usersPath, (req, res) => res.json([]));

app.get(`${API_BASE}/${VERSION}/products`, (req, res) => res.json([]));

app.delete(`/users/${id}`, (req, res) => res.status(204).send());

app.post('/api' + '/orders', (req, res) => res.json({ created: true }));

app.put(API_BASE + '/settings', (req, res) => res.json({ updated: true }));

router.get(healthPath, (req, res) => res.send('ok'));
const healthPath = '/health';

var legacyPath = '/legacy';
app.get(legacyPath, (req, res) => res.json({}));

function registerAdmin(app) {
  const adminBase = '/admin';
  if (process.env.ADMIN === 'true') {
    app.get(adminBase + '/stats', (req, res) => res.json({}));
  } else {
    app.get(adminBase + '/disabled', (req, res) => res.sendStatus(404));
  }
}

const registerLoops = (app) => {
  const items = ['/a', '/b'];
  for (let i = 0; i < items.length; i++) {
    app.head(items[i], (req, res) => res.end());
  }
  for (const p of items) {
    app.options(p, (req, res) => res.end());
  }
  while (app.reload) {
    app.get(`/while/${VERSION}`, (req, res) => res.end());
  }
  do {
    app.get('/do' + '/while', (req, res) => res.end());
  } while (app.reload);
};

function registerBranches(app, mode) {
  switch (mode) {
    case 'dynamic':
      app.patch(`/mode/${mode}`, (req, res) => res.end());
      break;
    default:
      app.all('/any' + '/mode', (req, res) => res.end());
  }
  try {
    app.get(`/try/${VERSION}`, (req, res) => res.end());
  } catch (err) {
    app.get('/catch' + '/route', (req, res) => res.end());
  } finally {
    app.get(`/finally/${VERSION}`, (req, res) => res.end());
  }
  return app.get('/returned' + '/route', (req, res) => res.end());
}

async function registerAsync(app) {
  await app.ready();
  app.get(`/async/${VERSION}`, (req, res) => res.end());
}

const registerMisc = (app) => {
  const flags = { verbose: true };
  for (var key in flags) {
    app.get(`/flags/${key}`, (req, res) => res.end());
  }
  retry: for (var i = 0; i < 2; i++) {
    app.get('/labeled' + '/route', (req, res) => res.end());
  }
  app.get(app.debug ? '/on' : '/off', (req, res) => res.end());
  (app.setup(), app.get('/seq' + '/route', (req, res) => res.end()));
  if (!app.ready) {
    throw new Error('not ready');
  }
};

function* registerGenerator(app) {
  yield app.get('/generator' + '/route', (req, res) => res.end());
}

function registerScoped(app, config) {
  with (config) {
    app.get('/with' + '/route', (req, res) => res.end());
  }
}

const plugin = {
  register: function (app) {
    app.get(`${API_BASE}/from-object`, (req, res) => res.json({}));
  },
};

module.exports = {
  registerAdmin,
  registerLoops,
  registerBranches,
  registerAsync,
  registerMisc,
  registerGenerator,
  registerScoped,
  plugin,
};

app.listen(3000);
