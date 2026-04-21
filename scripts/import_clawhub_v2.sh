#!/bin/bash
set -e

# Import skills from ClawHub.ai to AitHub
# Usage: ./import_clawhub_v2.sh [limit]

LIMIT=${1:-20}
AITHUB_URL=${DOMAIN:-http://localhost:8080}
ADMIN_TOKEN=${ADMIN_TOKEN}

if [ -z "$ADMIN_TOKEN" ]; then
    echo "Error: ADMIN_TOKEN not set"
    exit 1
fi

# Check dependencies
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed"
    exit 1
fi

if ! command -v unzip &> /dev/null; then
    echo "Error: unzip is required but not installed"
    exit 1
fi

echo "=== ClawHub → AitHub Importer v2 ==="
echo "Target: $AITHUB_URL"
echo "Fetching top $LIMIT skills from ClawHub..."

# Create temp directory
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

# Fetch skills from ClawHub using search
# Use a broad query to get popular skills
SKILLS=$(curl -s "https://clawhub.ai/api/v1/search?q=code&limit=$LIMIT")

# Check if we got results
SKILL_COUNT=$(echo "$SKILLS" | jq '.results | length')
if [ "$SKILL_COUNT" -eq 0 ]; then
    echo "No skills found or API error"
    exit 1
fi

echo "Found $SKILL_COUNT skills"
echo ""

IMPORTED=0
FAILED=0
SKIPPED=0

# Parse and import each skill
echo "$SKILLS" | jq -r '.results[] | @json' | while read -r skill; do
    SLUG=$(echo "$skill" | jq -r '.slug')
    NAME=$(echo "$skill" | jq -r '.displayName')
    SUMMARY=$(echo "$skill" | jq -r '.summary')

    echo "[$((IMPORTED + FAILED + SKIPPED + 1))/$SKILL_COUNT] $NAME ($SLUG)"

    # Download skill package
    ZIP_FILE="$TMPDIR/$SLUG.zip"
    HTTP_CODE=$(curl -s -w "%{http_code}" -o "$ZIP_FILE" "https://clawhub.ai/api/v1/download?slug=$SLUG")

    if [ "$HTTP_CODE" != "200" ]; then
        echo "  ✗ Failed to download (HTTP $HTTP_CODE)"
        FAILED=$((FAILED + 1))
        continue
    fi

    # Extract SKILL.md
    SKILL_MD="$TMPDIR/$SLUG-SKILL.md"
    if ! unzip -p "$ZIP_FILE" SKILL.md > "$SKILL_MD" 2>/dev/null; then
        echo "  ✗ Failed to extract SKILL.md"
        FAILED=$((FAILED + 1))
        continue
    fi

    # Read SKILL.md content
    CONTENT=$(cat "$SKILL_MD")

    # Extract metadata from SKILL.md frontmatter
    TAGS=$(echo "$CONTENT" | grep -A 20 "^---$" | grep "^tags:" | sed 's/tags: *//' | tr -d '[]"' || echo "")

    # Add required fields if missing (ClawHub skills don't have version/framework)
    # Check if frontmatter has version field
    if ! echo "$CONTENT" | grep -q "^version:"; then
        # Insert version and framework after description line
        CONTENT=$(echo "$CONTENT" | awk '
            /^---$/ { print; in_fm=1; next }
            in_fm && /^description:/ { print; print "version: 1.0.0"; print "framework: openclaw"; next }
            in_fm && /^---$/ { in_fm=0 }
            { print }
        ')
    fi

    # Ensure tags field exists
    if ! echo "$CONTENT" | grep -q "^tags:"; then
        CONTENT=$(echo "$CONTENT" | awk '
            /^---$/ { print; in_fm=1; next }
            in_fm && /^framework:/ { print; print "tags: [openclaw, imported]"; next }
            in_fm && /^---$/ { in_fm=0 }
            { print }
        ')
    fi

    # Clean name for AitHub
    CLEAN_NAME=$(echo "$SLUG" | tr '[:upper:]' '[:lower:]' | tr ' ' '-' | sed 's/[^a-z0-9-]/-/g')

    # Submit to AitHub
    # Note: ClawHub skills are designed for OpenClaw framework
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$AITHUB_URL/v1/skills" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -d @- <<EOF
{
    "namespace": "clawhub",
    "name": "$CLEAN_NAME",
    "description": "$SUMMARY",
    "content": $(echo "$CONTENT" | jq -Rs .),
    "tags": "openclaw,$TAGS",
    "framework": "openclaw"
}
EOF
)

    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | head -n-1)

    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
        echo "  ✓ Imported successfully"
        IMPORTED=$((IMPORTED + 1))
    elif echo "$BODY" | grep -q "already exists"; then
        echo "  ⊘ Already exists, skipping"
        SKIPPED=$((SKIPPED + 1))
    else
        echo "  ✗ Failed (HTTP $HTTP_CODE): $(echo "$BODY" | jq -r '.error // .' 2>/dev/null || echo "$BODY")"
        FAILED=$((FAILED + 1))
    fi

    # Rate limit: ~3 req/sec
    sleep 0.35
done

echo ""
echo "=== Import Complete ==="
echo "Imported: $IMPORTED"
echo "Skipped: $SKIPPED"
echo "Failed: $FAILED"
echo "Total: $SKILL_COUNT"
