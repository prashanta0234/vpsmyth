let appConfig = { apps: [] };

function renderApps(config) {
    const appsGrid = document.getElementById('apps-grid');
    const emptyState = document.getElementById('empty-state');

    if (!config.apps || config.apps.length === 0) {
        appsGrid.style.display = 'none';
        emptyState.style.display = 'flex';
        document.getElementById('active-apps-count').textContent = '0';
        return;
    }

    appsGrid.style.display = 'grid';
    emptyState.style.display = 'none';
    appsGrid.innerHTML = '';

    config.apps.forEach(app => {
        const card = document.createElement('div');
        card.className = 'app-card';
        card.innerHTML = `
            <div class="app-header">
                <span class="app-name">${app.app_name}</span>
                <span class="status-badge status-running">${app.status}</span>
            </div>
            <div class="app-details">
                <p>Port: ${app.port}</p>
                <p>Container: ${app.container_id}</p>
            </div>
            <div class="app-actions">
                <a href="apps.html?app=${app.app_name}" class="btn-outline" style="text-align: center; text-decoration: none; display: flex; align-items: center; justify-content: center;">Settings</a>
            </div>
        `;
        appsGrid.appendChild(card);
    });

    document.getElementById('active-apps-count').textContent = config.apps.length;
}

document.addEventListener('DOMContentLoaded', () => {
    const deployModal = document.getElementById('deploy-modal');
    const openModalBtn = document.getElementById('open-deploy-modal');
    const emptyDeployBtn = document.getElementById('empty-deploy-btn');
    const closeModalBtn = document.getElementById('close-modal');
    const cancelBtn = document.getElementById('cancel-deploy');
    const deployForm = document.getElementById('deploy-form');

    const toggleModal = (show) => {
        deployModal.style.display = show ? 'flex' : 'none';
    };

    openModalBtn.addEventListener('click', () => toggleModal(true));
    emptyDeployBtn.addEventListener('click', () => toggleModal(true));
    closeModalBtn.addEventListener('click', () => toggleModal(false));
    cancelBtn.addEventListener('click', () => toggleModal(false));

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

    const fetchApps = async () => {
        try {
            const response = await fetch('/apps');
            if (response.ok) {
                appConfig = await response.json();
                renderApps(appConfig);
            }
        } catch (err) {
            console.error('Failed to fetch apps:', err);
        }
    };

    const fetchStats = async () => {
        try {
            const response = await fetch('/stats');
            if (response.ok) {
                const stats = await response.json();
                const statValues = document.querySelectorAll('.stat-value');
                if (statValues.length >= 4) {
                    statValues[0].textContent = stats.cpuUsage;
                    statValues[1].textContent = stats.memory;
                    statValues[2].textContent = stats.activeApps;
                    statValues[3].textContent = stats.uptime;
                }
            }
        } catch (err) {
            console.error('Failed to fetch stats:', err);
        }
    };

    deployForm.addEventListener('submit', async (e) => {
        e.preventDefault();

        const formData = new FormData(deployForm);
        const btnText = document.getElementById('btn-text');
        const btnSpinner = document.getElementById('btn-spinner');
        const submitBtn = document.getElementById('submit-deploy');

        let env = {};
        try {
            const envText = formData.get('env').trim();
            if (envText) {
                env = JSON.parse(envText);
            }
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

            const result = await response.json();

            if (response.ok) {
                alert('Deployment started successfully!');
                toggleModal(false);
                deployForm.reset();
                frameworkSelect.innerHTML = '<option value="" disabled selected>Select Framework</option>';
                fetchApps();
            } else {
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

    // Sidebar Navigation Handling
    const navItems = document.querySelectorAll('.nav-item');
    navItems.forEach(item => {
        item.addEventListener('click', (e) => {
            const href = item.getAttribute('href');
            if (href.startsWith('#')) {
                e.preventDefault();
                const targetId = href.substring(1);
                const targetElement = document.getElementById(targetId);
                if (targetElement) {
                    targetElement.scrollIntoView({ behavior: 'smooth' });

                    // Update active state
                    navItems.forEach(nav => nav.classList.remove('active'));
                    item.classList.add('active');
                }
            }
        });
    });

    // Initial load
    fetchApps();
    fetchStats();
    // Periodic refresh
    setInterval(fetchStats, 5000);
});
