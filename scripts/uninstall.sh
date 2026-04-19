#!/usr/bin/env bash
# SkillHub Uninstaller
# Usage: bash <(curl -fsSL https://your-domain.com/uninstall)

set -euo pipefail

echo "╔══════════════════════════════════════╗"
echo "║       SkillHub Uninstaller           ║"
echo "╚══════════════════════════════════════╝"
echo ""

# Remove skills from all detected frameworks
declare -A FRAMEWORK_DIRS
FRAMEWORK_DIRS=(
  ["gstack"]="$HOME/.gstack/skills/skillhub"
  ["openclaw"]="$HOME/.openclaw/skills/skillhub"
  ["hermes"]="$HOME/.hermes/skills/skillhub"
  ["claude-code"]="$HOME/.claude/skills/skillhub"
  ["cursor"]="$HOME/.cursor/skills/skillhub"
  ["windsurf"]="$HOME/.windsurf/skills/skillhub"
)

for fw in "${!FRAMEWORK_DIRS[@]}"; do
  dir="${FRAMEWORK_DIRS[$fw]}"
  if [ -d "$dir" ]; then
    rm -rf "$dir"
    echo "  ✓ Removed $fw Discovery Skill"
  fi
done

# Remove environment variables from shell config
for rc in "$HOME/.zshrc" "$HOME/.bashrc"; do
  if [ -f "$rc" ]; then
    sed -i '/SKILLHUB_TOKEN/d' "$rc" 2>/dev/null || true
    sed -i '/SKILLHUB_AUTO_CONTRIBUTE/d' "$rc" 2>/dev/null || true
    sed -i '/SKILLHUB_CONTRIBUTE/d' "$rc" 2>/dev/null || true
    sed -i '/SKILLHUB_DEFAULT_VISIBILITY/d' "$rc" 2>/dev/null || true
    echo "  ✓ Cleaned $rc"
  fi
done

unset SKILLHUB_TOKEN

echo ""
echo "✓ SkillHub uninstalled. Your namespace and uploaded skills remain on the server."
echo "  To delete your account, contact support or use the API."
