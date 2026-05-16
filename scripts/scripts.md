
## Purpose
System-level Bash scripts for automation.

### Files

#### `install.sh`
- Install dependencies (curl, git, nginx, ufw)
- Download VPSMyth binary
- Create directories and users
- Setup systemd service
- Start VPSMyth

#### `uninstall.sh`
- Stop VPSMyth service
- Remove binaries
- Clean configuration
- Optional: remove apps, Docker, Node.js (user decides)

#### `update.sh`
- Pull latest source from GitHub
- Exit early if already on latest commit
- Build new binary
- Stop service, swap binary and UI, restart
- Auto-rollback if service fails to start after update
- Preserves database, credentials, panel path, and Nginx config

---

## Best Practices
- Scripts should be safe
- Avoid deleting user apps silently
- Print all steps for user visibility
