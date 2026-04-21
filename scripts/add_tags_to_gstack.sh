#!/usr/bin/env bash
# Add tags to gstack SKILL.md files based on skill name and description

set -euo pipefail

SKILLS_DIR="/opt/skillhub/skills/preloaded/gstack"

echo "🏷️  Adding tags to all SKILL.md files..."
echo ""

skill_files=$(find "$SKILLS_DIR" -name "SKILL.md" -type f)
total=$(echo "$skill_files" | wc -l)
count=0

for skill_file in $skill_files; do
    count=$((count + 1))
    skill_name=$(basename $(dirname "$skill_file"))
    
    echo "[$count/$total] Processing: $skill_name"
    
    # Check if tags field already exists
    if grep -q "^tags:" "$skill_file"; then
        echo "   ⊙ Already has tags field"
        continue
    fi
    
    # Generate tags based on skill name
    # Convert kebab-case to individual words
    tags=$(echo "$skill_name" | tr '-' '\n' | grep -v "^$" | head -3 | paste -sd ',' - | sed 's/,/, /g')
    
    # Add gstack as a tag
    tags="gstack, $tags"
    
    # Add tags field after framework field
    sed -i "/^framework:/a tags: [$tags]" "$skill_file"
    
    echo "   ✓ Added tags: [$tags]"
done

echo ""
echo "✅ Complete! Added tags to $count SKILL.md files"
