#!/bin/bash
set -e

# Import skills from ClawHub.ai to AitHub
# Usage: ./import_clawhub.sh [limit]

LIMIT=${1:-50}
AITHUB_URL=${DOMAIN:-http://localhost:8080}
ADMIN_TOKEN=${ADMIN_TOKEN}

if [ -z "$ADMIN_TOKEN" ]; then
    echo "Error: ADMIN_TOKEN not set"
    exit 1
fi

echo "=== ClawHub → AitHub Importer ==="
echo "Fetching top $LIMIT skills from ClawHub..."

# Fetch skills from ClawHub using search (empty query returns all)
SKILLS=$(curl -s "https://clawhub.ai/api/v1/search?q=&limit=$LIMIT")

# Check if we got results
if [ -z "$SKILLS" ] || [ "$SKILLS" = '{"results":[]}' ]; then
    echo "No skills found or API error"
    exit 1
fi

# Parse and import each skill
echo "$SKILLS" | jq -r '.results[] | @json' | while read -r skill; do
    SLUG=$(echo "$skill" | jq -r '.slug')
    NAME=$(echo "$skill" | jq -r '.displayName')
    SUMMARY=$(echo "$skill" | jq -r '.summary')

    echo ""
    echo "Processing: $NAME ($SLUG)"

    # Fetch skill details
    DETAIL=$(curl -s "https://clawhub.ai/api/v1/skills/$SLUG")

    # Extract content
    CONTENT=$(echo "$DETAIL" | jq -r '.content // ""')
    TAGS=$(echo "$DETAIL" | jq -r '.tags[]? // ""' | tr '\n' ',' | sed 's/,$//')

    # Build SKILL.md if content is empty
    if [ -z "$CONTENT" ] || [ "$CONTENT" = "null" ]; then
        CONTENT="---
name: $NAME
description: $SUMMARY
tags: $TAGS
framework: openclaw
---

# $NAME

$SUMMARY

Imported from ClawHub.ai (slug: $SLUG)
"
    fi

    # Clean name for AitHub
    CLEAN_NAME=$(echo "$SLUG" | tr '[:upper:]' '[:lower:]' | tr ' ' '-')

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
    "tags": "$TAGS",
    "framework": "openclaw"
}
EOF
)

    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | head -n-1)

    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
        echo "  ✓ Imported successfully"
    else
        echo "  ✗ Failed (HTTP $HTTP_CODE): $BODY"
    fi

    # Rate limit: ~3 req/sec
    sleep 0.35
done

echo ""
echo "=== Import Complete ==="
