function renderApps(config) {
    const appsList = document.getElementById('apps-list');
    const emptyState = document.getElementById('empty-state');

    if (!config.apps || config.apps.length === 0) {
        appsList.style.display = 'none';
        emptyState.style.display = 'flex';
        return;
    }

    appsList.style.display = 'grid';
    emptyState.style.display = 'none';
    appsList.innerHTML = '';

    config.apps.forEach(app => {
        const card = document.createElement('div');
        card.className = 'app-card';
        card.id = `app-card-${app.app_name}`;
        card.innerHTML = `
            <div class="app-header">
                <span class="app-name">${app.app_name}</span>
                <span class="status-badge status-running">${app.status}</span>
            </div>
            <div class="app-details">
                <p><strong>Port:</strong> ${app.port}</p>
                <p><strong>Container ID:</strong> ${app.container_id}</p>
            </div>
            <div class="app-actions" style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.5rem;">
                <button class="btn-outline" onclick="handleAction('${app.app_name}', 'restart')">Restart</button>
                <button class="btn-outline" onclick="handleAction('${app.app_name}', 'stop')">Stop</button>
                <button class="btn-outline" onclick="showLogs('${app.app_name}')">Logs</button>
                <button class="btn-outline" onclick="showEditEnv('${app.app_name}', ${JSON.stringify(app.env).replace(/"/g, '&quot;')})">Edit Envs</button>
                <button class="btn-outline" style="color: #ef4444; border-color: #fca5a5; grid-column: span 2;" onclick="handleAction('${app.app_name}', 'delete')">Delete</button>
            </div>
        `;
        appsList.appendChild(card);
    });

    // Check for app highlight in URL
    const urlParams = new URLSearchParams(window.location.search);
    const highlightApp = urlParams.get('app');
    if (highlightApp) {
        const targetCard = document.getElementById(`app-card-${highlightApp}`);
        if (targetCard) {
            targetCard.style.borderColor = 'var(--accent-primary)';
            targetCard.style.boxShadow = '0 0 0 2px var(--accent-primary)';
            targetCard.scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
    }
}

async function handleAction(appName, action) {
    if (action === 'delete' && !confirm(`Are you sure you want to delete ${appName}?`)) return;

    try {
        const response = await fetch(`/apps/${action}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ appName })
        });

        if (response.ok) {
            alert(`App ${action}ed successfully!`);
            fetchApps();
        } else {
            const result = await response.json();
            alert(`Failed to ${action} app: ${result.error || 'Unknown error'}`);
        }
    } catch (err) {
        alert(`Error: ${err.message}`);
    }
}

async function showLogs(appName) {
    const modal = document.getElementById('logs-modal');
    const content = document.getElementById('logs-content');
    modal.style.display = 'flex';
    content.textContent = 'Loading logs...';

    try {
        const response = await fetch(`/apps/logs?appName=${appName}`);
        if (response.ok) {
            const result = await response.json();
            content.textContent = result.logs || 'No logs found.';
        } else {
            content.textContent = 'Failed to load logs.';
        }
    } catch (err) {
        content.textContent = `Error: ${err.message}`;
    }
}

function showEditEnv(appName, env) {
    const modal = document.getElementById('edit-env-modal');
    const appNameInput = document.getElementById('edit-env-app-name');
    const envJsonInput = document.getElementById('edit-env-json');

    appNameInput.value = appName;
    envJsonInput.value = JSON.stringify(env || {}, null, 2);
    modal.style.display = 'flex';
}

async function fetchApps() {
    try {
        const response = await fetch('/apps');
        if (response.ok) {
            const data = await response.json();
            renderApps(data);
        }
    } catch (err) {
        console.error('Failed to fetch apps:', err);
    }
}

document.addEventListener('DOMContentLoaded', () => {
    fetchApps();

    // Shared Modal Logic
    const deployModal = document.getElementById('deploy-modal');
    const openModalBtn = document.getElementById('open-deploy-modal');
    const closeModalBtn = document.getElementById('close-deploy-modal');
    const cancelBtn = document.getElementById('cancel-deploy');
    const deployForm = document.getElementById('deploy-form');

    const toggleModal = (show) => {
        deployModal.style.display = show ? 'flex' : 'none';
    };

    openModalBtn.addEventListener('click', () => toggleModal(true));
    closeModalBtn.addEventListener('click', () => toggleModal(false));
    cancelBtn.addEventListener('click', () => toggleModal(false));

    // Logs Modal Close
    document.getElementById('close-logs-modal').addEventListener('click', () => {
        document.getElementById('logs-modal').style.display = 'none';
    });

    // Edit Env Modal Close
    const editEnvModal = document.getElementById('edit-env-modal');
    document.getElementById('close-edit-env-modal').addEventListener('click', () => {
        editEnvModal.style.display = 'none';
    });
    document.getElementById('cancel-edit-env').addEventListener('click', () => {
        editEnvModal.style.display = 'none';
    });

    // Edit Env Form Submission
    const editEnvForm = document.getElementById('edit-env-form');
    editEnvForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(editEnvForm);
        const appName = formData.get('appName');
        let env = {};

        try {
            env = JSON.parse(formData.get('env'));
        } catch (err) {
            alert('Invalid JSON in environment variables');
            return;
        }

        const btnText = document.getElementById('edit-btn-text');
        const btnSpinner = document.getElementById('edit-btn-spinner');
        const submitBtn = document.getElementById('submit-edit-env');

        submitBtn.disabled = true;
        btnText.style.display = 'none';
        btnSpinner.style.display = 'block';

        try {
            const response = await fetch('/apps/update-env', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ appName, env })
            });

            if (response.ok) {
                alert('Environment variables updated and app restarted!');
                editEnvModal.style.display = 'none';
                fetchApps();
            } else {
                const result = await response.json();
                alert('Failed to update envs: ' + (result.error || 'Unknown error'));
            }
        } catch (err) {
            alert('Error: ' + err.message);
        } finally {
            submitBtn.disabled = false;
            btnText.style.display = 'block';
            btnSpinner.style.display = 'none';
        }
    });

    // Framework Selection Logic (Shared)
    const categorySelect = document.getElementById('category');
    const frameworkSelect = document.getElementById('framework');
    const frameworks = {
        frontend: [
            { value: 'nextjs', label: 'Next.js' },
            { value: 'react', label: 'React (Vite/CRA)' },
            { value: 'html', label: 'Static HTML' }
        ],
        backend: [
            { value: 'nestjs', label: 'NestJS' },
            { value: 'express', label: 'Express' },
            { value: 'nodejs', label: 'Generic Node.js' }
        ]
    };

    categorySelect.addEventListener('change', () => {
        const category = categorySelect.value;
        frameworkSelect.innerHTML = '<option value="" disabled selected>Select Framework</option>';
        if (frameworks[category]) {
            frameworks[category].forEach(fw => {
                const option = document.createElement('option');
                option.value = fw.value;
                option.textContent = fw.label;
                frameworkSelect.appendChild(option);
            });
        }
    });

    deployForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(deployForm);
        const btnText = document.getElementById('btn-text');
        const btnSpinner = document.getElementById('btn-spinner');
        const submitBtn = document.getElementById('submit-deploy');

        let env = {};
        try {
            const envText = formData.get('env').trim();
            if (envText) env = JSON.parse(envText);
        } catch (err) {
            alert('Invalid JSON in environment variables');
            return;
        }

        const data = {
            appName: formData.get('appName'),
            category: formData.get('category'),
            framework: formData.get('framework'),
            repoURL: formData.get('repoURL'),
            port: parseInt(formData.get('port')),
            env: env
        };

        submitBtn.disabled = true;
        btnText.style.display = 'none';
        btnSpinner.style.display = 'block';

        try {
            const response = await fetch('/apps/deploy', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            });

            if (response.ok) {
                alert('Deployment started successfully!');
                toggleModal(false);
                deployForm.reset();
                fetchApps();
            } else {
                const result = await response.json();
                alert('Deployment failed: ' + (result.error || 'Unknown error'));
            }
        } catch (err) {
            alert('Error connecting to server: ' + err.message);
        } finally {
            submitBtn.disabled = false;
            btnText.style.display = 'block';
            btnSpinner.style.display = 'none';
        }
    });
});
