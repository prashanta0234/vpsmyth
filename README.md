# VPSMyth

VPSMyth is a lightweight, all-in-one VPS management platform. It allows you to deploy, monitor, and manage multiple applications with one click, without needing deep DevOps knowledge. Think of it as your personal VPS control plane that is simple, secure, and easy to use.

## MVP Features

* One-click app deployment for Node.js and Go applications
* Deploy from GitHub repo or ZIP upload
* Automatic port allocation
* Environment management per app
* System and app monitoring (CPU, RAM, Disk, uptime)
* Resource management (CPU and memory limits per app)
* Cron job management with logging
* Database management (SQLite, Redis, MySQL, MongoDB, PostgreSQL)
* Port management
* Firewall setup and rule management
* IP whitelisting and access control
* One-click SSL and domain setup
* Lightweight and VPS-friendly (Go backend, minimal memory usage, Docker required)

## Installation

Run this command to install:

```bash
curl -fsSL https://raw.githubusercontent.com/prashanta0234/vpsmyth/main/scripts/install.sh | sudo bash
```

Or if you want to inspect the script first:

```bash
curl -fsSL https://raw.githubusercontent.com/prashanta0234/vpsmyth/main/scripts/install.sh -o install.sh
less install.sh
sudo bash install.sh
```

During installation, you will be prompted to choose a custom dashboard path:

```
========================================
   Welcome to VPSMyth Installer
========================================

Enter dashboard path [default: admin]: myadminPanel

  Dashboard will be available at: http://YOUR_SERVER_IP/myadminPanel
```

Press Enter to use `admin` as the default, or type any path you want (letters, numbers, hyphens, underscores). After installation completes:

```
========================================
   VPSMyth installation complete!
========================================

  Dashboard: http://203.0.113.5/myadminPanel

  Panel path saved to: /opt/vpsmyth/panel-path
  To check it later: cat /opt/vpsmyth/panel-path
```

The backend runs internally on port **2026** and is never exposed publicly. Nginx handles all traffic on port 80 (and 443 with SSL) and proxies to the backend.

## Updating

Run this command on your server to pull the latest version and apply it:

```bash
curl -fsSL https://raw.githubusercontent.com/prashanta0234/vpsmyth/main/scripts/update.sh | sudo bash
```

Or inspect first:

```bash
curl -fsSL https://raw.githubusercontent.com/prashanta0234/vpsmyth/main/scripts/update.sh -o update.sh
less update.sh
sudo bash update.sh
```

The update script:

* Pulls the latest code from GitHub
* If already on the latest commit, exits early with no changes
* Builds the new binary
* Stops the service, swaps the binary and UI files, restarts
* **Preserves** your database, credentials, panel path, and Nginx config
* Automatically rolls back to the previous binary if the service fails to start after the update

A backup of the previous binary is kept at `/opt/vpsmyth/vpsmyth.bak`. To roll back manually:

```bash
sudo systemctl stop vpsmyth
sudo cp /opt/vpsmyth/vpsmyth.bak /opt/vpsmyth/vpsmyth
sudo systemctl start vpsmyth
```

## Uninstallation

Run this command to uninstall:

```bash
curl -fsSL https://raw.githubusercontent.com/prashanta0234/vpsmyth/main/scripts/uninstall.sh | sudo bash
```

Or inspect first:

```bash
curl -fsSL https://raw.githubusercontent.com/prashanta0234/vpsmyth/main/scripts/uninstall.sh -o uninstall.sh
less uninstall.sh
sudo bash uninstall.sh
```

The uninstaller will ask you a series of questions before removing anything:

```
Are you sure you want to uninstall VPSMyth? [y/N]:
Remove all Docker containers deployed by VPSMyth (your apps)? [y/N]:
Remove Docker from this server? [y/N]:
Remove Node.js from this server? [y/N]:
```

**What is always removed** (no prompt):

* VPSMyth systemd service
* VPSMyth binary and files (`/opt/vpsmyth`)
* All Nginx configuration added by VPSMyth

**What you decide:**

| Prompt | What it removes |
|--------|----------------|
| App containers | All Docker containers and images created by VPSMyth |
| Docker | Docker engine and all its data (`/var/lib/docker`) |
| Node.js | Node.js and the NodeSource apt repository |

At the end, the uninstaller prints a summary of what was kept and what was removed.

## Directory Structure

```
vpsmyth/
├── cmd/           # Backend entry points and server setup
│   └── server/    # HTTP server and API documentation
├── internal/      # Core logic (deploy, monitor, cron, config, firewall, db)
├── ui/            # Dashboard UI
├── scripts/       # Installation and helper scripts
├── tests/         # Backend tests
├── .github/       # GitHub workflows and PR templates
├── README.md
└── LICENSE
```

Full explanation of modules is documented in the component-specific markdown files under `cmd/`, `internal/`, `ui/`, `scripts/`, and `tests/`.

---

## Database Management

VPSMyth manages all databases as isolated Docker containers with security hardening applied automatically at launch.

### SQLite

Built-in, file-based database. No container required. Used for internal VPSMyth state.

### Redis

Launched as a Docker container with a randomly generated password. Bound to `127.0.0.1` only (not exposed to the public internet). Persistent volume is attached automatically.

### MySQL

When you click **MySQL** in the dashboard, VPSMyth:

1. Pulls the official `mysql:8` Docker image
2. Generates a strong random `root` password and a separate app-level user with limited privileges
3. Binds MySQL to `127.0.0.1` only — never exposed on a public port
4. Mounts a named Docker volume for data persistence
5. Sets `--character-set-server=utf8mb4` and `--collation-server=utf8mb4_unicode_ci` by default
6. Disables `LOAD DATA LOCAL INFILE` to prevent local file read attacks
7. Runs the container as a non-root user inside Docker (`mysql` user)
8. Provides a web UI panel to create/drop databases, manage users, and view connection strings

Credentials are stored in the VPSMyth encrypted config — never in plaintext on disk.

### MongoDB

When you click **MongoDB** in the dashboard, VPSMyth:

1. Pulls the official `mongo:7` Docker image
2. Creates an admin user and a per-database user with least-privilege roles
3. Enables authentication (`--auth` flag) — anonymous access is disabled
4. Binds to `127.0.0.1` only
5. Mounts a persistent volume for data
6. Disables JavaScript execution in queries (`security.javascriptEnabled: false`) where supported
7. Provides a panel to create databases, collections, manage users, and view connection URIs

### PostgreSQL

When you click **PostgreSQL** in the dashboard, VPSMyth:

1. Pulls the official `postgres:16` Docker image
2. Creates a superuser and a restricted app user with `CONNECT` + schema-level privileges only
3. Binds to `127.0.0.1` only
4. Mounts a persistent volume for data
5. Sets `pg_hba.conf` to use `scram-sha-256` password authentication
6. Disables remote superuser login
7. Provides a panel to create/drop databases, manage roles, run queries, and view connection strings

---

## Firewall Management

VPSMyth has a built-in firewall manager powered by **UFW (Uncomplicated Firewall)**. UFW wraps Linux `iptables` at the OS level — it controls **all traffic on all ports**, including SSH (port 22). Changes apply immediately with no service restart.

### What you can do

* **Enable / disable** UFW with one click
* **Add rules** — allow, deny, or rate-limit any port or port range
* **Set direction** — incoming, outgoing, or both
* **Filter by source IP** — e.g. allow port 22 only from your IP
* **Port ranges** — e.g. `8000:9000` covers all ports in that range
* **Protocol control** — TCP, UDP, or both
* **Delete any rule** by clicking Remove next to it
* **Quick presets** — common rules ready to apply in one click

### Quick presets

| Preset | What it does |
|--------|--------------|
| Allow SSH (22) | Opens port 22 for all IPs |
| Limit SSH (22) | Rate-limits SSH — blocks IPs that connect too fast, stops brute-force attacks |
| Allow HTTP (80) | Opens port 80 |
| Allow HTTPS (443) | Opens port 443 |
| Deny HTTP (80) | Blocks port 80 |
| Allow my IP on SSH | Allows only your current IP on port 22 — everyone else is blocked |

### Actions explained

| Action | Meaning |
|--------|---------|
| **Allow** | Let traffic through |
| **Deny** | Drop the packet silently — the sender gets no response |
| **Limit** | Allow, but rate-limit: if an IP makes 6+ connections in 30 seconds, block it temporarily. Use this on SSH to stop brute-force attacks. |

### How to add a rule

1. Go to **Firewall** in the sidebar
2. Enter your **sudo password** when prompted — required every time you open the page
3. Choose Action, Direction, Port, Protocol, and optionally a source IP
4. Click **Add Rule** — UFW applies it immediately

### Sudo password requirement

Every firewall operation (viewing rules, enabling/disabling, adding/deleting) requires your server's sudo password. This is because UFW requires root access.

**How it works:**

- When you open the Firewall page, a password prompt appears before anything loads
- You enter your sudo password once per page visit
- The password is held in memory for that session only — it is **never saved** to the database, cookies, or local storage
- When you leave the page or reload, the password is gone and you will be asked again next time
- If you enter the wrong password, it tells you immediately and lets you retry

This is intentional. Firewall changes are high-impact (a wrong rule can lock you out of SSH), so requiring the password every time adds a deliberate friction that prevents accidental changes.

### Important notes

* **UFW must be enabled** for rules to take effect. The status bar at the top of the Firewall page shows whether it is active. Rules you add are saved even when UFW is disabled — they activate when you enable it.
* **SSH lockout risk** — if you add a deny rule on port 22 without first allowing your IP, you will lose SSH access. Always add an allow rule for your IP before adding any deny rule on port 22.
* **VPSMyth backend port (2026)** is bound to `127.0.0.1` only — it is never reachable from outside regardless of firewall rules. You do not need to add a UFW rule for it.
* **Docker bypasses UFW** by default — Docker writes its own `iptables` rules directly, so `ufw deny` on a Docker-published port may not work as expected. To fix this, do not publish Docker ports to `0.0.0.0`; instead bind them to `127.0.0.1` (e.g. `-p 127.0.0.1:3306:3306`). VPSMyth does this automatically for all database containers it manages.

### Recovery if you lock yourself out of SSH

If you accidentally block SSH via UFW, you need to access your server through your hosting provider's **web console** (VNC/KVM console — available in most VPS dashboards like Hetzner, DigitalOcean, Contabo, etc.) and run:

```bash
sudo ufw delete <rule-number>
# or to disable UFW entirely:
sudo ufw disable
```

---

## IP Whitelisting

VPSMyth lets you lock down the admin panel so only specific IP addresses can access it — before any login prompt is shown.

### When does it activate?

The whitelist has two modes:

| Whitelist state | What happens |
|-----------------|--------------|
| **Empty (default)** | All IPs are allowed through. Everyone still needs to log in. |
| **One or more entries added** | Only listed IPs can reach the panel. All others get `403 Forbidden` immediately — no login page, no API access. |

The switch between these two modes is automatic. Add the first entry and the restriction is live. Remove all entries and the restriction is gone.

### What it blocks

Once the whitelist has entries, every request to the panel is checked — including:

* The dashboard (`/yourPanelPath/`)
* All API calls (`/api/...`)
* Static files (`/css/`, `/js/`, `/assets/`)

A blocked IP sees a plain `403 Forbidden` from Nginx. There is no login page, no error details, nothing to interact with.

### How it cannot be bypassed

Three layers work together:

1. **Backend binds to `127.0.0.1` only** — port 2026 is invisible from the internet. Direct port access is impossible regardless of firewall rules.
2. **Nginx enforces the whitelist** — `/etc/nginx/vpsmyth/whitelist.conf` is written by VPSMyth and included in every proxied location block. Blocked IPs are rejected at the network layer, before Go sees the request.
3. **Go middleware double-checks** — `IPWhitelistMiddleware` reads the `X-Real-IP` header injected by Nginx and rejects anything not in the list. Even if Nginx were misconfigured, the backend still enforces the same rules.

### Lockout protection

The Security page shows your current IP and warns you if it is not covered by any entry in the list. An **Add My IP** button lets you whitelist yourself with one click before saving any changes.

> **Important:** If you add entries that don't include your own IP, you will be locked out immediately. The lockout warning is there to prevent this. `127.0.0.1` (loopback) is always allowed so the server stays functional.

### How to configure

1. Go to **Security** in the sidebar
2. Your current IP is shown at the top
3. Enter an IP (`203.0.113.5`) or CIDR range (`10.0.0.0/24`) and an optional label
4. Click **Add to Whitelist** — Nginx reloads within seconds and the rule is live

To remove a restriction, click **Remove** next to any entry. When the last entry is removed, the whitelist is empty again and all IPs are allowed.

### Supported formats

| Input | Stored as | Matches |
|-------|-----------|---------|
| `1.2.3.4` | `1.2.3.4/32` | Exactly that IPv4 address |
| `2001:db8::1` | `2001:db8::1/128` | Exactly that IPv6 address |
| `10.0.0.0/8` | `10.0.0.0/8` | Entire `10.x.x.x` subnet |
| `192.168.1.0/24` | `192.168.1.0/24` | All `192.168.1.x` addresses |

### When you should use it

Use IP whitelisting when you have a **stable, predictable IP**:

* **Office or workplace** — your office router always has the same public IP. Whitelist it and the whole team is covered.
* **Home with a static IP** — some ISPs provide a fixed IP (check with your ISP). If yours does, whitelist it.
* **VPN with a fixed exit IP** — set up WireGuard or buy a VPN that gives you a dedicated IP. Whitelist that VPN IP and connect through it whenever you need the panel.

### When you should NOT use it

Do not use IP whitelisting if your IP changes regularly:

* **Home WiFi with a dynamic IP** — most ISPs (especially in countries like Bangladesh, India, etc.) reassign your public IP every time your router reconnects. Today your IP is `144.48.109.24`, tomorrow it could be `144.48.137.91`. If you whitelist `/32` (a single IP) and your IP changes, you are locked out.
* **Mobile data** — mobile IPs change constantly. Never whitelist a mobile IP as `/32`.
* **Shared or public networks** — whitelisting a shared network IP would let everyone on that network in.

In these cases, skip IP whitelisting entirely. The panel is already protected by login authentication and brute-force lockout — that is good enough for most self-hosted setups.

### Using a subnet range as a safer middle ground

If your ISP always assigns you IPs from the same block (e.g. always somewhere in `144.48.109.x`), you can whitelist the entire `/24` subnet instead of a single IP:

```
144.48.109.0/24
```

This allows any IP from `144.48.109.0` to `144.48.109.255` — so even if your IP changes within that range, you stay in. To find your current subnet, look at your IP (shown on the Security page) and replace the last number with `0`, then add `/24`.

> This is a tradeoff — it is less strict than a single `/32` but much more practical than nothing when you have a dynamic IP. If your ISP regularly moves you across different subnets, this still won't help and you are better off with a VPN.

### Recovery if you get locked out

If you accidentally lock yourself out, SSH into your server and run:

```bash
sqlite3 /opt/vpsmyth/vpsmyth.db "DELETE FROM ip_whitelist;"
echo "" | sudo tee /etc/nginx/vpsmyth/whitelist.conf
sudo nginx -s reload
```

This clears the whitelist completely. The panel is accessible to all IPs again immediately.

---

## Resource Management

VPSMyth lets you set CPU and memory limits for each deployed app and each database container.

### Per-App Limits

| Resource | Default | Configurable |
|----------|---------|--------------|
| CPU      | Unlimited | Yes (e.g., `0.5` = 50% of 1 core) |
| Memory   | Unlimited | Yes (e.g., `256m`, `1g`) |
| Swap     | Disabled | Optional |

* Limits are enforced via Docker `--cpus` and `--memory` flags
* If an app exceeds its memory limit, Docker kills and restarts it automatically
* The dashboard shows real-time CPU and memory usage per app

### System Overview

* Total CPU, RAM, and Disk usage visible on the main dashboard
* Per-container resource breakdown in the **Monitoring** section
* Historical graphs for the last 1h, 6h, 24h, and 7d

---

## Security

VPSMyth is designed with security as a first-class concern, not an afterthought.

### Authentication

* Single admin account with bcrypt-hashed password
* JWT-based session tokens with expiry
* Brute-force protection: account locks after repeated failed login attempts
* Secure cookie flags (`HttpOnly`, `Secure`, `SameSite=Strict`)

### Network Security

* Backend handles all privileged operations — the UI only calls APIs
* All database containers bound to `127.0.0.1` — no public database ports
* Nginx reverse proxy handles all public traffic; apps never bind directly to public interfaces
* Automatic UFW firewall setup on install

### Data Security

* Environment variables encrypted at rest using AES-256
* Database passwords generated with a cryptographically secure random generator
* Credentials never logged or exposed in API responses
* VPSMyth config file readable only by the `vpsmyth` system user

### Application Security

* Cron job commands are validated and sandboxed before execution
* Deploy scripts do not delete apps silently — all destructive actions require explicit confirmation
* No remote code execution from the UI — all operations go through validated API endpoints
* Input validation on all API endpoints to prevent injection attacks

### SSL / TLS

* One-click SSL via Let's Encrypt (Certbot)
* Auto-renewal configured via cron on install
* HTTP to HTTPS redirect enforced automatically

---

## Contributing

We welcome contributions. Please read CONTRIBUTING.md for guidelines. Follow the PR template for consistent pull requests. Run tests before submitting and keep changes modular and documented.

## Contributors

Thanks to all the amazing people who have contributed to this project:

[![Contributors](https://contrib.rocks/image?repo=prashanta0234/vpsmyth)](https://github.com/prashanta0234/vpsmyth/graphs/contributors)

## Documentation

* Server and API: cmd/server/server.md
* Core internals and services: internal/internal.md
* UI overview: ui/ui.md
* Scripts and automation: scripts/scripts.md
* Tests and examples: tests/tests.md
* Pull request guidelines: .github/pull_request_template.md

---

## Example Usage

### Deploy a Node.js App

1. Go to the dashboard and click **Deploy App**
2. Enter Name, Runtime (Node.js), Source (GitHub URL), Env variables
3. Set CPU and memory limits if needed
4. Click **Deploy** — the app starts and appears in monitoring

### Launch a MySQL Database

1. Go to **Databases > MySQL**
2. Click **Create Instance**
3. VPSMyth generates credentials, pulls the image, and starts the container
4. Copy the connection string from the panel and use it in your app's env variables

### Configure Firewall

1. Go to **Firewall**
2. Review current rules
3. Click **Add Rule** to allow or block a port
4. Click **Apply**

### Schedule a Cron Job

1. Go to **Cron Management**
2. Add new job with App, Schedule, Command
3. Logs will be tracked automatically

---

## Roadmap

* Add support for Python, Rust, R
* Multi-user support with roles
* Advanced metrics and alerts
* Docker support (optional containerized deploy)
* Marketplace for pre-built apps
* Audit log for all admin actions
* Two-factor authentication (2FA)
* Webhook notifications (Slack, Discord, email)

---

## Why VPSMyth

* No DevOps knowledge required
* Lightweight and fast
* Modular and contributor-friendly
* Open-source and free
* Perfect for VPS automation enthusiasts, students, and developers

## License

VPSMyth is licensed under MIT License.
