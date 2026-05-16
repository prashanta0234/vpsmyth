const themeToggle = document.getElementById('theme-toggle');
const body = document.body;

// Check for saved theme preference
const savedTheme = localStorage.getItem('theme');
if (savedTheme === 'dark') {
    body.classList.add('dark');
}

themeToggle.addEventListener('click', () => {
    body.classList.toggle('dark');
    const currentTheme = body.classList.contains('dark') ? 'dark' : 'light';
    localStorage.setItem('theme', currentTheme);
});

function getLoginURL() {
    const parts = window.location.pathname.split('/');
    parts[parts.length - 1] = 'login.html';
    return parts.join('/');
}

// Logout logic
const logoutBtn = document.getElementById('logout-btn');
if (logoutBtn) {
    logoutBtn.addEventListener('click', async (e) => {
        e.preventDefault();
        try {
            const response = await fetch('/api/auth/logout', { method: 'POST' });
            if (response.ok) {
                window.location.href = getLoginURL();
            }
        } catch (err) {
            console.error('Logout failed:', err);
        }
    });
}

function handleAuthError() {
    window.location.href = getLoginURL();
}
