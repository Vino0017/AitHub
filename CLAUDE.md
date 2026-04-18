# SkillHub Project

## gstack

For all web browsing tasks, use the `/browse` skill from gstack. Never use `mcp__claude-in-chrome__*` tools.

### Available gstack skills:

- `/office-hours` - YC Office Hours mode for startup/product review
- `/plan-ceo-review` - CEO/founder-mode plan review
- `/plan-eng-review` - Engineering manager-mode plan review
- `/plan-design-review` - Designer's eye plan review
- `/design-consultation` - Complete design system consultation
- `/design-shotgun` - Generate multiple AI design variants
- `/design-html` - Generate production-quality HTML/CSS
- `/review` - Pre-landing PR review
- `/ship` - Ship workflow (merge, test, review, bump version, changelog)
- `/land-and-deploy` - Land and deploy workflow with canary checks
- `/canary` - Post-deploy canary monitoring
- `/benchmark` - Performance regression detection
- `/browse` - Fast headless browser for QA and testing
- `/connect-chrome` - Launch AI-controlled Chromium browser
- `/qa` - Systematically QA test and fix bugs
- `/qa-only` - Report-only QA testing
- `/design-review` - Designer's eye QA review
- `/setup-browser-cookies` - Import cookies from real browser
- `/setup-deploy` - Configure deployment settings
- `/retro` - Weekly engineering retrospective
- `/investigate` - Systematic debugging with root cause analysis
- `/document-release` - Post-ship documentation update
- `/codex` - OpenAI Codex CLI wrapper
- `/cso` - Chief Security Officer security audit
- `/autoplan` - Auto-review pipeline (CEO, design, eng, DX reviews)
- `/plan-devex-review` - Developer experience plan review
- `/devex-review` - Live developer experience audit
- `/careful` - Safety guardrails for destructive commands
- `/freeze` - Restrict file edits to specific directory
- `/guard` - Full safety mode (destructive warnings + scoped edits)
- `/unfreeze` - Clear freeze boundary
- `/gstack-upgrade` - Upgrade gstack to latest version
- `/learn` - Manage project learnings

## Skill routing

When the user's request matches an available skill, ALWAYS invoke it using the Skill
tool as your FIRST action. Do NOT answer directly, do NOT use other tools first.
The skill has specialized workflows that produce better results than ad-hoc answers.

Key routing rules:
- Product ideas, "is this worth building", brainstorming → invoke office-hours
- Bugs, errors, "why is this broken", 500 errors → invoke investigate
- Ship, deploy, push, create PR → invoke ship
- QA, test the site, find bugs → invoke qa
- Code review, check my diff → invoke review
- Update docs after shipping → invoke document-release
- Weekly retro → invoke retro
- Design system, brand → invoke design-consultation
- Visual audit, design polish → invoke design-review
- Architecture review → invoke plan-eng-review
- Save progress, checkpoint, resume → invoke checkpoint
- Code quality, health check → invoke health
