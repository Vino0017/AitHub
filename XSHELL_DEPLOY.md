# SkillHub 快速部署命令（在Xshell中执行）

## 1. 连接服务器
```bash
# 在Xshell中新建会话
# Host: 192.227.235.131
# User: root
# Password: Vino
```

## 2. 检查当前状态
```bash
systemctl status skillhub
journalctl -u skillhub -n 20 --no-pager
curl http://localhost:8080/health
```

## 3. 更新代码（如果有Git仓库）
```bash
cd /opt/skillhub
git pull  # 如果是Git部署
# 或者手动上传新文件
```

## 4. 重新构建（如果需要）
```bash
cd /opt/skillhub
go build -o skillhub cmd/api/main.go
```

## 5. 更新环境变量
```bash
cd /opt/skillhub

# 生成新的ADMIN_TOKEN
ADMIN_TOKEN=$(openssl rand -hex 32)
echo "New ADMIN_TOKEN: $ADMIN_TOKEN"

# 更新.env文件
cat > .env << 'EOF'
DATABASE_URL=postgresql://skillhub:skillhub_password@localhost:5432/skillhub?sslmode=disable
PORT=8080
AUTO_MIGRATE=true
SEED_DATA=false
ADMIN_TOKEN=YOUR_ADMIN_TOKEN_HERE
AI_REVIEW_ENABLED=true
LLM_PROVIDER=openrouter
LLM_API_KEY=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
LLM_MODEL=google/gemma-4-31b-it
LLM_BASE_URL=https://openrouter.ai/api/v1
EOF

# 替换ADMIN_TOKEN
sed -i "s/YOUR_ADMIN_TOKEN_HERE/$ADMIN_TOKEN/" .env

# 保存token
echo "$ADMIN_TOKEN" > /root/skillhub_admin_token.txt
chmod 600 /root/skillhub_admin_token.txt
```

## 6. 重启服务
```bash
systemctl restart skillhub
sleep 3
systemctl status skillhub --no-pager
```

## 7. 测试新功能
```bash
# 健康检查
curl http://localhost:8080/health

# Bootstrap端点（新功能）
curl http://localhost:8080/v1/bootstrap/discovery | head -20
curl http://localhost:8080/v1/bootstrap/check

# 创建token
curl -X POST http://localhost:8080/v1/tokens -H "Content-Type: application/json" -d '{}'

# 搜索（需要token）
TOKEN="your_token_here"
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/skills?limit=5"
```

## 8. 查看日志
```bash
# 实时日志
journalctl -u skillhub -f

# 最近100行
journalctl -u skillhub -n 100 --no-pager

# 查找错误
journalctl -u skillhub -n 500 | grep -i error
```

## 快速一键部署（复制整段执行）
```bash
cd /opt/skillhub && \
ADMIN_TOKEN=$(openssl rand -hex 32) && \
echo "ADMIN_TOKEN: $ADMIN_TOKEN" && \
echo "$ADMIN_TOKEN" > /root/skillhub_admin_token.txt && \
cat > .env << EOF
DATABASE_URL=postgresql://skillhub:skillhub_password@localhost:5432/skillhub?sslmode=disable
PORT=8080
AUTO_MIGRATE=true
SEED_DATA=false
ADMIN_TOKEN=$ADMIN_TOKEN
AI_REVIEW_ENABLED=true
LLM_PROVIDER=openrouter
LLM_API_KEY=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
LLM_MODEL=google/gemma-4-31b-it
LLM_BASE_URL=https://openrouter.ai/api/v1
EOF
systemctl restart skillhub && \
sleep 3 && \
systemctl status skillhub --no-pager && \
curl http://localhost:8080/health
```
