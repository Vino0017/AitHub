#!/usr/bin/env bash
# Import gstack skills into SkillHub database

set -euo pipefail

API_URL="http://localhost:8080"
TOKEN="${TOKEN:-sk_43d4f01d688c9d753af9f7ebcc82d4027a7301ef8bf03e6ab9a9a60854b081a7}"
SKILLS_DIR="/opt/skillhub/skills/preloaded/gstack"

echo "🚀 Importing gstack skills into SkillHub..."
echo ""

# Find all SKILL.md files
skill_files=$(find "$SKILLS_DIR" -name "SKILL.md" -type f)
total=$(echo "$skill_files" | wc -l)
count=0
success=0
failed=0

for skill_file in $skill_files; do
    count=$((count + 1))
    skill_dir=$(dirname "$skill_file")
    skill_name=$(basename "$skill_dir")
    
    echo "[$count/$total] Importing: $skill_name"
    
    # Create temp file for JSON payload
    tmpfile=$(mktemp)
    
    # Build JSON using jq with file input
    jq -n \
        --arg name "$skill_name" \
        --arg namespace "gstack" \
        --rawfile content "$skill_file" \
        '{
            namespace: $namespace,
            name: $name,
            content: $content
        }' > "$tmpfile"
    
    # Submit to API
    response=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/v1/skills" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d @"$tmpfile")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    rm -f "$tmpfile"
    
    if [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
        echo "   ✓ Success"
        success=$((success + 1))
    else
        echo "   ✗ Failed (HTTP $http_code)"
        failed=$((failed + 1))
    fi
    
    # Rate limiting
    sleep 0.1
done

echo ""
echo "✅ Import complete!"
echo "   Total: $total"
echo "   Success: $success"
echo "   Failed: $failed"
