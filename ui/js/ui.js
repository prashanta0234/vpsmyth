/* ===========================================================================
   VPSMyth — application shell
   ---------------------------------------------------------------------------
   Renders the sidebar and top bar for every page from one definition, so the
   navigation can never drift between screens, and owns the mobile drawer,
   modal dismissal and toast behaviour.

   Load this BEFORE theme.js and any page script: it creates #theme-toggle and
   #logout-btn, which those scripts bind to.
   =========================================================================== */
(function () {
    'use strict';

    var icon = {
        dashboard: '<path d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"/>',
        apps: '<path d="M13 10V3L4 14h7v7l9-11h-7z"/>',
        database: '<path d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"/>',
        containers: '<path d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/>',
        monitoring: '<path d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>',
        cron: '<path d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>',
        firewall: '<path d="M17.657 18.657A8 8 0 016.343 7.343S7 9 9 10c0-2 .5-5 2.986-7C14 5 16.09 5.777 17.656 7.343A7.975 7.975 0 0120 13a7.975 7.975 0 01-2.343 5.657z"/><path d="M9.879 16.121A3 3 0 1012.015 11L11 14H9c0 .768.293 1.536.879 2.121z"/>',
        security: '<path d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/>',
        system: '<path d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"/>',
        settings: '<path d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><path d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>',
        logout: '<path d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"/>',
        menu: '<path d="M4 7h16M4 12h16M4 17h16"/>',
        close: '<path d="M6 18L18 6M6 6l12 12"/>',
        search: '<path d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>',
        sun: '<path d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364-6.364l-.707.707M6.343 17.657l-.707.707m12.728 0l-.707-.707M6.343 6.364l-.707-.707M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>',
        moon: '<path d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"/>'
    };

    /* Grouped so related tools read as one job rather than a flat list of ten. */
    var NAV = [
        { section: 'Overview', items: [{ href: 'index.html', label: 'Dashboard', icon: 'dashboard' }] },
        {
            section: 'Deploy',
            items: [
                { href: 'apps.html', label: 'Applications', icon: 'apps' },
                { href: 'databases.html', label: 'Databases', icon: 'database' },
                { href: 'containers.html', label: 'Docker Containers', icon: 'containers' }
            ]
        },
        {
            section: 'Operate',
            items: [
                { href: 'monitoring.html', label: 'Monitoring', icon: 'monitoring' },
                { href: 'cron.html', label: 'Cron Jobs', icon: 'cron' }
            ]
        },
        {
            section: 'Protect',
            items: [
                { href: 'firewall.html', label: 'Firewall', icon: 'firewall' },
                { href: 'security.html', label: 'Security', icon: 'security' }
            ]
        },
        {
            section: 'System',
            items: [
                { href: 'system.html', label: 'System Tools', icon: 'system' },
                { href: 'settings.html', label: 'Settings', icon: 'settings' }
            ]
        }
    ];

    function svg(name, extra) {
        return (
            '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" ' +
            'stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"' +
            (extra ? ' class="' + extra + '"' : '') +
            '>' + icon[name] + '</svg>'
        );
    }

    function currentPage() {
        var file = window.location.pathname.split('/').pop();
        return file === '' ? 'index.html' : file;
    }

    /* ---------------------------------------------------------------------
       Sidebar
       --------------------------------------------------------------------- */
    function buildSidebar() {
        var here = currentPage();
        var html =
            '<div class="brand">' +
            '<img src="/assets/logo.png" alt="" class="brand-logo">' +
            '<span class="brand-name">VPSMyth</span>' +
            '<button class="sidebar-close" type="button" data-nav-close aria-label="Close navigation">' +
            svg('close') +
            '</button>' +
            '</div><nav aria-label="Main">';

        NAV.forEach(function (group) {
            html += '<p class="nav-section label">' + group.section + '</p>';
            group.items.forEach(function (item) {
                var active = item.href === here;
                html +=
                    '<a href="' + item.href + '" class="nav-item' + (active ? ' active' : '') + '"' +
                    (active ? ' aria-current="page"' : '') + '>' +
                    svg(item.icon) + '<span>' + item.label + '</span></a>';
            });
        });

        html +=
            '</nav>' +
            '<div class="sidebar-footer">' +
            '<a href="#" class="nav-item danger" id="logout-btn">' + svg('logout') + '<span>Logout</span></a>' +
            '<div class="agent-status"><span class="dot"></span>Agent connected</div>' +
            '<div class="agent-host">' + window.location.host + '</div>' +
            '</div>';

        var aside = document.createElement('aside');
        aside.id = 'app-sidebar';
        aside.innerHTML = html;
        return aside;
    }

    /* ---------------------------------------------------------------------
       Top bar
       --------------------------------------------------------------------- */
    function buildTopbar(host) {
        var title = host.getAttribute('data-title') || document.title.replace(/^VPSMyth\s*\|\s*/, '');
        var meta = host.getAttribute('data-meta') || '';
        var search = host.getAttribute('data-search');
        var searchId = host.getAttribute('data-search-id');
        var user = host.getAttribute('data-user') || 'Admin';
        var initials = user
            .split(/[\s-_.]+/)
            .slice(0, 2)
            .map(function (part) { return part.charAt(0).toUpperCase(); })
            .join('');

        var html =
            '<button class="nav-toggle" type="button" data-nav-open aria-label="Open navigation" aria-expanded="false" aria-controls="app-sidebar">' +
            svg('menu') +
            '</button>' +
            '<div class="topbar-title"><h1>' + title + '</h1>' +
            (meta ? '<p class="topbar-meta">' + meta + '</p>' : '') +
            '</div>';

        if (search !== null) {
            html +=
                '<div class="search-container">' + svg('search') +
                '<input type="search" placeholder="' + (search || 'Search…') + '"' +
                (searchId ? ' id="' + searchId + '"' : '') +
                ' aria-label="' + (search || 'Search') + '"></div>';
        }

        html +=
            '<div class="actions">' +
            '<button class="theme-toggle" id="theme-toggle" type="button" title="Toggle theme" aria-label="Toggle theme">' +
            svg('sun', 'sun-icon') + svg('moon', 'moon-icon') +
            '</button>' +
            '<div class="user-info"><span class="user-name">' + user + '</span>' +
            '<div class="user-avatar" aria-hidden="true">' + initials + '</div></div>' +
            '</div>';

        host.className = 'topbar';
        host.innerHTML = html;
    }

    /* ---------------------------------------------------------------------
       Mobile drawer
       --------------------------------------------------------------------- */
    var lastFocused = null;

    function openNav() {
        lastFocused = document.activeElement;
        document.body.classList.add('nav-open');
        setExpanded(true);
        var first = document.querySelector('#app-sidebar .nav-item');
        if (first) first.focus();
    }

    function closeNav() {
        if (!document.body.classList.contains('nav-open')) return;
        document.body.classList.remove('nav-open');
        setExpanded(false);
        if (lastFocused && lastFocused.focus) lastFocused.focus();
    }

    function setExpanded(state) {
        var toggles = document.querySelectorAll('[data-nav-open]');
        for (var i = 0; i < toggles.length; i++) {
            toggles[i].setAttribute('aria-expanded', String(state));
        }
    }

    /* ---------------------------------------------------------------------
       Modals — dismissal behaviour every dialog inherits
       --------------------------------------------------------------------- */
    function visibleOverlays() {
        var open = [];
        var overlays = document.querySelectorAll('.modal-overlay');
        for (var i = 0; i < overlays.length; i++) {
            var style = window.getComputedStyle(overlays[i]);
            if (style.display !== 'none' && style.visibility !== 'hidden') open.push(overlays[i]);
        }
        return open;
    }

    function closeOverlay(overlay) {
        overlay.classList.remove('open');
        if (overlay.style.display && overlay.style.display !== 'none') overlay.style.display = 'none';
    }

    /* ---------------------------------------------------------------------
       Toasts
       --------------------------------------------------------------------- */
    function toast(message, type) {
        var stack = document.querySelector('.toast-stack');
        if (!stack) {
            stack = document.createElement('div');
            stack.className = 'toast-stack';
            stack.setAttribute('role', 'status');
            stack.setAttribute('aria-live', 'polite');
            document.body.appendChild(stack);
        }
        var el = document.createElement('div');
        el.className = 'toast' + (type ? ' ' + type : '');
        el.appendChild(document.createTextNode(message));
        stack.appendChild(el);
        setTimeout(function () { el.remove(); }, 4200);
    }

    /* ---------------------------------------------------------------------
       Boot
       --------------------------------------------------------------------- */
    function mount() {
        document.body.classList.add('app');

        var slot = document.getElementById('sidebar-root');
        var sidebar = buildSidebar();
        if (slot) {
            slot.parentNode.replaceChild(sidebar, slot);
        } else {
            document.body.insertBefore(sidebar, document.body.firstChild);
        }

        var scrim = document.createElement('div');
        scrim.className = 'scrim';
        scrim.setAttribute('data-nav-close', '');
        scrim.setAttribute('aria-hidden', 'true');
        sidebar.insertAdjacentElement('afterend', scrim);

        var topbars = document.querySelectorAll('[data-topbar]');
        for (var i = 0; i < topbars.length; i++) buildTopbar(topbars[i]);

        /* Drawer */
        document.addEventListener('click', function (event) {
            if (event.target.closest('[data-nav-open]')) {
                openNav();
            } else if (event.target.closest('[data-nav-close]')) {
                closeNav();
            } else if (event.target.closest('#app-sidebar .nav-item')) {
                closeNav();
            } else if (event.target.classList.contains('modal-overlay')) {
                closeOverlay(event.target);
            }
        });

        document.addEventListener('keydown', function (event) {
            if (event.key !== 'Escape') return;
            var open = visibleOverlays();
            if (open.length) {
                closeOverlay(open[open.length - 1]);
            } else {
                closeNav();
            }
        });

        var wide = window.matchMedia('(min-width: 1024px)');
        var onWide = function (event) { if (event.matches) closeNav(); };
        if (wide.addEventListener) wide.addEventListener('change', onWide);
        else if (wide.addListener) wide.addListener(onWide);
    }

    window.VPSMyth = { toast: toast, closeNav: closeNav, openNav: openNav };

    if (document.body) mount();
    else document.addEventListener('DOMContentLoaded', mount);
})();
