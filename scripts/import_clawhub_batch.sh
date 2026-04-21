#!/bin/bash
set -e

# Import skills from ClawHub.ai to AitHub with multiple search queries
# Usage: ./import_clawhub_batch.sh

AITHUB_URL=${DOMAIN:-http://localhost:8080}
ADMIN_TOKEN=${ADMIN_TOKEN}

if [ -z "$ADMIN_TOKEN" ]; then
    echo "Error: ADMIN_TOKEN not set"
    exit 1
fi

# Check dependencies
for cmd in jq unzip curl; do
    if ! command -v $cmd &> /dev/null; then
        echo "Error: $cmd is required but not installed"
        exit 1
    fi
done

echo "=== ClawHub Batch Importer ==="
echo "Target: $AITHUB_URL"
echo ""

# Create temp directory
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

# Different search queries to get diverse skills
QUERIES=(
    "python"
    "javascript"
    "react"
    "api"
    "database"
    "testing"
    "security"
    "devops"
    "docker"
    "kubernetes"
    "git"
    "web"
    "mobile"
    "ai"
    "machine learning"
    "data"
    "frontend"
    "backend"
    "automation"
    "monitoring"
)

TOTAL_IMPORTED=0
TOTAL_FAILED=0
TOTAL_SKIPPED=0
TOTAL_PROCESSED=0

for QUERY in "${QUERIES[@]}"; do
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Searching for: $QUERY"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # Fetch skills from ClawHub
    SKILLS=$(curl -s "https://clawhub.ai/api/v1/search?q=$QUERY&limit=20")

    SKILL_COUNT=$(echo "$SKILLS" | jq '.results | length')
    if [ "$SKILL_COUNT" -eq 0 ]; then
        echo "No skills found for query: $QUERY"
        continue
    fi

    echo "Found $SKILL_COUNT skills"

    # Parse and import each skill
    echo "$SKILLS" | jq -r '.results[] | @json' | while read -r skill; do
        SLUG=$(echo "$skill" | jq -r '.slug')
        NAME=$(echo "$skill" | jq -r '.displayName')
        SUMMARY=$(echo "$skill" | jq -r '.summary')

        TOTAL_PROCESSED=$((TOTAL_PROCESSED + 1))
        echo "[$TOTAL_PROCESSED] $NAME ($SLUG)"

        # Download skill package
        ZIP_FILE="$TMPDIR/$SLUG.zip"
        HTTP_CODE=$(curl -s -w "%{http_code}" -o "$ZIP_FILE" "https://clawhub.ai/api/v1/download?slug=$SLUG")

        if [ "$HTTP_CODE" != "200" ]; then
            echo "  ✗ Failed to download (HTTP $HTTP_CODE)"
            TOTAL_FAILED=$((TOTAL_FAILED + 1))
            continue
        fi

        # Extract SKILL.md
        SKILL_MD="$TMPDIR/$SLUG-SKILL.md"
        if ! unzip -p "$ZIP_FILE" SKILL.md > "$SKILL_MD" 2>/dev/null; then
            echo "  ✗ Failed to extract SKILL.md"
            TOTAL_FAILED=$((TOTAL_FAILED + 1))
            continue
        fi

        # Read SKILL.md content
        CONTENT=$(cat "$SKILL_MD")

        # Add required fields if missing
        if ! echo "$CONTENT" | grep -q "^version:"; then
            CONTENT=$(echo "$CONTENT" | awk '
                /^---$/ { print; in_fm=1; next }
                in_fm && /^description:/ { print; print "version: 1.0.0"; print "framework: openclaw"; next }
                in_fm && /^---$/ { in_fm=0 }
                { print }
            ')
        fi

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
        RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$AITHUB_URL/v1/skills" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $ADMIN_TOKEN" \
            -d @- <<EOF
{
    "namespace": "clawhub",
    "name": "$CLEAN_NAME",
    "description": "$SUMMARY",
    "content": $(echo "$CONTENT" | jq -Rs .),
    "tags": "openclaw,$QUERY",
    "framework": "openclaw"
}
EOF
)

        HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
        BODY=$(echo "$RESPONSE" | head -n-1)

        if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
            echo "  ✓ Imported successfully"
            TOTAL_IMPORTED=$((TOTAL_IMPORTED + 1))
        elif echo "$BODY" | grep -q "already exists"; then
            echo "  ⊘ Already exists, skipping"
            TOTAL_SKIPPED=$((TOTAL_SKIPPED + 1))
        else
            echo "  ✗ Failed (HTTP $HTTP_CODE): $(echo "$BODY" | jq -r '.error // .' 2>/dev/null || echo "$BODY")"
            TOTAL_FAILED=$((TOTAL_FAILED + 1))
        fi

        # Rate limit
        sleep 0.35
    done

    echo ""
done

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "=== Batch Import Complete ==="
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Imported: $TOTAL_IMPORTED"
echo "Skipped: $TOTAL_SKIPPED"
echo "Failed: $TOTAL_FAILED"
echo "Total Processed: $TOTAL_PROCESSED"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
