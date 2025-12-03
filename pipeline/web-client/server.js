const express = require('express');
const cors = require('cors');
const bodyParser = require('body-parser');
const path = require('path');

const axios = require('axios');

const app = express();
app.use(cors());
app.use(bodyParser.json());

const PORT = process.env.PORT || 3000;

// Set your Go backend base URL here
const GO_BACKEND_BASE = process.env.GO_BACKEND_BASE || 'http://localhost:8080';

app.get('/api/health', async (req, res) => {
  try {
    const r = await axios.get(`${GO_BACKEND_BASE}/health`);
    res.json(r.data);
  } catch (e) {
    res.status(500).json({ error: e.message });
  }
});

// List app packages
// List app packages
app.get('/api/app-packages', async (req, res) => {
  try {
    const r = await axios.get(`${GO_BACKEND_BASE}/app-packages`, {
      params: req.query,
      headers: req.headers
    });
    res.status(r.status).json(r.data);
  } catch (e) {
    res.status((e.response && e.response.status) || 500).json((e.response && e.response.data) || { error: e.message });
  }
});

// List devices
app.get('/api/devices', async (req, res) => {
  try {
    const r = await axios.get(`${GO_BACKEND_BASE}/devices`, {
      params: req.query,
      headers: req.headers
    });
    res.status(r.status).json(r.data);
  } catch (e) {
  res.status((e.response && e.response.status) || 500).json((e.response && e.response.data) || { error: e.message });
  }
});

// List deployments
app.get('/api/deployments', async (req, res) => {
  try {
    const r = await axios.get(`${GO_BACKEND_BASE}/app-deployments`, {
      params: req.query,
      headers: req.headers
    });
    res.status(r.status).json(r.data);
  } catch (e) {
    res.status(e.response?.status || 500).json(e.response?.data || { error: e.message });
  }
});

// Delete app package
app.delete('/api/app-packages/:id', async (req, res) => {
  try {
    const r = await axios.delete(`${GO_BACKEND_BASE}/app-packages/${req.params.id}`, {
      params: req.query,
      headers: req.headers
    });
    res.status(r.status).json(r.data);
  } catch (e) {
    res.status(e.response?.status || 500).json(e.response?.data || { error: e.message });
  }
});

// Delete deployment
app.delete('/api/deployments/:id', async (req, res) => {
  try {
    const r = await axios.delete(`${GO_BACKEND_BASE}/app-deployments/${req.params.id}`, {
      params: req.query,
      headers: req.headers
    });
    res.status(r.status).json(r.data);
  } catch (e) {
    res.status(e.response?.status || 500).json(e.response?.data || { error: e.message });
  }
});

// Onboard app package
app.post('/api/app-packages', async (req, res) => {
  try {
    const r = await axios.post(`${GO_BACKEND_BASE}/app-packages`, req.body, {
      headers: { ...req.headers, 'Content-Type': 'application/json' }
    });
    res.status(r.status).json(r.data);
  } catch (e) {
    res.status(e.response?.status || 500).json(e.response?.data || { error: e.message });
  }
});

// Create deployment
app.post('/api/deployments', async (req, res) => {
  try {
    const r = await axios.post(`${GO_BACKEND_BASE}/app-deployments`, req.body, {
      headers: { ...req.headers, 'Content-Type': 'application/json' }
    });
    res.status(r.status).json(r.data);
  } catch (e) {
    res.status(e.response?.status || 500).json(e.response?.data || { error: e.message });
  }
});

// Serve static frontend
app.use('/', express.static(path.join(__dirname, 'public')));

app.listen(PORT, () => {
  console.log(`WFM Web Client listening on http://localhost:${PORT}`);
});
