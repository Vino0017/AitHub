#!/bin/bash
# SkillHub Remote Deploy Script
# Run this on the server after copying files

set -euo pipefail

echo "=== SkillHub Server Setup ==="

# Install Docker if not present
if ! command -v docker &>/dev/null; then
    echo "→ Installing Docker..."
    curl -fsSL https://get.docker.com | sh
    systemctl enable docker
    systemctl start docker
else
    echo "→ Docker already installed: $(docker --version)"
fi

# Install Docker Compose plugin if not present
if ! docker compose version &>/dev/null; then
    echo "→ Installing Docker Compose plugin..."
    apt-get update && apt-get install -y docker-compose-plugin
else
    echo "→ Docker Compose already installed: $(docker compose version)"
fi

# Create app directory
mkdir -p /opt/skillhub
cd /opt/skillhub

# Copy .env if not exists
if [ ! -f .env ]; then
    cp .env.example .env
    # Generate a random admin token
    ADMIN_TOKEN=$(openssl rand -hex 32)
    sed -i "s/ADMIN_TOKEN=change-me-in-production/ADMIN_TOKEN=$ADMIN_TOKEN/" .env
    echo "→ Generated .env with admin token"
fi

# Build and start
echo "→ Building and starting services..."
docker compose up -d --build

echo ""
echo "=== Deployment Complete ==="
echo "→ API: http://$(hostname -I | awk '{print $1}'):8080"
echo "→ Health: curl http://localhost:8080/health"
docker compose ps
