// Dummy configuration data
const dummyConfig = {
    apps: [
        {
            name: "my-node-app",
            port: 3000,
            status: "running",
            runtime: "Node.js"
        },
        {
            name: "my-go-app",
            port: 8080,
            status: "running",
            runtime: "Go"
        }
    ]
};

/**
 * Renders the list of applications on the dashboard.
 * @param {Object} config - The configuration object containing apps.
 */
function renderApps(config) {
    const appsGrid = document.getElementById('apps-grid');
    const emptyState = document.getElementById('empty-state');

    if (!config.apps || config.apps.length === 0) {
        appsGrid.style.display = 'none';
        emptyState.style.display = 'flex';
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
                <span class="app-name">${app.name}</span>
                <span class="status-badge status-running">${app.status}</span>
            </div>
            <div class="app-details">
                <p>Runtime: ${app.runtime}</p>
                <p>Port: ${app.port}</p>
            </div>
            <div class="app-actions">
                <button class="btn-outline">Logs</button>
                <button class="btn-outline">Settings</button>
            </div>
        `;
        appsGrid.appendChild(card);
    });

    // Update stats
    document.getElementById('active-apps-count').textContent = config.apps.length;
}

// Initialize dashboard
document.addEventListener('DOMContentLoaded', () => {
    renderApps(dummyConfig);
});
