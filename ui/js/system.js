document.addEventListener('DOMContentLoaded', () => {
    const installNodeBtn = document.getElementById('install-node-btn');
    const installText = document.getElementById('install-text');
    const installSpinner = document.getElementById('install-spinner');
    const nodeVersionDisplay = document.getElementById('node-version');

    const installDockerBtn = document.getElementById('install-docker-btn');
    const installDockerText = document.getElementById('install-docker-text');
    const installDockerSpinner = document.getElementById('install-docker-spinner');
    const dockerStatusDisplay = document.getElementById('docker-status');

    const installGoBtn = document.getElementById('install-go-btn');
    const installGoText = document.getElementById('install-go-text');
    const installGoSpinner = document.getElementById('install-go-spinner');
    const goStatusDisplay = document.getElementById('go-status');

    const checkVersions = async () => {
        try {
            const response = await fetch('/api/system/status');
            if (response.status === 401) return handleAuthError();
            if (response.ok) {
                const status = await response.json();

                updateToolUI('Node.js', status.node, installNodeBtn, installText, nodeVersionDisplay);
                updateToolUI('Docker', status.docker, installDockerBtn, installDockerText, dockerStatusDisplay);
                updateToolUI('Go', status.go, installGoBtn, installGoText, goStatusDisplay);
            }
        } catch (err) {
            console.error('Failed to fetch system status:', err);
        }
    };

    const updateToolUI = (name, status, btn, btnText, versionDisplay) => {
        if (status.installed) {
            versionDisplay.textContent = `v${status.version}`;
            btn.disabled = true;
            btn.classList.add('btn-outline');
            btn.classList.remove('btn-primary');
            btnText.textContent = 'Already Installed';
        } else {
            versionDisplay.textContent = 'Not Installed';
            btn.disabled = false;
            btn.classList.add('btn-primary');
            btn.classList.remove('btn-outline');
            btnText.textContent = `Install ${name}`;
        }
    };

    const handleInstall = async (tool, btn, text, spinner, endpoint) => {
        if (!confirm(`This will install ${tool} on your host VPS. Continue?`)) return;

        btn.disabled = true;
        text.style.display = 'none';
        spinner.style.display = 'block';

        try {
            const response = await fetch('/api' + endpoint, { method: 'POST' });
            if (response.status === 401) return handleAuthError();
            if (response.ok) {
                alert(`${tool} installed successfully on host!`);
                checkVersions();
            } else {
                const result = await response.json();
                alert('Installation failed: ' + (result.error || 'Unknown error'));
            }
        } catch (err) {
            alert('Error connecting to server: ' + err.message);
        } finally {
            btn.disabled = false;
            text.style.display = 'block';
            spinner.style.display = 'none';
        }
    };

    installNodeBtn.addEventListener('click', () => handleInstall('Node.js', installNodeBtn, installText, installSpinner, '/system/install-node'));
    installDockerBtn.addEventListener('click', () => handleInstall('Docker', installDockerBtn, installDockerText, installDockerSpinner, '/system/install-docker'));
    installGoBtn.addEventListener('click', () => handleInstall('Go', installGoBtn, installGoText, installGoSpinner, '/system/install-go'));

    checkVersions();
});
