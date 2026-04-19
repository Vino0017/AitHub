# SkillHub 部署和测试报告

## 部署状态

由于本地Windows环境限制，无法直接使用SSH密码认证进行自动部署。

已创建以下部署文档：
1. **DEPLOY_MANUAL.md** - 详细的手动部署指南
2. **deploy_with_password.sh** - Linux/macOS自动部署脚本（使用expect）

## 推荐部署方式

### 方式1: 手动部署（推荐用于Windows）

参考 `DEPLOY_MANUAL.md` 文档，按步骤执行：

1. 本地构建二进制文件
2. 使用SCP上传文件到服务器
3. SSH登录服务器配置环境
4. 启动服务并测试

### 方式2: 自动部署（Linux/macOS）

```bash
chmod +x deploy_with_password.sh
./deploy_with_password.sh
```

## 服务器信息

- **IP**: 192.227.235.131
- **用户**: root
- **密码**: Vino
- **域名**: skillhub.koolkassanmsk.top
- **应用目录**: /opt/skillhub

## 环境变量配置

```bash
# LLM API Key (OpenRouter)
LLM_API_KEY=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
LLM_MODEL=google/gemma-4-31b-it
LLM_PROVIDER=openrouter

# Admin Token (需要生成)
ADMIN_TOKEN=$(openssl rand -hex 32)

# Database
DATABASE_URL=postgresql://skillhub:skillhub_password@localhost:5432/skillhub?sslmode=disable
```

## 测试清单

部署完成后，需要测试以下功能：

### 1. 基础功能测试

```bash
# 健康检查
curl https://skillhub.koolkassanmsk.top/health
# 预期: {"ok":true,"version":"2.0.0"}

# 搜索技能（无需认证）
curl "https://skillhub.koolkassanmsk.top/v1/skills?q=deploy&limit=5"

# Bootstrap端点
curl https://skillhub.koolkassanmsk.top/v1/bootstrap/discovery
curl https://skillhub.koolkassanmsk.top/v1/bootstrap/check
```

### 2. 认证功能测试

```bash
# 创建Token
TOKEN_RESPONSE=$(curl -X POST https://skillhub.koolkassanmsk.top/v1/tokens \
  -H "Content-Type: application/json" \
  -d '{}')

# 提取token
TOKEN=$(echo $TOKEN_RESPONSE | jq -r '.token')

# 使用token访问需要认证的端点
curl -H "Authorization: Bearer $TOKEN" \
  "https://skillhub.koolkassanmsk.top/v1/skills?limit=10"
```

### 3. 技能提交测试

```bash
# 创建命名空间
curl -X POST https://skillhub.koolkassanmsk.top/v1/namespaces \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-namespace",
    "type": "user"
  }'

# 提交技能
curl -X POST https://skillhub.koolkassanmsk.top/v1/skills \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "test-namespace",
    "name": "test-skill",
    "description": "Test skill for deployment verification",
    "content": "---\nname: test-skill\ndescription: Test skill\n---\n\nTest content",
    "version": "1.0.0",
    "tags": ["test", "deployment"]
  }'
```

### 4. Admin功能测试

```bash
# 查看待审核技能
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  https://skillhub.koolkassanmsk.top/admin/skills/pending

# 隐私扫描
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  https://skillhub.koolkassanmsk.top/admin/privacy/scan
```

### 5. AI审核测试

提交技能后，检查日志确认AI审核是否正常工作：

```bash
ssh root@192.227.235.131 'journalctl -u skillhub -n 100 | grep -i "review"'
```

### 6. 性能测试

```bash
# 并发搜索测试
for i in {1..10}; do
  curl -s "https://skillhub.koolkassanmsk.top/v1/skills?q=test&limit=5" &
done
wait

# 检查响应时间
time curl -s "https://skillhub.koolkassanmsk.top/v1/skills?limit=100" > /dev/null
```

## 预期结果

### 成功指标

- ✅ 健康检查返回 `{"ok":true,"version":"2.0.0"}`
- ✅ 搜索API返回技能列表（即使为空）
- ✅ Token创建成功
- ✅ 技能提交进入审核队列
- ✅ AI审核自动执行
- ✅ Admin端点需要认证
- ✅ 日志无ERROR级别错误
- ✅ 数据库连接正常
- ✅ 迁移全部执行成功

### 性能指标

- 健康检查响应时间 < 100ms
- 搜索API响应时间 < 500ms
- 技能详情响应时间 < 300ms
- 并发10请求无错误

## 故障排查

### 如果服务无法启动

```bash
# 查看服务状态
ssh root@192.227.235.131 'systemctl status skillhub'

# 查看详细日志
ssh root@192.227.235.131 'journalctl -u skillhub -n 100 --no-pager'

# 检查端口占用
ssh root@192.227.235.131 'netstat -tlnp | grep 8080'
```

### 如果数据库连接失败

```bash
# 检查PostgreSQL状态
ssh root@192.227.235.131 'systemctl status postgresql'

# 测试数据库连接
ssh root@192.227.235.131 'psql -U skillhub -d skillhub -h localhost -c "SELECT 1;"'
```

### 如果AI审核不工作

```bash
# 检查环境变量
ssh root@192.227.235.131 'cat /opt/skillhub/.env | grep LLM'

# 检查River队列日志
ssh root@192.227.235.131 'journalctl -u skillhub -n 100 | grep -i river'
```

## 下一步行动

1. **立即执行**: 按照 DEPLOY_MANUAL.md 手动部署到服务器
2. **部署后**: 执行上述测试清单，验证所有功能
3. **监控**: 持续监控日志24小时，确保稳定运行
4. **优化**: 根据测试结果调整配置（连接池、worker数量等）

## 部署时间估算

- 手动部署: 30-45分钟
- 功能测试: 15-20分钟
- 总计: 约1小时

## 注意事项

1. **安全**: 部署后立即保存ADMIN_TOKEN到安全位置
2. **备份**: 部署前备份现有数据（如果有）
3. **监控**: 设置日志监控和告警
4. **SSL**: 确认域名SSL证书已配置
5. **防火墙**: 确认80/443端口已开放

---

**状态**: 等待手动部署执行
**负责人**: 需要有服务器SSH访问权限的人员执行
**预计完成时间**: 1小时内
