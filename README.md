# VPSMyth

VPSMyth is a lightweight, all-in-one VPS management platform. It allows you to deploy, monitor, and manage multiple applications with one click, without needing deep DevOps knowledge. Think of it as your personal VPS control plane that is simple, secure, and easy to use.

## MVP Features

* One-click app deployment for Node.js and Go applications
* Deploy from GitHub repo or ZIP upload
* Automatic port allocation
* Environment management per app
* System and app monitoring (CPU, RAM, Disk, uptime)
* Cron job management with logging
* Database management (SQLite and Redis)
* Port management
* One-click SSL and domain setup
* Lightweight and VPS-friendly (Go backend, minimal memory usage, no Docker required)

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

After installation, open the dashboard at:

```
http://YOUR_SERVER_IP
```

## Directory Structure

```
vpsmyth/
├── cmd/           # Backend entry points and server setup
│   └── server/    # HTTP server and API documentation
├── internal/      # Core logic (deploy, monitor, cron, config)
├── ui/            # Dashboard UI
├── scripts/       # Installation and helper scripts
├── tests/         # Backend tests
├── .github/       # GitHub workflows and PR templates
├── README.md
└── LICENSE
```

Full explanation of modules is documented in the component-specific markdown files under `cmd/`, `internal/`, `ui/`, `scripts/`, and `tests/`.

## Contributing

We welcome contributions. Please read CONTRIBUTING.md for guidelines. Follow the PR template for consistent pull requests. Run tests before submitting and keep changes modular and documented.

## Documentation

* Server and API: cmd/server/server.md
* Core internals and services: internal/internal.md
* UI overview: ui/ui.md
* Scripts and automation: scripts/scripts.md
* Tests and examples: tests/tests.md
* Pull request guidelines: .github/pull_request_template.md

## Example Usage

Deploy a Node.js app:

1. Go to the dashboard and click Deploy App
2. Enter Name, Runtime (Node.js), Source (GitHub URL), Env variables
3. Click Deploy
4. The app will start and appear in monitoring

Schedule a Cron Job:

1. Go to Cron Management
2. Add new job with App, Schedule, Command
3. Logs will be tracked automatically

## Roadmap

* Add support for Python, Rust, R
* Multi-user support with roles
* Advanced metrics and alerts
* Docker support (optional)
* Marketplace for pre-built apps

## Security

* Backend handles privileged tasks, not the UI
* Environment variables are stored securely
* Cron jobs are validated before execution
* Scripts will not delete apps silently

## Why VPSMyth

* No DevOps knowledge required
* Lightweight and fast
* Modular and contributor-friendly
* Open-source and free
* Perfect for VPS automation enthusiasts, students, and developers

## License

VPSMyth is licensed under MIT License.
