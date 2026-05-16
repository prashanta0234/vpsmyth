'use strict';

let engineMeta = {};
let selectedEngine = 'mysql';
let deletingID = null;
let pollTimer = null;

const ENGINE_ICONS = { mysql: '/assets/MySQL.png', postgres: '/assets/Postgresql.png', mongodb: '/assets/mongodb.png', redis: '/assets/redis.png' };
const ENGINE_LABELS = { mysql: 'MySQL', postgres: 'PostgreSQL', mongodb: 'MongoDB', redis: 'Redis' };
const BADGE_CLASS = { mysql: 'badge-mysql', postgres: 'badge-postgres', mongodb: 'badge-mongodb', redis: 'badge-redis' };

// ── API helper ────────────────────────────────────────────────────────────────

async function api(path, opts = {}) {
    const res = await fetch(path, opts);
    if (res.status === 401) { handleAuthError(); throw new Error('unauthorized'); }
    return res;
}

// ── Load & render ─────────────────────────────────────────────────────────────

async function loadDatabases() {
    try {
        const res = await api('/api/databases');
        if (!res.ok) return;
        const dbs = await res.json();
        renderDatabases(dbs);

        // Keep polling while any DB is still deploying
        const anyDeploying = dbs.some(d => d.deploy_status === 'deploying');
        if (anyDeploying) {
            clearTimeout(pollTimer);
            pollTimer = setTimeout(loadDatabases, 3000);
        }
    } catch { /* ignore */ }
}

function renderDatabases(dbs) {
    const container = document.getElementById('db-grid-container');
    if (!dbs || dbs.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <div class="empty-icon">🗄️</div>
                <p>No databases yet.</p>
                <p>Click <strong>+ Deploy Database</strong> to get started.</p>
            </div>`;
        return;
    }

    const cards = dbs.map(d => buildCard(d)).join('');
    container.innerHTML = `<div class="db-grid">${cards}</div>`;

    dbs.forEach(d => wireCard(d));
}

function buildCard(d) {
    const isReady = d.deploy_status === 'ready';
    const isError = d.deploy_status === 'error';
    const isDeploying = d.deploy_status === 'deploying';

    let statusHtml = '';
    if (isDeploying) {
        statusHtml = `<span class="status-pill pill-deploying"><span class="pulse">●</span> Deploying...</span>`;
    } else if (isError) {
        statusHtml = `<span class="status-pill pill-error">✗ Error</span>`;
    } else {
        const ds = d.docker_status || 'unknown';
        const cls = ds === 'running' ? 'pill-running' : ds === 'exited' ? 'pill-stopped' : 'pill-notfound';
        const label = ds === 'running' ? '● Running' : ds === 'exited' ? '○ Stopped' : ds;
        statusHtml = `<span class="status-pill ${cls}">${label}</span>`;
    }

    const running = isReady && d.docker_status === 'running';
    const stopped = isReady && (d.docker_status === 'exited' || d.docker_status === 'stopped');

    const disAll = isDeploying ? 'disabled' : '';
    const disRun = (!stopped) ? 'disabled' : '';
    const disStop = (!running) ? 'disabled' : '';

    const errLine = isError ? `<div class="deploy-progress" style="background:#fef2f2;border-color:#fca5a5;color:#dc2626;font-size:0.78rem;margin-top:0.25rem;word-break:break-all;">${escHtml(d.deploy_error)}</div>` : '';

    return `<div class="db-card" id="card-${d.id}">
        <div class="db-card-header">
            <div class="db-engine-badge ${BADGE_CLASS[d.engine] || ''}"><img src="${ENGINE_ICONS[d.engine] || ''}" alt="${d.engine}" style="width:26px;height:26px;object-fit:contain;"></div>
            <div>
                <div class="db-name">${escHtml(d.name)}</div>
                <div class="db-meta">${ENGINE_LABELS[d.engine] || d.engine} ${d.version} · port ${d.port} · ${d.bind_ip}</div>
            </div>
        </div>
        <div class="db-status-row">
            ${statusHtml}
            <span style="font-size:0.75rem;color:var(--muted,#94a3b8);">${formatDate(d.created_at)}</span>
        </div>
        ${isDeploying ? '<div class="deploy-progress"><div class="spinner"></div>Pulling image and starting container…</div>' : ''}
        ${errLine}
        <div class="db-actions">
            <button class="btn-sm btn-start" data-id="${d.id}" ${disAll} ${disRun}>▶ Start</button>
            <button class="btn-sm btn-stop" data-id="${d.id}" ${disAll} ${disStop}>■ Stop</button>
            <button class="btn-sm btn-restart" data-id="${d.id}" ${disAll} ${(!running) ? 'disabled' : ''}>↺ Restart</button>
            <button class="btn-sm btn-conn" data-id="${d.id}" data-name="${escHtml(d.name)}" ${isDeploying ? 'disabled' : ''}>⊕ Connect</button>
            <button class="btn-sm btn-del" data-id="${d.id}" data-name="${escHtml(d.name)}" ${isDeploying ? 'disabled' : ''}>Delete</button>
        </div>
    </div>`;
}

function wireCard(d) {
    const card = document.getElementById(`card-${d.id}`);
    if (!card) return;

    const act = (action) => dbAction(d.id, action);
    card.querySelector('.btn-start')?.addEventListener('click', () => act('start'));
    card.querySelector('.btn-stop')?.addEventListener('click', () => act('stop'));
    card.querySelector('.btn-restart')?.addEventListener('click', () => act('restart'));
    card.querySelector('.btn-conn')?.addEventListener('click', () => showConnInfo(d.id, d.name));
    card.querySelector('.btn-del')?.addEventListener('click', () => openDeleteModal(d.id, d.name));
}

// ── Actions ───────────────────────────────────────────────────────────────────

async function dbAction(id, action) {
    try {
        const res = await api('/api/databases/action', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id, action }),
        });
        if (!res.ok) { alert('Failed: ' + await res.text()); return; }
        await loadDatabases();
    } catch { /* ignore */ }
}

// ── Connection info ───────────────────────────────────────────────────────────

let pwVisible = false;

async function showConnInfo(id, name) {
    pwVisible = false;
    document.getElementById('conn-modal-title').textContent = `Connect — ${name}`;
    document.getElementById('conn-modal-body').innerHTML = '<p style="color:var(--muted,#94a3b8);font-size:0.875rem;">Loading...</p>';
    document.getElementById('conn-modal').classList.add('open');

    try {
        const res = await api(`/api/databases/conninfo?id=${id}`);
        if (!res.ok) { document.getElementById('conn-modal-body').innerHTML = '<p style="color:#ef4444;">Failed to load.</p>'; return; }
        const info = await res.json();
        renderConnInfo(info);
    } catch { /* ignore */ }
}

function renderConnInfo(info) {
    const maskedConn = info.connection_string.replace(info.password, '●●●●●●●●');
    document.getElementById('conn-modal-body').innerHTML = `
        <p class="conn-label">Connection String</p>
        <div class="conn-block" id="conn-str-block">
            <span id="conn-str-text">${escHtml(maskedConn)}</span>
            <button class="copy-btn" onclick="copyText('conn-str-text')">Copy</button>
        </div>
        <div class="conn-label" style="margin-top:0.9rem;">Details</div>
        <div>
            <div class="conn-row"><span class="conn-key">Host</span><span class="conn-val">${escHtml(info.host)}</span></div>
            <div class="conn-row"><span class="conn-key">Port</span><span class="conn-val">${info.port}</span></div>
            <div class="conn-row">
                <span class="conn-key">Password</span>
                <span class="conn-val">
                    <span id="pw-display" class="pw-mask" title="Click to reveal">●●●●●●●●</span>
                    <button onclick="togglePw('${escHtml(info.password)}')" style="margin-left:0.5rem;font-size:0.72rem;color:#4f46e5;background:none;border:none;cursor:pointer;">Show</button>
                </span>
            </div>
        </div>
        <p style="font-size:0.78rem;color:var(--muted,#94a3b8);margin-top:1rem;">
            💡 For remote access from your dev machine, use SSH tunneling:<br>
            <code style="font-size:0.75rem;">ssh -L ${info.port}:127.0.0.1:${info.port} user@your-server</code>
        </p>`;
    // Store for real copy
    document.getElementById('conn-str-text').dataset.real = info.connection_string;
}

function togglePw(password) {
    pwVisible = !pwVisible;
    document.getElementById('pw-display').textContent = pwVisible ? password : '●●●●●●●●';
    const btn = document.getElementById('pw-display').nextElementSibling;
    btn.textContent = pwVisible ? 'Hide' : 'Show';
    // Also update connection string display
    const connText = document.getElementById('conn-str-text');
    if (connText) {
        connText.textContent = pwVisible ? connText.dataset.real : connText.dataset.real.replace(password, '●●●●●●●●');
    }
}

function copyText(elId) {
    const el = document.getElementById(elId);
    const text = el.dataset.real || el.textContent;
    navigator.clipboard.writeText(text).then(() => {
        const copyBtns = document.querySelectorAll('.copy-btn');
        copyBtns.forEach(b => { b.textContent = 'Copied!'; setTimeout(() => b.textContent = 'Copy', 1500); });
    });
}

// ── Delete modal ──────────────────────────────────────────────────────────────

function openDeleteModal(id, name) {
    deletingID = id;
    document.getElementById('delete-db-name').textContent = name;
    document.getElementById('delete-volume-cb').checked = false;
    document.getElementById('delete-password').value = '';
    document.getElementById('delete-pw-error').style.display = 'none';
    document.getElementById('delete-modal').classList.add('open');
    setTimeout(() => document.getElementById('delete-password').focus(), 100);
}

async function confirmDelete() {
    if (!deletingID) return;
    const deleteVolume = document.getElementById('delete-volume-cb').checked;
    const password = document.getElementById('delete-password').value;
    const errEl = document.getElementById('delete-pw-error');
    errEl.style.display = 'none';

    if (!password) {
        errEl.textContent = 'Password is required.';
        errEl.style.display = 'block';
        return;
    }

    document.getElementById('delete-confirm').disabled = true;
    try {
        const res = await api('/api/databases/action', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', 'X-Admin-Password': password },
            body: JSON.stringify({ id: deletingID, action: 'delete', delete_volume: deleteVolume }),
        });
        if (res.status === 401) {
            errEl.textContent = 'Incorrect password.';
            errEl.style.display = 'block';
            return;
        }
        if (!res.ok) { alert('Failed: ' + await res.text()); return; }
        document.getElementById('delete-modal').classList.remove('open');
        deletingID = null;
        await loadDatabases();
    } catch { /* ignore */ }
    finally { document.getElementById('delete-confirm').disabled = false; }
}

// ── Deploy modal ──────────────────────────────────────────────────────────────

function openDeployModal() {
    document.getElementById('deploy-error').style.display = 'none';
    document.getElementById('f-db-name').value = '';
    document.getElementById('f-db-password').value = generatePassword();
    document.getElementById('f-db-bind').value = '127.0.0.1';
    document.getElementById('bind-warning').classList.remove('show');
    selectEngine('mysql');
    document.getElementById('deploy-modal').classList.add('open');
    document.getElementById('f-db-name').focus();
}

function selectEngine(engine) {
    selectedEngine = engine;
    document.querySelectorAll('.engine-card').forEach(c => c.classList.toggle('selected', c.dataset.engine === engine));

    const meta = engineMeta[engine] || {};
    const versions = meta.versions || [engine === 'mysql' ? '8' : engine === 'postgres' ? '16' : engine === 'mongodb' ? '7' : '7-alpine'];
    const sel = document.getElementById('f-db-version');
    sel.innerHTML = versions.map(v => `<option value="${v}">${v}</option>`).join('');
    document.getElementById('f-db-port').value = meta.default_port || '';
}

async function submitDeploy() {
    const errEl = document.getElementById('deploy-error');
    errEl.style.display = 'none';

    const name = document.getElementById('f-db-name').value.trim();
    const version = document.getElementById('f-db-version').value;
    const password = document.getElementById('f-db-password').value.trim();
    const port = parseInt(document.getElementById('f-db-port').value);
    const bindIP = document.getElementById('f-db-bind').value;

    if (!name) { errEl.textContent = 'Name is required.'; errEl.style.display = 'block'; return; }
    if (!password) { errEl.textContent = 'Password is required.'; errEl.style.display = 'block'; return; }
    if (!port || port < 1 || port > 65535) { errEl.textContent = 'Invalid port.'; errEl.style.display = 'block'; return; }

    const btn = document.getElementById('deploy-submit');
    btn.disabled = true; btn.textContent = 'Deploying...';

    try {
        const res = await api('/api/databases/deploy', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, engine: selectedEngine, version, password, port, bind_ip: bindIP }),
        });
        if (!res.ok) {
            errEl.textContent = await res.text();
            errEl.style.display = 'block';
            return;
        }
        document.getElementById('deploy-modal').classList.remove('open');
        await loadDatabases();
    } catch (e) {
        errEl.textContent = 'Network error.';
        errEl.style.display = 'block';
    } finally {
        btn.disabled = false; btn.textContent = 'Deploy';
    }
}

// ── Utilities ─────────────────────────────────────────────────────────────────

function generatePassword() {
    const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789!@#$';
    return Array.from({ length: 20 }, () => chars[Math.floor(Math.random() * chars.length)]).join('');
}

function escHtml(s) {
    return String(s || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

function formatDate(str) {
    if (!str) return '';
    return new Date(str).toLocaleDateString();
}

// ── Event wiring ──────────────────────────────────────────────────────────────

document.getElementById('btn-deploy').addEventListener('click', openDeployModal);
document.getElementById('deploy-close').addEventListener('click', () => document.getElementById('deploy-modal').classList.remove('open'));
document.getElementById('deploy-cancel').addEventListener('click', () => document.getElementById('deploy-modal').classList.remove('open'));
document.getElementById('deploy-submit').addEventListener('click', submitDeploy);

document.getElementById('conn-close').addEventListener('click', () => document.getElementById('conn-modal').classList.remove('open'));
document.getElementById('conn-dismiss').addEventListener('click', () => document.getElementById('conn-modal').classList.remove('open'));

document.getElementById('delete-close').addEventListener('click', () => document.getElementById('delete-modal').classList.remove('open'));
document.getElementById('delete-cancel').addEventListener('click', () => document.getElementById('delete-modal').classList.remove('open'));
document.getElementById('delete-confirm').addEventListener('click', confirmDelete);

document.querySelectorAll('.engine-card').forEach(card => {
    card.addEventListener('click', () => selectEngine(card.dataset.engine));
});

document.getElementById('f-db-bind').addEventListener('change', e => {
    document.getElementById('bind-warning').classList.toggle('show', e.target.value === '0.0.0.0');
});

// Close modals on overlay click
['deploy-modal','conn-modal','delete-modal'].forEach(id => {
    document.getElementById(id).addEventListener('click', e => {
        if (e.target.id === id) document.getElementById(id).classList.remove('open');
    });
});

// ── Init ──────────────────────────────────────────────────────────────────────

async function init() {
    try {
        const res = await api('/api/databases/engines');
        if (res.ok) engineMeta = await res.json();
    } catch { /* ignore */ }
    selectEngine('mysql');
    await loadDatabases();
}

init();
