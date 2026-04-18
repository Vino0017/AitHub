#!/bin/bash
# SkillHub Deployment Script

set -e

echo "🚀 SkillHub Deployment Starting..."

# Configuration
SERVER="root@192.227.235.131"
DOMAIN="skillhub.koolkassanmsk.top"
APP_DIR="/opt/skillhub"
SERVICE_NAME="skillhub"

# Build locally
echo "📦 Building application..."
GOOS=linux GOARCH=amd64 go build -o skillhub-linux cmd/api/main.go

# Upload to server
echo "📤 Uploading to server..."
ssh $SERVER "mkdir -p $APP_DIR"
scp skillhub-linux $SERVER:$APP_DIR/skillhub
scp -r migrations $SERVER:$APP_DIR/
scp -r scripts $SERVER:$APP_DIR/

# Create .env on server
echo "⚙️  Configuring environment..."
ssh $SERVER "cat > $APP_DIR/.env << 'EOF'
# Database
DATABASE_URL=postgresql://skillhub:skillhub_password@localhost:5432/skillhub?sslmode=disable

# Server
PORT=8080
AUTO_MIGRATE=true
SEED_DATA=false

# Admin
ADMIN_TOKEN=change-me-in-production

# AI Review
AI_REVIEW_ENABLED=true
LLM_PROVIDER=openrouter
LLM_API_KEY=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
LLM_MODEL=google/gemma-4-31b-it
LLM_BASE_URL=https://openrouter.ai/api/v1

# Email (optional)
# SMTP_HOST=
# SMTP_PORT=
# SMTP_USER=
# SMTP_PASSWORD=
# SMTP_FROM=
EOF"

# Create systemd service
echo "🔧 Creating systemd service..."
ssh $SERVER "cat > /etc/systemd/system/$SERVICE_NAME.service << 'EOF'
[Unit]
Description=SkillHub API Server
After=network.target postgresql.service

[Service]
Type=simple
User=root
WorkingDirectory=$APP_DIR
ExecStart=$APP_DIR/skillhub
Restart=always
RestartSec=5
Environment=PATH=/usr/local/bin:/usr/bin:/bin

[Install]
WantedBy=multi-user.target
EOF"

# Reload systemd and restart service
echo "🔄 Restarting service..."
ssh $SERVER "systemctl daemon-reload && systemctl enable $SERVICE_NAME && systemctl restart $SERVICE_NAME"

# Wait for service to start
echo "⏳ Waiting for service to start..."
sleep 5

# Check service status
echo "✅ Checking service status..."
ssh $SERVER "systemctl status $SERVICE_NAME --no-pager"

# Test health endpoint
echo "🏥 Testing health endpoint..."
curl -f https://$DOMAIN/health || echo "⚠️  Health check failed"

echo "✨ Deployment complete!"
echo "🌐 Service available at: https://$DOMAIN"
echo "📊 Health check: https://$DOMAIN/health"
echo "📝 Logs: ssh $SERVER 'journalctl -u $SERVICE_NAME -f'"
