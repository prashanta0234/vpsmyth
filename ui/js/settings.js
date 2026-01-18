document.addEventListener('DOMContentLoaded', () => {
    const dockerUser = document.getElementById('docker-username');
    const dockerPass = document.getElementById('docker-password');
    const saveDockerBtn = document.getElementById('save-dockerhub');
    const githubToken = document.getElementById('github-token');
    const githubStatus = document.getElementById('github-status');
    const saveGithubBtn = document.getElementById('save-github');
    const secretsList = document.getElementById('secrets-list');
    const secretKey = document.getElementById('secret-key');
    const secretValue = document.getElementById('secret-value');
    const addSecretBtn = document.getElementById('add-secret');

    const fetchSettings = async () => {
        // Fetch DockerHub username
        try {
            const response = await fetch('/api/system/settings/dockerhub');
            if (response.status === 401) return handleAuthError();
            if (response.ok) {
                const data = await response.json();
                dockerUser.value = data.username || '';
            }
        } catch (err) { console.error(err); }

        // Fetch GitHub status
        try {
            const res = await fetch('/api/system/settings/github');
            if (res.status === 401) return handleAuthError();
            if (res.ok) {
                const data = await res.json();
                if (data.hasToken) {
                    githubStatus.innerHTML = '<span style="color: #4ade80;">✓ Token is configured</span>';
                } else {
                    githubStatus.innerHTML = '<span style="color: #f87171;">✗ No token configured</span>';
                }
            }
        } catch (err) { console.error(err); }

        // Fetch Secrets
        try {
            const res = await fetch('/api/system/settings/secrets');
            if (res.status === 401) return handleAuthError();
            if (res.ok) {
                const secrets = await res.json();
                renderSecrets(secrets);
            }
        } catch (err) { console.error(err); }
    };

    const renderSecrets = (secrets) => {
        secretsList.innerHTML = '';
        Object.entries(secrets).forEach(([key, value]) => {
            const item = document.createElement('div');
            item.className = 'secret-item';
            item.style = 'display: flex; justify-content: space-between; align-items: center; background: var(--bg-secondary); padding: 0.5rem 1rem; border-radius: 8px; margin-bottom: 0.5rem; font-size: 0.875rem;';
            item.innerHTML = `
                <div>
                    <span style="font-weight: 600; color: var(--text-primary);">${key}</span>
                    <span style="color: var(--text-secondary); margin-left: 0.5rem;">••••••••</span>
                </div>
                <button onclick="deleteSecret('${key}')" style="color: #ef4444; background: none; border: none; cursor: pointer;">
                    <svg width="18" height="18" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
                </button>
            `;
            secretsList.appendChild(item);
        });
    };

    saveDockerBtn.addEventListener('click', async () => {
        saveDockerBtn.disabled = true;
        saveDockerBtn.textContent = 'Logging in...';
        try {
            const res = await fetch('/api/system/settings/dockerhub', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username: dockerUser.value, password: dockerPass.value })
            });
            if (res.status === 401) return handleAuthError();
            if (res.ok) {
                alert('DockerHub credentials saved and logged in successfully!');
            } else {
                const err = await res.text();
                alert('Failed to save credentials: ' + err);
            }
        } catch (err) { alert(err.message); }
        saveDockerBtn.disabled = false;
        saveDockerBtn.textContent = 'Save & Login';
    });

    saveGithubBtn.addEventListener('click', async () => {
        if (!githubToken.value) return;
        saveGithubBtn.disabled = true;
        saveGithubBtn.textContent = 'Saving...';
        try {
            const res = await fetch('/api/system/settings/github', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ token: githubToken.value })
            });
            if (res.status === 401) return handleAuthError();
            if (res.ok) {
                alert('GitHub token saved successfully!');
                githubToken.value = '';
                fetchSettings();
            } else {
                alert('Failed to save token');
            }
        } catch (err) { alert(err.message); }
        saveGithubBtn.disabled = false;
        saveGithubBtn.textContent = 'Save Token';
    });

    addSecretBtn.addEventListener('click', async () => {
        if (!secretKey.value || !secretValue.value) return;
        try {
            const res = await fetch('/api/system/settings/secrets', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ action: 'save', key: secretKey.value, value: secretValue.value })
            });
            if (res.status === 401) return handleAuthError();
            if (res.ok) {
                secretKey.value = '';
                secretValue.value = '';
                fetchSettings();
            }
        } catch (err) { alert(err.message); }
    });

    window.deleteSecret = async (key) => {
        if (!confirm(`Delete secret ${key}?`)) return;
        try {
            const res = await fetch('/api/system/settings/secrets', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ action: 'delete', key })
            });
            if (res.status === 401) return handleAuthError();
            if (res.ok) fetchSettings();
        } catch (err) { alert(err.message); }
    };

    fetchSettings();
});
