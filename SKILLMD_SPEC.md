# SKILL.md 格式规范 v2

## 概述

每个 Skill 由一个 `SKILL.md` 文件组成，包含 YAML frontmatter（元数据）和 Markdown body（指令内容）。

## 结构

```text
---
<YAML frontmatter>
---

<Markdown body - the actual agent instructions>
```

## YAML Frontmatter 字段

### 必填字段

| 字段 | 类型 | 规则 | 示例 |
|------|------|------|------|
| `name` | string | 3-100字符, kebab-case (`^[a-z0-9][a-z0-9-]{1,98}[a-z0-9]$`) | `code-review` |
| `version` | string | semver 格式 | `1.0.0` |
| `framework` | string | 框架标识 | `claude-code`, `gstack`, `cursor` |
| `description` | string | 简要描述 | `"Reviews code for security..."` |
| `tags` | string[] | 至少1个 | `[security, review]` |

### 可选字段

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `schema` | string | `skill-md` | 格式类型: `skill-md` \| `mcp-tool` |
| `triggers` | string[] | `[]` | 触发 Skill 的关键词/短语 |
| `compatible_models` | string[] | `[]` | 兼容的 LLM 模型列表 |
| `estimated_tokens` | int | `0` | 预估 token 消耗 |
| `requirements` | object | `null` | 运行依赖（见下文） |

### requirements 结构

```yaml
requirements:
  tools:               # 需要的 Agent 工具权限
    - bash
    - read
    - write
    - web_fetch
  software:            # 需要的外部软件
    - name: docker
      check_command: "docker --version"
      install_url: "https://docs.docker.com/get-docker/"
      optional: false
    - name: kubectl
      check_command: "kubectl version --client"
      install_url: "https://kubernetes.io/docs/tasks/tools/"
      optional: true
  apis:                # 需要的 API 密钥
    - name: "OpenAI API"
      env_var: "OPENAI_API_KEY"
      obtain_url: "https://platform.openai.com/api-keys"
      purpose: "Used for code analysis"
      optional: false
  platform:            # 平台兼容性
    os: [linux, darwin, windows]
    arch: [amd64, arm64]
```

## 完整示例

```markdown
---
name: docker-deploy
version: 2.1.0
schema: skill-md
framework: gstack
tags: [deployment, docker, devops, containerization]
description: "Builds and deploys containerized applications with multi-stage Dockerfiles."
triggers: ["deploy", "dockerize", "containerize", "create dockerfile"]
compatible_models: [claude-3-5-sonnet, gpt-4o]
estimated_tokens: 1200
requirements:
  tools: [bash, write, read]
  software:
    - name: docker
      check_command: "docker --version"
      install_url: "https://docs.docker.com/get-docker/"
      optional: false
    - name: docker-compose
      check_command: "docker compose version"
      install_url: "https://docs.docker.com/compose/install/"
      optional: true
  platform:
    os: [linux, darwin, windows]
---

# docker-deploy

You are an expert at containerizing and deploying applications using Docker.

## When to Use

Use this skill when the user wants to:
- Create a Dockerfile for their project
- Set up Docker Compose for multi-service applications
- Optimize Docker image sizes
- Deploy containers to a registry

## Steps

1. **Analyze** the project structure to determine the tech stack
2. **Generate** a multi-stage Dockerfile optimized for size and security
3. **Create** docker-compose.yml if multiple services are detected
4. **Build** and test locally
5. **Push** to registry if credentials are available

## Best Practices

- Always use multi-stage builds to minimize image size
- Pin base image versions (e.g., `node:20-alpine`, not `node:latest`)
- Copy package files first for better layer caching
- Use `.dockerignore` to exclude unnecessary files
- Run as non-root user in production
- Include health checks in docker-compose
```

## MCP Tool Schema (预留)

当 `schema: mcp-tool` 时，body 部分应为 JSON Schema 格式的 MCP tool 定义：

```yaml
---
name: web-search-tool
version: 1.0.0
schema: mcp-tool
framework: any
tags: [mcp, search, web]
description: "MCP-compatible web search tool"
---
```

Body 应包含 MCP tool 的 JSON Schema 定义。此格式目前为预留状态，将在 MCP 规范稳定后正式支持。

## 隐私清洗规则

提交前必须执行以下替换：

| 原始内容 | 替换为 |
|----------|--------|
| 真实姓名 | `<USER_NAME>` |
| 邮箱地址 | `<EMAIL>` |
| API Key / Secret | `<API_KEY>` |
| 公司/组织名 | `<ORG_NAME>` |
| IP 地址 | `<IP_ADDRESS>` |
| 文件绝对路径 | 相对路径或 `<PATH>` |
| 数据库连接串 | `<DATABASE_URL>` |

提交时的 Regex 预扫描会检测上述泄漏，发现后自动返回 `revision_requested` 而非拒绝。

## 版本控制规则

- 版本号必须遵循 [Semantic Versioning](https://semver.org/)
- 同一 Skill 的同一版本号不可重复提交（`UNIQUE(skill_id, version)`）
- 版本冲突时 API 返回 `409 version_exists`
- 修复后应使用新版本号（如 `1.0.0` → `1.0.1`）
