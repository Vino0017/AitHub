#!/usr/bin/env bash
# Preload popular Claude Code skill collections into SkillHub
# This script fetches and imports skills from the most popular repositories

set -euo pipefail

SKILLS_DIR="skills/preloaded"
mkdir -p "$SKILLS_DIR"

echo "🚀 Preloading popular Claude Code skills..."
echo ""

# 1. gstack by Garry Tan (66K+ stars)
echo "📦 [1/5] Fetching gstack by Garry Tan..."
if [ ! -d "$SKILLS_DIR/gstack" ]; then
    git clone --depth 1 https://github.com/garrytan/gstack.git "$SKILLS_DIR/gstack"
    echo "   ✓ gstack cloned (23 specialist skills)"
else
    echo "   ✓ gstack already exists"
fi

# 2. awesome-claude-skills by travisvn (curated collection)
echo "📦 [2/5] Fetching awesome-claude-skills..."
if [ ! -d "$SKILLS_DIR/awesome-claude-skills" ]; then
    git clone --depth 1 https://github.com/travisvn/awesome-claude-skills.git "$SKILLS_DIR/awesome-claude-skills"
    echo "   ✓ awesome-claude-skills cloned (curated collection)"
else
    echo "   ✓ awesome-claude-skills already exists"
fi

# 3. claude-code-skills by levnikolaevich (production-ready)
echo "📦 [3/5] Fetching claude-code-skills..."
if [ ! -d "$SKILLS_DIR/claude-code-skills" ]; then
    git clone --depth 1 https://github.com/levnikolaevich/claude-code-skills.git "$SKILLS_DIR/claude-code-skills"
    echo "   ✓ claude-code-skills cloned (full delivery workflow)"
else
    echo "   ✓ claude-code-skills already exists"
fi

# 4. claude-skills by Jeffallan (65 specialized skills)
echo "📦 [4/5] Fetching claude-skills by Jeffallan..."
if [ ! -d "$SKILLS_DIR/jeffallan-claude-skills" ]; then
    git clone --depth 1 https://github.com/Jeffallan/claude-skills.git "$SKILLS_DIR/jeffallan-claude-skills"
    echo "   ✓ claude-skills cloned (65 full-stack skills)"
else
    echo "   ✓ claude-skills already exists"
fi

# 5. claude-code-plugins-plus-skills by jeremylongshore (1367 skills)
echo "📦 [5/5] Fetching claude-code-plugins-plus-skills..."
if [ ! -d "$SKILLS_DIR/claude-code-plugins-plus-skills" ]; then
    git clone --depth 1 https://github.com/jeremylongshore/claude-code-plugins-plus-skills.git "$SKILLS_DIR/claude-code-plugins-plus-skills"
    echo "   ✓ claude-code-plugins-plus-skills cloned (1367 skills)"
else
    echo "   ✓ claude-code-plugins-plus-skills already exists"
fi

echo ""
echo "✅ All popular skill collections preloaded!"
echo ""
echo "📊 Summary:"
echo "   • gstack: 23 specialist skills (CEO, Designer, Eng Manager, etc.)"
echo "   • awesome-claude-skills: Curated collection with resources"
echo "   • claude-code-skills: Production-ready delivery workflow"
echo "   • claude-skills: 65 full-stack developer skills"
echo "   • claude-code-plugins-plus-skills: 1367 agent skills + 340 plugins"
echo ""
echo "🔗 Next steps:"
echo "   1. Review skills in $SKILLS_DIR/"
echo "   2. Import selected skills to SkillHub database"
echo "   3. Test skills with Claude Code"
