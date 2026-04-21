# ClawHub Skills Importer

将ClawHub.ai上的OpenClaw技能导入到AitHub平台。

## 关于ClawHub

[ClawHub.ai](https://clawhub.ai) 是OpenClaw的官方技能注册中心，拥有超过13,000个社区贡献的技能。这些技能涵盖开发工具、生产力、通信、智能家居、AI模型集成等领域。

## 技能框架说明

- **OpenClaw**: ClawHub上的所有技能都是为OpenClaw框架设计的
- **Hermes**: 是另一个独立的AI agent框架（Nous Research开发）
- ClawHub的SKILL.md格式**不包含**`framework`字段，导入时会自动添加`framework: openclaw`

## 前置要求

1. 已创建`clawhub` namespace
2. 拥有该namespace的有效token
3. 安装依赖：`jq`, `unzip`, `curl`

## 使用方法

### 1. 创建namespace和token

```bash
# 通过数据库创建namespace
source .env
psql "$DATABASE_URL" -c "INSERT INTO namespaces (name, type) VALUES ('clawhub', 'org') ON CONFLICT (name) DO NOTHING;"

# 通过API创建token
CLAWHUB_TOKEN=$(curl -s -X POST "$DOMAIN/v1/tokens" \
  -H "Content-Type: application/json" \
  -d '{"namespace": "clawhub"}' | jq -r '.token')

# 关联token到namespace
psql "$DATABASE_URL" -c "UPDATE tokens SET namespace_id = (SELECT id FROM namespaces WHERE name = 'clawhub') WHERE token_hash = encode(digest('$CLAWHUB_TOKEN', 'sha256'), 'hex');"

# 保存token
echo "$CLAWHUB_TOKEN" > /tmp/clawhub_token.txt
```

### 2. 运行导入脚本

```bash
# 导入前5个技能（测试）
source .env
export ADMIN_TOKEN=$(cat /tmp/clawhub_token.txt)
./scripts/import_clawhub_v2.sh 5

# 导入前50个技能
./scripts/import_clawhub_v2.sh 50

# 导入前100个技能
./scripts/import_clawhub_v2.sh 100
```

## 导入过程

脚本会自动：

1. 从ClawHub搜索API获取技能列表
2. 下载每个技能的ZIP包
3. 提取SKILL.md文件
4. **自动添加缺失的字段**：
   - `version: 1.0.0` (ClawHub技能没有版本号)
   - `framework: openclaw` (标识为OpenClaw技能)
   - `tags: [openclaw, imported]` (如果没有tags)
5. 提交到AitHub API
6. 处理重复、错误等情况

## 导入统计

导入完成后会显示：

```
=== Import Complete ===
Imported: 45
Skipped: 3
Failed: 2
Total: 50
```

- **Imported**: 成功导入的技能数
- **Skipped**: 已存在的技能（跳过）
- **Failed**: 导入失败的技能（格式错误、验证失败等）

## 常见问题

### Q: 为什么有些技能导入失败？

A: 可能原因：
- SKILL.md格式不符合AitHub规范
- 缺少必需字段（name, description等）
- 技能名称不符合命名规则（必须是kebab-case）
- 内容包含敏感信息被隐私扫描拦截

### Q: 导入的技能会自动审核吗？

A: 是的，所有导入的技能都会经过：
1. 格式验证
2. 隐私扫描（PII检测）
3. 双层AI审核（如果启用）
   - 正则预扫描（<1ms）
   - LLM深度审计

### Q: 如何查看导入的技能？

```bash
# 搜索clawhub namespace的技能
curl "https://aithub.space/v1/skills?namespace=clawhub&limit=20"

# 查看特定技能
curl "https://aithub.space/v1/skills/clawhub/explain-code"

# 获取技能内容
curl "https://aithub.space/v1/skills/clawhub/explain-code/content"
```

### Q: 可以导入所有13,000+技能吗？

A: 技术上可以，但建议：
- 先导入热门技能（按下载量、星标排序）
- 分批导入，避免API限流
- 监控存储空间和数据库性能

## API限流

ClawHub API限制：
- 无认证：180 req/min（读取）
- 有认证：900 req/min（读取）

脚本已内置速率限制（~3 req/sec），符合API要求。

## 技术细节

### ClawHub API端点

- 搜索：`GET https://clawhub.ai/api/v1/search?q=...&limit=N`
- 下载：`GET https://clawhub.ai/api/v1/download?slug=SLUG`
- 详情：`GET https://clawhub.ai/api/v1/skills/SLUG`

### SKILL.md格式转换

ClawHub格式：
```yaml
---
name: python
description: Python coding guidelines
---
```

转换为AitHub格式：
```yaml
---
name: python
description: Python coding guidelines
version: 1.0.0
framework: openclaw
tags: [openclaw, imported]
---
```

## 相关资源

- [ClawHub官网](https://clawhub.ai)
- [OpenClaw GitHub](https://github.com/openclaw/clawhub)
- [ClawHub技能格式文档](https://github.com/openclaw/clawhub/blob/main/docs/skill-format.md)
- [Hermes Agent](https://github.com/nous-research/hermes) - 另一个AI agent框架

## 贡献

欢迎改进导入脚本：
- 更好的错误处理
- 批量导入优化
- 增量更新支持
- 版本同步机制

---

**Sources:**
- [Best ClawHub Skills: A Complete Guide](https://www.datacamp.com/blog/best-clawhub-skills)
- [Centralized Skills Registry for OpenClaw Agents](https://sparkco.ai/blog/clawhub-the-skills-registry-for-openclaw-agents)
- [ClawHub.ai: The official skill store of 220k-star OpenClaw](https://help.apiyi.com/en/clawhub-ai-openclaw-skills-registry-guide-en.html)
- [Hermes Agent Complete Guide](https://kevnu.com/en/posts/hermes-agent-complete-guide-installation-skills-mechanism-and-comparison-with-openclaw)
