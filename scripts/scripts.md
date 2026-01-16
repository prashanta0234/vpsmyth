
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
- Optional: remove apps

---

## Best Practices
- Scripts should be safe
- Avoid deleting user apps silently
- Print all steps for user visibility
