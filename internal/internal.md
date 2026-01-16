
## Purpose
Contains **core backend logic**.  
Packages inside `internal` are **private** and **cannot be imported externally**.

### Subfolders

#### `deploy/`
- Deploy apps (Node.js, Go)
- Handle GitHub repo cloning, ZIP extraction
- Port allocation
- Start, stop, restart apps
- Manage environment variables

#### `monitor/`
- Collect system and app metrics
- CPU, RAM, Disk usage
- App uptime and health checks
- Logs resource usage

#### `cron/`
- Manage scheduled jobs
- Add/remove cron jobs
- Log execution and failures
- Validate cron expressions

#### `config/`
- Load global configuration
- Load app-specific environment variables
- Validate configs
- Store sensitive values securely


#### `utils/`
- Helper functions for common tasks
- Logging, error handling, file operations
- System information gathering