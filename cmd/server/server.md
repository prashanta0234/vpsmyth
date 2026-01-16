
## Purpose
Contains the **entry point of the backend**.

### `server/`
- Starts the Go backend server
- Initializes services:
  - Deploy
  - Monitor
  - Cron
  - Config
- Handles routing and API endpoints
- Should **not contain business logic** (keep logic in `internal/`)

---

## Best Practices
- Keep it minimal
- Only wiring code
- Do not add utilities or helper functions here
