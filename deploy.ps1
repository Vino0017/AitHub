# SkillHub 部署脚本 (PowerShell)
# 使用方法: .\deploy.ps1

$SERVER = "root@192.227.235.131"
$PASSWORD = "Vino"
$DOMAIN = "skillhub.koolkassanmsk.top"
$APP_DIR = "/opt/skillhub"
$LLM_API_KEY = "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"

Write-Host "🚀 SkillHub 部署开始..." -ForegroundColor Green
Write-Host ""

# 生成ADMIN_TOKEN
$ADMIN_TOKEN = -join ((48..57) + (97..102) | Get-Random -Count 64 | ForEach-Object {[char]$_})
Write-Host "🔑 生成的 ADMIN_TOKEN: $ADMIN_TOKEN" -ForegroundColor Yellow
$ADMIN_TOKEN | Out-File -FilePath ".admin_token.txt" -Encoding UTF8
Write-Host "   (已保存到 .admin_token.txt)" -ForegroundColor Gray
Write-Host ""

# 检查plink是否可用 (PuTTY的命令行工具)
$plinkPath = Get-Command plink -ErrorAction SilentlyContinue
if (-not $plinkPath) {
    Write-Host "❌ 需要安装 PuTTY (plink)" -ForegroundColor Red
    Write-Host "   下载: https://www.putty.org/" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "或者使用手动部署方式:" -ForegroundColor Yellow
    Write-Host "1. 打开 Xshell 或其他SSH客户端" -ForegroundColor Gray
    Write-Host "2. 连接到 $SERVER (密码: $PASSWORD)" -ForegroundColor Gray
    Write-Host "3. 按照 DEPLOY_MANUAL.md 执行命令" -ForegroundColor Gray
    exit 1
}

Write-Host "✅ 找到 plink，继续部署..." -ForegroundColor Green
Write-Host ""

# 使用plink执行远程命令
Write-Host "📦 在服务器上构建和部署..." -ForegroundColor Cyan

$deployScript = @"
cd /opt/skillhub || mkdir -p /opt/skillhub && cd /opt/skillhub

# 克隆或更新代码
if [ -d .git ]; then
    git pull
else
    git clone https://github.com/yourusername/skillhub.git .
fi

# 构建
go build -o skillhub cmd/api/main.go

# 创建.env
cat > .env << 'EOF'
DATABASE_URL=postgresql://skillhub:skillhub_password@localhost:5432/skillhub?sslmode=disable
PORT=8080
AUTO_MIGRATE=true
SEED_DATA=false
ADMIN_TOKEN=$ADMIN_TOKEN
AI_REVIEW_ENABLED=true
LLM_PROVIDER=openrouter
LLM_API_KEY=$LLM_API_KEY
LLM_MODEL=google/gemma-4-31b-it
LLM_BASE_URL=https://openrouter.ai/api/v1
EOF

# 重启服务
systemctl restart skillhub
sleep 3
systemctl status skillhub --no-pager
"@

# 执行部署
echo $PASSWORD | plink -ssh -batch -pw $PASSWORD $SERVER $deployScript

Write-Host ""
Write-Host "✨ 部署完成！" -ForegroundColor Green
Write-Host ""
Write-Host "📊 测试服务:" -ForegroundColor Cyan
Write-Host "   curl https://$DOMAIN/health" -ForegroundColor Gray

# 测试健康检查
Start-Sleep -Seconds 2
$health = Invoke-RestMethod -Uri "https://$DOMAIN/health" -ErrorAction SilentlyContinue
if ($health.ok) {
    Write-Host "✅ 健康检查通过: version $($health.version)" -ForegroundColor Green
} else {
    Write-Host "⚠️  健康检查失败" -ForegroundColor Yellow
}
