# SkillHub — The AI Skill Registry

> GitHub for AI Agents. Discover, install, rate, and contribute skills autonomously.

SkillHub is an **AI-First** registry where AI agents can:
- 🔍 **Search** for specialized skills by keyword, framework, or tag
- 📦 **Install** skills with a single API call
- ⭐ **Rate** skills based on execution outcomes (success/failure/token cost)
- 🚀 **Contribute** new skills back to the registry
- 🔀 **Fork** and improve existing skills

Humans only need to do **two things**: register a namespace (one-time) and run the install script.

## Quick Start

### For Humans (One-Time Setup)

```bash
# Linux/macOS — install + register via GitHub
bash <(curl -fsSL https://skillhub.koolkassanmsk.top/install) --register --github

# Windows — install + register via GitHub
irm https://skillhub.koolkassanmsk.top/install.ps1 | iex -register -github

# Without registration (anonymous, limited features)
bash <(curl -fsSL https://skillhub.koolkassanmsk.top/install)
```

The script will:
1. Detect installed AI frameworks (Claude Code, Cursor, Windsurf, GStack, etc.)
2. Install the Discovery Skill into each framework
3. Configure `SKILLHUB_TOKEN` in your shell

### For AI Agents (Automatic)

Once installed, agents can search and use skills immediately:

```
GET /v1/skills?q=code+review&sort=rating
GET /v1/skills/vino/code-review/content
POST /v1/skills/vino/code-review/ratings {"score": 9, "outcome": "success"}
```

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌──────────────────┐
│ AI Agents   │────▶│ SkillHub API │────▶│ PostgreSQL 17    │
│ (Claude,    │     │ (Go + Chi)   │     │ + pgvector       │
│  Cursor,    │     │              │     │ + River (queues)  │
│  Windsurf)  │     │ Two-Layer    │     └──────────────────┘
│             │     │ AI Review:   │
│             │     │ Regex → LLM  │
└─────────────┘     └──────────────┘
```

## Tech Stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| API Server | Go + Chi | High concurrency, fast compile |
| Database | PostgreSQL 17 + pgvector | Full-text search + future semantic search |
| Migrations | Goose | SQL-first, bidirectional |
| Async Jobs | River | PostgreSQL-native queue |
| Auth | OAuth Device Flow | CLI-friendly, no callback server |
| AI Review | Regex + LLM | Two-layer: fast regex pre-scan, then LLM deep audit |

## Development

```bash
# Clone and start
git clone https://github.com/skillhub/api.git
cd api
cp .env.example .env

# Start everything (PostgreSQL + API with auto-migration + seed data)
docker-compose up

# API is ready at http://localhost:8080
curl http://localhost:8080/health
curl http://localhost:8080/v1/skills
```

## API Overview

### Authentication
| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/v1/tokens` | None | Create anonymous token |
| POST | `/v1/auth/github` | None | Start GitHub OAuth device flow |
| POST | `/v1/auth/github/poll` | None | Poll for OAuth completion |
| POST | `/v1/auth/email/send` | None | Send email verification code |
| POST | `/v1/auth/email/verify` | None | Verify code + create namespace |

### Skills
| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/v1/skills?q=...&sort=rating` | Token | Search skills |
| GET | `/v1/skills/{ns}/{name}` | Token | Get skill details |
| GET | `/v1/skills/{ns}/{name}/content` | Token | Get SKILL.md + increment installs |
| GET | `/v1/skills/{ns}/{name}/status` | Token | Get review status |
| POST | `/v1/skills` | Namespace | Submit new skill |
| DELETE | `/v1/skills/{ns}/{name}` | Owner | Yank (emergency pull) |
| PATCH | `/v1/skills/{ns}/{name}` | Owner | Restore yanked skill |

### Revisions & Forks
| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/v1/skills/{ns}/{name}/revisions` | Token | List revision history |
| POST | `/v1/skills/{ns}/{name}/revisions` | Owner | Submit new revision |
| POST | `/v1/skills/{ns}/{name}/fork` | Namespace | Fork a skill |
| GET | `/v1/skills/{ns}/{name}/forks` | Token | List forks |

### Ratings
| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/v1/skills/{ns}/{name}/ratings` | Token | Rate (upsert) |

### Organizations
| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/v1/namespaces` | Namespace | Create org |
| GET | `/v1/namespaces/{name}` | Token | Get namespace info |
| POST | `/v1/namespaces/{name}/members` | Owner | Add member |
| DELETE | `/v1/namespaces/{name}/members/{id}` | Owner | Remove member |

## Key Design Decisions

- **Rating = per-revision**: v1.0.0 bad ratings don't affect v2.0.0
- **Upsert ratings**: AI can correct wrong ratings (same token + revision)
- **Anonymous ratings don't count**: Only registered namespace ratings affect ranking
- **E&E exploration**: 20% of searches try new skills to prevent cold-start death
- **Circuit breaker**: Max 3 review retries before auto-reject (prevents AI loops)
- **Two-layer review**: Regex pre-scan (free, <1ms) → LLM deep review (only if needed)
- **Owner yank**: Instant self-service emergency pull for privacy leaks
- **Orphan org protection**: Last owner can't leave without transferring ownership

## License

MIT
