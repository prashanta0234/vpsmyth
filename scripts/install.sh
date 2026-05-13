#!/bin/bash

# VPSMyth Installation Script (Infrastructure Layer)
set -e

echo ""
echo "========================================"
echo "   Welcome to VPSMyth Installer"
echo "========================================"
echo ""

#################################
# 0. Dashboard Path Setup
#################################
DEFAULT_PANEL_PATH="admin"

while true; do
    # Read from /dev/tty explicitly so this works when piped via curl | bash
    read -rp "Enter dashboard path [default: ${DEFAULT_PANEL_PATH}]: " PANEL_PATH_INPUT </dev/tty
    PANEL_PATH="${PANEL_PATH_INPUT:-$DEFAULT_PANEL_PATH}"

    # Strip any leading/trailing slashes entered by user
    PANEL_PATH="${PANEL_PATH#/}"
    PANEL_PATH="${PANEL_PATH%/}"

    # Validate: only allow alphanumeric, hyphens, underscores. 2-40 chars.
    if [[ "$PANEL_PATH" =~ ^[a-zA-Z0-9_-]{2,40}$ ]]; then
        break
    else
        echo "  Invalid path. Use only letters, numbers, hyphens, or underscores (2-40 characters). No slashes."
    fi
done

echo ""
echo "  Dashboard will be available at: http://YOUR_SERVER_IP/${PANEL_PATH}"
echo ""

echo "Starting VPSMyth installation..."

#################################
# 1. Install Docker
#################################
if ! command -v docker >/dev/null 2>&1; then
    echo "Installing Docker..."
    curl -fsSL https://get.docker.com | sudo sh
    sudo usermod -aG docker $USER
    echo "Docker installed."
else
    echo "Docker already installed."
fi

#################################
# 2. Install Node.js 20.x
#################################
if ! command -v node >/dev/null 2>&1; then
    echo "Installing Node.js 20.x..."
    curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
    sudo apt-get install -y nodejs
    echo "Node.js installed."
else
    echo "Node.js already installed."
fi

#################################
# 3. Install Go 1.21+
#################################
if ! command -v go >/dev/null 2>&1; then
    echo "Installing Go..."
    GO_VERSION="1.21.5"
    curl -LO "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    export PATH=$PATH:/usr/local/go/bin
    rm go${GO_VERSION}.linux-amd64.tar.gz
    echo "Go installed."
else
    echo "Go already installed."
fi

#################################
# 4. Install Nginx
#################################
if ! command -v nginx >/dev/null 2>&1; then
    echo "Installing Nginx..."
    sudo apt update
    sudo apt install -y nginx
    sudo systemctl enable nginx
    sudo systemctl start nginx
    echo "Nginx installed."
else
    echo "Nginx already installed."
fi

#################################
# 5. Base Nginx Configuration (Catch-All)
#################################
echo "Configuring base Nginx (catch-all router)..."

# Clean defaults
sudo rm -f /etc/nginx/sites-enabled/default
sudo rm -f /etc/nginx/sites-available/default

# Create directories for VPSMyth-managed apps
sudo mkdir -p /etc/nginx/vpsmyth/apps

# Base catch-all server (owned by VPSMyth)
sudo tee /etc/nginx/sites-available/vpsmyth-base >/dev/null <<'EOF'
server {
    listen 80 default_server;
    server_name _;

    # Infrastructure placeholder
    location / {
        return 200 "VPSMyth is running.\n";
        add_header Content-Type text/plain;
    }
}
EOF

sudo ln -sf /etc/nginx/sites-available/vpsmyth-base /etc/nginx/sites-enabled/vpsmyth-base

#################################
# 6. Include App Configs (future use)
#################################
sudo tee /etc/nginx/conf.d/vpsmyth.conf >/dev/null <<'EOF'
# VPSMyth managed applications
include /etc/nginx/vpsmyth/apps/*.conf;
EOF

#################################
# 7. Test & Reload Nginx
#################################
sudo nginx -t
sudo systemctl reload nginx

echo "Nginx base router configured."

#################################
# 8. Build & Install VPSMyth
#################################
echo "Building VPSMyth..."

# Ensure we are in the project root
#
# Normal usage:
#   - Run from project root:          ./scripts/install.sh
#   - Or from scripts directory:      bash install.sh
#
# Remote usage (what you're doing):
#   curl -fsSL https://raw.githubusercontent.com/prashanta0234/vpsmyth/main/scripts/install.sh | sudo bash
#
# For remote usage there is no local checkout, so if go.mod is not found
# we clone the repository into /opt/vpsmyth-src and build from there.
if [ -f "go.mod" ]; then
    PROJECT_ROOT="."
elif [ -f "../go.mod" ]; then
    PROJECT_ROOT=".."
else
    echo "go.mod not found in current or parent directory. Attempting to clone VPSMyth repository..."
    PROJECT_ROOT="/opt/vpsmyth-src"

    # Ensure git is available
    if ! command -v git >/dev/null 2>&1; then
        echo "git not found. Installing git..."
        sudo apt-get update
        sudo apt-get install -y git
    fi

    if [ ! -d "$PROJECT_ROOT/.git" ]; then
        echo "Cloning repository into $PROJECT_ROOT..."
        sudo rm -rf "$PROJECT_ROOT"
        sudo git clone https://github.com/prashanta0234/vpsmyth.git "$PROJECT_ROOT"
    else
        echo "Existing repository found at $PROJECT_ROOT, pulling latest changes..."
        cd "$PROJECT_ROOT"
        sudo git pull --ff-only
        cd - >/dev/null 2>&1 || true
    fi
fi

cd "$PROJECT_ROOT"

# Build the binary
/usr/local/go/bin/go build -o vpsmyth cmd/server/main.go

# Install binary to /opt/vpsmyth
sudo mkdir -p /opt/vpsmyth
sudo cp vpsmyth /opt/vpsmyth/
sudo cp -r ui /opt/vpsmyth/
# Copy internal config or other assets if needed
# sudo cp config.json /opt/vpsmyth/ 2>/dev/null || true

# Create systemd service
echo "Creating systemd service..."
sudo tee /etc/systemd/system/vpsmyth.service >/dev/null <<EOF
[Unit]
Description=VPSMyth Management Portal
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/vpsmyth
ExecStart=/opt/vpsmyth/vpsmyth
Restart=on-failure
Environment="PORT=2026"
# Set these if you want to auto-create admin on first run
# Environment="ADMIN_USERNAME=admin"
# Environment="ADMIN_PASSWORD=changeme"

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable vpsmyth
sudo systemctl start vpsmyth

echo "VPSMyth service installed and started."

#################################
# 9. Update Nginx Config
#################################
echo "Updating Nginx configuration..."

# Save panel path for reference
echo "$PANEL_PATH" | sudo tee /opt/vpsmyth/panel-path >/dev/null

sudo tee /etc/nginx/sites-available/vpsmyth-base >/dev/null <<EOF
server {
    listen 80 default_server;
    server_name _;

    # Block root — visiting IP directly reveals nothing
    location = / {
        return 404;
    }

    # Redirect panel path without trailing slash
    location = /${PANEL_PATH} {
        return 301 /${PANEL_PATH}/;
    }

    # Dashboard entry point — strip panel prefix before proxying to backend
    location /${PANEL_PATH}/ {
        proxy_pass http://localhost:2026/;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    # Static assets — HTML references these as absolute paths so they must stay at root
    location /css/ {
        proxy_pass http://localhost:2026/css/;
        proxy_set_header Host \$host;
    }

    location /js/ {
        proxy_pass http://localhost:2026/js/;
        proxy_set_header Host \$host;
    }

    location /assets/ {
        proxy_pass http://localhost:2026/assets/;
        proxy_set_header Host \$host;
    }

    # API routes — protected by JWT on the backend
    location /api/ {
        proxy_pass http://localhost:2026/api/;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    # Block everything else
    location / {
        return 404;
    }
}
EOF

sudo nginx -t
sudo systemctl reload nginx

echo "Nginx configured to proxy to VPSMyth."

# Detect server IP for the final message
SERVER_IP=$(hostname -I | awk '{print $1}')

echo ""
echo "========================================"
echo "   VPSMyth installation complete!"
echo "========================================"
echo ""
echo "  Dashboard: http://${SERVER_IP}/${PANEL_PATH}"
echo ""
echo "  Panel path saved to: /opt/vpsmyth/panel-path"
echo "  To check it later: cat /opt/vpsmyth/panel-path"
echo ""
