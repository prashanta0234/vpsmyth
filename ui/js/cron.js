'use strict';

let editingID = null;
let activeType = 'command';

// ── Schedule helpers ──────────────────────────────────────────────────────────

const DAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

function cronToHuman(expr) {
    const f = expr.trim().split(/\s+/);
    if (f.length !== 5) return '';
    const [min, hr, dom, mon, dow] = f;

    if (expr === '* * * * *') return 'Every minute';
    if (min.startsWith('*/') && hr === '*' && dom === '*' && mon === '*' && dow === '*') {
        const n = min.slice(2);
        return `Every ${n} minute${n === '1' ? '' : 's'}`;
    }
    if (hr.startsWith('*/') && min === '0' && dom === '*' && mon === '*' && dow === '*') {
        const n = hr.slice(2);
        return `Every ${n} hour${n === '1' ? '' : 's'}`;
    }
    if (hr === '*' && min === '0' && dom === '*' && mon === '*' && dow === '*') {
        return 'Every hour at :00';
    }

    let desc = '';
    // Time part
    const minNum = /^\d+$/.test(min) ? parseInt(min) : null;
    const hrNum  = /^\d+$/.test(hr)  ? parseInt(hr)  : null;
    if (minNum !== null && hrNum !== null) {
        const ampm = hrNum >= 12 ? 'PM' : 'AM';
        const h12 = hrNum % 12 || 12;
        const mm = String(minNum).padStart(2, '0');
        desc = `at ${h12}:${mm} ${ampm}`;
    } else if (minNum !== null && hr === '*') {
        desc = `at minute ${minNum} of every hour`;
    }

    // Frequency part
    if (dom === '*' && mon === '*' && dow === '*') {
        return `Every day ${desc}`.trim();
    }
    if (dom === '*' && mon === '*' && /^\d+$/.test(dow)) {
        return `Every ${DAYS[parseInt(dow)] || 'weekday'} ${desc}`.trim();
    }
    if (/^\d+$/.test(dom) && mon === '*' && dow === '*') {
        const d = parseInt(dom);
        const suffix = d === 1 ? 'st' : d === 2 ? 'nd' : d === 3 ? 'rd' : 'th';
        return `On the ${d}${suffix} of every month ${desc}`.trim();
    }
    if (/^\d+$/.test(dom) && /^\d+$/.test(mon) && dow === '*') {
        return `On ${MONTHS[parseInt(mon) - 1]} ${dom} ${desc}`.trim();
    }

    return expr;
}

// ── Fetch helpers ─────────────────────────────────────────────────────────────

async function api(path, opts = {}) {
    const res = await fetch(path, opts);
    if (res.status === 401) { handleAuthError(); throw new Error('unauthorized'); }
    return res;
}

// ── Jobs list ─────────────────────────────────────────────────────────────────

async function loadJobs() {
    try {
        const res = await api('/api/cron/jobs');
        if (!res.ok) return;
        const jobs = await res.json();
        renderJobs(jobs);
    } catch { /* ignore */ }
}

function renderJobs(jobs) {
    const container = document.getElementById('jobs-container');
    if (!jobs || jobs.length === 0) {
        container.innerHTML = '<p class="jobs-empty">No cron jobs yet. Click <strong>+ Add Job</strong> to create one.</p>';
        return;
    }

    const rows = jobs.map(j => {
        const human = cronToHuman(j.schedule);
        const typeCls = `type-${j.type}`;
        const typeLabel = j.type.charAt(0).toUpperCase() + j.type.slice(1);

        let statusHtml = '<span class="status-badge"><span class="dot dot-never"></span>Never run</span>';
        if (j.last_exit_code !== null && j.last_exit_code !== undefined) {
            const ok = j.last_exit_code === 0;
            statusHtml = `<span class="status-badge"><span class="dot ${ok ? 'dot-ok' : 'dot-fail'}"></span>${ok ? 'OK' : 'Failed'} (exit ${j.last_exit_code})</span>`;
        }

        const lastRun = j.last_run_at ? new Date(j.last_run_at).toLocaleString() : '—';

        return `<tr>
            <td><strong>${escHtml(j.name)}</strong></td>
            <td><span class="type-badge ${typeCls}">${typeLabel}</span></td>
            <td>
                <span class="schedule-text" title="${escHtml(j.schedule)}">${escHtml(j.schedule)}</span>
                ${human ? `<br><span style="font-size:0.72rem; color:var(--muted,#94a3b8);">${escHtml(human)}</span>` : ''}
            </td>
            <td style="font-size:0.8rem; color:var(--muted,#94a3b8);">${lastRun}</td>
            <td>${statusHtml}</td>
            <td>
                <label class="toggle" title="${j.enabled ? 'Enabled' : 'Disabled'}">
                    <input type="checkbox" ${j.enabled ? 'checked' : ''} data-id="${j.id}" class="toggle-cb">
                    <span class="toggle-slider"></span>
                </label>
            </td>
            <td>
                <div class="action-group">
                    <button class="btn-action btn-run" data-id="${j.id}" title="Run now">▶ Run</button>
                    <button class="btn-action btn-history" data-id="${j.id}" data-name="${escHtml(j.name)}" title="History">History</button>
                    <button class="btn-action btn-edit" data-id="${j.id}" title="Edit">Edit</button>
                    <button class="btn-action btn-del" data-id="${j.id}" title="Delete">Delete</button>
                </div>
            </td>
        </tr>`;
    }).join('');

    container.innerHTML = `
        <table class="jobs-table">
            <thead>
                <tr>
                    <th>Name</th><th>Type</th><th>Schedule</th>
                    <th>Last Run</th><th>Status</th><th>On</th><th></th>
                </tr>
            </thead>
            <tbody>${rows}</tbody>
        </table>`;

    container.querySelectorAll('.toggle-cb').forEach(cb => {
        cb.addEventListener('change', () => toggleJob(parseInt(cb.dataset.id), cb.checked));
    });
    container.querySelectorAll('.btn-run').forEach(btn => {
        btn.addEventListener('click', () => runNow(parseInt(btn.dataset.id), btn));
    });
    container.querySelectorAll('.btn-history').forEach(btn => {
        btn.addEventListener('click', () => showHistory(parseInt(btn.dataset.id), btn.dataset.name));
    });
    container.querySelectorAll('.btn-edit').forEach(btn => {
        btn.addEventListener('click', () => openEditModal(parseInt(btn.dataset.id)));
    });
    container.querySelectorAll('.btn-del').forEach(btn => {
        btn.addEventListener('click', () => deleteJob(parseInt(btn.dataset.id)));
    });
}

// ── Job actions ───────────────────────────────────────────────────────────────

async function toggleJob(id, enabled) {
    try {
        await api('/api/cron/jobs/toggle', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id, enabled }),
        });
    } catch { /* ignore */ }
}

async function runNow(id, btn) {
    btn.disabled = true;
    btn.textContent = '...';
    try {
        const res = await api('/api/cron/jobs/run', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id }),
        });
        if (res.ok) {
            btn.textContent = '✓ Queued';
            setTimeout(() => { btn.textContent = '▶ Run'; btn.disabled = false; loadJobs(); }, 3000);
        } else {
            btn.textContent = '▶ Run'; btn.disabled = false;
            alert('Failed: ' + await res.text());
        }
    } catch { btn.textContent = '▶ Run'; btn.disabled = false; }
}

async function deleteJob(id) {
    if (!confirm('Delete this cron job?')) return;
    try {
        const res = await api('/api/cron/jobs/delete', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id }),
        });
        if (res.ok) loadJobs();
        else alert('Failed: ' + await res.text());
    } catch { /* ignore */ }
}

// ── History modal ─────────────────────────────────────────────────────────────

async function showHistory(id, name) {
    document.getElementById('history-title').textContent = `Run History — ${name}`;
    document.getElementById('history-container').innerHTML = '<p class="jobs-empty">Loading...</p>';
    document.getElementById('history-modal').classList.add('open');

    try {
        const res = await api(`/api/cron/jobs/runs?id=${id}`);
        if (!res.ok) { document.getElementById('history-container').innerHTML = '<p class="jobs-empty">Failed to load.</p>'; return; }
        const runs = await res.json();
        renderHistory(runs);
    } catch { /* ignore */ }
}

function renderHistory(runs) {
    const container = document.getElementById('history-container');
    if (!runs || runs.length === 0) {
        container.innerHTML = '<p class="jobs-empty">No runs recorded yet.</p>';
        return;
    }
    const rows = runs.map(r => {
        const ok = r.exit_code === 0;
        const d = new Date(r.ran_at).toLocaleString();
        const dur = r.duration_ms < 1000 ? `${r.duration_ms}ms` : `${(r.duration_ms / 1000).toFixed(1)}s`;
        const outputId = `out-${r.id}`;
        const hasOutput = r.output && r.output.trim() !== '';
        return `<tr>
            <td style="white-space:nowrap;">${d}</td>
            <td style="white-space:nowrap;">${dur}</td>
            <td class="${ok ? 'exit-ok' : 'exit-fail'}">${r.exit_code}</td>
            <td>
                ${hasOutput
                    ? `<button class="expand-btn" onclick="toggleOutput('${outputId}')">Show output</button>
                       <pre class="output-pre" id="${outputId}" style="display:none;">${escHtml(r.output)}</pre>`
                    : '<span style="color:var(--muted,#94a3b8); font-size:0.75rem;">(no output)</span>'
                }
            </td>
        </tr>`;
    }).join('');

    container.innerHTML = `
        <table class="history-table">
            <thead><tr><th>Time</th><th>Duration</th><th>Exit</th><th>Output</th></tr></thead>
            <tbody>${rows}</tbody>
        </table>`;
}

function toggleOutput(id) {
    const el = document.getElementById(id);
    const btn = el.previousElementSibling;
    if (el.style.display === 'none') {
        el.style.display = 'block';
        btn.textContent = 'Hide output';
    } else {
        el.style.display = 'none';
        btn.textContent = 'Show output';
    }
}

// ── Add / Edit modal ──────────────────────────────────────────────────────────

function openAddModal() {
    editingID = null;
    document.getElementById('modal-title').textContent = 'Add Cron Job';
    document.getElementById('modal-save').textContent = 'Save Job';
    clearModal();
    document.getElementById('job-modal').classList.add('open');
    document.getElementById('f-name').focus();
}

async function openEditModal(id) {
    try {
        const res = await api('/api/cron/jobs');
        if (!res.ok) return;
        const jobs = await res.json();
        const job = jobs.find(j => j.id === id);
        if (!job) return;

        editingID = id;
        document.getElementById('modal-title').textContent = 'Edit Cron Job';
        document.getElementById('modal-save').textContent = 'Update Job';
        clearModal();

        document.getElementById('f-name').value = job.name;
        document.getElementById('f-schedule').value = job.schedule;
        updateSchedulePreview(job.schedule);
        highlightPreset(job.schedule);

        setActiveType(job.type);

        if (job.type === 'command') {
            document.getElementById('f-command').value = job.command;
        } else if (job.type === 'script') {
            document.getElementById('f-script').value = job.script_content;
        } else if (job.type === 'curl') {
            document.getElementById('f-curl-method').value = job.curl_method || 'GET';
            document.getElementById('f-curl-url').value = job.curl_url;
            document.getElementById('f-curl-body').value = job.curl_body;
            document.getElementById('f-curl-status').value = job.curl_expected_status || 200;
            // Populate headers
            const headers = document.getElementById('headers-list');
            headers.innerHTML = '';
            if (job.curl_headers && job.curl_headers !== '{}') {
                try {
                    const h = JSON.parse(job.curl_headers);
                    Object.entries(h).forEach(([k, v]) => addHeaderRow(k, v));
                } catch { /* ignore */ }
            }
        }

        document.getElementById('job-modal').classList.add('open');
    } catch { /* ignore */ }
}

function clearModal() {
    document.getElementById('f-name').value = '';
    document.getElementById('f-schedule').value = '';
    document.getElementById('f-command').value = '';
    document.getElementById('f-script').value = '';
    document.getElementById('f-curl-method').value = 'GET';
    document.getElementById('f-curl-url').value = '';
    document.getElementById('f-curl-body').value = '';
    document.getElementById('f-curl-status').value = '200';
    document.getElementById('headers-list').innerHTML = '';
    document.getElementById('schedule-preview').textContent = '';
    document.getElementById('form-error').style.display = 'none';
    document.querySelectorAll('.btn-preset').forEach(b => b.classList.remove('active'));
    setActiveType('command');
}

function closeModal() {
    document.getElementById('job-modal').classList.remove('open');
}

async function saveJob() {
    const saveBtn = document.getElementById('modal-save');
    const errEl = document.getElementById('form-error');
    errEl.style.display = 'none';

    const name = document.getElementById('f-name').value.trim();
    const schedule = document.getElementById('f-schedule').value.trim();

    const job = {
        name,
        schedule,
        type: activeType,
        command: document.getElementById('f-command').value.trim(),
        script_content: document.getElementById('f-script').value,
        curl_url: document.getElementById('f-curl-url').value.trim(),
        curl_method: document.getElementById('f-curl-method').value,
        curl_headers: getHeadersJSON(),
        curl_body: document.getElementById('f-curl-body').value.trim(),
        curl_expected_status: parseInt(document.getElementById('f-curl-status').value) || 200,
    };

    if (editingID) job.id = editingID;

    saveBtn.disabled = true;
    saveBtn.textContent = 'Saving...';
    try {
        const url = editingID ? '/api/cron/jobs/update' : '/api/cron/jobs/add';
        const res = await api(url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(job),
        });
        if (!res.ok) {
            errEl.textContent = await res.text();
            errEl.style.display = 'block';
            return;
        }
        closeModal();
        loadJobs();
    } catch (e) {
        errEl.textContent = 'Network error. Please try again.';
        errEl.style.display = 'block';
    } finally {
        saveBtn.disabled = false;
        saveBtn.textContent = editingID ? 'Update Job' : 'Save Job';
    }
}

// ── Type tabs ─────────────────────────────────────────────────────────────────

function setActiveType(type) {
    activeType = type;
    document.querySelectorAll('.tab-btn').forEach(b => b.classList.toggle('active', b.dataset.type === type));
    document.querySelectorAll('.tab-panel').forEach(p => p.classList.toggle('active', p.id === `panel-${type}`));
}

// ── CURL headers ──────────────────────────────────────────────────────────────

function addHeaderRow(key = '', value = '') {
    const list = document.getElementById('headers-list');
    const row = document.createElement('div');
    row.className = 'header-row';
    row.innerHTML = `
        <input type="text" placeholder="Header name" value="${escHtml(key)}" class="hdr-key">
        <input type="text" placeholder="Value" value="${escHtml(value)}" class="hdr-val">
        <button class="btn-remove-header" title="Remove">&times;</button>`;
    row.querySelector('.btn-remove-header').addEventListener('click', () => row.remove());
    list.appendChild(row);
}

function getHeadersJSON() {
    const rows = document.getElementById('headers-list').querySelectorAll('.header-row');
    const h = {};
    rows.forEach(row => {
        const k = row.querySelector('.hdr-key').value.trim();
        const v = row.querySelector('.hdr-val').value.trim();
        if (k) h[k] = v;
    });
    return JSON.stringify(h);
}

// ── Schedule preview ──────────────────────────────────────────────────────────

function updateSchedulePreview(expr) {
    const preview = document.getElementById('schedule-preview');
    const human = cronToHuman(expr.trim());
    preview.textContent = human || '';
}

function highlightPreset(expr) {
    document.querySelectorAll('.btn-preset').forEach(b => {
        b.classList.toggle('active', b.dataset.expr === expr.trim());
    });
}

// ── Utilities ─────────────────────────────────────────────────────────────────

function escHtml(str) {
    return String(str || '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

// ── Event wiring ──────────────────────────────────────────────────────────────

document.getElementById('btn-add-job').addEventListener('click', openAddModal);
document.getElementById('modal-close').addEventListener('click', closeModal);
document.getElementById('modal-cancel').addEventListener('click', closeModal);
document.getElementById('modal-save').addEventListener('click', saveJob);

document.getElementById('history-close').addEventListener('click', () => {
    document.getElementById('history-modal').classList.remove('open');
});
document.getElementById('history-dismiss').addEventListener('click', () => {
    document.getElementById('history-modal').classList.remove('open');
});

// Close modals on overlay click
document.getElementById('job-modal').addEventListener('click', e => {
    if (e.target === document.getElementById('job-modal')) closeModal();
});
document.getElementById('history-modal').addEventListener('click', e => {
    if (e.target === document.getElementById('history-modal'))
        document.getElementById('history-modal').classList.remove('open');
});

// Type tabs
document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.addEventListener('click', () => setActiveType(btn.dataset.type));
});

// Schedule presets
document.querySelectorAll('.btn-preset').forEach(btn => {
    btn.addEventListener('click', () => {
        document.getElementById('f-schedule').value = btn.dataset.expr;
        updateSchedulePreview(btn.dataset.expr);
        highlightPreset(btn.dataset.expr);
    });
});

// Schedule input live preview
document.getElementById('f-schedule').addEventListener('input', e => {
    updateSchedulePreview(e.target.value);
    highlightPreset(e.target.value);
});

// CURL add header
document.getElementById('btn-add-header').addEventListener('click', () => addHeaderRow());

// Init
loadJobs();
