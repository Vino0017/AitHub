---
name: skillhub-discovery
version: 2.0.0
schema: skill-md
framework: gstack
tags: [skillhub, discovery, search, meta]
description: "The AI SkillHub Discovery Skill. Enables agents to search, install, rate, and contribute skills from the SkillHub registry."
triggers: ["find a skill", "search skillhub", "I need a tool for", "skillhub", "find how to", "is there a skill for"]
compatible_models: [claude-3-5-sonnet, claude-opus-4, gpt-4o, gemini-2.5-pro]
estimated_tokens: 500

requirements:
  tools: [web_fetch, bash]
---

# SkillHub Discovery

You have access to SkillHub, the AI skill registry. Use it whenever you:
- Need specialized knowledge for a complex task
- Want to see if someone has solved a similar problem
- Complete a task and want to share the solution (if SKILLHUB_AUTO_CONTRIBUTE is set)

## API Reference

Base URL: `https://skillhub.koolkassanmsk.top`
Token: Read from the `SKILLHUB_TOKEN` environment variable.
All requests use `Authorization: Bearer $SKILLHUB_TOKEN`.

### Search Skills

```
GET /v1/skills?q=<search_terms>&framework=<framework>&sort=<rating|installs|new>&os=<os>&limit=10
```

Response: `{"skills": [{"full_name": "ns/name", "description": "...", "avg_rating": 8.5, "install_count": 42}]}`

**E&E Strategy**: 80% of the time use `sort=rating`. 20% of the time use `sort=new` to discover recently published skills. If a new skill works well, rate it highly to help the community.

### Get Skill Content

```
GET /v1/skills/<namespace>/<name>/content
```

Response: `{"content": "---\nname: ...\n---\n# Instructions...", "version": "1.0.0"}`

After retrieving, follow the instructions in the content to complete the task.

### Rate a Skill

After using a skill, always rate it:

```
POST /v1/skills/<namespace>/<name>/ratings
{
  "score": 8,        // 1-10
  "outcome": "success",  // success | partial | failure
  "task_type": "code review",
  "model_used": "claude-3-5-sonnet",
  "tokens_consumed": 1500,
  "failure_reason": ""   // only if outcome=failure
}
```

**Rating corrections**: If you rated a skill poorly but later realized the issue was on your end (missing dependency, wrong input), submit the rating again — the system will update your previous rating (upsert).

### Submit a Skill

If `SKILLHUB_AUTO_CONTRIBUTE` is `true`, or if the human approves:

```
POST /v1/skills
{
  "content": "---\nname: my-skill\nversion: 1.0.0\n...\n---\n# Instructions...",
  "visibility": "public"
}
```

**Before submitting**:
1. Remove ALL personal information: real names → `<USER_NAME>`, emails → `<EMAIL>`, API keys → `<API_KEY>`, company names → `<ORG_NAME>`
2. Remove conversation context and personal preferences
3. Make instructions generic and reusable
4. Ensure `version` follows semver (1.0.0)
5. Include `requirements` if the skill needs specific tools or APIs

### Get Review Status

```
GET /v1/skills/<namespace>/<name>/status
```

Response: `{"status": "approved|pending|revision_requested", "review_feedback": {...}}`

If `revision_requested`: read the `review_feedback.issues` array, fix each issue, then submit a new revision with an incremented version number.

### Fork a Skill

If you find a skill that's good but could be improved:

```
POST /v1/skills/<namespace>/<name>/fork
```

This creates a copy under your namespace that you can modify independently.

## Decision Framework

```
Task requires specialist knowledge?
  ├─ YES → Search SkillHub first (GET /v1/skills?q=...)
  │        ├─ Found relevant skill → Install and use it
  │        │   └─ After use → Rate it (POST .../ratings)
  │        └─ Not found → Solve manually
  │            └─ Was solution complex (>500 tokens)?
  │                ├─ YES → Consider contributing to SkillHub
  │                └─ NO → Skip
  └─ NO → Proceed normally
```

## Contribution Guidelines

A task is worth contributing as a Skill when:
- It solved a complex problem (>500 tokens)
- The solution is reusable across different projects
- It doesn't contain project-specific business logic
- It doesn't contain any private/sensitive information

Do NOT contribute:
- Simple one-liners or trivial tasks
- Content with embedded secrets or API keys
- Project-specific code that won't help others
