'use strict';

// Stored in memory only — cleared when the page is left or reloaded.
let sudoPassword = '';
let currentRules = [];
let myIPForPreset = '';

// ─── Password gate ────────────────────────────────────────────────────────────

function showGate(errorMsg) {
    document.getElementById('pw-gate').style.display = 'block';
    document.getElementById('fw-main').style.display = 'none';
    const errEl = document.getElementById('pw-error');
    if (errorMsg) {
        errEl.textContent = errorMsg;
        errEl.style.display = 'block';
    } else {
        errEl.style.display = 'none';
    }
    document.getElementById('pw-input').value = '';
    document.getElementById('pw-input').focus();
}

function hideGate() {
    document.getElementById('pw-gate').style.display = 'none';
    document.getElementById('fw-main').style.display = 'block';
}

async function authenticate() {
    const pw = document.getElementById('pw-input').value;
    if (!pw) return;

    const btn = document.getElementById('pw-submit');
    btn.disabled = true;
    btn.textContent = 'Verifying...';

    try {
        const res = await fetch('/api/firewall/status', {
            headers: { 'X-Admin-Password': pw },
        });
        if (res.status === 401) {
            showGate('Incorrect password. Please try again.');
            return;
        }
        if (!res.ok) {
            showGate('Error: ' + (await res.text()));
            return;
        }
        sudoPassword = pw;
        const data = await res.json();
        currentRules = data.rules || [];
        hideGate();
        renderStatus(data.enabled);
        renderRules(currentRules);
    } catch {
        showGate('Network error. Please try again.');
    } finally {
        btn.disabled = false;
        btn.textContent = 'Unlock';
    }
}

// Called when any action gets a 401 — forces re-authentication.
function handleAuthFailure() {
    sudoPassword = '';
    showGate('Incorrect password. Please enter your VPSMyth admin password again.');
}

// ─── Firewall helpers ─────────────────────────────────────────────────────────

function authHeaders() {
    return { 'X-Admin-Password': sudoPassword };
}

function authJsonHeaders() {
    return { 'Content-Type': 'application/json', 'X-Admin-Password': sudoPassword };
}

async function loadStatus() {
    try {
        const res = await fetch('/api/firewall/status', { headers: authHeaders() });
        if (res.status === 401) { handleAuthFailure(); return; }
        if (!res.ok) return;
        const data = await res.json();
        currentRules = data.rules || [];
        renderStatus(data.enabled);
        renderRules(currentRules);
    } catch { /* ignore */ }
}

function renderStatus(enabled) {
    const pill = document.getElementById('status-pill');
    const text = document.getElementById('status-text');
    const warning = document.getElementById('fw-warning');
    const countEl = document.getElementById('rule-count');

    pill.className = 'status-pill ' + (enabled ? 'active' : 'inactive');
    text.textContent = enabled ? 'Active' : 'Inactive';
    warning.classList.toggle('hidden', enabled);

    const ipv4Count = currentRules.filter(r => !r.is_ipv6).length;
    countEl.textContent = ipv4Count ? `${ipv4Count} rule${ipv4Count !== 1 ? 's' : ''}` : 'No rules';
}

function renderRules(rules) {
    const container = document.getElementById('rules-container');
    if (!rules || rules.length === 0) {
        container.innerHTML = '<p class="rules-empty">No rules configured.</p>';
        return;
    }

    const rows = rules.map(r => `
        <tr>
            <td class="num-cell">${r.number}</td>
            <td>${escHtml(r.to)}${r.is_ipv6 ? '<span class="ipv6-tag">IPv6</span>' : ''}</td>
            <td class="action-cell"><span class="action-badge ${badgeClass(r.action)}">${escHtml(r.action)}</span></td>
            <td>${escHtml(r.from)}</td>
            <td><button class="btn-delete-rule" data-num="${r.number}">Remove</button></td>
        </tr>
    `).join('');

    container.innerHTML = `
        <table class="rules-table">
            <thead>
                <tr>
                    <th>#</th>
                    <th>To (Port / Service)</th>
                    <th>Action</th>
                    <th>From</th>
                    <th></th>
                </tr>
            </thead>
            <tbody>${rows}</tbody>
        </table>
    `;

    container.querySelectorAll('.btn-delete-rule').forEach(btn => {
        btn.addEventListener('click', () => deleteRule(parseInt(btn.dataset.num, 10)));
    });
}

function badgeClass(action) {
    const a = (action || '').toUpperCase();
    if (a.includes('ALLOW')) return 'badge-allow';
    if (a.includes('LIMIT')) return 'badge-limit';
    if (a.includes('DENY') || a.includes('REJECT')) return 'badge-deny';
    return '';
}

// ─── Actions ──────────────────────────────────────────────────────────────────

async function toggleFirewall(enable) {
    const endpoint = enable ? '/api/firewall/enable' : '/api/firewall/disable';
    const btnEnable = document.getElementById('btn-enable');
    const btnDisable = document.getElementById('btn-disable');

    if (!enable && !confirm('Disable the firewall? All ports will be open until you re-enable it.')) return;

    btnEnable.disabled = true;
    btnDisable.disabled = true;
    try {
        const res = await fetch(endpoint, { method: 'POST', headers: authHeaders() });
        if (res.status === 401) { handleAuthFailure(); return; }
        if (!res.ok) { alert('Failed: ' + await res.text()); return; }
        await loadStatus();
    } catch {
        alert('Network error. Please try again.');
    } finally {
        btnEnable.disabled = false;
        btnDisable.disabled = false;
    }
}

async function addRule(action, direction, port, proto, fromIP) {
    const errEl = document.getElementById('rule-error');
    errEl.style.display = 'none';

    const btn = document.getElementById('btn-add-rule');
    btn.disabled = true;
    btn.textContent = 'Adding...';

    try {
        const res = await fetch('/api/firewall/rule/add', {
            method: 'POST',
            headers: authJsonHeaders(),
            body: JSON.stringify({ action, direction, port, proto, from_ip: fromIP }),
        });
        if (res.status === 401) { handleAuthFailure(); return; }
        if (!res.ok) {
            errEl.textContent = (await res.text()) || 'Failed to add rule.';
            errEl.style.display = 'block';
            return;
        }
        document.getElementById('f-port').value = '';
        document.getElementById('f-from').value = '';
        await loadStatus();
    } catch {
        errEl.textContent = 'Network error. Please try again.';
        errEl.style.display = 'block';
    } finally {
        btn.disabled = false;
        btn.textContent = 'Add Rule';
    }
}

async function deleteRule(num) {
    if (!confirm(`Remove rule #${num}?`)) return;
    try {
        const res = await fetch('/api/firewall/rule/delete', {
            method: 'POST',
            headers: authJsonHeaders(),
            body: JSON.stringify({ number: num }),
        });
        if (res.status === 401) { handleAuthFailure(); return; }
        if (!res.ok) { alert('Failed: ' + await res.text()); return; }
        await loadStatus();
    } catch {
        alert('Network error. Please try again.');
    }
}

// ─── Presets ──────────────────────────────────────────────────────────────────

async function applyPreset(key) {
    const presets = {
        'ssh-allow': { action: 'allow', direction: 'in', port: '22',  proto: 'tcp', from_ip: '' },
        'ssh-limit': { action: 'limit', direction: 'in', port: '22',  proto: 'tcp', from_ip: '' },
        'http':      { action: 'allow', direction: 'in', port: '80',  proto: 'tcp', from_ip: '' },
        'https':     { action: 'allow', direction: 'in', port: '443', proto: 'tcp', from_ip: '' },
        'http-deny': { action: 'deny',  direction: 'in', port: '80',  proto: 'tcp', from_ip: '' },
    };

    if (key === 'custom-ip') {
        if (!myIPForPreset) {
            document.getElementById('f-action').value = 'allow';
            document.getElementById('f-direction').value = 'in';
            document.getElementById('f-port').value = '22';
            document.getElementById('f-proto').value = 'tcp';
            document.getElementById('f-from').focus();
            return;
        }
        await addRule('allow', 'in', '22', 'tcp', myIPForPreset);
        return;
    }

    const p = presets[key];
    if (p) await addRule(p.action, p.direction, p.port, p.proto, p.from_ip);
}

// ─── Utilities ────────────────────────────────────────────────────────────────

function escHtml(str) {
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;');
}

async function fetchMyIP() {
    try {
        const res = await fetch('/api/security/myip');
        const data = await res.json();
        myIPForPreset = data.ip || '';
    } catch { /* ignore */ }
}

// ─── Event wiring ─────────────────────────────────────────────────────────────

document.getElementById('pw-submit').addEventListener('click', authenticate);
document.getElementById('pw-input').addEventListener('keydown', e => {
    if (e.key === 'Enter') authenticate();
});

document.getElementById('btn-enable').addEventListener('click', () => toggleFirewall(true));
document.getElementById('btn-disable').addEventListener('click', () => toggleFirewall(false));

document.getElementById('btn-add-rule').addEventListener('click', () => {
    const port = document.getElementById('f-port').value.trim();
    const errEl = document.getElementById('rule-error');
    if (!port) {
        errEl.textContent = 'Port is required.';
        errEl.style.display = 'block';
        return;
    }
    addRule(
        document.getElementById('f-action').value,
        document.getElementById('f-direction').value,
        port,
        document.getElementById('f-proto').value,
        document.getElementById('f-from').value.trim(),
    );
});

document.getElementById('f-port').addEventListener('keydown', e => {
    if (e.key === 'Enter') document.getElementById('btn-add-rule').click();
});

document.querySelectorAll('.btn-preset').forEach(btn => {
    btn.addEventListener('click', () => applyPreset(btn.dataset.preset));
});

document.getElementById('logout-btn').addEventListener('click', async e => {
    e.preventDefault();
    await fetch('/api/auth/logout', { method: 'POST' });
    const parts = window.location.pathname.split('/');
    parts[parts.length - 1] = 'login.html';
    window.location.href = parts.join('/');
});

// ─── Init ─────────────────────────────────────────────────────────────────────

fetchMyIP();
showGate(); // always start at the password gate
