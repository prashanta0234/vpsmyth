#!/bin/bash

# VPSMyth Uninstallation Script
set -e

echo ""
echo "========================================"
echo "   VPSMyth Uninstaller"
echo "========================================"
echo ""

#################################
# Helper: ask yes/no from /dev/tty
# Works even when piped via curl | bash
#################################
ask() {
    local prompt="$1"
    local answer
    while true; do
        read -rp "$prompt [y/N]: " answer </dev/tty
        case "${answer,,}" in
            y|yes) return 0 ;;
            n|no|"") return 1 ;;
            *) echo "  Please enter y or n." ;;
        esac
    done
}

#################################
# 1. Confirm uninstall
#################################
echo "This will remove VPSMyth and all its data from this server."
echo ""

if ! ask "Are you sure you want to uninstall VPSMyth?"; then
    echo "Uninstallation cancelled."
    exit 0
fi

echo ""

#################################
# 2. Ask about VPSMyth app containers
#################################
REMOVE_APP_CONTAINERS=false
if ask "Remove all Docker containers deployed by VPSMyth (your apps)?"; then
    REMOVE_APP_CONTAINERS=true
fi

echo ""

#################################
# 3. Ask about Docker
#################################
REMOVE_DOCKER=false
if ask "Remove Docker from this server?"; then
    REMOVE_DOCKER=true
fi

echo ""

#################################
# 4. Ask about Node.js
#################################
REMOVE_NODE=false
if ask "Remove Node.js from this server?"; then
    REMOVE_NODE=true
fi

echo ""
echo "========================================"
echo "   Starting uninstallation..."
echo "========================================"
echo ""

#################################
# 5. Stop and disable VPSMyth service
#################################
echo "[1/7] Stopping VPSMyth service..."

if systemctl is-active --quiet vpsmyth 2>/dev/null; then
    sudo systemctl stop vpsmyth
    echo "  Service stopped."
else
    echo "  Service was not running."
fi

if systemctl is-enabled --quiet vpsmyth 2>/dev/null; then
    sudo systemctl disable vpsmyth
    echo "  Service disabled."
fi

#################################
# 6. Remove systemd service file
#################################
echo "[2/7] Removing systemd service..."

if [ -f /etc/systemd/system/vpsmyth.service ]; then
    sudo rm -f /etc/systemd/system/vpsmyth.service
    sudo systemctl daemon-reload
    echo "  Service file removed."
else
    echo "  Service file not found, skipping."
fi

#################################
# 7. Remove VPSMyth binary and files
#################################
echo "[3/7] Removing VPSMyth files..."

if [ -d /opt/vpsmyth ]; then
    sudo rm -rf /opt/vpsmyth
    echo "  /opt/vpsmyth removed."
else
    echo "  /opt/vpsmyth not found, skipping."
fi

if [ -d /opt/vpsmyth-src ]; then
    sudo rm -rf /opt/vpsmyth-src
    echo "  /opt/vpsmyth-src removed."
fi

#################################
# 8. Remove Nginx configuration
#################################
echo "[4/7] Removing Nginx configuration..."

if [ -f /etc/nginx/sites-enabled/vpsmyth-base ]; then
    sudo rm -f /etc/nginx/sites-enabled/vpsmyth-base
fi

if [ -f /etc/nginx/sites-available/vpsmyth-base ]; then
    sudo rm -f /etc/nginx/sites-available/vpsmyth-base
fi

if [ -f /etc/nginx/conf.d/vpsmyth.conf ]; then
    sudo rm -f /etc/nginx/conf.d/vpsmyth.conf
fi

if [ -d /etc/nginx/vpsmyth ]; then
    sudo rm -rf /etc/nginx/vpsmyth
fi

if command -v nginx >/dev/null 2>&1; then
    sudo nginx -t 2>/dev/null && sudo systemctl reload nginx
    echo "  Nginx config removed and reloaded."
else
    echo "  Nginx not found, skipping."
fi

#################################
# 9. Remove VPSMyth-managed app containers
#################################
echo "[5/7] Handling app containers..."

if [ "$REMOVE_APP_CONTAINERS" = true ]; then
    if command -v docker >/dev/null 2>&1; then
        CONTAINERS=$(docker ps -a --filter "label=managed-by=vpsmyth" --format "{{.Names}}" 2>/dev/null || true)
        if [ -n "$CONTAINERS" ]; then
            echo "  Stopping and removing VPSMyth containers:"
            echo "$CONTAINERS" | while read -r name; do
                docker rm -f "$name" 2>/dev/null && echo "    - $name removed" || true
            done
        else
            echo "  No VPSMyth-managed containers found."
        fi

        # Remove VPSMyth-built images
        IMAGES=$(docker images --filter "reference=vpsmyth/*" --format "{{.Repository}}:{{.Tag}}" 2>/dev/null || true)
        if [ -n "$IMAGES" ]; then
            echo "  Removing VPSMyth-built images:"
            echo "$IMAGES" | while read -r img; do
                docker rmi -f "$img" 2>/dev/null && echo "    - $img removed" || true
            done
        fi
    else
        echo "  Docker not installed, skipping container removal."
    fi
else
    echo "  Skipping app container removal (user choice)."
fi

#################################
# 10. Remove Docker (optional)
#################################
echo "[6/7] Handling Docker..."

if [ "$REMOVE_DOCKER" = true ]; then
    if command -v docker >/dev/null 2>&1; then
        echo "  Removing Docker..."
        sudo systemctl stop docker 2>/dev/null || true
        sudo apt-get purge -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin 2>/dev/null || \
        sudo apt-get purge -y docker.io 2>/dev/null || true
        sudo apt-get autoremove -y 2>/dev/null || true
        sudo rm -rf /var/lib/docker /etc/docker
        sudo groupdel docker 2>/dev/null || true
        echo "  Docker removed."
    else
        echo "  Docker is not installed, skipping."
    fi
else
    echo "  Skipping Docker removal (user choice)."
fi

#################################
# 11. Remove Node.js (optional)
#################################
echo "[7/7] Handling Node.js..."

if [ "$REMOVE_NODE" = true ]; then
    if command -v node >/dev/null 2>&1; then
        echo "  Removing Node.js..."
        sudo apt-get purge -y nodejs 2>/dev/null || true
        sudo apt-get autoremove -y 2>/dev/null || true
        sudo rm -f /etc/apt/sources.list.d/nodesource.list
        sudo rm -f /usr/share/keyrings/nodesource.gpg
        echo "  Node.js removed."
    else
        echo "  Node.js is not installed, skipping."
    fi
else
    echo "  Skipping Node.js removal (user choice)."
fi

#################################
# Done
#################################
echo ""
echo "========================================"
echo "   VPSMyth uninstallation complete!"
echo "========================================"
echo ""
echo "  Summary:"
echo "    VPSMyth service and files : removed"
echo "    Nginx configuration       : removed"

if [ "$REMOVE_APP_CONTAINERS" = true ]; then
    echo "    App containers            : removed"
else
    echo "    App containers            : kept"
fi

if [ "$REMOVE_DOCKER" = true ]; then
    echo "    Docker                    : removed"
else
    echo "    Docker                    : kept"
fi

if [ "$REMOVE_NODE" = true ]; then
    echo "    Node.js                   : removed"
else
    echo "    Node.js                   : kept"
fi

echo ""
