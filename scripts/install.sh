#!/bin/bash

# VPSMyth Installation Script
# This script will install VPSMyth on your server.

set -e

echo "Starting VPSMyth installation..."

# 1. Check and Install Docker
if ! [ -x "$(command -v docker)" ]; then
    echo "Installing Docker..."
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    sudo usermod -aG docker $USER
    rm get-docker.sh
    echo "Docker installed successfully."
else
    echo "Docker is already installed."
fi

# 2. Check and Install Node.js (20.x)
if ! [ -x "$(command -v node)" ]; then
    echo "Installing Node.js 20.x..."
    curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
    sudo apt-get install -y nodejs
    echo "Node.js installed successfully."
else
    echo "Node.js is already installed."
fi

# 3. Check and Install Go (1.21+)
if ! [ -x "$(command -v go)" ]; then
    echo "Installing Go..."
    GO_VERSION="1.21.5"
    curl -LO "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
    sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    export PATH=$PATH:/usr/local/go/bin
    rm "go${GO_VERSION}.linux-amd64.tar.gz"
    echo "Go installed successfully."
else
    echo "Go is already installed."
fi

# 4. Setup VPSMyth (Placeholder for actual setup)
echo "Setting up VPSMyth..."
# mkdir -p /opt/vpsmyth
# cp vpsmyth /opt/vpsmyth/

echo "VPSMyth installation complete!"
