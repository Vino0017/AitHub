# SkillHub v2 — 代码库现状文档

> 最后更新: 2026-04-18

## 架构概览

```
cmd/api/main.go           ← 入口: 路由 + 启动 + 周期任务
internal/
├── crypto/                ← 统一加密工具 (token生成/hash)
├── db/                    ← 连接池 + goose迁移 + 种子数据
├── handler/               ← HTTP handlers (每个领域一个文件)
│   ├── admin.go           ← 管理员: 审批/拒绝/移除/封禁/刷新评分
│   ├── auth.go            ← GitHub OAuth Device Flow + 邮箱验证
│   ├── fork.go            ← Fork + 列出forks
│   ├── namespace.go       ← 组织创建/成员管理/孤儿防护
│   ├── rating.go          ← 评分 (upsert + 匿名权重 + 贝叶斯)
│   ├── revision.go        ← 版本历史 + 新版本提交
│   ├── skill_detail.go    ← 详情 + 内容/安装 + 状态
│   ├── skill_search.go    ← 全文搜索 + 可见性 + 排序
│   ├── skill_submit.go    ← 提交 + frontmatter解析
│   ├── skill_yank.go      ← Owner自行下架/恢复
│   ├── token.go           ← Token CRUD
│   └── web.go             ← 落地页 + 安装脚本端点
├── helpers/               ← HTTP工具 (JSON读写/错误响应)
├── llm/                   ← LLM客户端 (Anthropic + OpenAI)
├── middleware/            ← Auth中间件 (token验证/namespace检查/admin)
├── models/                ← 全部实体 + API请求响应类型
├── review/                ← 两层审核系统
│   ├── regex_scanner.go   ← Regex预扫描 (12个密钥模式 + 6个恶意模式)
│   └── reviewer.go        ← 审核协调器 (regex→LLM + 3次熔断)
└── skillformat/           ← YAML frontmatter 解析/验证

migrations/                ← 7个goose迁移文件
queries/                   ← 6个sqlc查询文件 (未generate, 作为参考)
scripts/
├── install.sh             ← Linux/macOS 安装脚本
├── install.ps1            ← Windows PowerShell 安装脚本
├── uninstall.sh           ← 卸载脚本
└── seed.sql               ← 种子数据 (3个示范Skill)
skills/skillhub/SKILL.md   ← Discovery Skill v2
```

## 数据库 Schema

### 7 张核心表

| 表 | 说明 | 关键约束 |
|----|------|---------|
| `namespaces` | 用户/组织 | UNIQUE(name), name格式CHECK |
| `org_members` | 组织成员 | PK(org_id, member_id) |
| `tokens` | API Token | UNIQUE(token_hash), namespace_id可null(匿名) |
| `skills` | 技能仓库 | UNIQUE(namespace_id, name), FTS GIN索引 |
| `revisions` | 版本 | UNIQUE(skill_id, version), review_retry_count |
| `ratings` | 评分 | UNIQUE(revision_id, token_id), score 1-10 |
| `email_verifications` | 邮箱验证码 | 10分钟过期 |
| `oauth_device_flows` | OAuth状态 | 15分钟过期 |

### 关键索引
- `idx_skills_search`: GIN 全文搜索索引 (name + description + tags)
- `idx_skills_rating`: B-tree 评分降序 (WHERE active)
- `idx_skills_installs`: B-tree 安装数降序 (WHERE active)
- `idx_namespaces_github_id`: 部分索引 (WHERE NOT NULL)

## API 端点 (29个)

### 公开端点 (无需token)
- `GET /` 落地页
- `GET /health` 健康检查
- `GET /install` bash安装脚本
- `GET /install.ps1` PowerShell安装脚本
- `GET /uninstall` 卸载脚本
- `POST /v1/tokens` 创建匿名token
- `POST /v1/auth/github` GitHub OAuth开始
- `POST /v1/auth/github/poll` GitHub OAuth轮询
- `POST /v1/auth/email/send` 发送验证码
- `POST /v1/auth/email/verify` 验证邮箱

### 认证端点 (匿名token可用)
- `GET /v1/skills` 搜索
- `GET /v1/skills/{ns}/{name}` 详情
- `GET /v1/skills/{ns}/{name}/content` 内容/安装
- `GET /v1/skills/{ns}/{name}/status` 审核状态
- `GET /v1/skills/{ns}/{name}/revisions` 版本历史
- `GET /v1/skills/{ns}/{name}/revisions/{ver}` 指定版本
- `GET /v1/skills/{ns}/{name}/forks` Fork列表
- `GET /v1/namespaces/{name}` 命名空间详情
- `POST /v1/skills/{ns}/{name}/ratings` 评分 (匿名可提交但不计排名)

### 注册namespace必需
- `GET /v1/tokens` 列出Token
- `DELETE /v1/tokens/{id}` 撤销Token
- `POST /v1/skills` 提交Skill
- `POST /v1/skills/{ns}/{name}/revisions` 提交新版本
- `POST /v1/skills/{ns}/{name}/fork` Fork
- `DELETE /v1/skills/{ns}/{name}` Yank(下架)
- `PATCH /v1/skills/{ns}/{name}` 恢复下架
- `POST /v1/namespaces` 创建组织
- `POST /v1/namespaces/{name}/members` 添加成员
- `DELETE /v1/namespaces/{name}/members/{id}` 移除成员

### 管理端点
- `GET /admin/skills/pending` 待审核列表
- `POST /admin/skills/{id}/approve` 批准
- `POST /admin/skills/{id}/reject` 拒绝
- `POST /admin/skills/{id}/remove` 移除
- `POST /admin/namespaces/{id}/ban` 封禁
- `POST /admin/ratings/refresh` 刷新评分

## 技术栈

| 层 | 技术 |
|----|------|
| 语言 | Go 1.24 |
| 路由 | chi/v5 |
| 数据库 | PostgreSQL 17 + pgvector |
| 连接池 | pgx/v5 |
| 迁移 | goose/v3 |
| 容器 | Docker + docker-compose |
| LLM | Anthropic / OpenAI 兼容 |

## 当前状态

### ✅ 已完成
- 完整数据层 (迁移 + 查询 + 种子数据)
- 全部 HTTP handlers (29个端点)
- 双层审核系统 (regex + LLM + 熔断)
- 评分系统 (upsert + 贝叶斯 + 匿名权重)
- 安装脚本 (bash + PowerShell)
- Discovery Skill v2
- 落地页
- 单元测试 (crypto + skillformat + regex_scanner)

### 🟡 待完成
- `go mod tidy` + 编译验证 (系统无Go, 需Docker构建)
- Google OAuth Device Flow
- River 队列替代 goroutine
- 邮件实际发送 (SMTP/Resend)
- OpenAPI 规范
- Related Skills API
