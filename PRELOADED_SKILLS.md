# Preloaded Skill Collections - Quick Reference

## Overview

SkillHub now includes curated collections from the best Claude Code skill repositories in the community. These collections provide instant access to thousands of production-ready skills.

## Installation

```bash
# Download all popular skill collections
./scripts/preload_popular_skills.sh
```

This will download ~356MB of skills into `skills/preloaded/`.

## Featured Collections

### 1. gstack (14MB)
**Author**: Garry Tan (Y Combinator CEO)
**Stars**: 66,000+
**Skills**: 23 specialist skills
**Focus**: Structured engineering team roles

**Key Skills**:
- `/office-hours` - YC-style product review
- `/ship` - Complete deployment workflow
- `/qa` - Systematic testing
- `/design-review` - UI/UX polish
- `/investigate` - Root cause analysis

**Repository**: https://github.com/garrytan/gstack

---

### 2. awesome-claude-skills (172KB)
**Author**: travisvn
**Type**: Curated collection
**Focus**: Resources & tools directory

**What's Inside**:
- Comprehensive skill directory
- Best practices and guides
- Tool recommendations
- Community resources

**Repository**: https://github.com/travisvn/awesome-claude-skills

---

### 3. claude-code-skills (22MB)
**Author**: levnikolaevich
**Type**: Production-ready workflow
**Focus**: Full delivery pipeline

**Coverage**:
- Research & discovery
- Epic planning
- Task breakdown
- Implementation
- Testing
- Code review
- Quality gates

**Repository**: https://github.com/levnikolaevich/claude-code-skills

---

### 4. claude-skills (9.8MB)
**Author**: Jeffallan
**Skills**: 65 specialized skills
**Focus**: Full-stack development

**Categories**:
- Frontend (React, Vue, Angular)
- Backend (Node, Python, Go)
- Database (SQL, NoSQL)
- DevOps (Docker, K8s, CI/CD)
- Testing (Unit, Integration, E2E)

**Repository**: https://github.com/Jeffallan/claude-skills

---

### 5. claude-code-plugins-plus-skills (310MB)
**Author**: jeremylongshore
**Skills**: 1,367 agent skills
**Plugins**: 340 plugins
**Focus**: Massive ecosystem

**Features**:
- CCPI package manager
- Interactive tutorials
- Production orchestration patterns
- Largest collection available

**Repository**: https://github.com/jeremylongshore/claude-code-plugins-plus-skills

---

## Directory Structure

```
skills/preloaded/
├── gstack/                              # 23 specialist skills
│   ├── office-hours/
│   ├── ship/
│   ├── qa/
│   └── ...
├── awesome-claude-skills/               # Curated directory
│   └── README.md
├── claude-code-skills/                  # Production workflow
│   ├── research/
│   ├── planning/
│   ├── implementation/
│   └── ...
├── jeffallan-claude-skills/             # 65 full-stack skills
│   ├── frontend/
│   ├── backend/
│   ├── database/
│   └── ...
└── claude-code-plugins-plus-skills/     # 1367 skills + 340 plugins
    ├── skills/
    ├── plugins/
    └── tutorials/
```

---

## Usage

### Browse Skills

```bash
# List all collections
ls -la skills/preloaded/

# Browse gstack skills
ls skills/preloaded/gstack/

# View a specific skill
cat skills/preloaded/gstack/office-hours/SKILL.md
```

### Import to SkillHub

Coming soon: Automated import tool to upload selected skills to your SkillHub instance.

### Use with Claude Code

Skills in these collections follow the standard SKILL.md format and can be used directly with Claude Code:

```bash
# Copy a skill to your Claude skills directory
cp -r skills/preloaded/gstack/office-hours ~/.claude/skills/

# Or symlink for easy updates
ln -s $(pwd)/skills/preloaded/gstack/office-hours ~/.claude/skills/
```

---

## Statistics

| Collection | Size | Skills | Stars | Focus |
|------------|------|--------|-------|-------|
| gstack | 14MB | 23 | 66K+ | Team roles |
| awesome-claude-skills | 172KB | N/A | N/A | Directory |
| claude-code-skills | 22MB | ~50 | N/A | Workflow |
| claude-skills | 9.8MB | 65 | N/A | Full-stack |
| claude-code-plugins-plus-skills | 310MB | 1,367 | N/A | Ecosystem |
| **Total** | **356MB** | **~1,500+** | **66K+** | **All** |

---

## Acknowledgments

Huge thanks to:
- **Garry Tan** - For open-sourcing gstack and inspiring AI-first development
- **travisvn** - For maintaining the definitive curated list
- **levnikolaevich** - For production-ready workflow skills
- **Jeffallan** - For comprehensive full-stack coverage
- **jeremylongshore** - For the largest skill ecosystem
- **Anthropic** - For creating Claude and the Agent Skills standard
- **The entire Claude Code community** - For building and sharing

---

## Resources

- [Anthropic's Agent Skills Documentation](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills)
- [Best Claude Skills for Coding (2026)](https://www.toolsforhumans.ai/skills/coding)
- [Claude Code Skills Ecosystem](https://botmonster.com/posts/awesome-claude-code-ecosystem-agentic-skills/)
- [Top 50 Claude Skills and GitHub Repos](https://www.blockchain-council.org/claude-ai/top-50-claude-skills-and-github-repos/)

---

## Next Steps

1. **Explore**: Browse the preloaded collections
2. **Test**: Try skills with Claude Code
3. **Import**: Upload your favorites to SkillHub (coming soon)
4. **Contribute**: Share your own skills with the community

---

**Questions?** Open an issue or join the discussion.
