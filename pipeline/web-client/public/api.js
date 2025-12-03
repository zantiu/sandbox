// api.js - JS API client for backend connectivity

let GO_BACKEND_BASE = null; // If set, use Go backend directly; otherwise use proxy

export function setBaseURL(baseURL) {
  // If user provides a custom backend URL, store it for direct API calls
  GO_BACKEND_BASE = baseURL.replace(/\/$/, ''); // Remove trailing slash
  console.log('Backend URL set to:', GO_BACKEND_BASE);
}

function getAPIBase() {
  // If a custom backend URL is configured, use it directly
  // Otherwise use the Node.js proxy (avoids CORS issues)
  return GO_BACKEND_BASE ? GO_BACKEND_BASE : '';
}

// --- App Packages ---
export async function listAppPackages() {
  const base = getAPIBase();
  const endpoint = GO_BACKEND_BASE ? '/app-packages' : '/api/app-packages';
  const res = await fetch(`${base}${endpoint}`);
  if (!res.ok) throw new Error('Failed to fetch app packages');
  return res.json();
}

export async function onboardAppPackage(payload) {
  const base = getAPIBase();
  const endpoint = GO_BACKEND_BASE ? '/app-packages' : '/api/app-packages';
  const res = await fetch(`${base}${endpoint}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  if (!res.ok) throw new Error('Failed to onboard app package');
  return res.json();
}

export async function deleteAppPackage(pkgId) {
  const base = getAPIBase();
  const endpoint = GO_BACKEND_BASE ? '/app-packages' : '/api/app-packages';
  const res = await fetch(`${base}${endpoint}/${encodeURIComponent(pkgId)}`, {
    method: 'DELETE'
  });
  if (!res.ok) throw new Error('Failed to delete app package');
  return res.json();
}

// --- Devices ---
export async function listDevices() {
  const base = getAPIBase();
  const endpoint = GO_BACKEND_BASE ? '/devices' : '/api/devices';
  const res = await fetch(`${base}${endpoint}`);
  if (!res.ok) throw new Error('Failed to fetch devices');
  return res.json();
}

// --- Deployments ---
export async function listDeployments() {
  const base = getAPIBase();
  const endpoint = GO_BACKEND_BASE ? '/app-deployments' : '/api/deployments';
  const res = await fetch(`${base}${endpoint}`);
  if (!res.ok) throw new Error('Failed to fetch deployments');
  return res.json();
}

export async function createDeployment(payload) {
  const base = getAPIBase();
  const endpoint = GO_BACKEND_BASE ? '/app-deployments' : '/api/deployments';
  const res = await fetch(`${base}${endpoint}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  if (!res.ok) throw new Error('Failed to create deployment');
  return res.json();
}

// --- Upload/Onboarding Helpers ---
export async function uploadAppPackageManifest(manifestYAML) {
  const base = getAPIBase();
  const endpoint = GO_BACKEND_BASE ? '/app-packages' : '/api/app-packages';
  const res = await fetch(`${base}${endpoint}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ manifest: manifestYAML })
  });
  if (!res.ok) throw new Error('Failed to upload app package');
  return res.json();
}
