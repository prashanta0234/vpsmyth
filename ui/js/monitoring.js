const chartDefaults = {
    responsive: true,
    maintainAspectRatio: false,
    animation: false,
    plugins: { legend: { display: false } },
    scales: {
        x: {
            ticks: { maxTicksLimit: 8, font: { size: 10 } },
            grid: { color: 'rgba(0,0,0,0.05)' }
        },
        y: {
            beginAtZero: true,
            ticks: { font: { size: 10 } },
            grid: { color: 'rgba(0,0,0,0.05)' }
        }
    }
};

const lineDataset = (label, data, color) => ({
    label,
    data,
    borderColor: color,
    backgroundColor: color + '20',
    fill: true,
    tension: 0.3,
    pointRadius: 0,
    borderWidth: 2,
});

function fmtTime(ts) {
    const d = new Date(ts * 1000);
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function fmtDate(ts) {
    const d = new Date(ts * 1000);
    return d.toLocaleDateString([], { month: 'short', day: 'numeric' }) + ' ' +
        d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

// ── System charts ────────────────────────────────────────────────────────────

let cpuChart, ramChart, diskChart;
let currentRange = '1h';

function initSystemCharts() {
    cpuChart = new Chart(document.getElementById('cpu-chart'), {
        type: 'line',
        data: { labels: [], datasets: [lineDataset('CPU %', [], '#4f46e5')] },
        options: { ...chartDefaults, scales: { ...chartDefaults.scales, y: { ...chartDefaults.scales.y, max: 100 } } }
    });
    ramChart = new Chart(document.getElementById('ram-chart'), {
        type: 'line',
        data: { labels: [], datasets: [lineDataset('Used MB', [], '#0ea5e9'), lineDataset('Total MB', [], '#cbd5e1')] },
        options: { ...chartDefaults }
    });
    diskChart = new Chart(document.getElementById('disk-chart'), {
        type: 'line',
        data: { labels: [], datasets: [lineDataset('Used GB', [], '#f59e0b'), lineDataset('Total GB', [], '#e2e8f0')] },
        options: { ...chartDefaults }
    });
}

async function loadSystemMetrics(range) {
    const res = await fetch(`/api/monitoring/system?range=${range}`);
    if (!res.ok) return;
    const { data } = await res.json();
    if (!data || data.length === 0) return;

    const isLong = range === '7d' || range === '30d';
    const labels = data.map(p => isLong ? fmtDate(p.ts) : fmtTime(p.ts));

    cpuChart.data.labels = labels;
    cpuChart.data.datasets[0].data = data.map(p => +p.cpu.toFixed(1));
    cpuChart.update();

    ramChart.data.labels = labels;
    ramChart.data.datasets[0].data = data.map(p => p.ram_used_mb);
    ramChart.data.datasets[1].data = data.map(p => p.ram_total_mb);
    ramChart.update();

    diskChart.data.labels = labels;
    diskChart.data.datasets[0].data = data.map(p => +p.disk_used_gb.toFixed(2));
    diskChart.data.datasets[1].data = data.map(p => +p.disk_total_gb.toFixed(2));
    diskChart.update();
}

document.getElementById('system-range-tabs').addEventListener('click', e => {
    const btn = e.target.closest('.range-tab');
    if (!btn) return;
    document.querySelectorAll('#system-range-tabs .range-tab').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    currentRange = btn.dataset.range;
    loadSystemMetrics(currentRange);
});

// ── Container cards ──────────────────────────────────────────────────────────

const containerCharts = {};

function pctColor(pct) {
    if (pct >= 80) return 'danger';
    if (pct >= 50) return 'warn';
    return '';
}

function stale(lastSeen) {
    return Date.now() / 1000 - lastSeen > 120; // not seen in last 2 minutes
}

async function loadContainers() {
    const res = await fetch('/api/monitoring/containers');
    if (!res.ok) return;
    const { containers } = await res.json();

    const grid = document.getElementById('container-grid');
    if (!containers || containers.length === 0) {
        grid.innerHTML = '<div class="no-data">No containers found yet. Data appears after the first 30-second collection cycle.</div>';
        return;
    }

    grid.innerHTML = '';
    containers.forEach(c => {
        const memPct = c.mem_limit_mb > 0 ? (c.mem_used_mb / c.mem_limit_mb) * 100 : 0;
        const isStale = stale(c.last_seen);
        const card = document.createElement('div');
        card.className = 'container-card';
        card.dataset.id = c.container_id;
        card.innerHTML = `
            <div class="container-card-header">
                <span class="container-name">${c.container_name}</span>
                <span class="container-badge ${isStale ? 'stale' : ''}">${isStale ? 'stopped' : 'running'}</span>
            </div>
            <div class="mini-stats">
                <div class="mini-stat"><span>CPU</span><strong>${c.cpu.toFixed(1)}%</strong></div>
                <div class="mini-stat"><span>RAM</span><strong>${c.mem_used_mb.toFixed(0)} MB</strong></div>
                <div class="mini-stat"><span>Limit</span><strong>${c.mem_limit_mb > 0 ? c.mem_limit_mb.toFixed(0) + ' MB' : '—'}</strong></div>
                <div class="mini-stat"><span>ID</span><strong style="font-size:0.75rem">${c.container_id.slice(0, 8)}</strong></div>
            </div>
            <div class="progress-bar">
                <div class="progress-fill ${pctColor(memPct)}" style="width:${Math.min(memPct, 100).toFixed(1)}%"></div>
            </div>
            <div class="container-chart-wrap" id="cw-${c.container_id}">
                <div class="range-tabs" style="margin: 0.75rem 0 0.5rem;">
                    <button class="range-tab active" data-range="1h">1H</button>
                    <button class="range-tab" data-range="6h">6H</button>
                    <button class="range-tab" data-range="24h">24H</button>
                    <button class="range-tab" data-range="7d">7D</button>
                    <button class="range-tab" data-range="30d">30D</button>
                </div>
                <canvas id="cc-${c.container_id}"></canvas>
            </div>
        `;

        card.addEventListener('click', e => {
            if (e.target.closest('.range-tab')) return;
            toggleContainerChart(c.container_id);
        });

        card.querySelectorAll('.range-tab').forEach(btn => {
            btn.addEventListener('click', e => {
                e.stopPropagation();
                card.querySelectorAll('.range-tab').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                loadContainerChart(c.container_id, btn.dataset.range);
            });
        });

        grid.appendChild(card);
    });
}

function toggleContainerChart(id) {
    const wrap = document.getElementById(`cw-${id}`);
    const isOpen = wrap.classList.toggle('open');
    if (isOpen && !containerCharts[id]) {
        loadContainerChart(id, '1h');
    }
}

async function loadContainerChart(id, range) {
    const res = await fetch(`/api/monitoring/container?id=${id}&range=${range}`);
    if (!res.ok) return;
    const { data } = await res.json();

    const canvas = document.getElementById(`cc-${id}`);
    const isLong = range === '7d' || range === '30d';
    const labels = (data || []).map(p => isLong ? fmtDate(p.ts) : fmtTime(p.ts));
    const cpuData = (data || []).map(p => +p.cpu.toFixed(1));
    const memData = (data || []).map(p => +p.mem_used_mb.toFixed(0));

    if (containerCharts[id]) {
        containerCharts[id].destroy();
    }

    containerCharts[id] = new Chart(canvas, {
        type: 'line',
        data: {
            labels,
            datasets: [
                lineDataset('CPU %', cpuData, '#4f46e5'),
                lineDataset('RAM MB', memData, '#0ea5e9'),
            ]
        },
        options: {
            ...chartDefaults,
            plugins: {
                legend: { display: true, labels: { font: { size: 10 }, boxWidth: 12 } }
            }
        }
    });
}

// ── App monitoring toggles ───────────────────────────────────────────────────

let pendingAppName = '';
let pendingEnabled = false;
let pendingToggleEl = null;

async function loadMonitoringApps() {
    const [appsRes, monRes] = await Promise.all([
        fetch('/api/apps'),
        fetch('/api/monitoring/apps'),
    ]);

    const { apps } = appsRes.ok ? await appsRes.json() : { apps: [] };
    const { apps: monApps } = monRes.ok ? await monRes.json() : { apps: [] };

    const monMap = {};
    (monApps || []).forEach(a => { monMap[a.app_name] = a.enabled; });

    const list = document.getElementById('app-monitoring-list');
    if (!apps || apps.length === 0) {
        list.innerHTML = '<div class="no-data">No deployed applications found.</div>';
        return;
    }

    list.innerHTML = '';
    apps.forEach(app => {
        const enabled = !!monMap[app.app_name];
        const row = document.createElement('div');
        row.className = 'app-toggle-row';
        row.innerHTML = `
            <div>
                <div style="font-weight:600;font-size:0.9rem">${app.app_name}</div>
                <div style="font-size:0.75rem;color:var(--text-secondary,#64748b);margin-top:0.2rem">${app.framework || 'Docker'} · port ${app.port || '—'}</div>
            </div>
            <label class="toggle-switch" title="${enabled ? 'Disable monitoring' : 'Enable monitoring'}">
                <input type="checkbox" ${enabled ? 'checked' : ''} data-app="${app.app_name}">
                <span class="toggle-slider"></span>
            </label>
        `;
        const checkbox = row.querySelector('input[type=checkbox]');
        checkbox.addEventListener('change', e => {
            e.preventDefault();
            pendingAppName = app.app_name;
            pendingEnabled = e.target.checked;
            pendingToggleEl = e.target;
            e.target.checked = !e.target.checked; // revert until confirmed

            if (pendingEnabled) {
                document.getElementById('modal-app-name').textContent = app.app_name;
                document.getElementById('enable-modal').classList.add('open');
            } else {
                confirmToggle(false);
            }
        });
        list.appendChild(row);
    });
}

async function confirmToggle(enabled) {
    if (!pendingAppName) return;
    await fetch('/api/monitoring/app/toggle', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ appName: pendingAppName, enabled }),
    });
    if (pendingToggleEl) pendingToggleEl.checked = enabled;
    pendingAppName = '';
    pendingToggleEl = null;
    document.getElementById('enable-modal').classList.remove('open');
}

document.getElementById('modal-confirm').addEventListener('click', () => confirmToggle(true));
document.getElementById('modal-cancel').addEventListener('click', () => {
    pendingAppName = '';
    pendingToggleEl = null;
    document.getElementById('enable-modal').classList.remove('open');
});

// ── Init ─────────────────────────────────────────────────────────────────────

initSystemCharts();
loadSystemMetrics('1h');
loadContainers();
loadMonitoringApps();

// Auto-refresh every 30 seconds to match collection interval
setInterval(() => {
    loadSystemMetrics(currentRange);
    loadContainers();
}, 30000);
