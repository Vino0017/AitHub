# SkillHub — AI-First 核心设计哲学

> **这是产品的根本指导原则。所有功能、格式、API 设计决策均从此出发。**

---

## 核心洞察

> **未来软件最大的用户不是人类，而是 AI。**

GitHub 是人类协作写代码的地方。  
npm 是人类发布、人类安装代码包的地方。  
SkillHub 是 **AI Agent 自主发现、安装、评价能力的地方**。

人类在整个系统中只做**一件事**：

```bash
# 把 SkillHub Discovery Skill 放进 Agent 框架
bash <(curl -fsSL https://skillhub.io/install) skillhub
```

之后的一切——搜索、评估、安装、反馈——**由 AI 自己完成**。

---

## 哲学对比

| 维度 | 传统注册表（npm / GitHub） | SkillHub（AI-First） |
|------|--------------------------|---------------------|
| **主要用户** | 人类开发者 | AI Agent |
| **发现方式** | 人类搜索 + 浏览 | Agent 自主查询 |
| **安装决策** | 人类决定装什么 | Agent 根据任务自动选择 |
| **反馈机制** | 人类写 review | Agent 执行后提交结构化评分 |
| **内容格式** | 对人易读（长 README，Markdown 渲染） | 对 AI 低 token、高信息密度 |
| **UI 优先级** | Web UI 是核心 | API 是核心，Web UI 是次要 |
| **错误信息** | 人类友好的自然语言 | 机器可读错误码 + 最短必要描述 |
| **文档风格** | 营销性、详细叙述 | 结构化、精确、无冗余 |

---

## 六大设计原则

### 原则 1：AI 优先，Token 最小化

每一个从 SkillHub 返回给 Agent 的响应，都应该：

- **信息密度最大**：去掉所有对 AI 判断无用的字段
- **格式可预测**：AI 不需要猜结构
- **无营销语言**：描述是机器可读的功能摘要，不是卖点
- **按需加载**：列表不返回 content，只有 `/skills/:id/content` 才返回全文

```json
// ❌ 人类友好的响应（冗余）
{
  "message": "Here are the skills matching your search!",
  "data": { "results": [...], "pagination": { "current_page": 1 } },
  "status": "success"
}

// ✅ AI-First 响应（精确）
{
  "skills": [...],
  "total": 47,
  "limit": 20,
  "offset": 0
}
```

### 原则 2：人类只做一件事

```
人类执行一次 → Agent 永久自治
```

安装 Discovery Skill 后，Agent 拥有的能力：
- 描述任务 → 自动搜索匹配 Skill
- 评估候选 → 自主选择最优
- 执行安装 → 无需人类介入
- 使用后评分 → 自动提交结构化反馈

### 原则 3：自主发现（Agent 不靠关键词，靠任务描述）

```
Agent："我需要做代码安全审查"
  → GET /v1/skills?q=security+code+review&framework=gstack&sort=rating&limit=5
  → 返回列表（含 avg_rating、install_count、outcome_success_rate）
  → Agent 自主判断：选 avg_rating 高 + success_rate 高的
  → GET /v1/skills/:id/install
  → 执行命令
```

不需要人类参与决策的任何环节。

### 原则 4：机器可读格式，优于人类易读格式

**目标 SKILL.md 格式（AI-First）**

不是给人读的 README，是给 AI 执行的指令集：

```yaml
---
name: code-review
version: 1.2.0
framework: gstack
tags: [code-quality, security, review]
description: "Reviews diffs for security vulnerabilities and quality issues. Returns structured P0-P3 findings."
triggers: ["review", "check this code", "security audit", "qa"]
compatible_models: [claude-3-5-sonnet, claude-opus-4, gpt-4o]
estimated_tokens: 800
---
```

关键字段：

| 字段 | 作用 |
|------|------|
| `triggers` | Agent 框架自动路由，无需人类写路由规则 |
| `compatible_models` | Agent 知道自己能否使用 |
| `estimated_tokens` | Agent 在 token budget 紧时可跳过 |
| `description` | 精确功能摘要，无营销语言 |

### 原则 5：Agent 评分，结构化反馈

当前评分 API 已经是 AI-First 的，需要增加：

```json
{
  "score": 9,
  "outcome": "success",
  "task_type": "security-audit",
  "model_used": "claude-opus-4",
  "tokens_consumed": 643
}
```

新增字段：
- `model_used`：不同模型对 Skill 兼容性不同，是重要数据
- `tokens_consumed`：实际消耗 token 数（效率指标）

### 原则 6：即插即用，对 Agent 完全透明

```
Agent 工作时
  → 自动感知可用 Skill（从 ~/.claude/skills/ 读取）
  → 匹配任务触发词
  → 调用对应 Skill 执行
  → 完成后自动评分
```

整个过程不需要人类参与，不需要人类了解细节。

---

## API 重设计（AI-First 视角）

### GET /v1/skills — Agent 的主要入口

响应设计目标：
- `limit` 默认 5，Agent 不需要翻页
- 含 `outcome_success_rate` 字段（Agent 选 Skill 的核心决策信号）
- 含 `install_count`（信任信号，等同 npm 下载量）
- **不含** `content`（避免 token 浪费）

```json
{
  "skills": [
    {
      "id": "uuid",
      "name": "code-review",
      "description": "Reviews code for security and quality. Returns structured P0-P3 findings.",
      "framework": "gstack",
      "tags": ["security", "code-quality"],
      "version": "1.2.0",
      "avg_rating": 8.4,
      "rating_count": 142,
      "install_count": 1893,
      "outcome_success_rate": 0.87
    }
  ],
  "total": 3,
  "limit": 5,
  "offset": 0
}
```

### GET /v1/skills/:id — Agent 评估详情

加入 `outcome_distribution` 和 `top_task_types`，让 Agent 做决策：

```json
{
  "id": "uuid",
  "name": "code-review",
  "version": "1.2.0",
  "avg_rating": 8.4,
  "install_count": 1893,
  "outcome_distribution": {
    "success": 0.87,
    "partial": 0.09,
    "failure": 0.04
  },
  "top_task_types": ["security-audit", "pr-review", "code-quality"],
  "compatible_models": ["claude-3-5-sonnet", "claude-opus-4", "gpt-4o"],
  "estimated_tokens": 800
}
```

### GET /v1/skills/:id/install — Agent 拿命令直接执行

```json
{
  "command": "mkdir -p ~/.claude/skills/code-review && curl -s https://skillhub.io/v1/skills/uuid/content > ~/.claude/skills/code-review/SKILL.md",
  "framework": "gstack",
  "content_url": "https://skillhub.io/v1/skills/uuid/content"
}
```

---

## 人类的完整操作路径

```
步骤 1（人类）：
  bash <(curl -fsSL https://skillhub.io/install) skillhub

步骤 2（自动）：
  SkillHub Discovery Skill → ~/.claude/skills/skillhub/SKILL.md

步骤 3（Agent 自治，永久）：
  遇到任务 → 查询 SkillHub → 选 Skill → 安装 → 使用 → 评分
```

**人类不需要：**
- 知道 SkillHub API 文档
- 浏览 Web UI 搜索 Skill
- 决定安装哪个 Skill
- 手动提交评分

---

## 架构优先级（从此哲学推导）

### 必须要有（AI-First 核心）

| 功能 | 原因 |
|------|------|
| 高质量全文搜索 | Agent 能找到对的 Skill |
| `outcome_success_rate` | Agent 选 Skill 的核心信号 |
| `install_count` 统计 | 信任信号 |
| `compatible_models` 字段 | Agent 知道能否使用 |
| 异步 AI 审核 | 提交不阻塞 |
| Semver 版本管理 | Agent 可锁定版本 |
| 机器可读错误码 | Agent 能程序化处理错误 |
| SKILL.md 格式规范 + 验证 | 格式一致性，Agent 可靠解析 |

### 低优先级（人类才需要）

| 功能 | 理由 |
|------|------|
| 精美 Web UI | Agent 不用浏览器 |
| 复杂账号/OAuth | Agent 用 Token 即可 |
| 评论系统 | 结构化评分更好 |
| 个人主页/社交 | 人类才需要 |
| Stars 收藏 | 次要，install_count 更有意义 |

### 必须存在但可以简单（人类层）

- 最简 Web 页（人类确认 Skill 存在，看安装量）
- Token 申请页（人类注册获得 Token）
- 管理后台（人类审核恶意 Skill）

---

## 与 GitHub/npm 的根本区别

```
GitHub：人类 → 提交代码 → 人类 → 发现 → 人类 → 使用
npm：    人类 → 发布包   → 人类 → 安装 → 人类 → 使用

SkillHub：
  人类 → 提交 Skill（一次性）
  AI   → 发现 Skill（自主）
  AI   → 安装 Skill（自主）
  AI   → 使用 Skill（自主）
  AI   → 评分反馈（自主）
  系统 → 优胜劣汰（自动进化）
```

这是一个 **Agent 驱动的自进化能力注册表**。

人类的角色：贡献高质量 Skill。  
系统的角色：让最好的 Skill 自动浮到顶端。  
AI 的角色：发现、使用、反馈，形成飞轮。

---

## 产品 Slogan

推荐：
> **"The registry AI agents use themselves."**

备选：
> `"AI agents don't wait for humans to configure them."`  
> `"npm for AI agents. Operated by AI agents."`

---

## 任何功能决策的判断标准

> **这个功能的主要受益者是 AI 还是人类？**
> - 如果是 AI → 立即做，这是核心
> - 如果是人类 → 做最简版，能推迟就推迟

---

*本文档是 SkillHub 所有设计讨论的起点和审核标准。*
