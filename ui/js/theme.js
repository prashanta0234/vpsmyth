/* Theme + session helpers.
   The theme class is applied to <html> by the inline snippet in each page's
   <head> so there is no flash of the wrong palette; this file keeps <body> in
   sync for legacy selectors and owns the toggle and logout wiring. */

(function () {
    'use strict';

    function apply(theme) {
        var dark = theme === 'dark';
        document.documentElement.classList.toggle('dark', dark);
        if (document.body) document.body.classList.toggle('dark', dark);
        var meta = document.querySelector('meta[name="theme-color"]');
        if (meta) meta.setAttribute('content', dark ? '#080a12' : '#f7f8fa');
    }

    function stored() {
        try {
            return localStorage.getItem('theme');
        } catch (err) {
            return null;
        }
    }

    var media = window.matchMedia('(prefers-color-scheme: dark)');
    apply(stored() || (media.matches ? 'dark' : 'light'));

    // Follow the OS until the user makes an explicit choice.
    var onSystemChange = function (event) {
        if (!stored()) apply(event.matches ? 'dark' : 'light');
    };
    if (media.addEventListener) media.addEventListener('change', onSystemChange);
    else if (media.addListener) media.addListener(onSystemChange);

    var themeToggle = document.getElementById('theme-toggle');
    if (themeToggle) {
        themeToggle.addEventListener('click', function () {
            var next = document.documentElement.classList.contains('dark') ? 'light' : 'dark';
            apply(next);
            try {
                localStorage.setItem('theme', next);
            } catch (err) {
                /* storage disabled — the choice just won't persist */
            }
        });
    }
})();

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
