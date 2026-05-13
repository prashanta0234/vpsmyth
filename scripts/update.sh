#!/bin/bash

# VPSMyth Update Script
# Pulls the latest code, rebuilds the binary, and restarts the service.
# Preserves your database, credentials, panel path, and Nginx config.
set -e

echo ""
echo "========================================"
echo "   VPSMyth Updater"
echo "========================================"
echo ""

#################################
# 1. Verify VPSMyth is installed
#################################
if [ ! -f /opt/vpsmyth/vpsmyth ]; then
    echo "Error: VPSMyth does not appear to be installed (/opt/vpsmyth/vpsmyth not found)."
    echo "Run the installer first:"
    echo "  curl -fsSL https://raw.githubusercontent.com/prashanta0234/vpsmyth/main/scripts/install.sh | sudo bash"
    exit 1
fi

if ! systemctl list-units --type=service | grep -q "vpsmyth.service"; then
    echo "Error: vpsmyth systemd service not found."
    exit 1
fi

#################################
# 2. Ensure Go is available
#################################
GO_BIN=""
if command -v go >/dev/null 2>&1; then
    GO_BIN="go"
elif [ -x /usr/local/go/bin/go ]; then
    GO_BIN="/usr/local/go/bin/go"
else
    echo "Error: Go is not installed. Cannot build the update."
    exit 1
fi

echo "Using Go: $($GO_BIN version)"
echo ""

#################################
# 3. Pull latest source
#################################
PROJECT_ROOT="/opt/vpsmyth-src"

if [ -d "$PROJECT_ROOT/.git" ]; then
    echo "[1/4] Pulling latest changes from GitHub..."
    cd "$PROJECT_ROOT"
    CURRENT=$(git rev-parse --short HEAD)
    sudo git fetch origin
    sudo git reset --hard origin/$(git rev-parse --abbrev-ref HEAD)
    LATEST=$(git rev-parse --short HEAD)
    if [ "$CURRENT" = "$LATEST" ]; then
        echo "  Already up to date ($CURRENT). No changes to apply."
        echo ""
        echo "VPSMyth is already on the latest version."
        exit 0
    fi
    echo "  Updated: $CURRENT → $LATEST"
else
    echo "[1/4] Source not found at $PROJECT_ROOT. Cloning repository..."
    if ! command -v git >/dev/null 2>&1; then
        sudo apt-get update -qq && sudo apt-get install -y git
    fi
    sudo rm -rf "$PROJECT_ROOT"
    sudo git clone https://github.com/prashanta0234/vpsmyth.git "$PROJECT_ROOT"
    cd "$PROJECT_ROOT"
    LATEST=$(git rev-parse --short HEAD)
    echo "  Cloned at $LATEST"
fi

echo ""

#################################
# 4. Build new binary
#################################
echo "[2/4] Building new binary..."
cd "$PROJECT_ROOT"
sudo $GO_BIN build -o vpsmyth_new cmd/server/main.go
echo "  Build successful."
echo ""

#################################
# 5. Stop service, swap files, restart
#################################
echo "[3/4] Installing update..."

# Stop service
sudo systemctl stop vpsmyth
echo "  Service stopped."

# Backup current binary
sudo cp /opt/vpsmyth/vpsmyth /opt/vpsmyth/vpsmyth.bak
echo "  Backup saved to /opt/vpsmyth/vpsmyth.bak"

# Replace binary
sudo cp "$PROJECT_ROOT/vpsmyth_new" /opt/vpsmyth/vpsmyth
sudo rm -f "$PROJECT_ROOT/vpsmyth_new"

# Replace UI files (preserves db, panel-path, and other runtime files)
sudo rm -rf /opt/vpsmyth/ui
sudo cp -r "$PROJECT_ROOT/ui" /opt/vpsmyth/ui

echo "  Binary and UI updated."

# Restart service
sudo systemctl start vpsmyth
echo "  Service restarted."
echo ""

#################################
# 6. Verify
#################################
echo "[4/4] Verifying..."
sleep 2

if systemctl is-active --quiet vpsmyth; then
    echo "  vpsmyth service is running."
else
    echo "  Warning: service did not start. Rolling back..."
    sudo systemctl stop vpsmyth 2>/dev/null || true
    sudo cp /opt/vpsmyth/vpsmyth.bak /opt/vpsmyth/vpsmyth
    sudo systemctl start vpsmyth
    echo "  Rollback complete. Check logs: sudo journalctl -u vpsmyth -n 50"
    exit 1
fi

PANEL_PATH=$(cat /opt/vpsmyth/panel-path 2>/dev/null || echo "admin")
SERVER_IP=$(hostname -I | awk '{print $1}')

echo ""
echo "========================================"
echo "   VPSMyth updated successfully!"
echo "========================================"
echo ""
echo "  Version : $LATEST"
echo "  Dashboard: http://${SERVER_IP}/${PANEL_PATH}"
echo ""
echo "  If something looks wrong, roll back with:"
echo "    sudo systemctl stop vpsmyth"
echo "    sudo cp /opt/vpsmyth/vpsmyth.bak /opt/vpsmyth/vpsmyth"
echo "    sudo systemctl start vpsmyth"
echo ""
