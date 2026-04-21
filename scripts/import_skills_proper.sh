#!/usr/bin/env bash
# Import skills from preloaded collections into SkillHub database
# Uses proper schema: skills + revisions tables

set -euo pipefail

SKILLS_DIR="${1:-/opt/skillhub/skills/preloaded/gstack}"
NAMESPACE="${2:-gstack}"

echo "🚀 Importing skills from $SKILLS_DIR into namespace '$NAMESPACE'..."
echo ""

# Get or create namespace
NAMESPACE_ID=$(sudo -u postgres psql -d skillhub -t -A -c "
    INSERT INTO namespaces (id, name, created_at)
    VALUES (gen_random_uuid(), '$NAMESPACE', NOW())
    ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
    RETURNING id;
" | head -1)

if [ -z "$NAMESPACE_ID" ]; then
    echo "Error: Failed to get/create namespace"
    exit 1
fi

echo "Namespace: $NAMESPACE (ID: $NAMESPACE_ID)"
echo ""

# Find all SKILL.md files
skill_files=$(find "$SKILLS_DIR" -name "SKILL.md" -type f)
total=$(echo "$skill_files" | wc -l)
count=0
success=0
failed=0
skipped=0

for skill_file in $skill_files; do
    count=$((count + 1))
    skill_dir=$(dirname "$skill_file")
    skill_name=$(basename "$skill_dir")

    echo "[$count/$total] Processing: $skill_name"

    # Read skill content
    skill_content=$(cat "$skill_file")

    # Extract description from first line after ---
    description=$(echo "$skill_content" | grep -A 1 "^---$" | tail -1 | sed 's/^# //' | head -c 200)
    if [ -z "$description" ]; then
        description="$skill_name skill"
    fi

    # Escape for SQL
    skill_content_escaped=$(echo "$skill_content" | sed "s/'/''/g")
    description_escaped=$(echo "$description" | sed "s/'/''/g")

    # Check if skill already exists
    existing_skill_id=$(sudo -u postgres psql -d skillhub -t -c "
        SELECT id FROM skills
        WHERE namespace_id = '$NAMESPACE_ID' AND name = '$skill_name';
    " | tr -d ' ')

    if [ -n "$existing_skill_id" ]; then
        echo "   ⚠ Skill already exists (ID: $existing_skill_id), skipping"
        skipped=$((skipped + 1))
        continue
    fi

    # Insert skill and revision in transaction
    result=$(sudo -u postgres psql -d skillhub -t -A << SQL
WITH new_skill AS (
    INSERT INTO skills (
        id,
        namespace_id,
        name,
        description,
        tags,
        framework,
        visibility,
        install_count,
        avg_rating,
        rating_count,
        outcome_success_rate,
        latest_version,
        fork_count,
        status,
        created_at,
        updated_at
    ) VALUES (
        gen_random_uuid(),
        '$NAMESPACE_ID',
        '$skill_name',
        '$description_escaped',
        '{}',
        '$NAMESPACE',
        'public',
        0,
        0,
        0,
        0,
        '1.0.0',
        0,
        'active',
        NOW(),
        NOW()
    )
    RETURNING id
),
new_revision AS (
    INSERT INTO revisions (
        id,
        skill_id,
        version,
        content,
        change_summary,
        review_status,
        schema_type,
        triggers,
        compatible_models,
        estimated_tokens,
        created_at,
        breaking_change
    )
    SELECT
        gen_random_uuid(),
        id,
        '1.0.0',
        '$skill_content_escaped',
        'Initial import from $NAMESPACE collection',
        'approved',
        'skill-md',
        '{}',
        '{}',
        0,
        NOW(),
        false
    FROM new_skill
    RETURNING skill_id
)
SELECT id FROM new_skill;
SQL
)

    if [ $? -eq 0 ] && [ -n "$result" ]; then
        skill_id=$(echo "$result" | head -1)
        echo "   ✓ Success (ID: $skill_id)"
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
echo "   Skipped: $skipped"
echo "   Failed: $failed"
