document.addEventListener('DOMContentLoaded', () => {
    const containersList = document.getElementById('containers-list');
    const emptyState = document.getElementById('empty-state');
    const refreshBtn = document.getElementById('refresh-containers');
    const searchInput = document.getElementById('container-search');

    let allContainers = [];

    const fetchContainers = async () => {
        try {
            const response = await fetch('/system/containers');
            if (response.ok) {
                const data = await response.json();
                allContainers = data.containers || [];
                renderContainers(allContainers);
            }
        } catch (err) {
            console.error('Failed to fetch containers:', err);
        }
    };

    const renderContainers = (containers) => {
        if (containers.length === 0) {
            containersList.style.display = 'none';
            emptyState.style.display = 'flex';
            return;
        }

        containersList.style.display = 'grid';
        emptyState.style.display = 'none';
        containersList.innerHTML = '';

        containers.forEach(container => {
            const card = document.createElement('div');
            card.className = 'app-card';
            const statusClass = container.running ? 'status-running' : 'status-stopped';

            card.innerHTML = `
                <div class="app-header">
                    <span class="app-name" title="${container.name}">${container.name}</span>
                    <span class="status-badge ${statusClass}">${container.status}</span>
                </div>
                <div class="app-details">
                    <p><strong>Image:</strong> ${container.image}</p>
                    <p><strong>ID:</strong> ${container.id.substring(0, 12)}</p>
                    <p><strong>Ports:</strong> ${container.ports || 'None'}</p>
                </div>
                <div class="app-actions" style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.5rem;">
                    ${container.running ?
                    `<button class="btn-outline" onclick="handleContainerAction('${container.id}', 'stop')">Stop</button>` :
                    `<button class="btn-outline" onclick="handleContainerAction('${container.id}', 'start')">Start</button>`
                }
                    <button class="btn-outline" onclick="handleContainerAction('${container.id}', 'restart')">Restart</button>
                    <button class="btn-outline" onclick="showContainerLogs('${container.id}')">Logs</button>
                    <button class="btn-outline" style="color: #ef4444; border-color: #fca5a5;" onclick="handleContainerAction('${container.id}', 'delete')">Delete</button>
                </div>
            `;
            containersList.appendChild(card);
        });
    };

    window.handleContainerAction = async (id, action) => {
        if (action === 'delete' && !confirm(`Are you sure you want to delete container ${id}? This action is irreversible.`)) return;

        try {
            const response = await fetch(`/system/containers/${action}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ id })
            });

            if (response.ok) {
                fetchContainers();
            } else {
                const result = await response.json();
                alert(`Failed to ${action} container: ${result.error || 'Unknown error'}`);
            }
        } catch (err) {
            alert(`Error: ${err.message}`);
        }
    };

    window.showContainerLogs = async (id) => {
        const modal = document.getElementById('logs-modal');
        const content = document.getElementById('logs-content');
        modal.style.display = 'flex';
        content.textContent = 'Loading logs...';

        try {
            const response = await fetch(`/system/containers/logs?id=${id}`);
            if (response.ok) {
                const result = await response.json();
                content.textContent = result.logs || 'No logs found.';
            } else {
                content.textContent = 'Failed to load logs.';
            }
        } catch (err) {
            content.textContent = `Error: ${err.message}`;
        }
    };

    // Close Modal
    document.getElementById('close-logs-modal').addEventListener('click', () => {
        document.getElementById('logs-modal').style.display = 'none';
    });

    // Search
    searchInput.addEventListener('input', (e) => {
        const term = e.target.value.toLowerCase();
        const filtered = allContainers.filter(c =>
            c.name.toLowerCase().includes(term) ||
            c.image.toLowerCase().includes(term) ||
            c.id.toLowerCase().includes(term)
        );
        renderContainers(filtered);
    });

    refreshBtn.addEventListener('click', fetchContainers);

    // Initial load
    fetchContainers();
});
