'use strict';

let myIP = '';
let whitelist = [];

async function loadMyIP() {
    try {
        const res = await fetch('/api/security/myip');
        const data = await res.json();
        myIP = data.ip || '';
        document.getElementById('my-ip-display').textContent = myIP || 'Unknown';
    } catch {
        document.getElementById('my-ip-display').textContent = 'Unknown';
    }
}

async function loadWhitelist() {
    try {
        const res = await fetch('/api/security/whitelist');
        whitelist = await res.json();
    } catch {
        whitelist = [];
    }
    renderWhitelist();
    updateIPStatus();
}

function renderWhitelist() {
    const container = document.getElementById('whitelist-container');
    if (!whitelist || whitelist.length === 0) {
        container.innerHTML = '<p class="whitelist-empty">No entries — all IPs are currently allowed.</p>';
        return;
    }

    const rows = whitelist.map(e => `
        <tr>
            <td>${escHtml(e.cidr)}</td>
            <td class="label-cell">${escHtml(e.label || '—')}</td>
            <td class="date-cell">${formatDate(e.created_at)}</td>
            <td>
                <button class="btn-delete" data-id="${e.id}" title="Remove">Remove</button>
            </td>
        </tr>
    `).join('');

    container.innerHTML = `
        <table class="whitelist-table">
            <thead>
                <tr>
                    <th>IP / CIDR</th>
                    <th>Label</th>
                    <th>Added</th>
                    <th></th>
                </tr>
            </thead>
            <tbody>${rows}</tbody>
        </table>
    `;

    container.querySelectorAll('.btn-delete').forEach(btn => {
        btn.addEventListener('click', () => deleteEntry(parseInt(btn.dataset.id, 10)));
    });
}

function updateIPStatus() {
    const warning = document.getElementById('lockout-warning');
    const statusEl = document.getElementById('ip-status');
    const addMyIPBtn = document.getElementById('add-my-ip-btn');

    if (!myIP || whitelist.length === 0) {
        warning.classList.add('hidden');
        statusEl.textContent = whitelist.length === 0 ? '(no restrictions active)' : '';
        addMyIPBtn.style.display = 'none';
        return;
    }

    const covered = isIPCovered(myIP, whitelist);
    if (covered) {
        warning.classList.add('hidden');
        statusEl.innerHTML = '<span class="status-dot"></span>Your IP is whitelisted';
        addMyIPBtn.style.display = 'none';
    } else {
        warning.classList.remove('hidden');
        document.getElementById('warning-ip').textContent = myIP;
        statusEl.innerHTML = '<span class="status-dot locked"></span>Your IP is NOT in the whitelist';
        addMyIPBtn.style.display = 'inline-block';
    }
}

// Simple client-side CIDR containment check for the "Add My IP" helper button.
function isIPCovered(ip, entries) {
    // We do a basic prefix check — exact match or subnet match for display only.
    // The server is the authoritative source; this is just a UX hint.
    return entries.some(e => {
        if (e.cidr === ip + '/32' || e.cidr === ip + '/128' || e.cidr === ip) return true;
        // For subnets just check if the stored CIDR starts with the same network prefix
        const slash = e.cidr.indexOf('/');
        if (slash === -1) return false;
        const bits = parseInt(e.cidr.slice(slash + 1), 10);
        if (bits === 32 || bits === 128) return e.cidr.startsWith(ip + '/');
        // For partial subnets we can't do full math in JS easily — ask server
        return false;
    });
}

async function addEntry(cidr, label) {
    const cidrErr = document.getElementById('cidr-error');
    cidrErr.style.display = 'none';

    const btn = document.getElementById('add-btn');
    btn.disabled = true;
    btn.textContent = 'Adding...';

    try {
        const res = await fetch('/api/security/whitelist/add', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ cidr, label }),
        });
        if (!res.ok) {
            const msg = await res.text();
            cidrErr.textContent = msg || 'Failed to add entry.';
            cidrErr.style.display = 'block';
            return;
        }
        document.getElementById('input-cidr').value = '';
        document.getElementById('input-label').value = '';
        await loadWhitelist();
    } catch {
        cidrErr.textContent = 'Network error. Please try again.';
        cidrErr.style.display = 'block';
    } finally {
        btn.disabled = false;
        btn.textContent = 'Add to Whitelist';
    }
}

async function deleteEntry(id) {
    if (!confirm('Remove this IP from the whitelist?')) return;
    try {
        const res = await fetch(`/api/security/whitelist/delete?id=${id}`, { method: 'DELETE' });
        if (!res.ok) {
            alert('Failed to remove entry.');
            return;
        }
        await loadWhitelist();
    } catch {
        alert('Network error. Please try again.');
    }
}

function escHtml(str) {
    return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function formatDate(str) {
    if (!str) return '—';
    const d = new Date(str);
    if (isNaN(d)) return str;
    return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}

// Wire up events
document.getElementById('add-btn').addEventListener('click', () => {
    const cidr = document.getElementById('input-cidr').value.trim();
    const label = document.getElementById('input-label').value.trim();
    if (!cidr) {
        const err = document.getElementById('cidr-error');
        err.textContent = 'Please enter an IP address or CIDR.';
        err.style.display = 'block';
        return;
    }
    addEntry(cidr, label);
});

document.getElementById('input-cidr').addEventListener('keydown', e => {
    if (e.key === 'Enter') document.getElementById('add-btn').click();
});

document.getElementById('add-my-ip-btn').addEventListener('click', () => {
    document.getElementById('input-cidr').value = myIP;
    document.getElementById('input-label').value = 'My IP';
    document.getElementById('add-btn').click();
});

document.getElementById('logout-btn').addEventListener('click', async e => {
    e.preventDefault();
    await fetch('/api/auth/logout', { method: 'POST' });
    window.location.href = '/login.html';
});

// Init
(async () => {
    await loadMyIP();
    await loadWhitelist();
})();
