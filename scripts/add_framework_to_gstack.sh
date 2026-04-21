#!/usr/bin/env bash
# Add framework: gstack to all gstack SKILL.md files

set -euo pipefail

SKILLS_DIR="/opt/skillhub/skills/preloaded/gstack"

echo "🔧 Adding framework: gstack to all SKILL.md files..."
echo ""

skill_files=$(find "$SKILLS_DIR" -name "SKILL.md" -type f)
total=$(echo "$skill_files" | wc -l)
count=0

for skill_file in $skill_files; do
    count=$((count + 1))
    skill_name=$(basename $(dirname "$skill_file"))
    
    echo "[$count/$total] Processing: $skill_name"
    
    # Check if framework field already exists
    if grep -q "^framework:" "$skill_file"; then
        echo "   ⊙ Already has framework field"
        continue
    fi
    
    # Add framework: gstack after version field
    sed -i '/^version:/a framework: gstack' "$skill_file"
    
    echo "   ✓ Added framework: gstack"
done

echo ""
echo "✅ Complete! Added framework field to $count SKILL.md files"
