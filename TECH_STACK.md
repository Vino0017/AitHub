# SkillHub 技术栈

> 原则：极致性能 + 极致扩展性。只用成熟工具拼接，不造轮子。

---

## 核心运行时

| 层 | 选型 | 版本 | 理由 |
|----|------|------|------|
| **语言** | Go | 1.24 | 单二进制部署、goroutine 百万并发、~0.1ms 延迟、Ollama/Docker 同生态 |
| **HTTP 框架** | chi v5 | 5.x | 标准 net/http 兼容、最小化、可组合中间件、已有代码基础 |
| **数据库** | PostgreSQL | 17 | 100% 头部 AI 项目共识。关系 + 全文搜索 + pgvector 三合一 |
| **数据库驱动** | pgx/v5 + pgxpool | 5.x | Go 最快的 PostgreSQL 驱动，已有代码 |

## 数据层

| 需求 | 选型 | 理由 |
|------|------|------|
| **SQL 代码生成** | sqlc | SQL-first，编译时验证，零运行时开销，类型安全 |
| **数据库迁移** | goose | 简单直接，支持 SQL + Go 迁移，比 golang-migrate 更轻量 |
| **全文搜索** | PostgreSQL tsvector | 内置，不加额外服务 |
| **语义搜索（预留）** | pgvector 扩展 | 预装不强依赖，将来 Skill 量大时开启 |

## 异步处理

| 需求 | 选型 | 理由 |
|------|------|------|
| **任务队列** | River | Go-native，基于 PostgreSQL，**不加 Redis**。原子事务：提交 Skill 和入队审核在同一个 SQL 事务中 |
| **AI 审核** | River Worker | 提交立即返回 pending，后台 Worker 完成审核 |
| **周期任务** | River PeriodicJob | 替代现有 goroutine+ticker（评分刷新、日常重置） |

## AI/LLM 集成

| 需求 | 选型 | 理由 |
|------|------|------|
| **AI 审核** | 保留 internal/llm | 已有 Anthropic + OpenAI 兼容双供应商适配 |
| **Embedding（预留）** | OpenAI text-embedding-3-small 或 Ollama nomic-embed-text | 将来按需启用，通过 interface 解耦 |

## 基础设施

| 需求 | 选型 | 理由 |
|------|------|------|
| **日志** | log/slog | Go 1.21+ 标准库，零依赖，结构化输出 |
| **配置** | godotenv + os.Getenv | 已有，足够简单 |
| **YAML 解析** | gopkg.in/yaml.v3 | SKILL.md frontmatter 解析验证 |
| **CLI 工具（将来）** | cobra | Ollama 在用，行业标准 |
| **容器** | Docker + docker-compose | pgvector/pgvector:pg17 镜像自带 pgvector |
| **反向代理** | nginx（已有）或 Caddy | 已有 nginx.conf |

## 前端

| 需求 | 选型 | 理由 |
|------|------|------|
| **落地页** | 内嵌静态 HTML | AI-First：Agent 不需要前端，给人类一个最简安装入口 |
| **API 文档** | OpenAPI 3.1 YAML（手写） | Agent 可以自动发现 API 结构 |

## 不用的东西（以及为什么）

| 不用 | 为什么 |
|------|--------|
| Redis | River 基于 PostgreSQL，不需要额外消息中间件 |
| Elasticsearch / Meilisearch | PostgreSQL 全文搜索 + 将来 pgvector 足够 |
| GORM / Ent / Bun | sqlc 更轻更快，SQL-first 符合 Go 哲学 |
| Next.js / React | AI-First，不需要复杂前端 |
| Temporal | Postiz 用但太重，River 更轻量更 Go-native |
| golang-migrate | goose 更简单，支持 Go 代码迁移 |
| Prisma / Drizzle | TypeScript 生态，不适用 Go |

---

## 核心架构：Interface 适配器模式

```go
// === 框架适配器 ===
// 新增 Agent 框架 = 实现此接口 + 注册
type FrameworkAdapter interface {
    Name() string                          // "gstack", "openclaw", "hermes"
    InstallCommand(skill Skill) string     // 生成安装 shell 命令
    SkillDir() string                      // 目标目录 ~/.claude/skills/ 等
    Detect() bool                          // 检测本地是否安装此框架
}

// === AI 审核器 ===
// 换 LLM 供应商 = 实现此接口
type Reviewer interface {
    Review(ctx context.Context, content string) (ReviewResult, error)
}

// === 搜索引擎 ===
// 先全文搜索，将来加语义搜索 = 实现此接口
type Searcher interface {
    Search(ctx context.Context, query string, opts SearchOpts) ([]SkillSummary, int, error)
}

// === 内容存储 ===
// 先 PostgreSQL 存内容，将来换 S3 / R2 = 实现此接口
type ContentStore interface {
    Put(ctx context.Context, id string, content []byte) error
    Get(ctx context.Context, id string) ([]byte, error)
    Delete(ctx context.Context, id string) error
}

// === Embedding 生成器（预留） ===
// 将来启用语义搜索时实现
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
}
```

---

## 依赖清单（go.mod）

```
github.com/go-chi/chi/v5          # HTTP 路由
github.com/jackc/pgx/v5           # PostgreSQL 驱动
github.com/riverqueue/river        # 异步任务队列
github.com/pressly/goose/v3        # 数据库迁移
github.com/joho/godotenv           # 环境变量
gopkg.in/yaml.v3                   # YAML 解析
github.com/google/uuid             # UUID 生成
```

## Docker Compose（完整版）

```yaml
services:
  postgres:
    image: pgvector/pgvector:pg17   # 自带 pgvector 扩展
    environment:
      POSTGRES_DB: skillhub
      POSTGRES_USER: skillhub
      POSTGRES_PASSWORD: ${DB_PASSWORD:-skillhub_dev}
    ports: ["5432:5432"]
    volumes: ["pgdata:/var/lib/postgresql/data"]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U skillhub"]
      interval: 5s

  api:
    build: .
    depends_on:
      postgres: { condition: service_healthy }
    ports: ["8080:8080"]
    environment:
      DATABASE_URL: postgres://skillhub:${DB_PASSWORD:-skillhub_dev}@postgres:5432/skillhub
      ADMIN_TOKEN: ${ADMIN_TOKEN}
      LLM_PROVIDER: ${LLM_PROVIDER:-anthropic}
      LLM_API_KEY: ${LLM_API_KEY}
      LLM_MODEL: ${LLM_MODEL:-claude-sonnet-4-20250514}
    restart: unless-stopped

volumes:
  pgdata:
```
