#!/usr/bin/env bash
# Import gstack skills directly into database

set -euo pipefail

SKILLS_DIR="/opt/skillhub/skills/preloaded/gstack"

echo "🚀 Importing gstack skills directly into database..."
echo ""

# Get namespace ID
NAMESPACE_ID=$(sudo -u postgres psql -d skillhub -t -c "SELECT id FROM namespaces WHERE name = 'gstack';")
NAMESPACE_ID=$(echo $NAMESPACE_ID | tr -d ' ')

if [ -z "$NAMESPACE_ID" ]; then
    echo "Error: gstack namespace not found"
    exit 1
fi

echo "Namespace ID: $NAMESPACE_ID"
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
    
    # Read skill content and escape for SQL
    skill_content=$(cat "$skill_file" | sed "s/'/''/g")
    
    # Insert into database
    result=$(sudo -u postgres psql -d skillhub -t << SQL
INSERT INTO skills (
    id,
    namespace_id,
    name,
    content,
    status,
    rating,
    install_count,
    success_count,
    failure_count,
    created_at
) VALUES (
    gen_random_uuid(),
    '$NAMESPACE_ID',
    '$skill_name',
    '$skill_content',
    'approved',
    0.0,
    0,
    0,
    0,
    NOW()
)
ON CONFLICT (namespace_id, name) DO UPDATE
SET content = EXCLUDED.content
RETURNING id;
SQL
)
    
    if [ $? -eq 0 ]; then
        echo "   ✓ Success"
        success=$((success + 1))
    else
        echo "   ✗ Failed"
        failed=$((failed + 1))
    fi
done

echo ""
echo "✅ Import complete!"
echo "   Total: $total"
echo "   Success: $success"
echo "   Failed: $failed"
