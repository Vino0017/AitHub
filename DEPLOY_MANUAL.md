# SkillHub 手动部署指南

## 服务器信息
- IP: 192.227.235.131
- 用户: root
- 密码: Vino
- 域名: skillhub.koolkassanmsk.top

## 环境变量配置

### LLM API Key (用于AI审核)
```bash
export LLM_API_KEY="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"
export ADMIN_TOKEN="$(openssl rand -hex 32)"
```

## 部署步骤

### 1. 本地构建
```bash
# 在项目根目录执行
GOOS=linux GOARCH=amd64 go build -o skillhub-linux cmd/api/main.go
```

### 2. 上传文件到服务器
```bash
# 创建目录
ssh root@192.227.235.131 "mkdir -p /opt/skillhub"

# 上传二进制文件
scp skillhub-linux root@192.227.235.131:/opt/skillhub/skillhub

# 上传迁移文件
scp -r migrations root@192.227.235.131:/opt/skillhub/

# 上传脚本
scp -r scripts root@192.227.235.131:/opt/skillhub/
```

### 3. 在服务器上配置环境变量
```bash
ssh root@192.227.235.131

cd /opt/skillhub

# 创建 .env 文件
cat > .env << 'EOF'
# Database
DATABASE_URL=postgresql://skillhub:skillhub_password@localhost:5432/skillhub?sslmode=disable

# Server
PORT=8080
AUTO_MIGRATE=true
SEED_DATA=false

# Admin (生成强随机token)
ADMIN_TOKEN=YOUR_STRONG_RANDOM_TOKEN_HERE

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
EOF

# 生成强随机ADMIN_TOKEN
ADMIN_TOKEN=$(openssl rand -hex 32)
sed -i "s/YOUR_STRONG_RANDOM_TOKEN_HERE/$ADMIN_TOKEN/" .env

# 保存ADMIN_TOKEN供后续使用
echo "ADMIN_TOKEN: $ADMIN_TOKEN" > /root/skillhub_admin_token.txt
chmod 600 /root/skillhub_admin_token.txt
```

### 4. 配置PostgreSQL数据库
```bash
# 安装PostgreSQL 17 (如果未安装)
apt update
apt install -y postgresql-17 postgresql-contrib-17

# 启动PostgreSQL
systemctl start postgresql
systemctl enable postgresql

# 创建数据库和用户
sudo -u postgres psql << 'EOF'
CREATE USER skillhub WITH PASSWORD 'skillhub_password';
CREATE DATABASE skillhub OWNER skillhub;
GRANT ALL PRIVILEGES ON DATABASE skillhub TO skillhub;
\c skillhub
GRANT ALL ON SCHEMA public TO skillhub;
EOF
```

### 5. 创建systemd服务
```bash
cat > /etc/systemd/system/skillhub.service << 'EOF'
[Unit]
Description=SkillHub API Server
After=network.target postgresql.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/skillhub
ExecStart=/opt/skillhub/skillhub
Restart=always
RestartSec=5
Environment=PATH=/usr/local/bin:/usr/bin:/bin

[Install]
WantedBy=multi-user.target
EOF
```

### 6. 启动服务
```bash
# 给二进制文件执行权限
chmod +x /opt/skillhub/skillhub

# 重载systemd配置
systemctl daemon-reload

# 启用并启动服务
systemctl enable skillhub
systemctl start skillhub

# 等待5秒让服务启动
sleep 5

# 检查服务状态
systemctl status skillhub --no-pager
```

### 7. 检查日志
```bash
# 查看最近的日志
journalctl -u skillhub -n 50 --no-pager

# 实时查看日志
journalctl -u skillhub -f
```

### 8. 测试健康检查
```bash
# 本地测试
curl http://localhost:8080/health

# 远程测试 (如果配置了反向代理)
curl https://skillhub.koolkassanmsk.top/health
```

## 配置Nginx反向代理 (如果需要)

```bash
# 安装Nginx
apt install -y nginx

# 配置反向代理
cat > /etc/nginx/sites-available/skillhub << 'EOF'
server {
    listen 80;
    server_name skillhub.koolkassanmsk.top;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF

# 启用站点
ln -s /etc/nginx/sites-available/skillhub /etc/nginx/sites-enabled/
nginx -t
systemctl reload nginx
```

## 配置SSL证书 (使用Let's Encrypt)

```bash
# 安装certbot
apt install -y certbot python3-certbot-nginx

# 获取证书
certbot --nginx -d skillhub.koolkassanmsk.top

# 自动续期
certbot renew --dry-run
```

## 测试完整功能

### 1. 测试健康检查
```bash
curl https://skillhub.koolkassanmsk.top/health
# 预期: {"ok":true,"version":"2.0.0"}
```

### 2. 测试搜索 (无需认证)
```bash
curl "https://skillhub.koolkassanmsk.top/v1/skills?q=deploy&limit=5"
```

### 3. 创建Token
```bash
curl -X POST https://skillhub.koolkassanmsk.top/v1/tokens \
  -H "Content-Type: application/json" \
  -d '{}'
```

### 4. 测试Bootstrap
```bash
curl https://skillhub.koolkassanmsk.top/v1/bootstrap/discovery
curl https://skillhub.koolkassanmsk.top/v1/bootstrap/check
```

### 5. 测试Admin端点
```bash
# 使用保存的ADMIN_TOKEN
ADMIN_TOKEN=$(cat /root/skillhub_admin_token.txt | cut -d: -f2 | tr -d ' ')

curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  https://skillhub.koolkassanmsk.top/admin/skills/pending
```

## 故障排查

### 服务无法启动
```bash
# 查看详细日志
journalctl -u skillhub -n 100 --no-pager

# 检查端口占用
netstat -tlnp | grep 8080

# 检查数据库连接
sudo -u postgres psql -c "\l" | grep skillhub
```

### 数据库连接失败
```bash
# 测试数据库连接
psql -U skillhub -d skillhub -h localhost -c "SELECT 1;"

# 检查PostgreSQL状态
systemctl status postgresql
```

### 迁移失败
```bash
# 手动运行迁移
cd /opt/skillhub
./skillhub  # 会自动运行迁移 (如果AUTO_MIGRATE=true)
```

## 监控和维护

### 查看服务状态
```bash
systemctl status skillhub
```

### 重启服务
```bash
systemctl restart skillhub
```

### 查看实时日志
```bash
journalctl -u skillhub -f
```

### 数据库备份
```bash
# 备份数据库
sudo -u postgres pg_dump skillhub > /root/skillhub_backup_$(date +%Y%m%d).sql

# 恢复数据库
sudo -u postgres psql skillhub < /root/skillhub_backup_20260419.sql
```

## 完成部署后的验证清单

- [ ] 服务正常运行 (`systemctl status skillhub`)
- [ ] 健康检查通过 (`curl /health`)
- [ ] 数据库连接正常
- [ ] 迁移已执行
- [ ] API端点可访问
- [ ] SSL证书已配置
- [ ] 日志正常输出
- [ ] ADMIN_TOKEN已保存
- [ ] 防火墙规则已配置 (如需要)

## 安全建议

1. 定期更新系统: `apt update && apt upgrade`
2. 配置防火墙: `ufw allow 80/tcp && ufw allow 443/tcp && ufw enable`
3. 定期备份数据库
4. 监控日志异常
5. 定期轮换ADMIN_TOKEN
6. 限制SSH访问 (使用密钥认证)
