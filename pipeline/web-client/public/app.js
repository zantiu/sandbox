
// SPA navigation and rendering for Margo TestBed UI
import * as api from './api.js';
const root = document.getElementById('app-root');

// --- CONFIG ---
// WFM Backend URL is ONLY used for API calls, NOT for loading the webpage
// The webpage is always served from the current domain/port (the Node.js proxy or web server)
let config = {
  wfmBaseURL: localStorage.getItem('wfmBaseURL') || ''
};

function setWfmBaseURL(url) {
  config.wfmBaseURL = url;
  localStorage.setItem('wfmBaseURL', url);
  api.setBaseURL(url);
}

// --- ROUTING ---
const SCREENS = {
  WELCOME: 'welcome',
  WORKLOAD_SUPPLIER: 'workload-supplier',
  WFM_SUPPLIER: 'wfm-supplier',
  DEVICE_SUPPLIER: 'device-supplier',
  END_USER: 'end-user',
  END_USER_MARKETPLACE: 'end-user-marketplace',
  END_USER_MARKETPLACE_DEPLOY: 'end-user-marketplace-deploy',
  // Add more as needed
};
let state = { screen: SCREENS.WELCOME, persona: null, context: {} };

function navigate(screen, context = {}) {
  state = { ...state, screen, context };
  render();
}

// --- COMPONENTS ---
function render() {
  root.innerHTML = ''; // Clear all previous content
  switch (state.screen) {
    case SCREENS.WELCOME:
      renderWelcome(); break;
    case SCREENS.WORKLOAD_SUPPLIER:
      renderWorkloadSupplier(); break;
    case SCREENS.WFM_SUPPLIER:
      renderWFMSupplier(); break;
    case SCREENS.DEVICE_SUPPLIER:
      renderDeviceSupplier(); break;
    case SCREENS.END_USER:
      renderEndUser(); break;
    case SCREENS.END_USER_MARKETPLACE:
      renderEndUserMarketplace(); break;
    case SCREENS.END_USER_MARKETPLACE_DEPLOY:
      renderEndUserMarketplaceDeploy(); break;
    default:
      root.innerHTML = '<div>Unknown screen</div>';
  }
}

function renderHeader(title, subtitle) {
  const div = document.createElement('div');
  div.className = 'header';
  const configHTML = `
    <div class="header-top">
      <div><h1>${title}</h1>${subtitle ? `<div class="subtitle">${subtitle}</div>` : ''}</div>
      <div class="header-config">
        <label for="wfm-url">WFM Backend API URL:</label>
        <input type="text" id="wfm-url" placeholder="e.g., http://10.139.9.90:8082" value="${config.wfmBaseURL}" />
        <button id="btn-set-wfm-url">Configure</button>
      </div>
    </div>
  `;
  div.innerHTML = configHTML;
  root.appendChild(div);
  
  // Setup event listener for WFM URL configuration
  div.querySelector('#btn-set-wfm-url').onclick = () => {
    const newURL = div.querySelector('#wfm-url').value.trim();
    if (!newURL) {
      alert('Please enter a valid WFM Backend API URL');
      return;
    }
    setWfmBaseURL(newURL);
    alert('WFM Backend API URL configured: ' + newURL);
  };
}


function renderWelcome() {
  renderHeader('Margo Standard TestBed');
  const card = document.createElement('div');
  card.className = 'card welcome-card';
  card.innerHTML = `
    <h2>Select your persona</h2>
    <button class="persona-btn" id="btn-workload">Workload Supplier</button>
    <button class="persona-btn" id="btn-wfm">WFM Supplier</button>
    <button class="persona-btn" id="btn-device">Device Supplier</button>
    <button class="persona-btn" id="btn-enduser">End User (Integrator, OT etc...)</button>
  `;
  root.appendChild(card);
  card.querySelector('#btn-workload').onclick = () => navigate(SCREENS.WORKLOAD_SUPPLIER);
  card.querySelector('#btn-wfm').onclick = () => navigate(SCREENS.WFM_SUPPLIER);
  card.querySelector('#btn-device').onclick = () => navigate(SCREENS.DEVICE_SUPPLIER);
  card.querySelector('#btn-enduser').onclick = () => navigate(SCREENS.END_USER);
}

async function renderWorkloadSupplier() {
  renderHeader('Margo Standard TestBed', '&gt; Workload Supplier Workflow');
  const card = document.createElement('div');
  card.className = 'card';
  card.innerHTML = `
    <button class="primary" id="btn-upload-app">Upload App Package</button>
    <button class="secondary" id="btn-refresh-wls">🔄 Refresh</button>
    <div style="margin-top:20px">
      <h3>List of App Packages</h3>
      <table class="data-table">
        <thead><tr><th>ID</th><th>Status</th><th>Name</th><th>Actions</th></tr></thead>
        <tbody id="wls-app-table"></tbody>
      </table>
    </div>
    <button class="back-btn">&larr; Back</button>
  `;
  root.appendChild(card);
  card.querySelector('.back-btn').onclick = () => navigate(SCREENS.WELCOME);
  card.querySelector('#btn-refresh-wls').onclick = () => navigate(SCREENS.WORKLOAD_SUPPLIER);
  // Fetch data from backend
  let data = [];
  try {
    const resp = await api.listAppPackages();
    if (resp && resp.Data && resp.Data[0] && resp.Data[0].items) {
      data = resp.Data[0].items.map(it => ({
        id: it.metadata.id,
        status: it.status || '-',
        name: it.metadata.name || '-'
      }));
    }
  } catch (e) {
    data = [];
  }
  const tbody = card.querySelector('#wls-app-table');
  data.forEach(row => {
    const tr = document.createElement('tr');
    tr.innerHTML = `<td>${row.id}</td><td>${row.status}</td><td>${row.name}</td><td><button class=\"danger\">Delete</button></td>`;
    tr.querySelector('button').onclick = async () => {
      if (confirm('Delete this app package?')) {
        await api.deleteAppPackage(row.id);
        renderWorkloadSupplier();
      }
    };
    tbody.appendChild(tr);
  });
  card.querySelector('#btn-upload-app').onclick = async () => {
    showUploadAppPackageModal();
  };
}

// Modal for uploading app package
async function showUploadAppPackageModal() {
  const modal = document.createElement('div');
  modal.className = 'modal-overlay';
  modal.innerHTML = `
    <div class="modal">
      <div class="modal-header">
        <h2>Upload App Package</h2>
        <button class="modal-close">&times;</button>
      </div>
      <div class="modal-body">
        <label>App Package Name:</label>
        <input type="text" id="upload-pkg-name" placeholder="e.g., my-app-package" />
        
        <label style="margin-top: 16px;">Package Manifest (YAML):</label>
        <textarea id="upload-pkg-manifest" placeholder="apiVersion: margo.org/v1-alpha1
kind: ApplicationPackage
metadata:
  name: my-app
  id: my-app-pkg-001
  version: 1.0.0
spec:
  sourceType: git
  source:
    url: https://github.com/user/repo.git
    branch: main" rows="12" style="width: 100%; font-family: monospace; padding: 8px; border-radius: 6px; border: 1px solid #ddd;"></textarea>
      </div>
      <div class="modal-footer">
        <button class="btn-modal-cancel">Cancel</button>
        <button class="btn-modal-upload">Upload Package</button>
      </div>
    </div>
  `;
  root.appendChild(modal);
  
  const closeBtn = modal.querySelector('.modal-close');
  const cancelBtn = modal.querySelector('.btn-modal-cancel');
  const uploadBtn = modal.querySelector('.btn-modal-upload');
  const nameInput = modal.querySelector('#upload-pkg-name');
  const manifestInput = modal.querySelector('#upload-pkg-manifest');
  
  const closeModal = () => modal.remove();
  closeBtn.onclick = closeModal;
  cancelBtn.onclick = closeModal;
  
  uploadBtn.onclick = async () => {
    const name = nameInput.value.trim();
    const manifest = manifestInput.value.trim();
    
    if (!name) {
      alert('Please enter an app package name');
      return;
    }
    if (!manifest) {
      alert('Please enter a package manifest');
      return;
    }
    
    try {
      uploadBtn.disabled = true;
      uploadBtn.textContent = 'Uploading...';
      await api.uploadAppPackageManifest(manifest);
      alert('App package uploaded successfully!');
      closeModal();
      navigate(SCREENS.WORKLOAD_SUPPLIER);
    } catch (e) {
      alert('Failed to upload: ' + e.message);
    } finally {
      uploadBtn.disabled = false;
      uploadBtn.textContent = 'Upload Package';
    }
  };
}

function renderWFMSupplier() {
  renderHeader('Margo Standard TestBed', '&gt; WFM Supplier Workflow');
  const card = document.createElement('div');
  card.className = 'card';
  card.innerHTML = `
    <button class="primary" id="btn-manage-app">Manage App Packages</button>
    <button class="primary" id="btn-manage-devices">Manage Devices</button>
    <button class="primary" id="btn-manage-deployments">Manage Deployments</button>
    <button class="back-btn">&larr; Back</button>
  `;
  root.appendChild(card);
  card.querySelector('.back-btn').onclick = () => navigate(SCREENS.WELCOME);
  // For missing flows, show alert
  card.querySelector('#btn-manage-app').onclick = () => alert('Manage App Packages not implemented');
  card.querySelector('#btn-manage-devices').onclick = () => alert('Manage Devices not implemented');
  card.querySelector('#btn-manage-deployments').onclick = () => alert('Manage Deployments not implemented');
}

async function renderDeviceSupplier() {
  renderHeader('Margo Standard TestBed', '&gt; Device Supplier Workflow');
  const card = document.createElement('div');
  card.className = 'card';
  card.innerHTML = `
    <h3>Device Management</h3>
    <button class="primary" id="btn-add-device">Add Device</button>
    <button class="secondary" id="btn-refresh-devices">🔄 Refresh</button>
    <div style="margin-top:20px">
      <table class="data-table">
        <thead><tr><th>ID</th><th>Name</th><th>Status</th><th>Actions</th></tr></thead>
        <tbody id="ds-device-table"></tbody>
      </table>
    </div>
    <button class="back-btn">&larr; Back</button>
  `;
  root.appendChild(card);
  card.querySelector('.back-btn').onclick = () => navigate(SCREENS.WELCOME);
  card.querySelector('#btn-refresh-devices').onclick = () => navigate(SCREENS.DEVICE_SUPPLIER);
  // Fetch data from backend
  let data = [];
  try {
    const resp = await api.listDevices();
    if (resp && resp.Data && resp.Data[0] && resp.Data[0].items) {
      data = resp.Data[0].items.map(it => ({
        id: it.metadata.id,
        name: it.metadata.name || '-',
        status: it.status || '-'
      }));
    }
  } catch (e) {
    data = [];
  }
  const tbody = card.querySelector('#ds-device-table');
  data.forEach(row => {
    const tr = document.createElement('tr');
    tr.innerHTML = `<td>${row.id}</td><td>${row.name}</td><td>${row.status}</td><td><button class=\"danger\">Delete</button></td>`;
    tr.querySelector('button').onclick = () => alert('Delete not implemented');
    tbody.appendChild(tr);
  });
  card.querySelector('#btn-add-device').onclick = () => alert('Add Device not implemented');
}

function renderEndUser() {
  renderHeader('Margo Standard TestBed', '&gt; End-User Supplier Workflow');
  const card = document.createElement('div');
  card.className = 'card';
  card.innerHTML = `
    <button class="primary" id="btn-browse-market">Browse Marketplace</button>
    <button class="primary" id="btn-browse-devices">Browse Devices</button>
    <button class="primary" id="btn-manage-deployments">Manage Deployments</button>
    <button class="back-btn">&larr; Back</button>
  `;
  root.appendChild(card);
  card.querySelector('.back-btn').onclick = () => navigate(SCREENS.WELCOME);
  card.querySelector('#btn-browse-market').onclick = () => navigate(SCREENS.END_USER_MARKETPLACE);
  card.querySelector('#btn-browse-devices').onclick = () => alert('Browse Devices not implemented');
  card.querySelector('#btn-manage-deployments').onclick = () => alert('Manage Deployments not implemented');
}

async function renderEndUserMarketplace() {
  renderHeader('Margo Standard TestBed', '&gt; End-User Supplier Workflow &gt; Browse Marketplace');
  const card = document.createElement('div');
  card.className = 'card';
  card.innerHTML = `
    <h3>App Packages</h3>
    <button class="secondary" id="btn-refresh-marketplace">🔄 Refresh</button>
    <table class="data-table">
      <thead><tr><th>ID</th><th>Name</th><th>Developer</th><th>Icon</th><th>Actions</th></tr></thead>
      <tbody id="eu-app-table"></tbody>
    </table>
    <button class="back-btn">&larr; Back</button>
  `;
  root.appendChild(card);
  card.querySelector('.back-btn').onclick = () => navigate(SCREENS.END_USER);
  card.querySelector('#btn-refresh-marketplace').onclick = () => navigate(SCREENS.END_USER_MARKETPLACE);
  // Fetch app packages from backend
  let data = [];
  try {
    const resp = await api.listAppPackages();
    if (resp && resp.Data && resp.Data[0] && resp.Data[0].items) {
      data = resp.Data[0].items.map(it => ({
        id: it.metadata.id,
        name: it.metadata.name || '-',
        developer: it.metadata.developer || '-',
        icon: '📦',
        raw: it
      }));
    }
  } catch (e) {
    data = [];
  }
  const tbody = card.querySelector('#eu-app-table');
  data.forEach(row => {
    const tr = document.createElement('tr');
    tr.innerHTML = `<td>${row.id}</td><td>${row.name}</td><td>${row.developer}</td><td>${row.icon}</td><td><button class=\"primary\">Deploy on device</button></td>`;
    tr.querySelector('button').onclick = () => navigate(SCREENS.END_USER_MARKETPLACE_DEPLOY, { app: row.raw });
    tbody.appendChild(tr);
  });
}

async function renderEndUserMarketplaceDeploy() {
  const app = state.context.app || { name: 'CacheDB' };
  renderHeader('Margo Standard TestBed', '&gt; End-User Supplier Workflow &gt; Browse Marketplace &gt; Deploy App on Device');
  const card = document.createElement('div');
  card.className = 'card';
  card.innerHTML = `
    <h3>Deploy App Package</h3>
    <div class="popup">
      <div><b>Name:</b> ${app.name || app.metadata?.name || '-'}</div>
      <div style="margin:8px 0"><label>Select Device: <select id="deploy-device"></select></label></div>
      <div style="margin:8px 0"><label>Choose Deployment Profile: <select id="deploy-profile"><option>HELM_V3</option></select></label></div>
      <div style="margin:8px 0"><label>Parameter Override Template (YAML):<br><textarea id="deploy-yaml" rows="6" style="width:100%"># all the yaml is rendered here and is editable</textarea></label></div>
      <button class="primary" id="btn-deploy">Deploy</button>
      <button class="back-btn">&larr; Back</button>
    </div>
  `;
  root.appendChild(card);
  card.querySelector('.back-btn').onclick = () => navigate(SCREENS.END_USER_MARKETPLACE);
  // Fetch devices for dropdown
  let devices = [];
  try {
    const resp = await api.listDevices();
    if (resp && resp.Data && resp.Data[0] && resp.Data[0].items) {
      devices = resp.Data[0].items.map(it => ({
        id: it.metadata.id,
        name: it.metadata.name || it.metadata.id
      }));
    }
  } catch (e) {
    devices = [];
  }
  const deviceSel = card.querySelector('#deploy-device');
  devices.forEach(d => {
    const opt = document.createElement('option');
    opt.value = d.id;
    opt.textContent = d.name;
    deviceSel.appendChild(opt);
  });
  card.querySelector('#btn-deploy').onclick = async () => {
    const deviceId = deviceSel.value;
    const yaml = card.querySelector('#deploy-yaml').value;
    await api.createDeployment({ appId: app.id || app.metadata?.id, deviceId, yaml });
    alert('Deployment requested!');
    navigate(SCREENS.END_USER_MARKETPLACE);
  };
}

// Initial render
render();
