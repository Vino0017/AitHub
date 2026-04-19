# Preloaded Skill Collections - Quick Reference

## Overview

SkillHub integrates with the **highest-quality** Claude Code skill repositories. Only collections with significant community adoption (10K+ stars) are included.

## Installation

```bash
# Download all high-quality skill collections
./scripts/preload_popular_skills.sh
```

This will download the 3 featured collections into `skills/preloaded/`.

## Featured Collections (168K+ ⭐ Combined)

### 1. gstack by Garry Tan
**66K+ ⭐ | 23 specialist skills | Y Combinator CEO's setup**

Garry Tan's personal Claude Code configuration. Transforms your AI into a structured engineering team with specialist roles.

**Key Skills**:
- **Office Hours**: YC-style product review and startup advice
- **Ship**: Complete deployment workflow with canary checks
- **QA**: Systematic testing and bug detection
- **Design Review**: Designer's eye for UI/UX polish
- **Investigate**: Root cause analysis for debugging
- **Plan Reviews**: CEO, Engineering Manager, and Designer perspectives

**Repository**: https://github.com/garrytan/gstack

---

### 2. Everything Claude Code
**100K+ ⭐ | 28 agents + 119 skills + 60 commands | Largest ecosystem**

The most comprehensive Claude Code configuration framework. 100,000 stars and growing.

**What's Included**:
- **28 Specialized Agents**: Full-stack, DevOps, Security, Data Science
- **119 Production Skills**: Battle-tested workflows
- **60 Commands**: Instant productivity boosters
- **Complete Framework**: Ready-to-use configuration

**Repository**: https://github.com/cline/everything-claude-code

---

### 3. Agency Agents by msitarzewski
**2K+ ⭐ | 112 specialized AI personas | Domain experts**

Transforms Claude Code into 112 specialized domain experts. Each persona has deep expertise in specific areas.

**Persona Categories**:
- **Engineering**: Senior Dev, Architect, DevOps, Security
- **Product**: PM, Designer, UX Researcher
- **Business**: Marketing, Sales, Analytics
- **Creative**: Writer, Editor, Content Strategist
- **Data**: Data Scientist, ML Engineer, Analyst

**Repository**: https://github.com/msitarzewski/agency-agents

---

## Directory Structure

```
skills/preloaded/
├── gstack/                              # 23 specialist skills
│   ├── office-hours/
│   ├── ship/
│   ├── qa/
│   └── ...
├── everything-claude-code/              # 28 agents + 119 skills + 60 commands
│   ├── agents/
│   ├── skills/
│   ├── commands/
│   └── ...
└── agency-agents/                       # 112 specialized AI personas
    ├── engineering/
    ├── product/
    ├── business/
    └── ...
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

| Collection | Stars | Skills/Agents | Focus |
|------------|-------|---------------|-------|
| gstack | 66K+ | 23 | Team roles |
| Everything Claude Code | 100K+ | 28 + 119 + 60 | Complete framework |
| Agency Agents | 2K+ | 112 | Domain experts |
| **Total** | **168K+** | **342+** | **All** |

---

## Acknowledgments

Huge thanks to:
- **Garry Tan** - For open-sourcing gstack and inspiring AI-first development
- **Cline Team** - For Everything Claude Code, the most comprehensive Claude Code framework
- **msitarzewski** - For Agency Agents, 112 specialized AI personas
- **Anthropic** - For creating Claude and the Agent Skills standard
- **The entire Claude Code community** - For building and sharing

---

## Resources

- [Anthropic's Agent Skills Documentation](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills)
- [Best Claude Skills for Coding (2026)](https://www.toolsforhumans.ai/skills/coding)
- [Everything Claude Code Explained](https://www.augmentcode.com/learn/everything-claude-code-github)
- [Top 50 Claude Skills and GitHub Repos](https://www.blockchain-council.org/claude-ai/top-50-claude-skills-and-github-repos/)

---

## Next Steps

1. **Explore**: Browse the preloaded collections
2. **Test**: Try skills with Claude Code
3. **Import**: Upload your favorites to SkillHub (coming soon)
4. **Contribute**: Share your own skills with the community

---

**Questions?** Open an issue or join the discussion.
