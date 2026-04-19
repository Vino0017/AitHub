#!/usr/bin/env bash
# SkillHub 自动部署脚本 (带密码认证)
# 使用方法: ./deploy_with_password.sh

set -e

# 配置
SERVER="root@192.227.235.131"
PASSWORD="Vino"
DOMAIN="skillhub.koolkassanmsk.top"
APP_DIR="/opt/skillhub"
SERVICE_NAME="skillhub"
LLM_API_KEY="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"

echo "🚀 SkillHub 自动部署开始..."
echo ""

# 检查是否安装了expect
if ! command -v expect &> /dev/null; then
    echo "❌ 需要安装 expect 工具"
    echo "   Ubuntu/Debian: sudo apt install expect"
    echo "   macOS: brew install expect"
    echo "   或者使用手动部署: 参考 DEPLOY_MANUAL.md"
    exit 1
fi

# 1. 本地构建
echo "📦 构建应用..."
GOOS=linux GOARCH=amd64 go build -o skillhub-linux cmd/api/main.go
echo "✅ 构建完成"
echo ""

# 2. 生成强随机ADMIN_TOKEN
ADMIN_TOKEN=$(openssl rand -hex 32)
echo "🔑 生成的 ADMIN_TOKEN: $ADMIN_TOKEN"
echo "   (已保存到 .admin_token.txt)"
echo "$ADMIN_TOKEN" > .admin_token.txt
chmod 600 .admin_token.txt
echo ""

# 3. 创建expect脚本用于上传文件
echo "📤 上传文件到服务器..."

# 创建临时expect脚本
cat > /tmp/deploy_upload.exp << 'EXPECT_EOF'
#!/usr/bin/expect -f
set timeout 60
set server [lindex $argv 0]
set password [lindex $argv 1]
set app_dir [lindex $argv 2]

# 创建目录
spawn ssh -o StrictHostKeyChecking=no $server "mkdir -p $app_dir"
expect {
    "password:" { send "$password\r"; exp_continue }
    eof
}

# 上传二进制文件
spawn scp skillhub-linux $server:$app_dir/skillhub
expect {
    "password:" { send "$password\r"; exp_continue }
    eof
}

# 上传migrations
spawn scp -r migrations $server:$app_dir/
expect {
    "password:" { send "$password\r"; exp_continue }
    eof
}

# 上传scripts
spawn scp -r scripts $server:$app_dir/
expect {
    "password:" { send "$password\r"; exp_continue }
    eof
}
EXPECT_EOF

chmod +x /tmp/deploy_upload.exp
expect /tmp/deploy_upload.exp "$SERVER" "$PASSWORD" "$APP_DIR"
rm /tmp/deploy_upload.exp

echo "✅ 文件上传完成"
echo ""

# 4. 在服务器上配置和启动
echo "⚙️  配置服务器环境..."

cat > /tmp/deploy_setup.exp << EXPECT_EOF
#!/usr/bin/expect -f
set timeout 120
set server "$SERVER"
set password "$PASSWORD"
set app_dir "$APP_DIR"
set admin_token "$ADMIN_TOKEN"
set llm_api_key "$LLM_API_KEY"
set service_name "$SERVICE_NAME"

spawn ssh -o StrictHostKeyChecking=no \$server bash
expect "password:" { send "\$password\r" }

expect "# " { send "cd \$app_dir\r" }

# 创建.env文件
expect "# " { send "cat > .env << 'ENV_EOF'
# Database
DATABASE_URL=postgresql://skillhub:skillhub_password@localhost:5432/skillhub?sslmode=disable

# Server
PORT=8080
AUTO_MIGRATE=true
SEED_DATA=false

# Admin
ADMIN_TOKEN=\$admin_token

# AI Review
AI_REVIEW_ENABLED=true
LLM_PROVIDER=openrouter
LLM_API_KEY=\$llm_api_key
LLM_MODEL=google/gemma-4-31b-it
LLM_BASE_URL=https://openrouter.ai/api/v1
ENV_EOF
\r" }

# 给二进制文件执行权限
expect "# " { send "chmod +x \$app_dir/skillhub\r" }

# 创建systemd服务
expect "# " { send "cat > /etc/systemd/system/\$service_name.service << 'SERVICE_EOF'
[Unit]
Description=SkillHub API Server
After=network.target postgresql.service

[Service]
Type=simple
User=root
WorkingDirectory=\$app_dir
ExecStart=\$app_dir/skillhub
Restart=always
RestartSec=5
Environment=PATH=/usr/local/bin:/usr/bin:/bin

[Install]
WantedBy=multi-user.target
SERVICE_EOF
\r" }

# 重载systemd并启动服务
expect "# " { send "systemctl daemon-reload\r" }
expect "# " { send "systemctl enable \$service_name\r" }
expect "# " { send "systemctl restart \$service_name\r" }
expect "# " { send "sleep 5\r" }

# 检查服务状态
expect "# " { send "systemctl status \$service_name --no-pager\r" }
expect "# " { send "exit\r" }

expect eof
EXPECT_EOF

chmod +x /tmp/deploy_setup.exp
expect /tmp/deploy_setup.exp
rm /tmp/deploy_setup.exp

echo "✅ 服务配置完成"
echo ""

# 5. 测试健康检查
echo "🏥 测试健康检查..."
sleep 3

# 本地测试
cat > /tmp/health_check.exp << EXPECT_EOF
#!/usr/bin/expect -f
set timeout 30
set server "$SERVER"
set password "$PASSWORD"

spawn ssh -o StrictHostKeyChecking=no \$server "curl -s http://localhost:8080/health"
expect "password:" { send "\$password\r" }
expect eof
EXPECT_EOF

chmod +x /tmp/health_check.exp
HEALTH_RESULT=$(expect /tmp/health_check.exp 2>/dev/null | grep -o '{"ok":true.*}' || echo "")
rm /tmp/health_check.exp

if [[ "$HEALTH_RESULT" == *"\"ok\":true"* ]]; then
    echo "✅ 健康检查通过: $HEALTH_RESULT"
else
    echo "⚠️  健康检查失败，请查看日志"
fi
echo ""

# 6. 显示部署信息
echo "✨ 部署完成！"
echo ""
echo "📊 部署信息:"
echo "   服务器: $SERVER"
echo "   域名: https://$DOMAIN"
echo "   健康检查: https://$DOMAIN/health"
echo "   ADMIN_TOKEN: $ADMIN_TOKEN (已保存到 .admin_token.txt)"
echo ""
echo "📝 查看日志:"
echo "   ssh $SERVER 'journalctl -u $SERVICE_NAME -f'"
echo ""
echo "🔍 测试API:"
echo "   curl https://$DOMAIN/health"
echo "   curl https://$DOMAIN/v1/skills?limit=5"
echo "   curl https://$DOMAIN/v1/bootstrap/discovery"
echo ""
