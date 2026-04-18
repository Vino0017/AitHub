# SkillHub 开发计划

> 按模块拆解，每个模块标注：涉及文件、依赖关系、验证方式。
> 预估基于单人全职开发。

---

## M1：基础架构（预估 2-3 天）

### M1.1 修复 Go 版本 + 依赖更新

```
涉及文件：
  [MODIFY] go.mod                    → go 1.26.2 改为 go 1.24
  [MODIFY] Dockerfile               → golang:1.26-alpine 改为 golang:1.24-alpine

新增依赖：
  github.com/pressly/goose/v3       # 数据库迁移
  github.com/riverqueue/river        # 异步任务
  gopkg.in/yaml.v3                   # YAML frontmatter 解析
  golang.org/x/oauth2                # GitHub/Google OAuth

验证：go build ./... 编译通过
```

### M1.2 Docker Compose 完整版 + 开箱即用

```
涉及文件：
  [MODIFY] docker-compose.yml       → 加 postgres 服务，用 pgvector/pgvector:pg17 镜像
  [NEW]    .env.example              → 环境变量模板
  [NEW]    scripts/docker-entrypoint.sh → API 容器启动时自动 goose up + 种子数据
  [NEW]    scripts/seed.sql           → 预置 3-5 个示范 Skill（带评分）

开箱即用目标：
  docker-compose up → 自动迁移 → 自动种子数据 → API 就绪
  新开发者 clone 后一条命令即可拥有完整运行环境

验证：docker-compose up → pg_isready 通过 → curl /v1/skills 返回种子 Skill
```

### M1.3 Goose 迁移体系

```
涉及文件：
  [DELETE] migrations/001_initial.sql                    → 旧迁移删除
  [NEW]    migrations/001_create_namespaces.sql
  [NEW]    migrations/002_create_tokens.sql
  [NEW]    migrations/003_create_skills.sql
  [NEW]    migrations/004_create_revisions.sql
  [NEW]    migrations/005_create_ratings.sql
  [NEW]    migrations/006_create_org_members.sql
  [NEW]    migrations/007_create_indexes.sql

表结构（对应 BLUEPRINT 第七章）：

  namespaces:
    id UUID PK, name TEXT UNIQUE, type TEXT (personal|org),
    github_id TEXT, google_id TEXT, email TEXT, created_at TIMESTAMPTZ

  org_members:
    org_id UUID FK→namespaces, member_id UUID FK→namespaces,
    role TEXT (owner|member), joined_at TIMESTAMPTZ

  tokens:
    id UUID PK, namespace_id UUID FK→namespaces (nullable, 匿名时为空),
    token_hash TEXT UNIQUE, label TEXT, daily_uses INT DEFAULT 0,
    last_used TIMESTAMPTZ, created_at TIMESTAMPTZ

  skills:
    id UUID PK, namespace_id UUID FK→namespaces,
    name TEXT, description TEXT, tags TEXT[], framework TEXT,
    ⚠️ framework 是纯 TEXT 列，不用 CHECK 约束也不用 ENUM 类型
    ⚠️ 校验在应用层（Go 代码）做，新增框架只改代码不改 DB
    visibility TEXT (public|private|org), forked_from UUID FK→skills,
    install_count INT DEFAULT 0, avg_rating NUMERIC(4,2),
    rating_count INT DEFAULT 0, outcome_success_rate NUMERIC(4,3),
    latest_version TEXT, fork_count INT DEFAULT 0,
    status TEXT (active|yanked|removed), created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
    UNIQUE(namespace_id, name)

  revisions:
    id UUID PK, skill_id UUID FK→skills,
    version TEXT, content TEXT, change_summary TEXT,
    author_token_id UUID FK→tokens,
    review_status TEXT (pending|approved|revision_requested|rejected),
    review_feedback JSONB, review_result JSONB,
    review_retry_count INT DEFAULT 0,     # 熔断：上限 3 次
    triggers TEXT[], compatible_models TEXT[],
    estimated_tokens INT, requirements JSONB,
    platform JSONB,                        # {os: [...], arch: [...]}
    created_at TIMESTAMPTZ
    UNIQUE(skill_id, version)              # 防止版本号冲突

  ratings:
    id UUID PK, skill_id UUID FK→skills, revision_id UUID FK→revisions,
    token_id UUID FK→tokens, score INT, outcome TEXT,
    task_type TEXT, model_used TEXT, tokens_consumed INT,
    failure_reason TEXT, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
    UNIQUE(revision_id, token_id)           # 同 Token 同版本只能 1 条评分（upsert）

前置依赖：无
验证：goose up 成功，goose down 回滚成功，所有表存在
```

### M1.4 sqlc 配置 + 查询生成

```
涉及文件：
  [NEW]    sqlc.yaml
  [NEW]    queries/namespaces.sql     → CRUD for namespaces
  [NEW]    queries/tokens.sql         → CRUD for tokens
  [NEW]    queries/skills.sql         → 搜索、详情、创建、更新
  [NEW]    queries/revisions.sql      → 创建、列表、按版本获取
  [NEW]    queries/ratings.sql        → 创建、聚合查询
  [NEW]    queries/org_members.sql    → 增删查成员
  [GENERATED] internal/sqlgen/        → sqlc 自动生成的代码

前置依赖：M1.3（需要 schema 定义）
验证：sqlc generate 成功，生成代码编译通过
```

### M1.5 核心数据模型

```
涉及文件：
  [MODIFY] internal/models/models.go → 重写，匹配新 schema

前置依赖：M1.4
验证：模型与 sqlc 生成类型兼容
```

---

## M2：注册与身份（预估 2 天）

### M2.1 匿名 Token 自动发放

```
涉及文件：
  [MODIFY] internal/handler/token.go  → POST /v1/tokens（创建匿名 Token）
  [MODIFY] internal/middleware/auth.go → 解析 Bearer Token，注入 namespace 信息

路径覆盖：H1（安装时自动获取）
前置依赖：M1
验证：curl -X POST /v1/tokens → 返回 token，用该 token 访问需认证 API 成功
```

### M2.2 GitHub OAuth Device Flow

```
涉及文件：
  [NEW]    internal/handler/auth_github.go  → POST /v1/auth/github
  [NEW]    internal/auth/github.go          → GitHub Device Flow 实现

流程：
  1. POST /v1/auth/github → 返回 {device_code, user_code, verification_uri}
  2. 客户端展示 user_code → 人类去 GitHub 授权
  3. POST /v1/auth/github/poll {device_code} → 轮询 → 返回 {namespace, token}

路径覆盖：H2
前置依赖：M2.1
验证：完整 OAuth 流程跑通，namespace 创建成功
```

### M2.3 Google OAuth Device Flow

```
涉及文件：
  [NEW]    internal/handler/auth_google.go  → POST /v1/auth/google
  [NEW]    internal/auth/google.go          → Google Device Flow 实现

结构同 M2.2，复用 auth 接口。

前置依赖：M2.1
验证：Google OAuth 流程跑通
```

### M2.4 邮箱注册

```
涉及文件：
  [NEW]    internal/handler/auth_email.go   → POST /v1/auth/email/send, /verify
  [NEW]    internal/email/sender.go         → 邮件发送抽象（interface）
  [NEW]    internal/email/smtp.go           → SMTP 实现
  [NEW]    internal/email/resend.go         → Resend API 实现（可选）

路径覆盖：H2（邮箱方式）
前置依赖：M2.1
验证：发送验证码 → 输入验证码 → namespace 创建成功
```

### M2.5 namespace_required 拦截

```
涉及文件：
  [MODIFY] internal/middleware/auth.go → 写操作检查 namespace，无则返回 403

路径覆盖：路径 5（触发注册）
前置依赖：M2.1
验证：匿名 Token 调 POST /v1/skills → 403 namespace_required
```

---

## M3：核心 Skill API（预估 3-4 天）

### M3.1 搜索 API

```
涉及文件：
  [NEW]    internal/handler/skill_search.go → GET /v1/skills
  [NEW]    internal/search/searcher.go      → Searcher interface
  [NEW]    internal/search/fulltext.go      → PostgreSQL 全文搜索实现

参数：q, framework, tag, visibility, os, arch,
      sort (rating|installs|recent|trending|new), limit, offset
可见性过滤：public + 当前 Token 有权限的 private/org
平台过滤：Agent 传入自己的 os 和 arch，过滤不兼容的 Skill
E&E 探索：sort=new 返回近 7 天新发布的 Skill，解决冷启动问题

路径覆盖：路径 1, 路径 8
前置依赖：M1, M2.1
验证：
  - 搜索返回正确结果
  - private Skill 对非 owner 不可见
  - org Skill 对非成员不可见
  - os=linux 过滤掉 os: [darwin] 的 Skill
  - sort=new 返回近期新 Skill
```

### M3.2 Skill 详情 API

```
涉及文件：
  [NEW]    internal/handler/skill_detail.go → GET /v1/skills/:namespace/:name

返回：完整 Skill 信息 + requirements + fork 信息
权限检查：visibility 过滤

路径覆盖：路径 1 步骤 4, 路径 7
前置依赖：M3.1
验证：请求详情返回完整字段
```

### M3.3 Skill 内容/安装 API

```
涉及文件：
  [NEW]    internal/handler/skill_content.go → GET /v1/skills/:namespace/:name/content
                                             → GET /v1/skills/:namespace/:name/install

content：返回 SKILL.md 全文，install_count+1
install：按 framework 生成安装 shell 命令

涉及 framework 适配器：
  [NEW]    internal/framework/adapter.go    → FrameworkAdapter interface
  [NEW]    internal/framework/gstack.go
  [NEW]    internal/framework/openclaw.go
  [NEW]    internal/framework/hermes.go
  [NEW]    internal/framework/claudecode.go
  [NEW]    internal/framework/cursor.go
  [NEW]    internal/framework/windsurf.go

路径覆盖：路径 1 步骤 5
前置依赖：M3.2
验证：GET content → 返回 SKILL.md，install_count 递增
```

### M3.4 Skill 提交 API

```
涉及文件：
  [NEW]    internal/handler/skill_submit.go  → POST /v1/skills
  [NEW]    internal/skillformat/validate.go  → YAML frontmatter 解析验证

frontmatter 必填字段：name, version, framework, description, tags
frontmatter 推荐字段：triggers, requirements, compatible_models, estimated_tokens
frontmatter 预留字段：schema (支持 "skill-md" | "mcp-tool"，默认 "skill-md")
  ⚠️ schema: mcp-tool 预留给 MCP 标准工具定义，让 Skill 不仅是指令仓库
     也可以是符合 JSON Schema 的标准 Tool 函数，直接喂给各大模型

流程：
  1. 验证 Token 有 namespace（否则 403 namespace_required）
  2. 解析 YAML frontmatter → 验证必填字段
  3. 提取 triggers, requirements, compatible_models, estimated_tokens, schema
  4. 写入 skills 表 + revisions 表（status=pending）
  5. 入队 River 审核任务（M4 就绪前用 goroutine fallback：5s 超时则直接 pending）
  6. 返回 {"id": "...", "status": "pending"}

路径覆盖：路径 3
前置依赖：M2.5
验证：提交 Skill → DB 中 skill + revision 存在 → status=pending
```

### M3.5 Skill 状态 API

```
涉及文件：
  [NEW]    internal/handler/skill_status.go → GET /v1/skills/:namespace/:name/status

返回：最新 revision 的 review_status + review_feedback（退回原因）

路径覆盖：路径 4 步骤 1-2
前置依赖：M3.4
验证：查询 pending/approved/revision_requested 状态
```

### M3.6 评分 API

```
涉及文件：
  [NEW]    internal/handler/rating.go → POST /v1/skills/:namespace/:name/ratings

字段：score, outcome, task_type, model_used, tokens_consumed, failure_reason
写入逻辑：
  - UPSERT：同 token_id + revision_id 已存在 → 更新（解决误评修正）
  - 不存在 → 新增
  - 权重：检查 token.namespace_id
    ├─ 有 namespace → 计入 avg_rating
    └─ 无（匿名） → 存储但不计入排名
  - avg_rating 仅基于最新 revision 的注册用户评分重算

路径覆盖：路径 2, 路径 10
前置依赖：M3.2
验证：
  - 提交评分 → avg_rating 更新
  - 同 Token 再次评分 → 覆盖而非新增
  - 匿名 Token 评分 → 不影响 avg_rating
  - v1 评分不影响 v2 的 avg_rating
```

### M3.7 Owner 自行下架 API

```
涉及文件：
  [NEW]    internal/handler/skill_yank.go → DELETE /v1/skills/:ns/:name
                                          → PATCH /v1/skills/:ns/:name

逻辑：
  - 验证 Token 是此 Skill 的 owner（同 namespace）
  - DELETE → status = yanked，从搜索结果消失
  - PATCH {status: "active"} → 恢复上线

路径覆盖：Owner 紧急拉闸
前置依赖：M3.2
验证：yank 后搜索不可见，恢复后可见
```

---

## M4：异步审核（预估 2 天）

### M4.1 River 集成

```
涉及文件：
  [MODIFY] cmd/api/main.go            → 初始化 River client + worker
  [MODIFY] internal/db/db.go           → 连接池配置适配 River
  [NEW]    internal/worker/setup.go    → River worker 注册

前置依赖：M1
验证：River tables 自动创建，worker 启动无报错
```

### M4.2 AI 审核 Worker（两层架构）

```
涉及文件：
  [NEW]    internal/worker/ai_review.go     → AIReviewWorker
  [NEW]    internal/review/reviewer.go      → Reviewer interface
  [NEW]    internal/review/llm_reviewer.go  → LLM 实现
  [NEW]    internal/review/regex_scanner.go  → 正则预扫描（快速拦截层）

审核两层架构（左移安全检测，省 LLM token）：

  第一层：Regex 预扫描（<1ms，零成本）
    基于 gitleaks 规则集的 Go 正则扫描器：
    - AWS Key: AKIA[0-9A-Z]{16}
    - GitHub Token: ghp_[a-zA-Z0-9]{36}
    - OpenAI Key: sk-[a-zA-Z0-9]{40,}
    - Private Key: -----BEGIN (RSA|EC|DSA|OPENSSH) PRIVATE KEY-----
    - 通用密码模式: password\s*[:=]\s*["'][^"']+["']
    如果正则命中：直接 revision_requested + 具体行号和类型
    不调 LLM，省时省钱。

  第二层：LLM 深度审核（正则未命中时才走）
    1. 恶意命令检测（rm -rf、反弹 shell、数据外泄、挖矿）
    2. 隐私泄露检测（正则漏掉的上下文相关隐私）
    3. 格式检测（frontmatter 完整性）
    4. 质量检测（内容是否有实质指令）

审核结果：
  - approved → 更新 status，更新 skill.latest_version
  - revision_requested → 存入结构化 review_feedback + 具体 issues
  - rejected → 严重恶意，存入原因

路径覆盖：路径 3 末尾, 路径 4
前置依赖：M4.1, M3.4, 保留 internal/llm/llm.go
验证：
  - 提交含 API key 的 Skill → regex 层直接 revision_requested（不调 LLM）
  - 提交含隐晦隐私的 Skill → LLM 层检出 → revision_requested
  - 提交正常 Skill → approved
  - 提交含 rm -rf 的 Skill → rejected
```

### M4.3 周期任务

```
涉及文件：
  [NEW]    internal/worker/rating_refresh.go → 每小时重算所有 rating
  [NEW]    internal/worker/daily_reset.go    → 每日重置 daily_uses
  [NEW]    internal/worker/trending.go       → 每小时计算 trending 得分

前置依赖：M4.1
验证：定时任务按时触发，rating 聚合正确
```

---

## M5：版本与 Fork（预估 2 天）

### M5.1 Revision 历史 API

```
涉及文件：
  [NEW]    internal/handler/revision.go → GET /v1/skills/:ns/:name/revisions
                                        → GET /v1/skills/:ns/:name/revisions/:version
                                        → POST /v1/skills/:ns/:name/revisions

POST 逻辑：
  - 验证 Token 是此 Skill 的 owner（同 namespace）
  - 版本号必须大于当前 latest_version
  - 创建新 revision → 入队审核

路径覆盖：路径 9
前置依赖：M3.4, M4.2
验证：提交新 revision → 审核通过 → latest_version 更新
```

### M5.2 Fork API

```
涉及文件：
  [NEW]    internal/handler/fork.go → POST /v1/skills/:ns/:name/fork
                                    → GET /v1/skills/:ns/:name/forks

逻辑：
  - 复制 latest revision 的 content
  - 在当前 Token 的 namespace 下创建新 Skill
  - forked_from 指向原 Skill ID
  - 原 Skill fork_count+1

路径覆盖：路径 6
前置依赖：M3.4
验证：Fork → 新 Skill 创建 → forked_from 正确 → 原 Skill fork_count 增加
```

---

## M6：安装脚本 + Discovery Skill（预估 2 天）

### M6.1 bash 安装脚本

```
涉及文件：
  [MODIFY] scripts/install-skill.sh → 完全重写

功能：
  1. 检测 OS (uname)
  2. 检测已安装框架（which claude, which openclaw 等）
  3. 解析命令行参数（--register, --github, --google, --email）
  4. 创建/升级 Token
  5. 下载 Discovery Skill 到各框架目录
  6. 写入 SKILLHUB_TOKEN 到 shell 配置
  7. 注册流程（如果带参数）

路径覆盖：H1, H2
前置依赖：M2, M3
验证：全新环境运行脚本 → Token 获取 → Discovery Skill 安装到正确位置
```

### M6.2 PowerShell 安装脚本

```
涉及文件：
  [NEW]    scripts/install-skill.ps1

功能同 M6.1，用 PowerShell 语法。
检测框架：Test-Path, Get-Command
写入变量：[Environment]::SetEnvironmentVariable

路径覆盖：H1, H2（Windows）
前置依赖：M2, M3
验证：Windows 环境运行脚本 → 全部通过
```

### M6.3 卸载脚本

```
涉及文件：
  [NEW]    scripts/uninstall.sh
  [NEW]    scripts/uninstall.ps1

路径覆盖：H5
前置依赖：M6.1, M6.2
验证：运行后 Discovery Skill 删除，环境变量清除
```

### M6.4 Discovery Skill v2

```
涉及文件：
  [MODIFY] skills/skillhub/SKILL.md → 按 BLUEPRINT 第六章完全重写

内容：
  - 搜索指令
  - 自动贡献指令 + 判断标准
  - 隐私清洗规则
  - 评分指令
  - namespace_required 错误处理指令
  - API 快速参考

路径覆盖：所有 AI 路径的基础
前置依赖：M3（API 都可用时）
验证：AI Agent 读取 SKILL.md 后能完成路径 1-10 的全部操作
```

### M6.5 安装脚本 API 端点

```
涉及文件：
  [NEW]    internal/handler/web.go → GET / (落地页)
                                   → GET /install (bash 脚本)
                                   → GET /install.ps1 (PowerShell 脚本)
                                   → GET /uninstall (bash 卸载)
                                   → GET /uninstall.ps1 (PowerShell 卸载)

前置依赖：M6.1, M6.2, M6.3
验证：curl https://.../install → 返回有效 bash 脚本
```

---

## M7：组织与可见性（预估 1-2 天）

### M7.1 Namespace 管理 API

```
涉及文件：
  [NEW]    internal/handler/namespace.go → POST /v1/namespaces（创建 org）
                                         → GET /v1/namespaces/:name
                                         → POST /v1/namespaces/:name/members
                                         → DELETE /v1/namespaces/:name/members/:id

孤儿组织防护：
  - 不允许最后一个 owner 退出 org
  - 退出前必须 transfer owner 给另一成员
  - 或解散整个 org（所有 org skill 转 private 或删除）

路径覆盖：H3
前置依赖：M2
验证：
  - 创建 org → 添加成员 → 成员 Token 能访问 org Skill
  - 最后一个 owner 尝试退出 → 403 拒绝
```

### M7.2 可见性过滤

```
涉及文件：
  [MODIFY] internal/search/fulltext.go   → 搜索时加 visibility 过滤
  [MODIFY] internal/handler/skill_detail.go → 详情时检查权限
  [MODIFY] internal/handler/skill_content.go → 内容获取时检查权限

逻辑：
  - public → 所有人可见
  - private → 仅 skill.namespace_id = token.namespace_id
  - org → token.namespace_id 是 org 成员

路径覆盖：路径 8
前置依赖：M7.1
验证：
  - private Skill 对非 owner 返回 403
  - org Skill 对非成员返回 403
  - org Skill 对成员正常返回
```

---

## M8：收尾与打磨（预估 2 天）

### M8.1 Trending API

```
涉及文件：
  [NEW]    internal/handler/trending.go → GET /v1/trending

逻辑：过去 7 天 install_count 增长最多的 Skill

前置依赖：M4.3（trending 计算任务）
验证：有安装数据后 trending 返回正确排序
```

### M8.2 Related Skills API

```
涉及文件：
  [NEW]    internal/handler/related.go → GET /v1/skills/:ns/:name/related

逻辑：同 tag 或同 framework 的其他 Skill

前置依赖：M3.1
验证：返回相关 Skill 列表
```

### M8.3 静态落地页

```
涉及文件：
  [NEW]    web/index.html → 最简 HTML 落地页

内容：
  - 项目一句话介绍
  - 安装命令（可复制）
  - GitHub 链接
  - API 文档链接

前置依赖：M6.5
验证：GET / → 返回渲染正确的 HTML
```

### M8.4 OpenAPI 规范

```
涉及文件：
  [NEW]    docs/openapi.yaml → 手写完整 OpenAPI 3.1 规范

覆盖：所有 API 端点
用途：Agent 自动发现 API 结构

前置依赖：所有 API 完成
验证：openapi-generator validate 通过
```

### M8.5 健康检查

```
涉及文件：
  [NEW]    internal/handler/health.go → GET /health, GET /ready

health：DB 连接 + River worker 状态
ready：Kubernetes readiness probe

前置依赖：M1
验证：服务健康时 200，DB 挂时 503
```

### M8.6 完善 README

```
涉及文件：
  [MODIFY] README.md → 完全重写

内容：
  - 项目定位（AI 的 GitHub）
  - 一键安装命令
  - API 概览
  - Self-hosting 部署指南
  - 贡献指南

前置依赖：全部完成
```

### M8.7 SKILL.md 格式规范文档

```
涉及文件：
  [NEW]    docs/SKILL_FORMAT.md → YAML frontmatter 字段完整定义

前置依赖：M3.4
```

---

## 主路由注册（贯穿所有 M）

```
涉及文件：
  [MODIFY] cmd/api/main.go → 路由注册总览

r := chi.NewRouter()

// 中间件
r.Use(middleware.Logger)
r.Use(middleware.RateLimit)

// 系统
r.Get("/", handler.LandingPage)
r.Get("/install", handler.InstallScript)
r.Get("/install.ps1", handler.InstallScriptPS)
r.Get("/uninstall", handler.UninstallScript)
r.Get("/uninstall.ps1", handler.UninstallScriptPS)
r.Get("/health", handler.Health)
r.Get("/ready", handler.Ready)
r.Get("/openapi.yaml", handler.OpenAPI)

// 认证（无需 Token）
r.Post("/v1/auth/github", handler.AuthGitHub)
r.Post("/v1/auth/github/poll", handler.AuthGitHubPoll)
r.Post("/v1/auth/google", handler.AuthGoogle)
r.Post("/v1/auth/google/poll", handler.AuthGooglePoll)
r.Post("/v1/auth/email/send", handler.AuthEmailSend)
r.Post("/v1/auth/email/verify", handler.AuthEmailVerify)

// Token（匿名创建）
r.Post("/v1/tokens", handler.CreateToken)

// 需要 Token 的 API
r.Group(func(r chi.Router) {
    r.Use(middleware.RequireToken)

    // Token 管理
    r.Get("/v1/tokens", handler.ListTokens)
    r.Delete("/v1/tokens/{id}", handler.DeleteToken)

    // Namespace
    r.Post("/v1/namespaces", handler.CreateNamespace)
    r.Get("/v1/namespaces/{name}", handler.GetNamespace)
    r.Post("/v1/namespaces/{name}/members", handler.AddOrgMember)
    r.Delete("/v1/namespaces/{name}/members/{memberId}", handler.RemoveOrgMember)

    // Skill 发现
    r.Get("/v1/skills", handler.SearchSkills)
    r.Get("/v1/trending", handler.Trending)
    r.Get("/v1/skills/{namespace}/{name}", handler.GetSkill)
    r.Get("/v1/skills/{namespace}/{name}/content", handler.GetSkillContent)
    r.Get("/v1/skills/{namespace}/{name}/install", handler.GetInstallCommand)
    r.Get("/v1/skills/{namespace}/{name}/status", handler.GetSkillStatus)
    r.Get("/v1/skills/{namespace}/{name}/related", handler.GetRelatedSkills)
    r.Get("/v1/skills/{namespace}/{name}/forks", handler.ListForks)
    r.Get("/v1/skills/{namespace}/{name}/revisions", handler.ListRevisions)
    r.Get("/v1/skills/{namespace}/{name}/revisions/{version}", handler.GetRevision)

    // Skill 写操作（需要 namespace）
    r.Group(func(r chi.Router) {
        r.Use(middleware.RequireNamespace)

        r.Post("/v1/skills", handler.SubmitSkill)
        r.Post("/v1/skills/{namespace}/{name}/ratings", handler.SubmitRating)
        r.Post("/v1/skills/{namespace}/{name}/fork", handler.ForkSkill)
        r.Post("/v1/skills/{namespace}/{name}/revisions", handler.SubmitRevision)
    })

    // Admin
    r.Group(func(r chi.Router) {
        r.Use(middleware.RequireAdmin)
        r.Post("/admin/skills/{id}/remove", handler.RemoveSkill)
        r.Post("/admin/namespaces/{id}/ban", handler.BanNamespace)
    })
})
```

---

## 依赖关系图

```
M1（基础架构）
 ├── M2（注册与身份）
 │    ├── M3（核心 Skill API）
 │    │    ├── M4（异步审核）
 │    │    │    └── M5（版本与 Fork）
 │    │    └── M6（安装脚本 + Discovery Skill）
 │    └── M7（组织与可见性）
 └── M8（收尾打磨）← 依赖一切
```

---

## 文件总数估算

| 类型 | 数量 |
|------|------|
| 新增 Go 文件 | ~35 |
| 修改 Go 文件 | ~5 |
| SQL 迁移文件 | 7 |
| SQL 查询文件 | 6 |
| 脚本文件 | 5 |
| 配置文件 | 3 |
| 文档文件 | 4 |
| HTML 文件 | 1 |
| **总计** | **~66 个文件** |

---

## 总预估

| 阶段 | 预估 | 累计 |
|------|------|------|
| M1 基础架构 | 2-3 天 | 2-3 天 |
| M2 注册身份 | 2 天 | 4-5 天 |
| M3 核心 API | 3-4 天 | 7-9 天 |
| M4 异步审核 | 2 天 | 9-11 天 |
| M5 版本 Fork | 2 天 | 11-13 天 |
| M6 安装脚本 | 2 天 | 13-15 天 |
| M7 组织可见性 | 1-2 天 | 14-17 天 |
| M8 收尾打磨 | 2 天 | 16-19 天 |
| **总计** | **16-19 天** | |

---

*每个模块开始前先看 BLUEPRINT.md 对应章节，结束后验证对应的 AI/人类路径。*
