#!/bin/bash

# VPSMyth Installation Script (Infrastructure Layer)
set -e

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

# Ensure we are in the project root (assuming script is run from project root or scripts dir)
# We'll try to find go.mod
if [ -f "go.mod" ]; then
    PROJECT_ROOT="."
elif [ -f "../go.mod" ]; then
    PROJECT_ROOT=".."
else
    echo "Error: Could not find project root (go.mod not found)."
    exit 1
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
Environment="PORT=8080"
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

sudo tee /etc/nginx/sites-available/vpsmyth-base >/dev/null <<'EOF'
server {
    listen 80 default_server;
    server_name _;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF

sudo nginx -t
sudo systemctl reload nginx

echo "Nginx configured to proxy to VPSMyth."

echo "VPSMyth installation complete 🚀"
echo "Access your management portal at http://<YOUR_SERVER_IP>"
