#!/usr/bin/env bash
# Preload HIGH-QUALITY Claude Code skill collections into SkillHub
# Only includes repositories with significant stars and community adoption

set -euo pipefail

SKILLS_DIR="skills/preloaded"
mkdir -p "$SKILLS_DIR"

echo "🚀 Preloading HIGH-QUALITY Claude Code skills..."
echo ""

# 1. gstack by Garry Tan (66K+ stars) - THE GOLD STANDARD
echo "📦 [1/3] Fetching gstack by Garry Tan (66K+ ⭐)..."
if [ ! -d "$SKILLS_DIR/gstack" ]; then
    git clone --depth 1 https://github.com/garrytan/gstack.git "$SKILLS_DIR/gstack"
    echo "   ✓ gstack cloned (23 specialist skills)"
else
    echo "   ✓ gstack already exists"
fi

# 2. Everything Claude Code (100K+ stars) - MASSIVE ECOSYSTEM
echo "📦 [2/3] Fetching Everything Claude Code (100K+ ⭐)..."
if [ ! -d "$SKILLS_DIR/everything-claude-code" ]; then
    git clone --depth 1 https://github.com/cline/everything-claude-code.git "$SKILLS_DIR/everything-claude-code"
    echo "   ✓ Everything Claude Code cloned (28 agents, 119 skills, 60 commands)"
else
    echo "   ✓ Everything Claude Code already exists"
fi

# 3. Agency Agents by msitarzewski (2K+ stars) - 112 SPECIALIZED PERSONAS
echo "📦 [3/3] Fetching Agency Agents by msitarzewski (2K+ ⭐)..."
if [ ! -d "$SKILLS_DIR/agency-agents" ]; then
    git clone --depth 1 https://github.com/msitarzewski/agency-agents.git "$SKILLS_DIR/agency-agents"
    echo "   ✓ Agency Agents cloned (112 specialized AI personas)"
else
    echo "   ✓ Agency Agents already exists"
fi

echo ""
echo "✅ All HIGH-QUALITY skill collections preloaded!"
echo ""
echo "📊 Summary:"
echo "   • gstack: 23 specialist skills (CEO, Designer, Eng Manager, etc.) - 66K+ ⭐"
echo "   • Everything Claude Code: 28 agents + 119 skills + 60 commands - 100K+ ⭐"
echo "   • Agency Agents: 112 specialized AI personas - 2K+ ⭐"
echo ""
echo "   Total GitHub Stars: 168K+"
echo "   Total Skills/Agents: 250+"
echo ""
echo "🔗 Next steps:"
echo "   1. Review skills in $SKILLS_DIR/"
echo "   2. Import selected skills to SkillHub database"
echo "   3. Test skills with Claude Code"
