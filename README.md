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

VPSMyth provides a built-in firewall management interface powered by **UFW (Uncomplicated Firewall)**.

### Features

* View all active firewall rules in the dashboard
* Add and remove rules (allow/deny) for any port or port range
* Protocol-level control (TCP, UDP, or both)
* One-click deny-all incoming with allow-all outgoing (recommended default)
* Preset rules for common services (HTTP 80, HTTPS 443, SSH 22, custom)
* Real-time rule status without SSH access

### Default Firewall Policy (applied on install)

| Direction | Policy |
|-----------|--------|
| Incoming  | Deny all |
| Outgoing  | Allow all |
| SSH (22)  | Allow (to prevent lockout) |
| HTTP (80) | Allow |
| HTTPS (443) | Allow |
| VPSMyth dashboard port | Allow (restricted by IP whitelist) |

### How to Add a Rule

1. Go to **Firewall** in the dashboard
2. Click **Add Rule**
3. Enter port, protocol, and action (allow/deny)
4. Click **Apply** — the rule is active immediately

---

## IP Whitelisting

VPSMyth supports IP-level access control for the dashboard and individual apps.

### Dashboard IP Whitelist

* Only IPs in the whitelist can access the VPSMyth dashboard
* By default, all IPs are allowed (open mode) — you should restrict this after install
* Supports single IPs (`203.0.113.5`) and CIDR ranges (`10.0.0.0/24`)
* Changes take effect immediately without a service restart

### Per-App IP Whitelist

* Each deployed app can have its own IP whitelist
* Traffic from non-whitelisted IPs returns `403 Forbidden` via the Nginx reverse proxy
* Whitelist is managed per-app from the app detail page

### How to Configure

1. Go to **Settings > Security > IP Whitelist**
2. Add your IP address or CIDR range
3. Click **Save** — the Nginx config is updated and reloaded automatically

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
